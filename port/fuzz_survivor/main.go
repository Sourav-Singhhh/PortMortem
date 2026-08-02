package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	picomatch "github.com/Sourav-Singhhh/PortMortem/port"
)

type jsMatcherRequest struct {
	ID      int                    `json:"id"`
	Pattern string                 `json:"pattern"`
	Input   string                 `json:"input"`
	Options map[string]interface{} `json:"options"`
}

type jsMatcherResponse struct {
	ID      int    `json:"id"`
	Success bool   `json:"success"`
	Result  bool   `json:"result"`
	Error   string `json:"error"`
}

func main() {
	durFlag := flag.Duration("duration", 60*time.Second, "Continuous fuzz execution duration")
	seedFlag := flag.Int64("seed", 1337, "Pseudorandom generator seed")
	flag.Parse()

	if err := RunSurvivor(*durFlag, *seedFlag); err != nil {
		fmt.Fprintf(os.Stderr, "Differential Fuzz Survivor Error: %v\n", err)
		os.Exit(1)
	}
}

// RunSurvivor executes the continuous differential fuzzing loop against Node.js daemon.
func RunSurvivor(duration time.Duration, seed int64) error {
	scriptPath := filepath.Join("testdata", "js_matcher.js")
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		scriptPath = filepath.Join("..", "testdata", "js_matcher.js")
	}

	absScript, err := filepath.Abs(scriptPath)
	if err != nil {
		absScript = scriptPath
	}

	cmd := exec.Command("node", absScript)
	cmd.Dir = filepath.Dir(absScript)
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to open stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to open stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Node daemon: %w", err)
	}
	defer func() {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
	}()

	scanner := bufio.NewScanner(stdout)
	encoder := json.NewEncoder(stdin)

	// Fetch Node version
	nodeVerOut, _ := exec.Command("node", "-v").Output()
	nodeVer := string(nodeVerOut)
	if len(nodeVer) > 0 && nodeVer[len(nodeVer)-1] == '\n' {
		nodeVer = nodeVer[:len(nodeVer)-1]
	}

	reporter, err := NewReporter(seed, nodeVer)
	if err != nil {
		return fmt.Errorf("failed to initialize reporter: %w", err)
	}
	defer reporter.Close()

	generator := NewGenerator(seed)
	classifier := NewClassifier()

	stats := Stats{
		StartTime: time.Now(),
	}

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	timer := time.NewTimer(duration)
	defer timer.Stop()

	fmt.Printf("Starting Differential Fuzz Survivor (Duration: %v, Seed: %d)...\n", duration, seed)

	reqID := 0
	running := true

	for running {
		select {
		case <-timer.C:
			running = false
		case <-stopChan:
			fmt.Println("\nReceived interrupt signal, stopping fuzz run...")
			running = false
		default:
			stats.TotalGeneratedInputs++
			cand := generator.Next()

			res := evaluateCandidate(cand, &reqID, encoder, scanner)
			if res.GoPanic {
				stats.GoPanics++
			}

			class, reason := classifier.Classify(cand, res)

			switch class {
			case ClassSharedAPIParity:
				stats.SharedAPIComparisons++
				stats.SharedAPIAgreement++
			case ClassDocumentedRE2:
				stats.SharedAPIComparisons++
				stats.DocumentedRE2++
			case ClassDocumentedSecurity:
				stats.SharedAPIComparisons++
				stats.DocumentedSecurity++
			case ClassExcludedSyntaxErr:
				stats.ExcludedSyntaxErr++
			case ClassUnexpectedDiff:
				stats.SharedAPIComparisons++
				stats.UnexpectedDiff++
				reporter.LogRecord(stats.TotalGeneratedInputs, cand, res, class, reason)
				fmt.Printf("[UNEXPECTED DIVERGENCE] Pattern: %q | Input: %q | Go: %v vs JS: %v\n", cand.Pattern, cand.Input, res.GoResult, res.NodeResult)
			case ClassGoPanic:
				stats.GoPanics++
				reporter.LogRecord(stats.TotalGeneratedInputs, cand, res, class, reason)
				fmt.Printf("[GO PANIC] Pattern: %q | Msg: %s\n", cand.Pattern, res.PanicMsg)
			case ClassNodeFailure:
				stats.NodeFailures++
			}
		}
	}

	stats.EndTime = time.Now()

	if err := reporter.WriteMarkdownReport(stats); err != nil {
		return fmt.Errorf("failed to write Markdown report: %w", err)
	}

	fmt.Printf("\nDifferential Fuzz Survivor Completed in %.2fs!\n", stats.EndTime.Sub(stats.StartTime).Seconds())
	fmt.Printf("Total Generated: %d | Shared API Agreement: %d | RE2 Exclusions: %d | Security Exclusions: %d | Unexpected Divergences: %d | Go Panics: %d\n",
		stats.TotalGeneratedInputs, stats.SharedAPIAgreement, stats.DocumentedRE2, stats.DocumentedSecurity, stats.UnexpectedDiff, stats.GoPanics)

	if stats.UnexpectedDiff > 0 || stats.GoPanics > 0 {
		return fmt.Errorf("fuzz run failed with %d unexpected divergences and %d panics", stats.UnexpectedDiff, stats.GoPanics)
	}

	return nil
}

func evaluateCandidate(cand Candidate, reqID *int, encoder *json.Encoder, scanner *bufio.Scanner) (res EvalResult) {
	defer func() {
		if r := recover(); r != nil {
			res.GoPanic = true
			res.PanicMsg = fmt.Sprintf("panic: %v", r)
		}
	}()

	m, err := picomatch.Compile(cand.Pattern, nil)
	if err != nil {
		res.GoSuccess = false
		res.GoErr = err
	} else {
		res.GoSuccess = true
		res.GoResult = m.Match(cand.Input)
	}

	*reqID++
	req := jsMatcherRequest{
		ID:      *reqID,
		Pattern: cand.Pattern,
		Input:   cand.Input,
		Options: cand.Options,
	}

	if err := encoder.Encode(req); err != nil {
		res.NodeSuccess = false
		res.NodeErr = err.Error()
		return res
	}

	if !scanner.Scan() {
		res.NodeSuccess = false
		res.NodeErr = "EOF from Node daemon"
		return res
	}

	var resp jsMatcherResponse
	if err := json.Unmarshal([]byte(scanner.Text()), &resp); err != nil {
		res.NodeSuccess = false
		res.NodeErr = err.Error()
		return res
	}

	res.NodeSuccess = resp.Success
	res.NodeResult = resp.Result
	res.NodeErr = resp.Error

	return res
}
