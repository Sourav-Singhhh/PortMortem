package picomatch

import (
	"fmt"
	"strings"
)

// SyntaxError generates an error formatted identically to syntaxError in parse.js:44-46.
func SyntaxError(typ string, char string) error {
	return fmt.Errorf(`Missing %s: "%s" - use "\%s" to match literal characters`, typ, char, char)
}

// EscapeLast prefixes the last unescaped occurrence of ch up to lastIdx with a backslash.
// Matches original picomatch/lib/utils.js exports.escapeLast lines 36-41.
func EscapeLast(input string, ch byte, lastIdx int) string {
	if len(input) == 0 {
		return input
	}
	if lastIdx < 0 || lastIdx >= len(input) {
		lastIdx = len(input) - 1
	}
	for i := lastIdx; i >= 0; i-- {
		if input[i] == ch {
			if i > 0 && input[i-1] == '\\' {
				continue
			}
			return input[:i] + "\\" + input[i:]
		}
	}
	return input
}

// HandleBracketTraversal executes active regex character class traversal when s.Brackets > 0.
// Matches original picomatch/lib/parse.js lines 718-758.
func HandleBracketTraversal(s *ParseState, value string) (bool, error) {
	if s.Brackets > 0 {
		prev := s.CurrentToken()
		if value != "]" || (prev != nil && (prev.Value == "[" || prev.Value == "[^")) {
			// TODO: Implement POSIX character class table translation (parse.js:719-741)

			if (value == "[" && s.Peek(1) != ':') || (value == "-" && s.Peek(1) == ']') {
				value = "\\" + value
			}

			if value == "]" && prev != nil && (prev.Value == "[" || prev.Value == "[^") {
				value = "\\" + value
			}

			if s.Opts != nil && s.Opts.Posix && value == "!" && prev != nil && prev.Value == "[" {
				value = "^"
			}

			if prev != nil {
				prev.Value += value
			}
			s.Append(&ParseToken{Value: value})
			return true, nil
		}
	}
	return false, nil
}

// HandleOpenBracket evaluates opening square bracket state transitions and syntax error constraints.
// Matches original picomatch/lib/parse.js lines 814-827.
func HandleOpenBracket(s *ParseState, value string) error {
	if (s.Opts != nil && s.Opts.NoBracket) || !strings.Contains(s.Remaining(), "]") {
		if (s.Opts == nil || !s.Opts.NoBracket) && (s.Opts != nil && s.Opts.StrictBrackets) {
			return SyntaxError("closing", "]")
		}
		value = "\\" + value
	} else {
		s.Increment("brackets")
	}

	s.PushToken(NewParseToken(TokenTypeBracket, value, ""))
	return nil
}

// HandleCloseBracket evaluates closing square bracket state transitions and opening balance verification.
// Matches original picomatch/lib/parse.js lines 829-875.
func HandleCloseBracket(s *ParseState, value string) error {
	prev := s.CurrentToken()
	if (s.Opts != nil && s.Opts.NoBracket) || (prev != nil && prev.Type == TokenTypeBracket && len(prev.Value) == 1) {
		tok := NewParseToken(TokenTypeText, value, "")
		tok.Output = "\\" + value
		tok.OutputSet = true
		s.PushToken(tok)
		return nil
	}

	if s.Brackets == 0 {
		if s.Opts != nil && s.Opts.StrictBrackets {
			return SyntaxError("opening", "[")
		}
		tok := NewParseToken(TokenTypeText, value, "")
		tok.Output = "\\" + value
		tok.OutputSet = true
		s.PushToken(tok)
		return nil
	}

	s.Decrement("brackets")

	if prev != nil {
		if len(prev.Value) > 1 {
			prevValue := prev.Value[1:]
			if !prev.Posix && len(prevValue) > 0 && prevValue[0] == '^' && !strings.Contains(prevValue, "/") {
				value = "/" + value
			}
		}
		prev.Value += value
	}
	s.Append(&ParseToken{Value: value})

	// TODO: Implement regex character class generation, hasRegexChars evaluation, and literalBrackets option rewriting (parse.js:854-874)
	return nil
}

// HandleUnclosedBrackets checks open bracket balances at EOF and escapes trailing unclosed brackets.
// Matches original picomatch/lib/parse.js lines 1286-1290.
func HandleUnclosedBrackets(s *ParseState) error {
	for s.Brackets > 0 {
		if s.Opts != nil && s.Opts.StrictBrackets {
			return SyntaxError("closing", "]")
		}
		s.Output = EscapeLast(s.Output, '[', len(s.Output)-1)
		s.Decrement("brackets")
	}
	return nil
}
