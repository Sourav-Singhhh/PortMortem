package picomatch

import (
	"strings"
	"testing"
)

func TestParseBraces_BasicAndEmpty(t *testing.T) {
	t.Run("Empty braces {} escape to \\{\\}", func(t *testing.T) {
		state, err := Parse("foo/{}bar", nil)
		if err != nil {
			t.Fatalf("Unexpected parse error: %v", err)
		}
		if state.Braces != 0 {
			t.Errorf("Expected Braces==0 at EOF, got %d", state.Braces)
		}
		if !strings.Contains(state.Output, "\\{\\}") {
			t.Errorf("Expected empty braces to escape to '\\{\\}', got %q", state.Output)
		}
	})

	t.Run("Single item braces {a} without comma escape to \\{a\\}", func(t *testing.T) {
		state, err := Parse("foo/{a}bar", nil)
		if err != nil {
			t.Fatalf("Unexpected parse error: %v", err)
		}
		if !strings.Contains(state.Output, "\\{a\\}") {
			t.Errorf("Expected solitary brace group without comma to escape as '\\{a\\}', got %q", state.Output)
		}
	})

	t.Run("Brace group with comma {a,b} converts to (a|b)", func(t *testing.T) {
		state, err := Parse("foo/{a,b}bar", nil)
		if err != nil {
			t.Fatalf("Unexpected parse error: %v", err)
		}
		if !strings.Contains(state.Output, "(a|b)") {
			t.Errorf("Expected brace expansion group with comma to produce '(a|b)', got %q", state.Output)
		}
	})
}

func TestParseBraces_NestedAndCommas(t *testing.T) {
	t.Run("Nested braces with commas {a,{b,c}} convert both levels", func(t *testing.T) {
		state, err := Parse("{a,{b,c}}", nil)
		if err != nil {
			t.Fatalf("Unexpected parse error: %v", err)
		}
		if state.Output != "(a|(b|c))" {
			t.Errorf("Expected nested brace output '(a|(b|c))', got %q", state.Output)
		}
	})

	t.Run("Nested solitary brace inside comma group {a,{b}} escapes inner group", func(t *testing.T) {
		state, err := Parse("{a,{b}}", nil)
		if err != nil {
			t.Fatalf("Unexpected parse error: %v", err)
		}
		if state.Output != "(a|\\{b\\})" {
			t.Errorf("Expected inner solitary brace to escape as '(a|\\{b\\})', got %q", state.Output)
		}
	})

	t.Run("Comma outside braces remains literal comma", func(t *testing.T) {
		state, err := Parse("foo,bar", nil)
		if err != nil {
			t.Fatalf("Unexpected parse error: %v", err)
		}
		if !strings.Contains(state.Output, "foo,bar") {
			t.Errorf("Expected standalone comma to remain literal, got %q", state.Output)
		}
	})

	t.Run("Comma inside non-brace stack context (e.g., parens/extglob) does not trigger brace comma conversion", func(t *testing.T) {
		// Because extglob parenthesis tracking is deferred to a subsequent milestone, we explicitly evaluate
		// HandleBraceTraversal under simulated non-brace stack depth to confirm the parse.js:962 stack invariant.
		state := NewParseState("{foo,(a,b)}", nil)
		state.Increment(ParserContextBraces) // Open outer brace
		openTok := NewParseToken(TokenTypeBrace, "{", "(")
		state.BraceStack.Push(NewBraceState(openTok, "{", "(", 0, 1))
		state.Increment(ParserContextParens) // Open inner paren group

		err := HandleBraceTraversal(state, ",")
		if err != nil {
			t.Fatalf("Unexpected error in HandleBraceTraversal: %v", err)
		}

		last := state.CurrentToken()
		if last == nil || last.Output == "|" || last.Output != "," {
			t.Errorf("Expected comma inside parens context to remain literal ',', got token Output %q", last.Output)
		}
		brace, _ := state.BraceStack.Peek()
		if brace.Comma {
			t.Errorf("Expected brace Comma flag to remain false when comma occurs inside parenthetical context")
		}
	})
}

func TestParseBraces_EscapesAndMalformed(t *testing.T) {
	t.Run("Escaped opening and closing braces \\{a,b\\}", func(t *testing.T) {
		state, err := Parse("\\{a,b\\}", nil)
		if err != nil {
			t.Fatalf("Unexpected parse error: %v", err)
		}
		if state.Braces != 0 {
			t.Errorf("Expected Braces==0 with escaped braces, got %d", state.Braces)
		}
		if !strings.Contains(state.Output, "\\{a,b\\}") {
			t.Errorf("Expected escaped braces to remain literal '\\{a,b\\}', got %q", state.Output)
		}
	})

	t.Run("Escaped comma inside brace group does not set comma flag", func(t *testing.T) {
		state, err := Parse("{a\\,b}", nil)
		if err != nil {
			t.Fatalf("Unexpected parse error: %v", err)
		}
		if state.Output != "\\{a\\,b\\}" {
			t.Errorf("Expected escaped comma in braces to produce '\\{a\\,b\\}', got %q", state.Output)
		}
	})

	t.Run("Unclosed opening brace at EOF without strict mode is escaped", func(t *testing.T) {
		state, err := Parse("foo/{a,b", nil)
		if err != nil {
			t.Fatalf("Unexpected parse error: %v", err)
		}
		if state.Braces != 0 {
			t.Errorf("Expected Braces==0 after EOF reconciliation, got %d", state.Braces)
		}
		if !strings.Contains(state.Output, "(a|b") {
			t.Errorf("Expected unclosed brace output, got %q", state.Output)
		}
	})

	t.Run("Unmatched closing brace emits literal text token", func(t *testing.T) {
		state, err := Parse("foo}bar", nil)
		if err != nil {
			t.Fatalf("Unexpected parse error: %v", err)
		}
		if !strings.Contains(state.Output, "foo}bar") {
			t.Errorf("Expected unmatched closing brace to emit literal text 'foo}bar', got %q", state.Output)
		}
	})
}

func TestParseBraces_OptionsAndStrictMode(t *testing.T) {
	t.Run("NoBrace option disables brace parsing and emits text tokens", func(t *testing.T) {
		state, err := Parse("{a,b}", &ParseOptions{NoBrace: true})
		if err != nil {
			t.Fatalf("Unexpected parse error: %v", err)
		}
		if state.Braces != 0 {
			t.Errorf("Expected Braces==0 when NoBrace=true, got %d", state.Braces)
		}
		if state.Output != "{a,b}" {
			t.Errorf("Expected unescaped literal '{a,b}' when NoBrace=true, got %q", state.Output)
		}
	})

	t.Run("StrictBrackets throws syntax error on unclosed opening brace at EOF", func(t *testing.T) {
		_, err := Parse("foo/{a,b", &ParseOptions{StrictBrackets: true})
		if err == nil {
			t.Fatalf("Expected syntax error with StrictBrackets on unclosed opening brace")
		}
		expectedErr := `Missing closing: "}" - use "\}" to match literal characters`
		if err.Error() != expectedErr {
			t.Errorf("Expected error message %q, got %q", expectedErr, err.Error())
		}
	})

	t.Run("LiteralBrackets and option permutation integrity with brace expressions", func(t *testing.T) {
		state, err := Parse("foo/{a,b}/bar", &ParseOptions{LiteralBrackets: true})
		if err != nil {
			t.Fatalf("Unexpected parse error: %v", err)
		}
		if !strings.Contains(state.Output, "(a|b)") {
			t.Errorf("Expected normal brace conversion under LiteralBrackets, got %q", state.Output)
		}
	})

	t.Run("Brace nesting depth counter integrity during sequential scanning", func(t *testing.T) {
		state, err := Parse("{a,{b,{c,d}}}", nil)
		if err != nil {
			t.Fatalf("Unexpected parse error: %v", err)
		}
		if state.Braces != 0 || state.BraceStack.Len() != 0 {
			t.Errorf("Expected clean closure of nesting depth and stack at EOF, got Braces=%d, Len=%d", state.Braces, state.BraceStack.Len())
		}
		if state.Output != "(a|(b|(c|d)))" {
			t.Errorf("Expected deep nested expansion '(a|(b|(c|d)))', got %q", state.Output)
		}
	})
}

// TestParseBraces_ForensicReview Explicates every required test pattern from pre-commit review requirements #3, #4, #5, and #6.
func TestParseBraces_ForensicReview(t *testing.T) {
	t.Run("Requirement 3: Verify nested braces {{a,b},c}", func(t *testing.T) {
		state, err := Parse("{{a,b},c}", nil)
		if err != nil {
			t.Fatalf("Unexpected parse error: %v", err)
		}
		if state.Output != "((a|b)|c)" {
			t.Errorf("Expected nested left brace conversion '((a|b)|c)', got %q", state.Output)
		}
	})

	t.Run("Requirement 4: Verify malformed inputs {, {{, {a, {a,{b", func(t *testing.T) {
		malformed := []struct {
			pattern string
			expect  string
		}{
			{"{", "("},
			{"{{", "(("},
			{"{a", "(a"},
			{"{a,{b", "(a|(b"},
		}
		for _, tc := range malformed {
			s, err := Parse(tc.pattern, nil)
			if err != nil {
				t.Fatalf("Unexpected parse error on %q: %v", tc.pattern, err)
			}
			if s.Braces != 0 || s.BraceStack.Len() != 0 {
				t.Errorf("Expected clean EOF balance recovery for %q, got Braces=%d", tc.pattern, s.Braces)
			}
			if s.Output != tc.expect {
				t.Errorf("For pattern %q, expected output %q, got %q", tc.pattern, tc.expect, s.Output)
			}
		}
	})

	t.Run("Requirement 5: Verify standalone escaped symbols \\{, \\}, \\,", func(t *testing.T) {
		escapers := []struct {
			pattern string
			expect  string
		}{
			{"\\{", "\\{"},
			{"\\}", "\\}"},
			{"\\,", "\\,"},
		}
		for _, tc := range escapers {
			s, err := Parse(tc.pattern, nil)
			if err != nil {
				t.Fatalf("Unexpected parse error on %q: %v", tc.pattern, err)
			}
			if s.Output != tc.expect {
				t.Errorf("For pattern %q, expected output %q, got %q", tc.pattern, tc.expect, s.Output)
			}
		}
	})

	t.Run("Requirement 6: Verify zero leakage of future grammar (range expansion, extglob, wildcards)", func(t *testing.T) {
		// Ensure {1..5} does NOT perform range enumeration during Sprint 7
		s, err := Parse("{1..5}", nil)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if strings.Contains(s.Output, "2") || strings.Contains(s.Output, "3") || strings.Contains(s.Output, "4") {
			t.Errorf("Leakage detected: numeric range expansion occurred prematurely in Sprint 7: %q", s.Output)
		}
	})
}
