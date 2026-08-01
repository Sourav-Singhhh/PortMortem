package picomatch

import (
	"bufio"
	"encoding/json"
	"os/exec"
	"testing"
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

func TestMatcherAPI(t *testing.T) {
	t.Run("empty pattern returns error", func(t *testing.T) {
		_, err := Compile("", nil)
		if err == nil || err.Error() != "expected pattern to be a non-empty string" {
			t.Errorf("expected non-empty string error, got %v", err)
		}
	})

	t.Run("caching behavior", func(t *testing.T) {
		opts := &ParseOptions{Dot: true}
		m1, err := Compile("*.js", opts)
		if err != nil {
			t.Fatalf("Compile failure: %v", err)
		}
		m2, err := Compile("*.js", opts)
		if err != nil {
			t.Fatalf("Compile failure: %v", err)
		}
		if m1 != m2 {
			t.Errorf("expected Compile to return cached matcher instance")
		}
	})

	t.Run("basic wildcard matching", func(t *testing.T) {
		matched, err := Match("*.js", "index.js", nil)
		if err != nil || !matched {
			t.Errorf("expected *.js to match index.js, got %v (err: %v)", matched, err)
		}
		matched, err = Match("*.js", "index.ts", nil)
		if err != nil || matched {
			t.Errorf("expected *.js to not match index.ts, got %v (err: %v)", matched, err)
		}
	})

	t.Run("matchBase option", func(t *testing.T) {
		matched, err := Match("*.js", "foo/bar/baz.js", &ParseOptions{MatchBase: true})
		if err != nil || !matched {
			t.Errorf("expected MatchBase=true to match foo/bar/baz.js against *.js")
		}
	})

	t.Run("nocase option", func(t *testing.T) {
		matched, err := Match("*.js", "INDEX.JS", &ParseOptions{Nocase: true})
		if err != nil || !matched {
			t.Errorf("expected Nocase=true to match INDEX.JS against *.js")
		}
	})

	t.Run("ignore option", func(t *testing.T) {
		opts := &ParseOptions{Ignore: []string{"*.spec.js", "test/*.js"}}
		matched, err := Match("*.js", "app.spec.js", opts)
		if err != nil || matched {
			t.Errorf("expected app.spec.js to be ignored")
		}
		matched, err = Match("*.js", "app.js", opts)
		if err != nil || !matched {
			t.Errorf("expected app.js to match")
		}
	})

	t.Run("cache eviction and reset", func(t *testing.T) {
		cacheMu.Lock()
		oldMax := maxCacheSize
		maxCacheSize = 1
		cacheMu.Unlock()

		_, _ = Compile("a*.js", nil)
		_, _ = Compile("b*.js", nil) // triggers eviction when size >= maxCacheSize

		cacheMu.Lock()
		maxCacheSize = oldMax
		cacheMu.Unlock()
		clearCache()
	})

	t.Run("special directory filtering and idxEquals", func(t *testing.T) {
		matched, _ := Match("./*", "./foo", nil)
		if matched {
			t.Errorf("expected ./* to not match ./foo per picomatch default rules")
		}
		matched, _ = Match("a/../*", "a/../bar", nil)
		if !matched {
			t.Errorf("expected a/../* to match a/../bar")
		}
		matched, _ = Match("*", ".", nil)
		if matched {
			t.Errorf("expected * to reject special directory .")
		}
		matched, _ = Match("foo/*", "foo/..", nil)
		if matched {
			t.Errorf("expected foo/* to reject special directory ..")
		}
		matched, _ = Match("../*", "../bar", nil)
		if !matched {
			t.Errorf("expected ../* to match ../bar")
		}
	})

	t.Run("custom Format option and Contains option", func(t *testing.T) {
		opts := &ParseOptions{
			Format: func(s string) string {
				return "formatted/" + s
			},
		}
		matched, _ := Match("formatted/*.js", "app.js", opts)
		if !matched {
			t.Errorf("expected custom Format option to alter input path before matching")
		}

		containsOpts := &ParseOptions{Contains: true}
		matched, _ = Match("bar", "foo/bar/baz.js", containsOpts)
		if !matched {
			t.Errorf("expected Contains=true to match substring bar")
		}
	})

	t.Run("Match error propagation", func(t *testing.T) {
		_, err := Match("", "input", nil)
		if err == nil {
			t.Errorf("expected error when calling Match with empty pattern")
		}
	})
}

func TestDifferentialMatcher(t *testing.T) {
	cmd := exec.Command("node", "testdata/js_matcher.js")
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

	type testcase struct {
		Pattern string
		Input   string
		Opts    *ParseOptions
	}

	corpus := []testcase{
		{"*.js", "index.js", nil},
		{"*.js", "index.ts", nil},
		{"*.js", ".js", nil},
		{"*.js", ".js", &ParseOptions{Dot: true}},
		{"foo/*.js", "foo/app.js", nil},
		{"foo/*.js", "foo/bar/app.js", nil},
		{"**/*.js", "foo/bar/app.js", nil},
		{"**/*.js", "app.js", nil},
		{"foo/**/bar/*.js", "foo/x/y/z/bar/app.js", nil},
		{"*", "foo", nil},
		{"*", "foo/bar", nil},
		{"*", ".hidden", nil},
		{"*", ".hidden", &ParseOptions{Dot: true}},
		{".*", ".bashrc", nil},
		{"!*.js", "app.js", nil},
		{"!*.js", "app.ts", nil},
		{"foo/{a,b,c}/*.js", "foo/a/app.js", nil},
		{"foo/{a,b,c}/*.js", "foo/d/app.js", nil},
		{"foo/[a-z]/*.js", "foo/x/app.js", nil},
		{"foo/[a-z]/*.js", "foo/1/app.js", nil},
		{"@(foo|bar)/*.js", "foo/app.js", nil},
		{"@(foo|bar)/*.js", "baz/app.js", nil},
		{"!(foo)", "bar", nil},
		{"!(foo)", "foo", nil},
		{"!(a|b)", "c", nil},
		{"!(a|b)", "a", nil},
		{"a/*", "a/b", nil},
		{"a/*", "a/b", &ParseOptions{Windows: true}},
		{"*.*", "foo.js", nil},
		{"*.*", ".hidden", nil},
		{"*.*", ".hidden", &ParseOptions{Dot: true}},
		{"foo/*", "foo/.hidden", nil},
		{"foo/*", "foo/.hidden", &ParseOptions{Dot: true}},
		{"foo/.*", "foo/.hidden", nil},
		{"[a-c]*", "b.js", nil},
		{"[a-c]*", "d.js", nil},
		{"[!a-c]*", "d.js", &ParseOptions{Posix: true}},
		{"[!a-c]*", "b.js", &ParseOptions{Posix: true}},
		{"*(foo|bar)", "foofobar", nil},
		{"*(foo|bar)", "foobar", nil},
		{"*(foo|bar)", "foobarx", nil},
		{"+(foo|bar)", "foobar", nil},
		{"+(foo|bar)", "", nil},
		{"?(foo|bar)", "foo", nil},
		{"?(foo|bar)", "bar", nil},
		{"?(foo|bar)", "foobar", nil},
		{"?(foo|bar)", "", nil},
		{"**/*.js", "foo/.bar/baz.js", nil},
		{"**/*.js", "foo/.bar/baz.js", &ParseOptions{Dot: true}},
		// Expanded differential suite addressing audit findings
		{"foo/!(bar|baz)/*.js", "foo/quux/test.js", nil},
		{"foo/!(bar|baz)/*.js", "foo/bar/test.js", nil},
		{"!(foo)/**/bar.ts", "quux/a/bar.ts", nil},
		{"!(foo)/**/bar.ts", "foo/a/bar.ts", nil},
		{"src/**/!(test)/*.go", "src/pkg/util/main.go", nil},
		{"src/**/!(test)/*.go", "src/pkg/test/main.go", nil},
		{"!(foo|!(bar))", "baz", nil},
		{"!(foo|!(bar))", "foo", nil},
		{"!(foo|!(bar))", "bar", nil},
		{"\\!(foo)*.js", "!(foo)app.js", nil},
		{"\\!(foo)*.js", "bar.js", nil},
		{"**/*.ts", "src/日本語/test.ts", nil},
		{"📁/*.txt", "📁/document.txt", nil},
		{"[[:alpha:]]*", "foo.js", nil},
		{"[[:alpha:]]*", "123.js", nil},
		{"[[:digit:]]*", "456.ts", nil},
		{"foo/{1..5}/*.js", "foo/3/app.js", nil},
		{"foo/{1..5}/*.js", "foo/8/app.js", nil},
		{"*.js", "foo/bar/app.js", &ParseOptions{MatchBase: true}},
		{"*.js", "foo/bar/app.js", &ParseOptions{Basename: true}},
		{"**/*.js", "foo/build/bundle.js", &ParseOptions{Ignore: []string{"**/build/**"}}},
		{"**/*.js", "foo/src/index.js", &ParseOptions{Ignore: []string{"**/build/**"}}},
		{"*.js", "APP.JS", &ParseOptions{Nocase: true}},
		{"foo//bar/*.js", "foo//bar/app.js", nil},
		{"foo/*/", "foo/bar/", nil},
		{".*", ".git", nil},
		{"**/*", "a/.b/c", nil},
		{"**/*", "a/.b/c", &ParseOptions{Dot: true}},
		{"a/../*", "a/../bar", nil},
		{"*", ".", nil},
		{"*", "..", nil},
		{"[a-", "[a-", nil},
		{"*(foo", "*(foo", nil},
		{"[a-", "b", &ParseOptions{StrictBrackets: true}},
	}

	for i, tc := range corpus {
		m, goErr := Compile(tc.Pattern, tc.Opts)
		var goResult bool
		if goErr == nil {
			goResult = m.Match(tc.Input)
		}

		jsOpts := make(map[string]interface{})
		if tc.Opts != nil {
			if tc.Opts.Dot {
				jsOpts["dot"] = true
			}
			if tc.Opts.Windows {
				jsOpts["windows"] = true
			}
			if tc.Opts.Posix {
				jsOpts["posix"] = true
			}
			if tc.Opts.MatchBase {
				jsOpts["matchBase"] = true
			}
			if tc.Opts.Basename {
				jsOpts["basename"] = true
			}
			if len(tc.Opts.Ignore) > 0 {
				jsOpts["ignore"] = tc.Opts.Ignore
			}
			if tc.Opts.Nocase {
				jsOpts["nocase"] = true
			}
			if tc.Opts.Bash {
				jsOpts["bash"] = true
			}
			if tc.Opts.NoExt {
				jsOpts["noext"] = true
			}
			if tc.Opts.NoExtglob {
				jsOpts["noextglob"] = true
			}
			if tc.Opts.Unescape {
				jsOpts["unescape"] = true
			}
			if tc.Opts.Contains {
				jsOpts["contains"] = true
			}
			if tc.Opts.StrictBrackets {
				jsOpts["strictBrackets"] = true
			}
			if tc.Opts.NoBracket {
				jsOpts["nobracket"] = true
			}
			if tc.Opts.LiteralBrackets {
				jsOpts["literalBrackets"] = true
			}
			if tc.Opts.NoBrace {
				jsOpts["nobrace"] = true
			}
			if tc.Opts.NoGlobstar {
				jsOpts["noglobstar"] = true
			}
			if tc.Opts.StrictSlashes {
				jsOpts["strictSlashes"] = true
			}
			if tc.Opts.NoNegate {
				jsOpts["nonegate"] = true
			}
		}

		req := jsMatcherRequest{
			ID:      i + 1,
			Pattern: tc.Pattern,
			Input:   tc.Input,
			Options: jsOpts,
		}

		if err := encoder.Encode(req); err != nil {
			t.Fatalf("Failed to encode request %d: %v", i, err)
		}

		if !scanner.Scan() {
			t.Fatalf("Node subprocess exited unexpectedly at test %d", i)
		}

		var resp jsMatcherResponse
		if err := json.Unmarshal([]byte(scanner.Text()), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response %d: %v", i, err)
		}

		if goErr != nil && !resp.Success {
			// Both Go and Node rejected the pattern as invalid syntax
			continue
		}
		if goErr != nil && resp.Success {
			t.Errorf("[Test %d] Go errored (%v) but Node succeeded (result: %v) for pattern %q on input %q", i, goErr, resp.Result, tc.Pattern, tc.Input)
			continue
		}
		if goErr == nil && !resp.Success {
			t.Errorf("[Test %d] Go succeeded (result: %v) but Node reported error (%s) for pattern %q on input %q", i, goResult, resp.Error, tc.Pattern, tc.Input)
			continue
		}

		if goResult != resp.Result {
			t.Errorf("[Test %d] MISMATCH for pattern %q against input %q (opts: %+v) | Go: %v, JS: %v | RawOutput: %q | RE2: %v",
				i, tc.Pattern, tc.Input, tc.Opts, goResult, resp.Result, m.State.Output, m.Regexp)
		}
	}
}
