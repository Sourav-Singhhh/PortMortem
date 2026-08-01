package picomatch

// isRegexGroupChar checks if a byte matches regex capture or lookaround identifiers /[!=<:]/ (parse.js:1055).
func isRegexGroupChar(b byte) bool {
	return b == '!' || b == '=' || b == '<' || b == ':'
}

// HandleExtglobOpen executes extglob opening state tracking, stack pushes, and token initialization.
// Matches original picomatch/lib/parse.js lines 523-537.
func HandleExtglobOpen(s *ParseState, extType TokenType, value string) error {
	outIdx := len(s.Output)
	tokIdx := len(s.Tokens)
	startIdx := s.Index
	parens := s.Parens
	output := s.Output

	tok := NewParseToken(extType, value, "")
	tok.Extglob = true
	tok.OutputIndex = outIdx
	tok.TokensIndex = tokIdx

	ext := NewExtglobState(tok, extType, value, parens, startIdx, tokIdx)
	ext.Output = output
	ext.Conditions = 1
	ext.Inner = ""

	s.Increment(ParserContextParens)
	s.PushToken(tok)

	parenCh := s.Advance()
	parenTok := NewParseToken(TokenTypeParen, string([]byte{parenCh}), "(")
	parenTok.Extglob = true
	s.PushToken(parenTok)

	s.ExtglobStack.Push(ext)
	return nil
}

// HandleExtglobClose executes extglob closing state transitions and stack pops.
// Matches original picomatch/lib/parse.js lines 539-600.
func HandleExtglobClose(s *ParseState, ext *ExtglobState, value string) error {
	// TODO: Implement ReDoS repeated extglob analysis and rollback fallback (parse.js:542-566)
	// TODO: Implement wildcard globstar and regex closing synthesis (parse.js:568-591)

	if ext.Type == TokenTypeNegate {
		if ext.Token != nil && ext.Token.Prev != nil && ext.Token.Prev.Type == TokenTypeBos {
			s.NegatedExtglob = true
		}
	}

	tok := NewParseToken(TokenTypeParen, value, ")")
	tok.Extglob = true
	s.PushToken(tok)

	s.Decrement(ParserContextParens)
	return nil
}

// HandleExtglobPrefix evaluates whether an operator symbol (?, !, +, @, *) initiates an extended glob expression.
// Matches original picomatch/lib/parse.js lines 1021-1026, 1053-1059, 1071-1075, 1095-1099, 1139-1143.
func HandleExtglobPrefix(s *ParseState, ch byte, value string) (bool, error) {
	if s.Opts != nil && s.Opts.NoExtglob {
		return false, nil
	}

	if s.Peek(1) != '(' {
		return false, nil
	}

	switch ch {
	case '?':
		// parse.js:1022-1026: DO NOT initiate extglob if previous token was an opening paren group or if peek(2) is '?'
		prev := s.CurrentToken()
		isGroup := prev != nil && prev.Value == "("
		if !isGroup && s.Peek(2) != '?' {
			if err := HandleExtglobOpen(s, TokenTypeQmark, value); err != nil {
				return true, err
			}
			return true, nil
		}

	case '!':
		// parse.js:1054-1059: Exclude regular expression lookahead/capture syntaxes such as !(?: or !(?!
		if s.Peek(2) != '?' || !isRegexGroupChar(s.Peek(3)) {
			if err := HandleExtglobOpen(s, TokenTypeNegate, value); err != nil {
				return true, err
			}
			return true, nil
		}

	case '+':
		// parse.js:1072-1075: Initiate plus extglob unless followed by question mark (+(?...)
		if s.Peek(2) != '?' {
			if err := HandleExtglobOpen(s, TokenTypePlus, value); err != nil {
				return true, err
			}
			return true, nil
		}

	case '@':
		// parse.js:1096-1099: In upstream parse.js, '@(...)' does not invoke extglobOpen; it pushes a stripped '@' token marked with extglob:true
		if s.Peek(2) != '?' {
			tok := NewParseToken(TokenTypeAt, value, "")
			tok.Extglob = true
			s.PushToken(tok)
			return true, nil
		}

	case '*':
		// parse.js:1140-1143: Matches /^\([^?]/.test(remaining()), requiring '(' followed by any character other than '?' or EOF
		if s.Peek(2) != 0 && s.Peek(2) != '?' {
			if err := HandleExtglobOpen(s, TokenTypeStar, value); err != nil {
				return true, err
			}
			return true, nil
		}
	}

	return false, nil
}

// HandleOpenParen evaluates regular opening parenthesis state transitions and tracking stack pushes.
// Matches original picomatch/lib/parse.js lines 788-792.
func HandleOpenParen(s *ParseState, value string) error {
	s.Increment(ParserContextParens)
	s.PushToken(NewParseToken(TokenTypeParen, value, value))
	return nil
}

// HandleCloseParen evaluates closing parenthesis transitions, extglob termination matching, and syntax validation.
// Matches original picomatch/lib/parse.js lines 794-808.
func HandleCloseParen(s *ParseState, value string) error {
	if s.Parens == 0 && s.Opts != nil && s.Opts.StrictBrackets {
		return SyntaxError("opening", "(")
	}

	if ext, ok := s.ExtglobStack.Peek(); ok && s.Parens == ext.Parens+1 {
		s.ExtglobStack.Pop()
		return HandleExtglobClose(s, ext, value)
	}

	output := "\\)"
	if s.Parens > 0 {
		output = ")"
	}

	s.PushToken(NewParseToken(TokenTypeParen, value, output))
	if s.Parens > 0 {
		s.Decrement(ParserContextParens)
	}
	return nil
}

// HandlePipe evaluates pipe ('|') symbols, incrementing active extglob condition counts and tokenizing as text.
// Matches original picomatch/lib/parse.js lines 946-952.
func HandlePipe(s *ParseState, value string) error {
	if !s.ExtglobStack.IsEmpty() {
		if ext, ok := s.ExtglobStack.Peek(); ok {
			ext.Conditions++
		}
	}
	s.PushToken(NewParseToken(TokenTypeText, value, value))
	return nil
}

// HandleUnclosedParens checks open parenthesis balances at EOF, reconciles unclosed extglob constructs, and escapes trailing unclosed parentheses.
// Matches original picomatch/lib/parse.js lines 1292-1296.
func HandleUnclosedParens(s *ParseState) error {
	for s.Parens > 0 {
		if s.Opts != nil && s.Opts.StrictBrackets {
			return SyntaxError("closing", ")")
		}
		s.Output = EscapeLast(s.Output, '(', len(s.Output)-1)
		s.Decrement(ParserContextParens)
		if !s.ExtglobStack.IsEmpty() {
			if ext, ok := s.ExtglobStack.Peek(); ok && ext.Parens == s.Parens {
				s.ExtglobStack.Pop()
			}
		}
	}
	s.ExtglobStack.Clear()
	return nil
}
