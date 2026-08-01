package picomatch

import (
	"testing"
)

func TestParseState_CursorEmptyInput(t *testing.T) {
	state := NewParseState("", nil)

	if !state.EOS() {
		t.Errorf("expected initial state on empty input to be at EOS immediately")
	}
	if ch := state.Advance(); ch != 0 {
		t.Errorf("Advance() on empty input must safely return 0 byte, got %q", ch)
	}
	if ch := state.Peek(0); ch != 0 {
		t.Errorf("Peek(0) on empty input must safely return 0 byte, got %q", ch)
	}
	if rem := state.Remaining(); rem != "" {
		t.Errorf("Remaining() on empty input must return empty string, got %q", rem)
	}
	if state.Index != 0 {
		t.Errorf("expected index to be incremented once by Advance() to 0, got %d", state.Index)
	}
	if !state.EOS() {
		t.Errorf("expected EOS to remain true after out-of-bounds advance")
	}
}

func TestParseState_CursorSequentialTraversal(t *testing.T) {
	input := "foo/bar"
	state := NewParseState(input, nil)

	if state.Index != -1 {
		t.Errorf("expected initial scanning offset to be -1, got %d", state.Index)
	}
	if state.EOS() {
		t.Errorf("non-empty input should not be at EOS initially")
	}
	if rem := state.Remaining(); rem != input {
		t.Errorf("initial Remaining() should match full input %q, got %q", input, rem)
	}

	// Advance through "f-o-o"
	expectedBytes := []byte{'f', 'o', 'o', '/', 'b', 'a', 'r'}
	for i, exp := range expectedBytes {
		if ch := state.Peek(1); ch != exp {
			t.Errorf("Peek(1) at iteration %d expected %q, got %q", i, exp, ch)
		}
		ch := state.Advance()
		if ch != exp {
			t.Errorf("Advance() at index %d expected %q, got %q", i, exp, ch)
		}
		if state.Index != i {
			t.Errorf("expected index tracking %d, got %d", i, state.Index)
		}
		if i < len(expectedBytes)-1 && state.EOS() {
			t.Errorf("premature EOS transition at character index %d", i)
		}
	}

	if !state.EOS() {
		t.Errorf("expected state to transition to EOS upon consuming final character")
	}

	// Verify out-of-bounds protection after reaching EOF
	if ch := state.Advance(); ch != 0 {
		t.Errorf("Advance() beyond EOF must safely return 0 byte, got %q", ch)
	}
}

func TestParseState_CursorConsumeAndRemaining(t *testing.T) {
	input := "alpha.beta"
	state := NewParseState(input, nil)

	// Consume first segment without character-by-character Advance loops
	state.Consume("alpha", 5)
	if state.Consumed != "alpha" {
		t.Errorf("expected Consumed accumulator %q, got %q", "alpha", state.Consumed)
	}
	if state.Index != 4 {
		t.Errorf("expected scanning offset index to move from -1 to 4 (5 offsets), got %d", state.Index)
	}
	if rem := state.Remaining(); rem != ".beta" {
		t.Errorf("expected Remaining() slice after consuming 5 characters to be %q, got %q", ".beta", rem)
	}

	// Advance past the dot delimiter
	dot := state.Advance()
	if dot != '.' || state.Index != 5 {
		t.Errorf("expected Advance() to consume dot delimiter at index 5, got %q at index %d", dot, state.Index)
	}
	if rem := state.Remaining(); rem != "beta" {
		t.Errorf("expected Remaining() after dot delimiter to be %q, got %q", "beta", rem)
	}
}

func TestParseState_CursorOutOfBoundsProtection(t *testing.T) {
	input := "a/b"
	state := NewParseState(input, nil)

	// Extreme negative offsets
	if ch := state.Peek(-50); ch != 0 {
		t.Errorf("Peek(-50) out of bounds must return 0 byte, got %q", ch)
	}
	// Extreme positive offsets
	if ch := state.Peek(500); ch != 0 {
		t.Errorf("Peek(500) out of bounds must return 0 byte, got %q", ch)
	}
	if ch := state.Peek(4); ch != 0 {
		t.Errorf("Peek(4) beyond length must return 0 byte, got %q", ch)
	}

	// Move index explicitly far beyond EOF
	state.Index = 1000
	if !state.EOS() {
		t.Errorf("expected EOS to be true when Index (%d) far exceeds input length", state.Index)
	}
	if rem := state.Remaining(); rem != "" {
		t.Errorf("Remaining() when index far out of bounds must return empty string, got %q", rem)
	}
}

func TestParseState_CursorNullByteAndUnicode(t *testing.T) {
	// Confirm cursor helpers pass through raw null bytes and special formatting characters faithfully
	// without enforcing grammar stops or premature EOF categorization.
	input := "val\u0000ue"
	state := NewParseState(input, nil)

	for i := 0; i < 3; i++ {
		state.Advance()
	}
	if rem := state.Remaining(); rem != "\u0000ue" {
		t.Errorf("expected Remaining() before null byte to be %q, got %q", "\u0000ue", rem)
	}
	nullByte := state.Advance()
	if nullByte != 0 {
		t.Errorf("expected Advance() on explicit string null byte to return 0, got %v", nullByte)
	}
	if state.EOS() {
		t.Errorf("encountering explicit string null byte must not trigger premature EOS")
	}
	if rem := state.Remaining(); rem != "ue" {
		t.Errorf("expected Remaining() after null byte to be %q, got %q", "ue", rem)
	}
}

func TestParseState_CursorExhaustiveBoundaryAndUTF8Parity(t *testing.T) {
	// 1. Rigorous boundary verification: index = -1, 0, len-1, len, > len
	input := "abcd" // len == 4
	state := NewParseState(input, nil)

	// index = -1 (initial state)
	if state.Index != -1 || state.EOS() {
		t.Errorf("boundary failure at index -1: Index=%d, EOS=%v", state.Index, state.EOS())
	}
	if rem := state.Remaining(); rem != "abcd" {
		t.Errorf("Remaining() at index -1 expected %q, got %q", "abcd", rem)
	}

	// index = 0
	state.Index = 0
	if state.EOS() || state.Peek(0) != 'a' || state.Remaining() != "bcd" {
		t.Errorf("boundary failure at index 0: EOS=%v, Peek(0)=%q, Remaining=%q", state.EOS(), state.Peek(0), state.Remaining())
	}

	// index = len - 1 (3)
	state.Index = len(input) - 1
	if !state.EOS() || state.Peek(0) != 'd' || state.Remaining() != "" {
		t.Errorf("boundary failure at index len-1: EOS=%v, Peek(0)=%q, Remaining=%q", state.EOS(), state.Peek(0), state.Remaining())
	}

	// index = len (4, exact EOF boundary)
	state.Index = len(input)
	if !state.EOS() || state.Peek(0) != 0 || state.Advance() != 0 || state.Remaining() != "" {
		t.Errorf("boundary failure at index len: EOS=%v, Peek(0)=%v, Advance()=%v, Remaining=%q", state.EOS(), state.Peek(0), state.Advance(), state.Remaining())
	}

	// index > len (e.g., 10)
	state.Index = 10
	if !state.EOS() || state.Peek(0) != 0 || state.Advance() != 0 || state.Remaining() != "" {
		t.Errorf("boundary failure at index > len: EOS=%v, Peek(0)=%v, Advance()=%v, Remaining=%q", state.EOS(), state.Peek(0), state.Advance(), state.Remaining())
	}

	// 2. Multibyte UTF-8 characters parity test
	utf8Input := "foo/★/🚀.txt"
	stateUTF8 := NewParseState(utf8Input, nil)

	for !stateUTF8.EOS() {
		stateUTF8.Advance()
	}
	if stateUTF8.Index != len(utf8Input)-1 {
		t.Errorf("expected sequential traversal of UTF-8 string to end at index %d, got %d", len(utf8Input)-1, stateUTF8.Index)
	}

	stateSlice := NewParseState("a★b", nil)
	stateSlice.Advance() // consume 'a' at index 0
	if rem := stateSlice.Remaining(); rem != "★b" {
		t.Errorf("Remaining() after ASCII char in UTF-8 string expected %q, got %q", "★b", rem)
	}
	stateSlice.Advance() // byte 1
	stateSlice.Advance() // byte 2
	stateSlice.Advance() // byte 3
	if stateSlice.Index != 3 || stateSlice.Remaining() != "b" {
		t.Errorf("Remaining() after 3-byte UTF-8 char expected %q, got %q (Index=%d)", "b", stateSlice.Remaining(), stateSlice.Index)
	}
}
