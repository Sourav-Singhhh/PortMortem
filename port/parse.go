package picomatch

// Parse executes the main single-pass parser loop across a pattern string, generating an AST of tokens
// and compiling regular expression output in accordance with original parse.js control flow.
//
// In this architectural skeleton milestone, all syntactic grammar evaluation, regex transformation,
// extglob tracking, and delimiter matching are deferred via explicit TODO markers. The parser loop
// purely traverses character by character, maintains safe boundaries, and terminates cleanly.
func Parse(pattern string, opts *ParseOptions) (*ParseState, error) {
	if opts == nil {
		opts = NewParseOptions()
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
		// TODO: Handle active quoted string literal traversal when state.Quotes == 1 (parse.js:765)

		// Switch structure precisely reflecting original parse.js syntactic branches
		switch ch {
		case '\\':
			// Backslashes are handled at parse.js:672 prior to active character class evaluation above
			state.PushToken(&ParseToken{Type: TokenTypeText, Value: tokVal})

		case '"':
			// TODO: Implement double quote state toggling and keepQuotes option evaluation (parse.js:776)
			state.PushToken(&ParseToken{Type: TokenTypeText, Value: tokVal})

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
			// TODO: Implement path slash tokenization and leading "./" prefix stripping (parse.js:975)
			state.PushToken(&ParseToken{Type: TokenTypeText, Value: tokVal})

		case '.':
			// TODO: Implement dot tokenization, directory dot rules, and consecutive dots ".." range formation (parse.js:997)
			state.PushToken(&ParseToken{Type: TokenTypeText, Value: tokVal})

		case '?':
			if handled, err := HandleExtglobPrefix(state, ch, tokVal); err != nil {
				return nil, err
			} else if handled {
				continue
			}
			// TODO: Implement question mark wildcard semantics (parse.js:1028-1046)
			state.PushToken(&ParseToken{Type: TokenTypeText, Value: tokVal})

		case '!':
			if handled, err := HandleExtglobPrefix(state, ch, tokVal); err != nil {
				return nil, err
			} else if handled {
				continue
			}
			// TODO: Implement negation prefix interpretation (parse.js:1061)
			state.PushToken(&ParseToken{Type: TokenTypeText, Value: tokVal})

		case '+':
			if handled, err := HandleExtglobPrefix(state, ch, tokVal); err != nil {
				return nil, err
			} else if handled {
				continue
			}
			// TODO: Implement plus literal interpretation (parse.js:1077-1088)
			state.PushToken(&ParseToken{Type: TokenTypeText, Value: tokVal})

		case '@':
			if handled, err := HandleExtglobPrefix(state, ch, tokVal); err != nil {
				return nil, err
			} else if handled {
				continue
			}
			// TODO: Implement '@' text symbol handling (parse.js:1101)
			state.PushToken(&ParseToken{Type: TokenTypeText, Value: tokVal})

		case '*':
			if handled, err := HandleExtglobPrefix(state, ch, tokVal); err != nil {
				return nil, err
			} else if handled {
				continue
			}
			// TODO: Implement star/globstar wildcard evaluation, consecutive "/**/" stripping, and regex generation (parse.js:1125)
			state.PushToken(&ParseToken{Type: TokenTypeText, Value: tokVal})

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

	return state, nil
}
