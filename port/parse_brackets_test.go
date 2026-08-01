package picomatch

import (
	"strings"
	"testing"
)

func TestParseBrackets_BasicAndUnclosed(t *testing.T) {
	t.Run("Basic closed bracket expression", func(t *testing.T) {
		state, err := Parse("foo[bar]baz", nil)
		if err != nil {
			t.Fatalf("Unexpected parse error: %v", err)
		}
		if state.Brackets != 0 {
			t.Errorf("Expected Brackets==0 at EOF, got %d", state.Brackets)
		}
		// Notice that during foundation before regex generation, [bar] remains uncompiled
		if !strings.Contains(state.Output, "[bar]") {
			t.Errorf("Expected output to contain '[bar]', got %q", state.Output)
		}
	})

	t.Run("Unclosed opening bracket without closing bracket in remaining", func(t *testing.T) {
		state, err := Parse("foo[bar", nil)
		if err != nil {
			t.Fatalf("Unexpected parse error: %v", err)
		}
		// Since remaining does not contain ']', '[' is immediately escaped to '\\['
		if !strings.Contains(state.Output, "\\[") {
			t.Errorf("Expected unclosed bracket to be escaped as '\\[', got %q", state.Output)
		}
	})

	t.Run("Empty brackets [] escape to \\[\\]", func(t *testing.T) {
		state, err := Parse("[]", nil)
		if err != nil {
			t.Fatalf("Unexpected parse error: %v", err)
		}
		if state.Output != "\\[\\]" {
			t.Errorf("Expected empty brackets to produce '\\[\\]', got %q", state.Output)
		}
	})

	t.Run("Negated bracket without slash automatically injects slash", func(t *testing.T) {
		state, err := Parse("[^abc]", nil)
		if err != nil {
			t.Fatalf("Unexpected parse error: %v", err)
		}
		if !strings.Contains(state.Output, "[^abc/]") {
			t.Errorf("Expected negated character class to inject path slash as '[^abc/]', got %q", state.Output)
		}
	})

	t.Run("Negated bracket with existing slash does not double inject", func(t *testing.T) {
		state, err := Parse("[^abc/]", nil)
		if err != nil {
			t.Fatalf("Unexpected parse error: %v", err)
		}
		if !strings.Contains(state.Output, "[^abc/]") || strings.Contains(state.Output, "//") {
			t.Errorf("Expected single slash in negated class, got %q", state.Output)
		}
	})
}

func TestParseBrackets_EscapedAndNested(t *testing.T) {
	t.Run("Escaped hyphen inside bracket class", func(t *testing.T) {
		state, err := Parse("[a\\-z]", nil)
		if err != nil {
			t.Fatalf("Unexpected parse error: %v", err)
		}
		if !strings.Contains(state.Output, "[a\\-z]") {
			t.Errorf("Expected escaped hyphen to be preserved inside class, got %q", state.Output)
		}
	})

	t.Run("Escaped closing bracket inside character class does not terminate class", func(t *testing.T) {
		state, err := Parse("[a\\]b]", nil)
		if err != nil {
			t.Fatalf("Unexpected parse error: %v", err)
		}
		if state.Brackets != 0 {
			t.Errorf("Expected Brackets==0 after real closing bracket, got %d", state.Brackets)
		}
		if !strings.Contains(state.Output, "[a\\]b]") {
			t.Errorf("Expected escaped closing bracket inside class '[a\\]b]', got %q", state.Output)
		}
	})

	t.Run("Closing bracket immediately after open bracket escapes without terminating", func(t *testing.T) {
		state, err := Parse("[\\]]", nil)
		if err != nil {
			t.Fatalf("Unexpected parse error: %v", err)
		}
		if state.Brackets != 0 {
			t.Errorf("Expected clean closure, got brackets %d", state.Brackets)
		}
	})
}

func TestParseBrackets_Options(t *testing.T) {
	t.Run("NoBracket option disables character class tracking", func(t *testing.T) {
		state, err := Parse("foo[bar]baz", &ParseOptions{NoBracket: true})
		if err != nil {
			t.Fatalf("Unexpected parse error: %v", err)
		}
		if state.Brackets != 0 {
			t.Errorf("Expected Brackets==0 when NoBracket=true, got %d", state.Brackets)
		}
		if !strings.Contains(state.Output, "\\[") || !strings.Contains(state.Output, "\\]") {
			t.Errorf("Expected brackets to be escaped when NoBracket is active, got %q", state.Output)
		}
	})

	t.Run("StrictBrackets throws syntax error on unclosed opening bracket", func(t *testing.T) {
		_, err := Parse("foo[bar", &ParseOptions{StrictBrackets: true})
		if err == nil {
			t.Fatalf("Expected syntax error with StrictBrackets on unclosed opening bracket")
		}
		expectedErr := `Missing closing: "]" - use "\]" to match literal characters`
		if err.Error() != expectedErr {
			t.Errorf("Expected error message %q, got %q", expectedErr, err.Error())
		}
	})

	t.Run("StrictBrackets throws syntax error on unmatched closing bracket", func(t *testing.T) {
		_, err := Parse("foo]bar", &ParseOptions{StrictBrackets: true})
		if err == nil {
			t.Fatalf("Expected syntax error with StrictBrackets on unmatched closing bracket")
		}
		expectedErr := `Missing opening: "[" - use "\[" to match literal characters`
		if err.Error() != expectedErr {
			t.Errorf("Expected error message %q, got %q", expectedErr, err.Error())
		}
	})

	t.Run("Unmatched closing bracket without StrictBrackets emits escaped text", func(t *testing.T) {
		state, err := Parse("foo]bar", nil)
		if err != nil {
			t.Fatalf("Unexpected parse error: %v", err)
		}
		if !strings.Contains(state.Output, "foo\\]bar") {
			t.Errorf("Expected escaped unmatched closing bracket 'foo\\]bar', got %q", state.Output)
		}
	})

	t.Run("Posix negation [! translates to [^ when Posix option is true", func(t *testing.T) {
		state, err := Parse("[!abc]", &ParseOptions{Posix: true})
		if err != nil {
			t.Fatalf("Unexpected parse error: %v", err)
		}
		if !strings.Contains(state.Output, "[^abc/]") {
			t.Errorf("Expected posix negation to translate to '[^abc/]', got %q", state.Output)
		}
	})
}

func TestParseBrackets_EscapeLast(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		ch      byte
		lastIdx int
		expect  string
	}{
		{"Empty string", "", '[', 10, ""},
		{"Negative index", "abc", '[', -5, "abc"},
		{"Out of bounds index defaults to len-1", "foo[bar", '[', 100, "foo\\[bar"},
		{"Skip already escaped occurrence", "foo\\[bar[baz", '[', 12, "foo\\[bar\\[baz"},
		{"Target character not found", "abcdef", '[', 5, "abcdef"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EscapeLast(tt.input, tt.ch, tt.lastIdx)
			if got != tt.expect {
				t.Errorf("EscapeLast(%q, %q, %d) = %q, expected %q", tt.input, tt.ch, tt.lastIdx, got, tt.expect)
			}
		})
	}
}

func TestParseBrackets_SyntaxError(t *testing.T) {
	err := SyntaxError("closing", "]")
	expected := `Missing closing: "]" - use "\]" to match literal characters`
	if err.Error() != expected {
		t.Errorf("SyntaxError output %q != expected %q", err.Error(), expected)
	}
}
