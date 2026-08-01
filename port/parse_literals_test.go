package picomatch

import (
	"testing"
)

func TestParseLiterals_PlainLiterals(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		expected string
	}{
		{"Simple ASCII", "hello_world_123", "hello_world_123"},
		{"Single character", "a", "a"},
		{"Anchor symbol dollar", "foo$bar", "foo\\$bar"},
		{"Anchor symbol caret", "foo^bar", "foo\\^bar"},
		{"Combined anchors", "a$b^c", "a\\$b\\^c"},
		{"Standalone anchor dollar", "$", "\\$"},
		{"Standalone anchor caret", "^", "\\^"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state, err := Parse(tt.pattern, nil)
			if err != nil {
				t.Fatalf("Parse(%q) returned unexpected error: %v", tt.pattern, err)
			}

			// We expect exactly 2 tokens: BOS and the merged text token
			if len(state.Tokens) != 2 {
				t.Fatalf("Expected 2 tokens (BOS + text), got %d: %+v", len(state.Tokens), state.Tokens)
			}
			tok := state.Tokens[1]
			if tok.Type != TokenTypeText || tok.Value != tt.expected {
				t.Errorf("Expected token (text, %q), got (%s, %q)", tt.expected, tok.Type, tok.Value)
			}
			if state.Output != tt.expected {
				t.Errorf("Expected state.Output %q, got %q", tt.expected, state.Output)
			}
		})
	}
}

func TestParseLiterals_EscapedLiterals(t *testing.T) {
	t.Run("Standard escaped ASCII", func(t *testing.T) {
		state, err := Parse("a\\bc", nil)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if len(state.Tokens) != 2 || state.Tokens[1].Value != "a\\bc" {
			t.Errorf("Expected consolidated token value 'a\\bc', got %+v", state.Tokens)
		}
	})

	t.Run("Escaped slash without bash option (default)", func(t *testing.T) {
		// Without bash option, \/ discards backslash. With slash grammar active in Sprint 9,
		// the unescaped slash emits a distinct TokenTypeSlash token between the plain text segments.
		state, err := Parse("foo\\/bar", nil)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if len(state.Tokens) != 4 || state.Tokens[1].Value != "foo" || state.Tokens[2].Type != TokenTypeSlash || state.Tokens[3].Value != "bar" {
			t.Errorf("Expected distinct text and slash tokens for foo\\/bar in Sprint 9, got %+v", state.Tokens)
		}
	})

	t.Run("Escaped slash with bash option", func(t *testing.T) {
		// With bash option, \/ treats the slash as an escaped literal text sequence
		state, err := Parse("foo\\/bar", &ParseOptions{Bash: true})
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if len(state.Tokens) != 2 || state.Tokens[1].Value != "foo\\/bar" {
			t.Errorf("Expected single consolidated token 'foo\\/bar' when Bash=true, got %+v", state.Tokens)
		}
	})

	t.Run("Escaped dot and semicolon discard backslash", func(t *testing.T) {
		state, err := Parse("a\\.b\\;c", nil)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		// In Sprint 5, before dot grammar is implemented, dropping backslashes merges tokens into "a.b;c"
		if len(state.Tokens) != 2 || state.Tokens[1].Value != "a.b;c" {
			t.Errorf("Expected consolidated text token 'a.b;c' for escaped dot/semicolon, got %+v", state.Tokens)
		}
	})

	t.Run("Unescape option active", func(t *testing.T) {
		state, err := Parse("a\\-b", &ParseOptions{Unescape: true})
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if len(state.Tokens) != 2 || state.Tokens[1].Value != "a-b" {
			t.Errorf("Expected unescaped token 'a-b', got %+v", state.Tokens)
		}
	})
}

func TestParseLiterals_EscapedWildcardsAndDelimiters(t *testing.T) {
	wildcards := []string{"\\*", "\\?", "\\+", "\\!", "\\@", "\\(", "\\)", "\\{", "\\}", "\\[", "\\]", "\\|", "\\,"}
	for _, w := range wildcards {
		t.Run("Escaped "+w, func(t *testing.T) {
			state, err := Parse(w, nil)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}
			if len(state.Tokens) != 2 || state.Tokens[1].Type != TokenTypeText || state.Tokens[1].Value != w {
				t.Errorf("Expected escaped wildcard to emit single text token %q, got %+v", w, state.Tokens)
			}
		})
	}
}

func TestParseLiterals_TrailingBackslash(t *testing.T) {
	tests := []struct {
		pattern string
		expect  string
	}{
		{"\\", "\\\\"},
		{"foo\\", "foo\\\\"},
		{"a\\b\\", "a\\b\\\\"},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			state, err := Parse(tt.pattern, nil)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}
			if len(state.Tokens) != 2 || state.Tokens[1].Value != tt.expect {
				t.Errorf("Expected trailing backslash token value %q, got %+v", tt.expect, state.Tokens)
			}
		})
	}
}

func TestParseLiterals_ConsecutiveBackslashCollapse(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		opts     *ParseOptions
		expected string
	}{
		// 2 backslashes followed by 'a': slashes in remaining == 1 (not > 2). \\ escapes \ then 'a'.
		{"Two backslashes", "\\\\a", nil, "\\\\a"},
		// 4 backslashes followed by 'a': start at L0 (\), remaining has 3 backslashes (slashes=3 > 2).
		// slashes % 2 != 0 -> value += "\", index skips 3 -> lands on last \ -> Advance consumes 'a' -> "\\a".
		{"Four backslashes collapse", "\\\\\\\\a", nil, "\\\\a"},
		// 5 backslashes followed by 'a': start at L0 (\), remaining has 4 backslashes (slashes=4 > 2).
		// slashes % 2 == 0 -> index skips 4 -> Advance consumes 'a' -> "\\a".
		{"Five backslashes collapse", "\\\\\\\\\\a", nil, "\\a"},
		{"Four backslashes with unescape", "\\\\\\\\a", &ParseOptions{Unescape: true}, "a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state, err := Parse(tt.pattern, tt.opts)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}
			if len(state.Tokens) != 2 || state.Tokens[1].Value != tt.expected {
				t.Errorf("For pattern %q, expected collapsed value %q, got %+v", tt.pattern, tt.expected, state.Tokens)
			}
		})
	}
}

func TestParseLiterals_UTF8Input(t *testing.T) {
	tests := []string{
		"Hello 🚀 World",
		"日本語のテスト",
		"foo\\★bar",
		"✨🎉",
	}

	for _, pattern := range tests {
		t.Run(pattern, func(t *testing.T) {
			state, err := Parse(pattern, nil)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}
			if len(state.Tokens) < 2 {
				t.Fatalf("Expected tokens for UTF-8 input %q, got %d", pattern, len(state.Tokens))
			}
			// Verify output reconstructs identical string length and content without panic or byte corruption
			if state.Output != pattern {
				t.Errorf("Expected output %q, got %q", pattern, state.Output)
			}
		})
	}
}

func TestParseLiterals_NullBytes(t *testing.T) {
	t.Run("Unescaped null bytes are filtered by primary loop", func(t *testing.T) {
		state, err := Parse("abc\u0000def\u0000ghi", nil)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		expected := "abcdefghi"
		if len(state.Tokens) != 2 || state.Tokens[1].Value != expected {
			t.Errorf("Expected unescaped null bytes to be filtered leaving %q, got %+v", expected, state.Tokens)
		}
	})

	t.Run("Escaped null byte is directly consumed by escape handler", func(t *testing.T) {
		state, err := Parse("abc\u0000def\\\u0000ghi", nil)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		// In parse.js, advance() inside escape handling directly consumes the escaped byte without null filtering
		expected := "abcdef\\\u0000ghi"
		if len(state.Tokens) != 2 || state.Tokens[1].Value != expected {
			t.Errorf("Expected escaped null byte to be preserved as %q, got %+v", expected, state.Tokens)
		}
	})
}

func TestParseLiterals_TokenMerging(t *testing.T) {
	// Alternating plain text and escaped characters should consolidate into exactly 1 text token
	pattern := "foo\\-bar\\_baz123"
	state, err := Parse(pattern, nil)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(state.Tokens) != 2 {
		t.Fatalf("Expected token consolidation to yield exactly 2 tokens (BOS + text), got %d", len(state.Tokens))
	}
	if state.Tokens[1].Type != TokenTypeText || state.Tokens[1].Value != pattern {
		t.Errorf("Expected merged text token %q, got (%s, %q)", pattern, state.Tokens[1].Type, state.Tokens[1].Value)
	}
	if err := state.ValidateTokenLinks(); err != nil {
		t.Errorf("Token link invariant validation failed: %v", err)
	}
}

func TestParseLiterals_BracketFallthrough(t *testing.T) {
	// When state.Brackets > 0, HandleEscape must return false to allow fallthrough to character class traversal
	state := NewParseState("test", nil)
	state.Brackets = 1
	_, handled := HandleEscape(state, "\\")
	if handled {
		t.Errorf("HandleEscape must return false when state.Brackets > 0 to permit fallthrough")
	}
}

func TestParseLiterals_IsNonSpecialChar(t *testing.T) {
	specials := map[byte]bool{
		0: true, '@': true, '!': true, '[': true, ']': true, '.': true, ',': true,
		'$': true, '*': true, '+': true, '?': true, '^': true, '{': true,
		'}': true, '(': true, ')': true, '|': true, '\\': true, '/': true,
	}

	for i := 0; i < 256; i++ {
		b := byte(i)
		isNonSpecial := IsNonSpecialChar(b)
		expected := !specials[b]
		if isNonSpecial != expected {
			t.Errorf("IsNonSpecialChar(%q) = %v, expected %v", b, isNonSpecial, expected)
		}
	}
}

func TestParseLiterals_MalformedEscapes(t *testing.T) {
	// Test sequences where escapes hit boundary constraints or repeated symbols
	tests := []string{
		"\\\\\\\\\\\\\\\\",
		"a\\",
		"\\\\\\.",
		"\\\\\\;",
	}

	for _, pattern := range tests {
		t.Run(pattern, func(t *testing.T) {
			_, err := Parse(pattern, nil)
			if err != nil {
				t.Errorf("Parse(%q) panicked or returned error: %v", pattern, err)
			}
		})
	}
}
