package picomatch

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os/exec"
	"reflect"
	"testing"
)

type jsToken struct {
	Value       string `json:"value"`
	Depth       int    `json:"depth"`
	IsGlob      bool   `json:"isGlob"`
	IsPrefix    bool   `json:"isPrefix"`
	IsBrace     bool   `json:"isBrace"`
	IsBracket   bool   `json:"isBracket"`
	IsExtglob   bool   `json:"isExtglob"`
	IsGlobstar  bool   `json:"isGlobstar"`
	Negated     bool   `json:"negated"`
	Backslashes bool   `json:"backslashes"`
}

type jsScanResult struct {
	Prefix         string    `json:"prefix"`
	Input          string    `json:"input"`
	Start          int       `json:"start"`
	Base           string    `json:"base"`
	Glob           string    `json:"glob"`
	MaxDepth       int       `json:"maxDepth"`
	IsBrace        bool      `json:"isBrace"`
	IsBracket      bool      `json:"isBracket"`
	IsGlob         bool      `json:"isGlob"`
	IsExtglob      bool      `json:"isExtglob"`
	IsGlobstar     bool      `json:"isGlobstar"`
	Negated        bool      `json:"negated"`
	NegatedExtglob bool      `json:"negatedExtglob"`
	Slashes        []int     `json:"slashes"`
	Parts          []string  `json:"parts"`
	Tokens         []jsToken `json:"tokens"`
}

type jsResponse struct {
	ID      int          `json:"id"`
	Success bool         `json:"success"`
	Result  jsScanResult `json:"result"`
	Error   string       `json:"error"`
}

type jsRequest struct {
	ID      int             `json:"id"`
	Pattern string          `json:"pattern"`
	Options map[string]bool `json:"options"`
}

func TestDifferentialScanner(t *testing.T) {
	cmd := exec.Command("node", "testdata/js_scanner.js")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("Failed to open stdin pipe to Node: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to open stdout pipe to Node: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start Node differential subprocess: %v", err)
	}
	defer func() {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
	}()

	scanner := bufio.NewScanner(stdout)
	encoder := json.NewEncoder(stdin)

	// Corpus of diverse patterns covering all scanner branches, invariants, and edge cases
	corpus := []string{
		"*.js",
		"foo/bar/*.js",
		"./foo/bar/*.js",
		"!foo/bar/*.js",
		"!./foo/bar/*.js",
		"!(foo)*",
		"!(foo)",
		"test/!(foo)/*",
		"./foo/@(foo)/*.js",
		"./foo/@(bar)/**/*.js",
		"foo/{a,b,c}/*.js",
		"foo/\\{a,b}/*.js",
		"foo/[a-z]/*.js",
		"foo/[a-z",
		"./foo/**/*.js",
		"/*.js",
		"/foo/bar",
		"foo/bar/",
		"/foo",
		"foo",
		"",
		"foo/(bar)",
		"foo/\\(bar)",
		"foo/[a\\-z]/bar",
		"foo/bar\\/baz",
		"**",
		"a/b/c/**/d",
		"!**/node_modules/**",
		"foo/{1..5}/*.js",
		"foo/bar/*.{js,ts}",
		"@(a|b|c)/*.js",
		"+(foo|bar)/baz",
		"*(foo|bar)/baz",
		"?(foo|bar)/baz",
		"foo/bar/baz.js",
		"./",
		"!./",
		"/",
		"//",
		"foo//bar",
		"foo\\bar",
		"foo/bar/\\[baz\\]",
	}

	optionVariants := []struct {
		name string
		opts *ScanOptions
		js   map[string]bool
	}{
		{"Default", nil, map[string]bool{}},
		{"Parts", &ScanOptions{Parts: true}, map[string]bool{"parts": true}},
		{"Tokens", &ScanOptions{Tokens: true}, map[string]bool{"tokens": true}},
		{"ScanToEnd", &ScanOptions{ScanToEnd: true}, map[string]bool{"scanToEnd": true}},
		{"Unescape", &ScanOptions{Unescape: true}, map[string]bool{"unescape": true}},
		{"NoExt", &ScanOptions{NoExt: true}, map[string]bool{"noext": true}},
		{"NoNegate", &ScanOptions{NoNegate: true}, map[string]bool{"nonegate": true}},
		{"NoParen", &ScanOptions{NoParen: true}, map[string]bool{"noparen": true}},
		{"TokensAndUnescape", &ScanOptions{Tokens: true, Unescape: true}, map[string]bool{"tokens": true, "unescape": true}},
	}

	testID := 0
	for _, pattern := range corpus {
		for _, variant := range optionVariants {
			testID++
			t.Run(fmt.Sprintf("%d_%s_%s", testID, variant.name, pattern), func(t *testing.T) {
				req := jsRequest{
					ID:      testID,
					Pattern: pattern,
					Options: variant.js,
				}

				if err := encoder.Encode(req); err != nil {
					t.Fatalf("Failed to encode request to JS: %v", err)
				}

				if !scanner.Scan() {
					t.Fatalf("Failed to read response from JS subprocess: %v", scanner.Err())
				}

				var res jsResponse
				if err := json.Unmarshal(scanner.Bytes(), &res); err != nil {
					t.Fatalf("Failed to unmarshal JSON response from JS: %v", err)
				}

				if !res.Success {
					t.Fatalf("JS scanner returned error: %s", res.Error)
				}

				goRes := Scan(pattern, variant.opts)
				jsRes := res.Result

				compareResults(t, pattern, variant.name, goRes, &jsRes)
			})
		}
	}
}

func compareResults(t *testing.T, pattern, variant string, goRes *ScanState, jsRes *jsScanResult) {
	t.Helper()

	if goRes.Prefix != jsRes.Prefix || goRes.Input != jsRes.Input || goRes.Start != jsRes.Start ||
		goRes.Base != jsRes.Base || goRes.Glob != jsRes.Glob || goRes.IsBrace != jsRes.IsBrace ||
		goRes.IsBracket != jsRes.IsBracket || goRes.IsGlob != jsRes.IsGlob || goRes.IsExtglob != jsRes.IsExtglob ||
		goRes.IsGlobstar != jsRes.IsGlobstar || goRes.Negated != jsRes.Negated || goRes.NegatedExtglob != jsRes.NegatedExtglob {
		t.Errorf("Mismatch in scalar fields for pattern %q (opts %s):\nGo: %+v\nJS: %+v", pattern, variant, goRes, jsRes)
	}

	expectedMaxDepth := goRes.MaxDepth
	if goRes.MaxInfinity {
		expectedMaxDepth = -1
	}
	if expectedMaxDepth != jsRes.MaxDepth {
		t.Errorf("MaxDepth mismatch for pattern %q (opts %s): Go=%d (inf=%v), JS=%d", pattern, variant, goRes.MaxDepth, goRes.MaxInfinity, jsRes.MaxDepth)
	}

	if !(len(goRes.Slashes) == 0 && len(jsRes.Slashes) == 0) && !reflect.DeepEqual(goRes.Slashes, jsRes.Slashes) {
		t.Errorf("Slashes mismatch for pattern %q (opts %s):\nGo: %v\nJS: %v", pattern, variant, goRes.Slashes, jsRes.Slashes)
	}
	if !(len(goRes.Parts) == 0 && len(jsRes.Parts) == 0) && !reflect.DeepEqual(goRes.Parts, jsRes.Parts) {
		t.Errorf("Parts mismatch for pattern %q (opts %s):\nGo: %v\nJS: %v", pattern, variant, goRes.Parts, jsRes.Parts)
	}

	if len(goRes.Tokens) != len(jsRes.Tokens) {
		t.Fatalf("Tokens length mismatch for pattern %q (opts %s): Go=%d, JS=%d", pattern, variant, len(goRes.Tokens), len(jsRes.Tokens))
	}

	for i, gt := range goRes.Tokens {
		jt := jsRes.Tokens[i]
		expectedDepth := gt.Depth
		if gt.Infinity {
			expectedDepth = -1
		}
		if gt.Value != jt.Value || expectedDepth != jt.Depth || gt.IsGlob != jt.IsGlob ||
			gt.IsPrefix != jt.IsPrefix || gt.IsBrace != jt.IsBrace || gt.IsBracket != jt.IsBracket ||
			gt.IsExtglob != jt.IsExtglob || gt.IsGlobstar != jt.IsGlobstar || gt.Negated != jt.Negated ||
			gt.Backslashes != jt.Backslashes {
			t.Errorf("Token[%d] mismatch for pattern %q (opts %s):\nGo: %+v\nJS: %+v", i, pattern, variant, gt, jt)
		}
	}
}
