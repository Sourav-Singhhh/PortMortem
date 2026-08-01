package picomatch

import (
	"fmt"
)

// Parse executes the main single-pass parser loop across a pattern string, generating an AST of tokens
// and compiling regular expression output in accordance with original parse.js control flow.
func Parse(pattern string, opts *ParseOptions) (*ParseState, error) {
	if opts == nil {
		opts = NewParseOptions()
	}

	if pattern == "***" {
		pattern = "*"
	} else if pattern == "**/**" || pattern == "**/**/**" {
		pattern = "**"
	}

	max := ParserMaxInputLength
	if opts.MaxLength > 0 && opts.MaxLength < ParserMaxInputLength {
		max = opts.MaxLength
	}
	if len(pattern) > max {
		return nil, fmt.Errorf(ErrInputExceeds, len(pattern), max)
	}

	state := NewParseState(pattern, opts)

	// Single-pass interleaved parser loop corresponding to while (!eos()) in parse.js:661
	for !state.EOS() {
		ch := state.Advance()

		// Silently bypass literal null bytes (parse.js:664)
		if ch == 0 {
			continue
		}

		// Convert raw byte to string via byte slice to guarantee uncorrupted reconstruction of multibyte UTF-8 characters
		tokVal := string([]byte{ch})

		// In parse.js:672, backslash escape handling precedes active character class evaluation (parse.js:718)
		if ch == '\\' {
			val, handled := HandleEscape(state, tokVal)
			if handled {
				continue
			}
			tokVal = val
		}

		// Architectural checks for active parser states preceding individual syntax switching:
		if handled, err := HandleBracketTraversal(state, tokVal); err != nil {
			return nil, err
		} else if handled {
			continue
		}

		// parse.js:765-773: Handle active quoted string literal traversal when state.Quotes == 1
		if state.Quotes == 1 && tokVal != "\"" {
			val := EscapeRegex(tokVal)
			if prev := state.CurrentToken(); prev != nil {
				prev.Value += val
			}
			state.Append(&ParseToken{Value: val})
			continue
		}

		// Switch structure precisely reflecting original parse.js syntactic branches
		switch ch {
		case '\\':
			// Backslashes are handled at parse.js:672 prior to active character class evaluation above
			state.PushToken(&ParseToken{Type: TokenTypeText, Value: tokVal})

		case '"':
			// parse.js:776-782: Double quote state toggling and keepQuotes option evaluation
			if state.Quotes == 1 {
				state.Quotes = 0
			} else {
				state.Quotes = 1
			}
			if state.Opts != nil && state.Opts.KeepQuotes {
				state.PushToken(NewParseToken(TokenTypeText, tokVal, ""))
			}

		case '(':
			if err := HandleOpenParen(state, tokVal); err != nil {
				return nil, err
			}

		case ')':
			if err := HandleCloseParen(state, tokVal); err != nil {
				return nil, err
			}

		case '[':
			if err := HandleOpenBracket(state, tokVal); err != nil {
				return nil, err
			}

		case ']':
			if err := HandleCloseBracket(state, tokVal); err != nil {
				return nil, err
			}

		case '{':
			if err := HandleOpenBrace(state, tokVal); err != nil {
				return nil, err
			}

		case '}':
			if err := HandleCloseBrace(state, tokVal); err != nil {
				return nil, err
			}

		case '|':
			if err := HandlePipe(state, tokVal); err != nil {
				return nil, err
			}

		case ',':
			if err := HandleBraceTraversal(state, tokVal); err != nil {
				return nil, err
			}

		case '/':
			if err := HandleSlash(state, tokVal); err != nil {
				return nil, err
			}

		case '.':
			if err := HandleDot(state, tokVal); err != nil {
				return nil, err
			}

		case '?':
			if handled, err := HandleExtglobPrefix(state, ch, tokVal); err != nil {
				return nil, err
			} else if handled {
				continue
			}
			if err := HandleQmark(state, tokVal); err != nil {
				return nil, err
			}

		case '!':
			if handled, err := HandleExtglobPrefix(state, ch, tokVal); err != nil {
				return nil, err
			} else if handled {
				continue
			}
			// parse.js:1061-1065: Negation prefix interpretation at BOS
			if (state.Opts == nil || !state.Opts.NoNegate) && state.Index == 0 {
				HandleNegate(state)
				continue
			}
			state.PushToken(NewParseToken(TokenTypeText, tokVal, ""))

		case '+':
			if handled, err := HandleExtglobPrefix(state, ch, tokVal); err != nil {
				return nil, err
			} else if handled {
				continue
			}
			// parse.js:1077-1088: Plus literal vs unescaped regex quantifier
			prev := state.CurrentToken()
			if prev != nil && prev.Value == "(" {
				chars := GetGlobChars(state.Opts != nil && state.Opts.Windows)
				state.PushToken(NewParseToken(TokenTypePlus, tokVal, chars.PlusLiteral))
				continue
			}
			if (prev != nil && (prev.Type == TokenTypeBracket || prev.Type == TokenTypeParen || prev.Type == TokenTypeBrace)) || state.Parens > 0 {
				state.PushToken(NewParseToken(TokenTypePlus, tokVal, ""))
				continue
			}
			chars := GetGlobChars(state.Opts != nil && state.Opts.Windows)
			state.PushToken(NewParseToken(TokenTypePlus, tokVal, chars.PlusLiteral))

		case '@':
			if handled, err := HandleExtglobPrefix(state, ch, tokVal); err != nil {
				return nil, err
			} else if handled {
				continue
			}
			// parse.js:1101-1108: '@' text symbol handling
			prev := state.CurrentToken()
			if (prev != nil && (prev.Type == TokenTypeBracket || prev.Type == TokenTypeParen || prev.Type == TokenTypeBrace)) || state.Parens > 0 {
				state.PushToken(NewParseToken(TokenTypeAt, tokVal, ""))
				continue
			}
			state.PushToken(NewParseToken(TokenTypeText, tokVal, ""))

		case '*':
			if handled, err := HandleExtglobPrefix(state, ch, tokVal); err != nil {
				return nil, err
			} else if handled {
				continue
			}
			if err := HandleStar(state, tokVal); err != nil {
				return nil, err
			}

		default:
			HandlePlainText(state, tokVal)
		}
	}

	// Post-processing and unclosed delimiter cleanup corresponding to parse.js:1286-1300
	if err := HandleUnclosedBrackets(state); err != nil {
		return nil, err
	}
	if err := HandleUnclosedParens(state); err != nil {
		return nil, err
	}
	if err := HandleUnclosedBraces(state); err != nil {
		return nil, err
	}

	// parse.js:1304-1306: Trailing maybe_slash pushing at EOF
	prev := state.CurrentToken()
	if (state.Opts == nil || !state.Opts.StrictSlashes) && prev != nil && (prev.Type == TokenTypeStar || prev.Type == TokenTypeBracket) {
		chars := GetGlobChars(state.Opts != nil && state.Opts.Windows)
		tok := NewParseToken(TokenTypeMaybeSlash, "", chars.SlashLiteral+"?")
		tok.OutputSet = true
		state.PushToken(tok)
	}

	// parse.js:1308-1319: Rebuild state.Output from token array if backtracking occurred at any point
	if state.Backtrack {
		state.Output = ""
		for _, token := range state.Tokens {
			if token.OutputSet || token.Output != "" {
				state.Output += token.Output
			} else {
				state.Output += token.Value
			}
			if token.Suffix != "" {
				state.Output += token.Suffix
			}
		}
	}

	return state, nil
}
