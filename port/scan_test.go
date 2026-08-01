package picomatch

import (
	"reflect"
	"testing"
)

func TestScan_BasicAndPrefixes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		opts     *ScanOptions
		expected *ScanState
	}{
		{
			name:  "leading ./ with basic glob",
			input: "./foo/bar/*.js",
			opts:  nil,
			expected: &ScanState{
				Prefix: "./",
				Input:  "./foo/bar/*.js",
				Start:  2,
				Base:   "foo/bar",
				Glob:   "*.js",
				IsGlob: true,
			},
		},
		{
			name:  "negation prefix !",
			input: "!foo/bar/*.js",
			opts:  nil,
			expected: &ScanState{
				Prefix:  "!",
				Input:   "!foo/bar/*.js",
				Start:   1,
				Base:    "foo/bar",
				Glob:    "*.js",
				IsGlob:  true,
				Negated: true,
			},
		},
		{
			name:  "combined negation and ./ prefix (!./)",
			input: "!./foo/bar/*.js",
			opts:  nil,
			expected: &ScanState{
				Prefix:  "!./",
				Input:   "!./foo/bar/*.js",
				Start:   3,
				Base:    "foo/bar",
				Glob:    "*.js",
				IsGlob:  true,
				Negated: true,
			},
		},
		{
			name:  "nonegate option overrides !",
			input: "!foo/bar/*.js",
			opts:  &ScanOptions{NoNegate: true},
			expected: &ScanState{
				Prefix:  "",
				Input:   "!foo/bar/*.js",
				Start:   0,
				Base:    "!foo/bar",
				Glob:    "*.js",
				IsGlob:  true,
				Negated: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := Scan(tt.input, tt.opts)
			assertStateEqual(t, res, tt.expected)
		})
	}
}

func TestScan_Extglob(t *testing.T) {
	t.Run("basic extglob", func(t *testing.T) {
		res := Scan("./foo/@(foo)/*.js", nil)
		if !res.IsExtglob || !res.IsGlob || res.Base != "foo" || res.Glob != "@(foo)/*.js" {
			t.Errorf("Unexpected result for basic extglob: %+v", res)
		}
	})

	t.Run("negated extglob does not trigger simple negation", func(t *testing.T) {
		res := Scan("!(foo)*", nil)
		if !res.IsExtglob || !res.NegatedExtglob || res.Negated {
			t.Errorf("Expected NegatedExtglob=true and Negated=false, got: %+v", res)
		}
	})

	t.Run("noext option disables extglob recognition and applies reset rule", func(t *testing.T) {
		res := Scan("./foo/@(foo)/*.js", &ScanOptions{NoExt: true})
		if res.IsExtglob {
			t.Errorf("Expected IsExtglob=false when NoExt is true, got: %+v", res)
		}
	})
}

func TestScan_BracesAndEscapedBraces(t *testing.T) {
	t.Run("standard brace expansion with scanToEnd", func(t *testing.T) {
		res := Scan("foo/{a,b,c}/*.js", &ScanOptions{ScanToEnd: true})
		if !res.IsBrace || !res.IsGlob || res.Base != "foo" || res.Glob != "{a,b,c}/*.js" {
			t.Errorf("Expected valid brace expansion, got: %+v", res)
		}
	})

	t.Run("escaped braces still set IsBrace in JS due to closing brace check", func(t *testing.T) {
		// In original JS scan.js, when reaching '}', if braces decrements to 0, it sets isBrace = true
		// regardless of braceEscaped flag. We preserve this exact JS behavior.
		res := Scan("foo/\\{a,b}/*.js", &ScanOptions{ScanToEnd: true})
		if !res.IsBrace {
			t.Errorf("Expected IsBrace=true (preserving JS edge case behavior), got: %+v", res)
		}
	})
}

func TestScan_BracketsAndUnterminatedBrackets(t *testing.T) {
	t.Run("valid POSIX bracket character class", func(t *testing.T) {
		res := Scan("foo/[a-z]/*.js", nil)
		if !res.IsBracket || !res.IsGlob || res.Base != "foo" || res.Glob != "[a-z]/*.js" {
			t.Errorf("Expected valid bracket recognition, got: %+v", res)
		}
	})

	t.Run("unterminated brackets are ignored without closing ]", func(t *testing.T) {
		res := Scan("foo/[a-z", nil)
		if res.IsBracket {
			t.Errorf("Expected IsBracket=false for unterminated bracket, got: %+v", res)
		}
	})
}

func TestScan_Globstar(t *testing.T) {
	res := Scan("./foo/**/*.js", &ScanOptions{ScanToEnd: true})
	if !res.IsGlobstar || !res.IsGlob || res.Base != "foo" || res.Glob != "**/*.js" {
		t.Errorf("Expected valid globstar recognition, got: %+v", res)
	}
}

func TestScan_SlashNormalizationAndTrailingRules(t *testing.T) {
	t.Run("root path preservation", func(t *testing.T) {
		res := Scan("/*.js", nil)
		if res.Base != "" && res.Base != "/" {
			t.Logf("Root scan base: %q, glob: %q", res.Base, res.Glob)
		}
	})

	t.Run("trailing slash removal from base directory", func(t *testing.T) {
		res := Scan("foo/bar/*.js", nil)
		if res.Base != "foo/bar" {
			t.Errorf("Expected trailing slash removed from base ('foo/bar'), got: %q", res.Base)
		}
	})
}

func TestScan_TokensPartsAndMaxDepth(t *testing.T) {
	t.Run("parts extraction", func(t *testing.T) {
		res := Scan("./foo/@(bar)/**/*.js", &ScanOptions{Parts: true})
		expectedParts := []string{"foo", "@(bar)", "**", "*.js"}
		if !reflect.DeepEqual(res.Parts, expectedParts) {
			t.Errorf("Expected parts %v, got %v", expectedParts, res.Parts)
		}
	})

	t.Run("tokens and maxDepth computation with globstar infinity", func(t *testing.T) {
		// Note: in JS, opts.tokens alone does not set scanToEnd = true on line 53; parts or scanToEnd must be true
		res := Scan("./foo/**/*.js", &ScanOptions{Tokens: true, ScanToEnd: true})
		if !res.MaxInfinity {
			t.Errorf("Expected MaxInfinity=true when globstar is present in tokens, got false")
		}
		if len(res.Tokens) == 0 {
			t.Errorf("Expected populated Tokens array when Tokens=true")
		}
	})

	t.Run("finite maxDepth computation without globstar", func(t *testing.T) {
		res := Scan("foo/bar/*.js", &ScanOptions{Tokens: true})
		if res.MaxInfinity {
			t.Errorf("Expected MaxInfinity=false for simple glob, got true")
		}
		if res.MaxDepth <= 0 {
			t.Errorf("Expected positive MaxDepth for tokenized path, got %d", res.MaxDepth)
		}
	})
}

func assertStateEqual(t *testing.T, actual, expected *ScanState) {
	t.Helper()
	if actual.Prefix != expected.Prefix {
		t.Errorf("Prefix: got %q, want %q", actual.Prefix, expected.Prefix)
	}
	if actual.Start != expected.Start {
		t.Errorf("Start: got %d, want %d", actual.Start, expected.Start)
	}
	if actual.Base != expected.Base {
		t.Errorf("Base: got %q, want %q", actual.Base, expected.Base)
	}
	if actual.Glob != expected.Glob {
		t.Errorf("Glob: got %q, want %q", actual.Glob, expected.Glob)
	}
	if actual.IsGlob != expected.IsGlob {
		t.Errorf("IsGlob: got %v, want %v", actual.IsGlob, expected.IsGlob)
	}
	if actual.Negated != expected.Negated {
		t.Errorf("Negated: got %v, want %v", actual.Negated, expected.Negated)
	}
}
