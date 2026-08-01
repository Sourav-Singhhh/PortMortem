package picomatch

import (
	"testing"
)

func TestParse_InitializationAndEOF(t *testing.T) {
	// Test nil options handling and empty string EOF termination
	state, err := Parse("", nil)
	if err != nil {
		t.Fatalf("Parse(\"\", nil) unexpected error: %v", err)
	}
	if !state.EOS() {
		t.Errorf("expected empty string parse to finish at EOS")
	}
	if len(state.Tokens) != 1 || state.Tokens[0].Type != TokenTypeBos {
		t.Errorf("expected empty input AST to contain exclusively initial BOS token, got %+v", state.Tokens)
	}

	// Test custom options application onto initial parse state and BOS token
	customOpts := NewParseOptions()
	customOpts.Dot = true
	customOpts.Prepend = "^"
	stateOpts, err := Parse("", customOpts)
	if err != nil {
		t.Fatalf("Parse(\"\", customOpts) unexpected error: %v", err)
	}
	if !stateOpts.Dot || len(stateOpts.Tokens) == 0 || stateOpts.Tokens[0].Output != "^" {
		t.Errorf("custom options were not properly applied to initial state: Dot=%v, BOS output=%q", stateOpts.Dot, stateOpts.Tokens[0].Output)
	}
}

func TestParse_ASCIISyntaxTraversalAndBranches(t *testing.T) {
	// Construct an input containing EVERY single grammatical syntax character defined in the switch statement,
	// plus plain text alphanumeric characters, guaranteeing 100% switch branch traversal and code coverage.
	input := `\"()[].{}/|,?!+@*abcdef`
	state, err := Parse(input, nil)
	if err != nil {
		t.Fatalf("Parse on all ASCII syntax branches failed: %v", err)
	}
	if !state.EOS() {
		t.Errorf("parser loop failed to terminate cleanly at EOS on ASCII syntax stream")
	}
	expectedConsumed := `\"()[\].{}/|,?!+@*abcdef`
	if state.Consumed != expectedConsumed {
		t.Errorf("expected consumed text accumulator %q, got %q", expectedConsumed, state.Consumed)
	}
	expectedOutput := `\"()\[\].{}/|,?!+@*abcdef`
	if state.Output != expectedOutput {
		t.Errorf("expected compiled output %q, got %q", expectedOutput, state.Output)
	}
	// With bracket and paren foundations active, [], (), and extglob structures emit distinct structural tokens separating plain text sequences
	if len(state.Tokens) != 5 {
		t.Errorf("expected 5 AST tokens (BOS, text, parens, brackets), got %d tokens", len(state.Tokens))
	}
}

func TestParse_MalformedInputsAndStability(t *testing.T) {
	// Verify that malformed or unmatched structural patterns traverse safely without panicking or hanging
	malformedPatterns := []struct {
		pattern  string
		expected string
	}{
		{"[unclosed-bracket", "\\[unclosed-bracket"},
		{"{unclosed,brace,dots..", "{unclosed,brace,dots.."},
		{"(unclosed-extglob|paren", "(unclosed-extglob|paren"},
		{"trailing-backslash\\", "trailing-backslash\\\\"},
		{")))(()}{][[][" + `\` + `\\\\\\`, ")))(()}{][\\[]\\[\\"},
		{"/*/**/***//", "/*/**/***//"},
	}

	for _, tt := range malformedPatterns {
		state, err := Parse(tt.pattern, nil)
		if err != nil {
			t.Errorf("Parse(%q) returned unexpected error: %v", tt.pattern, err)
		}
		if !state.EOS() {
			t.Errorf("Parse(%q) failed to cleanly reach EOS", tt.pattern)
		}
		if state.Consumed != tt.expected {
			t.Errorf("Parse(%q) incomplete consumption: got %q, expected %q", tt.pattern, state.Consumed, tt.expected)
		}
	}
}

func TestParse_UnicodeAndMultibyteTraversal(t *testing.T) {
	// Confirm that multibyte UTF-8 sequences traverse cleanly without breaking character boundaries or triggering false branches
	unicodePatterns := []string{
		"path/to/★/file.png",
		"emoji/🚀/test/*.md",
		"日本語/テスト/index.html",
	}

	for _, pattern := range unicodePatterns {
		state, err := Parse(pattern, nil)
		if err != nil {
			t.Errorf("Parse(%q) failed with error: %v", pattern, err)
		}
		if !state.EOS() {
			t.Errorf("Parse(%q) did not terminate at EOS", pattern)
		}
		if state.Consumed != pattern {
			t.Errorf("Parse(%q) accumulator mismatch: got %q, expected %q", pattern, state.Consumed, pattern)
		}
	}
}

func TestParse_NullByteFiltering(t *testing.T) {
	// Verify that embedded literal null bytes are silently discarded during parser loop traversal (parse.js:664)
	input := "before\u0000after\u0000"
	state, err := Parse(input, nil)
	if err != nil {
		t.Fatalf("Parse on null byte input failed: %v", err)
	}
	if !state.EOS() {
		t.Errorf("Parse did not reach EOS after null bytes")
	}
	expected := "beforeafter"
	if state.Consumed != expected || state.Output != expected {
		t.Errorf("null bytes were not properly filtered: got Consumed=%q, Output=%q (expected %q)", state.Consumed, state.Output, expected)
	}
}

func TestParse_RepeatedInvocationsAndStateIsolation(t *testing.T) {
	// Guarantee stable memory reuse and total state isolation across successive Parse calls
	pattern := "a*b?c"
	for i := 0; i < 10; i++ {
		state, err := Parse(pattern, nil)
		if err != nil || !state.EOS() || state.Consumed != pattern || state.Output != pattern {
			t.Fatalf("iteration %d failed state isolation checks: %+v, err=%v", i, state, err)
		}
	}
}
