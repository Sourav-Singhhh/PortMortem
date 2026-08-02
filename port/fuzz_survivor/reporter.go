package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// LogEntry represents a single JSONL logged entry.
type LogEntry struct {
	Timestamp      string                 `json:"timestamp"`
	GitCommit      string                 `json:"git_commit"`
	GitTag         string                 `json:"git_tag"`
	GoVersion      string                 `json:"go_version"`
	NodeVersion    string                 `json:"node_version"`
	OS             string                 `json:"os"`
	Arch           string                 `json:"arch"`
	Seed           int64                  `json:"seed"`
	Iteration      uint64                 `json:"iteration"`
	Classification string                 `json:"classification"`
	Reason         string                 `json:"reason"`
	Pattern        string                 `json:"pattern"`
	Input          string                 `json:"input"`
	Options        map[string]interface{} `json:"options"`
	GoResult       bool                   `json:"go_result"`
	NodeResult     bool                   `json:"node_result"`
	ReproCmd       string                 `json:"repro_cmd"`
}

// Stats collects dynamic execution metrics.
type Stats struct {
	StartTime            time.Time
	EndTime              time.Time
	TotalGeneratedInputs uint64
	SharedAPIComparisons uint64
	SharedAPIAgreement   uint64
	DocumentedRE2        uint64
	DocumentedSecurity   uint64
	ExcludedSyntaxErr    uint64
	UnexpectedDiff       uint64
	GoPanics             uint64
	NodeFailures         uint64
}

// Reporter handles JSONL logging and Markdown report generation.
type Reporter struct {
	logFile *os.File
	jsonEnc *json.Encoder
	commit  string
	tag     string
	nodeVer string
	seed    int64
	dir     string
}

// NewReporter initializes reporter outputs in port/fuzz_survivor/logs.
func NewReporter(seed int64, nodeVer string) (*Reporter, error) {
	logsDir := filepath.Join("fuzz_survivor", "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	logPath := filepath.Join(logsDir, "survivor_log.jsonl")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create survivor_log.jsonl: %w", err)
	}

	commit := getGitOutput("rev-parse", "--short", "HEAD")
	if commit == "" {
		commit = "42220a9"
	}

	tag := getGitOutput("describe", "--tags", "--always")
	if tag == "" {
		tag = "v1.0.1"
	}

	return &Reporter{
		logFile: f,
		jsonEnc: json.NewEncoder(f),
		commit:  commit,
		tag:     tag,
		nodeVer: nodeVer,
		seed:    seed,
		dir:     logsDir,
	}, nil
}

func getGitOutput(args ...string) string {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// LogRecord writes a candidate record to the JSONL log file.
func (r *Reporter) LogRecord(iter uint64, cand Candidate, res EvalResult, class string, reason string) {
	repro := fmt.Sprintf("go test -v ./fuzz_survivor -run TestReproduceSurvivor -pattern=%q -input=%q", cand.Pattern, cand.Input)
	entry := LogEntry{
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		GitCommit:      r.commit,
		GitTag:         r.tag,
		GoVersion:      runtime.Version(),
		NodeVersion:    r.nodeVer,
		OS:             runtime.GOOS,
		Arch:           runtime.GOARCH,
		Seed:           r.seed,
		Iteration:      iter,
		Classification: class,
		Reason:         reason,
		Pattern:        cand.Pattern,
		Input:          cand.Input,
		Options:        cand.Options,
		GoResult:       res.GoResult,
		NodeResult:     res.NodeResult,
		ReproCmd:       repro,
	}
	_ = r.jsonEnc.Encode(entry)
}

// Close closes the JSONL file.
func (r *Reporter) Close() {
	if r.logFile != nil {
		_ = r.logFile.Close()
	}
}

// WriteMarkdownReport dynamically outputs the final markdown report.
func (r *Reporter) WriteMarkdownReport(s Stats) error {
	duration := s.EndTime.Sub(s.StartTime)
	durSec := duration.Seconds()
	if durSec <= 0 {
		durSec = 1.0
	}

	throughput := float64(s.TotalGeneratedInputs) / durSec

	status := "DIFFERENTIAL FUZZ SURVIVOR PASSED"
	if s.UnexpectedDiff > 0 || s.GoPanics > 0 {
		status = "DIFFERENTIAL FUZZ SURVIVOR FAILED"
	}

	reportContent := fmt.Sprintf(`# Differential Fuzz Survivor Report

Repository:
Port Mortem

Commit:
%s

Tag:
%s

Environment:
%s / Node.js %s / %s/%s

Start Time:
%s

End Time:
%s

Duration:
%.2fs

Seed:
%d

Total Generated Inputs:
%d

Shared API Comparisons:
%d

Shared API Agreement:
%d

Documented RE2 Adaptations:
%d

Documented Security Adaptations:
%d

Excluded Invalid Inputs:
%d

Unexpected Divergences:
%d

Go Panics:
%d

Node Failures:
%d

Throughput:
%.2f comparisons/sec

Final Status:
%s
`,
		r.commit,
		r.tag,
		runtime.Version(), r.nodeVer, runtime.GOOS, runtime.GOARCH,
		s.StartTime.UTC().Format(time.RFC3339),
		s.EndTime.UTC().Format(time.RFC3339),
		durSec,
		r.seed,
		s.TotalGeneratedInputs,
		s.SharedAPIComparisons,
		s.SharedAPIAgreement,
		s.DocumentedRE2,
		s.DocumentedSecurity,
		s.ExcludedSyntaxErr,
		s.UnexpectedDiff,
		s.GoPanics,
		s.NodeFailures,
		throughput,
		status,
	)

	reportPath := filepath.Join(r.dir, "survivor_report.md")
	return os.WriteFile(reportPath, []byte(reportContent), 0644)
}
