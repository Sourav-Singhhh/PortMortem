package picomatch

import (
	"testing"
)

func TestParseRegex_PosixMapping(t *testing.T) {
	classes := []string{
		"alnum", "alpha", "ascii", "blank", "cntrl", "digit",
		"graph", "lower", "print", "punct", "space", "upper",
		"word", "xdigit",
	}

	for _, name := range classes {
		got, ok := GetPosixRegexSource(name)
		if !ok || got == "" {
			t.Errorf("GetPosixRegexSource(%q) failed to map valid POSIX class", name)
		}
	}

	if _, ok := GetPosixRegexSource("invalid_class"); ok {
		t.Errorf("Expected false for invalid POSIX class")
	}
}

func TestParseRegex_GlobCharsAndExtglobDefs(t *testing.T) {
	posixChars := GetGlobChars(false)
	if posixChars.SlashLiteral != `\/` {
		t.Errorf("Expected posix slash literal \\/, got %q", posixChars.SlashLiteral)
	}

	winChars := GetGlobChars(true)
	if winChars.Sep != `\` {
		t.Errorf("Expected windows sep \\, got %q", winChars.Sep)
	}

	extDefs := GetExtglobCharDefs(posixChars)
	for _, op := range []string{"!", "?", "+", "*", "@"} {
		if def, ok := extDefs[op]; !ok || def.Open == "" {
			t.Errorf("GetExtglobCharDefs missing valid definition for operator %q", op)
		}
	}
}

func TestParseRegex_IsRegexChar(t *testing.T) {
	regexChars := []string{"-", "*", "+", "?", ".", "^", "$", "{", "}", "(", ")", "|", "[", "]"}
	for _, ch := range regexChars {
		if !IsRegexChar(ch) {
			t.Errorf("Expected IsRegexChar(%q) to be true", ch)
		}
	}
	if IsRegexChar("a") || IsRegexChar("ab") || IsRegexChar("++") || IsRegexChar("") {
		t.Errorf("Expected false for non 1-byte regex chars")
	}
}

func TestParseRegex_EscapeRegex(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"a-b", `a\-b`},
		{"a*.+?^${}()|[]b", `a\*\.\+\?\^\$\{\}\(\)\|\[\]b`},
		{"plain", "plain"},
	}

	for _, tt := range tests {
		got := EscapeRegex(tt.input)
		if got != tt.expected {
			t.Errorf("EscapeRegex(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestParseRegex_HasRegexChars(t *testing.T) {
	if !HasRegexChars("a-b") {
		t.Errorf("Expected true for 'a-b'")
	}
	if HasRegexChars("abc") {
		t.Errorf("Expected false for 'abc'")
	}
}

func TestParseRegex_Globstar(t *testing.T) {
	chars := GetGlobChars(false)
	res1 := Globstar(nil, chars)
	if res1 == "" {
		t.Errorf("Expected non-empty Globstar result for nil opts")
	}
	res2 := Globstar(&ParseOptions{Dot: true}, chars)
	if res1 == res2 {
		t.Errorf("Expected different Globstar result when Dot=true")
	}
	res3 := Globstar(&ParseOptions{Capture: true}, chars)
	if res3[1:3] == "?:" {
		t.Errorf("Expected capture group without ?: when Capture=true, got %q", res3)
	}
}

func TestParseRegex_ExpandRange(t *testing.T) {
	t.Run("Standard alphabetical range", func(t *testing.T) {
		res := ExpandRange([]string{"a", "z"}, nil)
		if res != "[a-z]" {
			t.Errorf("Expected [a-z], got %q", res)
		}
	})

	t.Run("Reverse alphabetical range", func(t *testing.T) {
		res := ExpandRange([]string{"z", "a"}, nil)
		if res != "[a-z]" {
			t.Errorf("Expected sorted [a-z] for reverse range, got %q", res)
		}
	})

	t.Run("Custom ExpandRange function", func(t *testing.T) {
		opts := &ParseOptions{
			ExpandRange: func(left string, right string, opts *ParseOptions) string {
				return "([" + left + "-" + right + "])"
			},
		}
		res := ExpandRange([]string{"1", "5"}, opts)
		if res != "([1-5])" {
			t.Errorf("Expected ([1-5]), got %q", res)
		}
	})
}

func TestParseRegex_ReDoSAnalysis(t *testing.T) {
	testCases := []struct {
		expr    string
		isRisky bool
	}{
		{"*(a)|*(b)", true},
		{"a|b", false},
		{"+(a|b|a*)", true},
		{"*(*(a))", true},
		{"+(+(a|b))", true},
		{"a*|b*", false},
		{"a", false},
	}

	for _, tc := range testCases {
		res := AnalyzeRepeatedExtglob(tc.expr, nil)
		if res.Risky != tc.isRisky {
			t.Errorf("AnalyzeRepeatedExtglob(%q).Risky = %v, want %v", tc.expr, res.Risky, tc.isRisky)
		}
	}
}
