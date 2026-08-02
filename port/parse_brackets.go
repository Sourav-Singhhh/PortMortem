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
			// parse.js:719-741: POSIX character class table translation
			if (s.Opts == nil || s.Opts.Posix) && value == ":" && prev != nil && len(prev.Value) > 1 {
				inner := prev.Value[1:]
				if strings.Contains(inner, "[") {
					prev.Posix = true
					if strings.Contains(inner, ":") {
						idx := strings.LastIndex(prev.Value, "[")
						if idx >= 0 && idx+2 <= len(prev.Value) {
							pre := prev.Value[:idx]
							rest := prev.Value[idx+2:]
							if posix, ok := GetPosixRegexSource(rest); ok {
								prev.Value = pre + posix
								s.Backtrack = true
								s.Advance()

								if len(s.Tokens) > 1 && s.Tokens[1] == prev {
									bos := s.Tokens[0]
									if !bos.OutputSet && bos.Output == "" {
										chars := GetGlobChars(s.Opts != nil && s.Opts.Windows)
										bos.Output = chars.OneChar
										bos.OutputSet = true
									}
								}
								return true, nil
							}
						}
					}
				}
			}

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

	tok := NewParseToken(TokenTypeBracket, value, "")
	s.PushToken(tok)
	return nil
}

// HandleCloseBracket evaluates closing square bracket state transitions, path slash injection, and character class formation.
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
		prevValue := ""
		if len(prev.Value) > 1 {
			prevValue = prev.Value[1:]
			if !prev.Posix && len(prevValue) > 0 && prevValue[0] == '^' && !strings.Contains(prevValue, "/") {
				value = "/" + value
			}
		}
		prev.Value += value
		s.Append(&ParseToken{Value: value})

		// parse.js:854-874: Regex character class generation and literalBrackets option rewriting
		if len(prevValue) > 0 {
			if HasRegexChars(prevValue) {
				return nil
			}

			escaped := EscapeRegex(prev.Value)
			if len(s.Output) >= len(prev.Value) {
				s.Output = s.Output[:len(s.Output)-len(prev.Value)]
			}

			if s.Opts != nil && s.Opts.LiteralBrackets {
				s.Output += escaped
				prev.Value = escaped
				return nil
			}

			capture := "?:"
			if s.Opts != nil && s.Opts.Capture {
				capture = ""
			}
			prev.Value = "(" + capture + escaped + "|" + prev.Value + ")"
			s.Output += prev.Value
		}
		return nil
	}
	s.Append(&ParseToken{Value: value})
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
