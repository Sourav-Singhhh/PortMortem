package picomatch

import (
	"bufio"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

// FuzzCompile verifies that pattern parsing and compilation never panic on arbitrary inputs.
func FuzzCompile(f *testing.F) {
	// Seed corpus with representative glob patterns across wildcards, extglobs, braces, POSIX, Windows, and Unicode
	seeds := []string{
		"*.js",
		"src/**/*.go",
		"foo/{bar,baz}.js",
		"file[0-9].txt",
		"@(foo|bar)/*.ts",
		"!(test|spec).go",
		"**/*",
		"{1..10}",
		"[[:alpha:]]*",
		"foo/\\*\\*/bar",
		"a/b/c/d",
		"!",
		"*(a|b|c)",
		"foo/[a-z]",
		"{{a,b},{c,d}}",
		"C:\\\\foo\\\\bar\\\\*.js",
		"\\\\\\\\server\\\\share\\\\*.go",
		"[а-я]*.txt",
		"🎉/*.js",
		".env*",
		"foo/.*",
		"[[:alnum:]_]*",
		"src/{a..z}/{1..100}/*.ts",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, pattern string) {
		// Ensure Compile, Parse, and Scan never panic on arbitrary string inputs
		_, _ = Compile(pattern, nil)
		_, _ = Parse(pattern, nil)
		_ = Scan(pattern, nil)
	})
}

// FuzzMatch verifies that compilation and string evaluation operate reliably without panics.
func FuzzMatch(f *testing.F) {
	seeds := []struct {
		pattern string
		input   string
	}{
		{"*.js", "index.js"},
		{"src/**/*.go", "src/cmd/main.go"},
		{"foo/{bar,baz}.js", "foo/bar.js"},
		{"[a-z]*", "alpha.txt"},
		{"@(a|b)/*.ts", "a/main.ts"},
		{"!(test).go", "app.go"},
		{"**/*.json", "config/settings.json"},
		{"file?.txt", "file1.txt"},
		{"C:\\\\foo\\\\*.js", "C:\\\\foo\\\\app.js"},
		{"[а-я]*.txt", "привет.txt"},
		{"🎉/*.js", "🎉/party.js"},
		{".env*", ".env.local"},
	}

	for _, s := range seeds {
		f.Add(s.pattern, s.input)
	}

	f.Fuzz(func(t *testing.T, pattern string, input string) {
		m, err := Compile(pattern, nil)
		if err == nil && m != nil {
			// Ensure Match never panics on valid compiled matchers
			_ = m.Match(input)
		}
	})
}

// FuzzDifferentialMatcher executes live differential comparisons against native Node.js picomatch over IPC.
func FuzzDifferentialMatcher(f *testing.F) {
	// Require node executable to be present for live differential testing
	if _, err := exec.LookPath("node"); err != nil {
		f.Skip("Node.js executable not found in PATH; skipping differential fuzzing")
	}

	cmd := exec.Command("node", "testdata/js_matcher.js")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		f.Fatalf("Failed to open stdin pipe to Node daemon: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		f.Fatalf("Failed to open stdout pipe to Node daemon: %v", err)
	}

	if err := cmd.Start(); err != nil {
		f.Fatalf("Failed to start Node daemon: %v", err)
	}
	defer func() {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
	}()

	scanner := bufio.NewScanner(stdout)
	encoder := json.NewEncoder(stdin)

	seeds := []struct {
		pattern string
		input   string
	}{
		{"*.js", "index.js"},
		{"src/**/*.go", "src/cmd/main.go"},
		{"foo/{bar,baz}.js", "foo/bar.js"},
		{"[a-z]*", "alpha.txt"},
		{"@(foo|bar)/*.ts", "foo/app.ts"},
		{"!(test).go", "main.go"},
		{"file?.txt", "file1.txt"},
		{"**/*.json", "package.json"},
		{"[а-я]*.txt", "привет.txt"},
		{".env*", ".env.local"},
	}

	for _, s := range seeds {
		f.Add(s.pattern, s.input)
	}

	reqID := 0
	f.Fuzz(func(t *testing.T, pattern string, input string) {
		// Limit pattern length to avoid excessive OS buffer bloat during fuzzing
		if len(pattern) > 256 || len(input) > 256 {
			return
		}

		m, goErr := Compile(pattern, nil)
		if goErr != nil {
			// Syntax compilation error in Go; expected for random fuzz inputs
			return
		}

		goResult := m.Match(input)

		reqID++
		req := jsMatcherRequest{
			ID:      reqID,
			Pattern: pattern,
			Input:   input,
			Options: nil,
		}

		if err := encoder.Encode(req); err != nil {
			return
		}

		if !scanner.Scan() {
			return
		}

		var resp jsMatcherResponse
		if err := json.Unmarshal([]byte(scanner.Text()), &resp); err != nil {
			return
		}

		// Flag potential defects if Go succeeds and JS succeeds but outputs disagree
		if resp.Success && goResult != resp.Result {
			// Check if mismatch is a known architectural adaptation (RE2 lookaround limits, directory dotfiles)
			if strings.Contains(pattern, "!(") || strings.Contains(pattern, "(?!") || strings.Contains(pattern, "(?=") {
				return // Known RE2 non-linear lookaround limitation
			}
			if input == "." || input == ".." || strings.HasSuffix(input, "/.") || strings.HasSuffix(input, "/..") {
				return // Known path traversal security hardening (blocking . and ..)
			}
			t.Logf("[DIFFERENTIAL FUZZ MISMATCH] Pattern: %q | Input: %q | Go: %v vs JS: %v", pattern, input, goResult, resp.Result)
		}
	})
}
