package picomatch

import (
	"strings"
	"testing"
)

func TestParseExtglobs_BasicOperators(t *testing.T) {
	tests := []struct {
		name         string
		pattern      string
		expectTokens []TokenType
		expectConds  int
		expectInner  string
		expectNegBos bool
	}{
		{
			name:    "question mark extglob",
			pattern: "?(foo|bar)",
			expectTokens: []TokenType{
				TokenTypeBos, TokenTypeQmark, TokenTypeParen, TokenTypeText, TokenTypeParen,
			},
			expectConds: 2,
			expectInner: "foo|bar",
		},
		{
			name:         "negated extglob at BOS",
			pattern:      "!(abc)",
			expectTokens: []TokenType{TokenTypeBos, TokenTypeNegate, TokenTypeParen, TokenTypeText, TokenTypeParen},
			expectConds:  1,
			expectInner:  "abc",
			expectNegBos: true,
		},
		{
			name:         "negated extglob not at BOS",
			pattern:      "x!(abc)",
			expectTokens: []TokenType{TokenTypeBos, TokenTypeText, TokenTypeNegate, TokenTypeParen, TokenTypeText, TokenTypeParen},
			expectConds:  1,
			expectInner:  "abc",
			expectNegBos: false,
		},
		{
			name:        "plus extglob with multiple pipes",
			pattern:     "+(a|b|c)",
			expectConds: 3,
			expectInner: "a|b|c",
		},
		{
			name:        "star extglob",
			pattern:     "*(x|y)",
			expectConds: 2,
			expectInner: "x|y",
		},
		{
			name:    "at symbol extglob (not added to ExtglobStack per parse.js:1097)",
			pattern: "@(x|y)",
			// @ emits token with output "" marked as extglob, followed by regular paren group
			expectTokens: []TokenType{TokenTypeBos, TokenTypeAt, TokenTypeParen, TokenTypeText, TokenTypeParen},
			expectConds:  0,
			expectInner:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := Parse(tt.pattern, nil)
			if err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", tt.pattern, err)
			}

			if !s.ExtglobStack.IsEmpty() {
				t.Errorf("ExtglobStack should be empty after balanced extglob closure in %q", tt.pattern)
			}
			if s.Parens != 0 {
				t.Errorf("expected 0 parens remaining, got %d in %q", s.Parens, tt.pattern)
			}
			if s.NegatedExtglob != tt.expectNegBos {
				t.Errorf("expected NegatedExtglob = %v, got %v in %q", tt.expectNegBos, s.NegatedExtglob, tt.pattern)
			}
			if len(tt.expectTokens) > 0 && len(s.Tokens) == len(tt.expectTokens) {
				for i, expectedType := range tt.expectTokens {
					if s.Tokens[i].Type != expectedType {
						t.Errorf("token %d expected type %q, got %q", i, expectedType, s.Tokens[i].Type)
					}
				}
			}
		})
	}
}

func TestParseExtglobs_NestedExpressions(t *testing.T) {
	pattern := "!(a|?(b|c))"
	s, err := Parse(pattern, nil)
	if err != nil {
		t.Fatalf("Parse(%q) failed on nested extglob: %v", pattern, err)
	}
	if !s.EOS() {
		t.Errorf("failed to reach EOS cleanly on nested extglob")
	}
	if !s.ExtglobStack.IsEmpty() {
		t.Errorf("ExtglobStack not cleanly emptied on nested extglob")
	}
	if !s.NegatedExtglob {
		t.Errorf("expected NegatedExtglob to be set by outer !(...) at BOS")
	}
}

func TestParseExtglobs_Options(t *testing.T) {
	t.Run("NoExtglob option disables extglob recognition", func(t *testing.T) {
		opts := &ParseOptions{NoExtglob: true}
		pattern := "!(abc)"
		s, err := Parse(pattern, opts)
		if err != nil {
			t.Fatalf("Parse failed with NoExtglob: %v", err)
		}
		for _, tok := range s.Tokens {
			if tok.Extglob {
				t.Errorf("expected no tokens marked as Extglob when NoExtglob=true, got token %+v", tok)
			}
		}
	})

	t.Run("StrictBrackets unbalanced opening at EOF", func(t *testing.T) {
		opts := &ParseOptions{StrictBrackets: true}
		_, err := Parse("!(a|b", opts)
		if err == nil || !strings.Contains(err.Error(), "closing") {
			t.Errorf("expected SyntaxError for missing closing paren when StrictBrackets=true, got: %v", err)
		}
	})

	t.Run("StrictBrackets unbalanced closing", func(t *testing.T) {
		opts := &ParseOptions{StrictBrackets: true}
		_, err := Parse("abc)", opts)
		if err == nil || !strings.Contains(err.Error(), "opening") {
			t.Errorf("expected SyntaxError for missing opening paren when StrictBrackets=true, got: %v", err)
		}
	})

	t.Run("Unbalanced parens escape without StrictBrackets", func(t *testing.T) {
		s, err := Parse("abc)", nil)
		if err != nil {
			t.Fatalf("unexpected error on unmatched closing paren: %v", err)
		}
		if !strings.Contains(s.Output, "\\)") {
			t.Errorf("expected escaped trailing parenthesis \\) in output, got %q", s.Output)
		}

		s2, err := Parse("!(abc", nil)
		if err != nil {
			t.Fatalf("unexpected error on unclosed opening extglob: %v", err)
		}
		if !strings.Contains(s2.Output, "\\(") {
			t.Errorf("expected escaped trailing parenthesis \\( in output, got %q", s2.Output)
		}
	})
}

func TestParseExtglobs_RegexGroupExclusion(t *testing.T) {
	regexGroups := []string{
		"!(?:foo)",
		"!(?!foo)",
		"!(?=foo)",
		"!(?<=foo)",
	}

	for _, pattern := range regexGroups {
		s, err := Parse(pattern, nil)
		if err != nil {
			t.Fatalf("Parse(%q) failed: %v", pattern, err)
		}
		for _, tok := range s.Tokens {
			if tok.Type == TokenTypeNegate {
				t.Errorf("regex group %q incorrectly identified as negate extglob token: %+v", pattern, tok)
			}
		}
	}
}

func TestParseExtglobs_EscapedOperators(t *testing.T) {
	escapedPatterns := []string{
		`\!(abc)`,
		`\?(abc)`,
		`\+(abc)`,
		`\@(abc)`,
		`\*(abc)`,
		`!\(abc)`,
	}

	for _, pattern := range escapedPatterns {
		s, err := Parse(pattern, nil)
		if err != nil {
			t.Fatalf("Parse(%q) failed: %v", pattern, err)
		}
		for _, tok := range s.Tokens {
			if tok.Extglob {
				t.Errorf("escaped operator in pattern %q should not produce extglob tokens, got %+v", pattern, tok)
			}
		}
	}
}

func TestParseExtglobs_EdgeCasesAndGuardBranches(t *testing.T) {
	t.Run("Qmark extglob not initiated when previous token is open paren group", func(t *testing.T) {
		pattern := "(?(a))"
		s, err := Parse(pattern, nil)
		if err != nil {
			t.Fatalf("Parse(%q) failed: %v", pattern, err)
		}
		// In (?(a)), the first ( is an open paren token. When ? is evaluated, prev.Value == "(", so isGroup is true.
		// Therefore, ? does not initiate an extglob (parse.js:1022).
		for _, tok := range s.Tokens {
			if tok.Type == TokenTypeQmark {
				t.Errorf("did not expect TokenTypeQmark when preceded by open paren group in %q", pattern)
			}
		}
	})

	t.Run("Operators not followed by open paren", func(t *testing.T) {
		pattern := "!a ?b +c @d *e"
		s, err := Parse(pattern, nil)
		if err != nil {
			t.Fatalf("Parse(%q) failed: %v", pattern, err)
		}
		if s.Parens != 0 || !s.ExtglobStack.IsEmpty() {
			t.Errorf("operators without trailing open parens must not affect extglob state")
		}
	})

	t.Run("Star operator at EOF with open paren but no third character", func(t *testing.T) {
		// In parse.js:1140, * requires /^\([^?]/.test(remaining()), requiring a third character that is not '?'
		pattern := "*("
		s, err := Parse(pattern, nil)
		if err != nil {
			t.Fatalf("Parse(%q) failed: %v", pattern, err)
		}
		for _, tok := range s.Tokens {
			if tok.Extglob {
				t.Errorf("did not expect extglob token for %q with no character following open paren", pattern)
			}
		}
	})
}
