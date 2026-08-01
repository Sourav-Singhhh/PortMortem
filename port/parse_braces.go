package picomatch

// HandleOpenBrace evaluates opening curly brace state transitions, token initialization, and tracking stack pushes.
// Matches original picomatch/lib/parse.js lines 881-895.
func HandleOpenBrace(s *ParseState, value string) error {
	if s.Opts != nil && s.Opts.NoBrace {
		HandlePlainText(s, value)
		return nil
	}

	s.Increment(ParserContextBraces)

	open := NewParseToken(TokenTypeBrace, value, "(")
	outIdx := len(s.Output)
	tokIdx := len(s.Tokens)
	open.OutputIndex = outIdx
	open.TokensIndex = tokIdx

	brace := NewBraceState(open, value, "(", outIdx, tokIdx)
	s.BraceStack.Push(brace)
	s.PushToken(open)
	return nil
}

// HandleCloseBrace evaluates closing curly brace state transitions, range expansion evaluation, and backtracking fallback.
// Matches original picomatch/lib/parse.js lines 897-940.
func HandleCloseBrace(s *ParseState, value string) error {
	brace, ok := s.BraceStack.Peek()
	if (s.Opts != nil && s.Opts.NoBrace) || !ok {
		tok := NewParseToken(TokenTypeText, value, value)
		s.PushToken(tok)
		return nil
	}

	output := ")"

	// TODO: Implement brace range expansion via dots evaluation and backtracking (parse.js:907-923)
	if brace.Dots {
		// Deferred to subsequent range expansion milestone
	}

	if !brace.Comma && !brace.Dots {
		outIdx := brace.OutputIndex
		if outIdx < 0 || outIdx > len(s.Output) {
			outIdx = len(s.Output)
		}
		tokIdx := brace.TokensIndex
		if tokIdx < 0 || tokIdx > len(s.Tokens) {
			tokIdx = len(s.Tokens)
		}

		out := s.Output[:outIdx]
		toks := s.Tokens[tokIdx:]

		brace.Value = "\\{"
		brace.Output = "\\{"
		if brace.Token != nil {
			brace.Token.Value = "\\{"
			brace.Token.Output = "\\{"
			brace.Token.OutputSet = true
		}

		value = "\\}"
		output = "\\}"
		s.Output = out

		for _, t := range toks {
			if t.OutputSet || t.Output != "" {
				s.Output += t.Output
			} else {
				s.Output += t.Value
			}
		}
	}

	s.PushToken(NewParseToken(TokenTypeBrace, value, output))
	s.Decrement(ParserContextBraces)
	s.BraceStack.Pop()
	return nil
}

// HandleBraceTraversal evaluates comma delimiter tracking inside active brace structures.
// Matches original picomatch/lib/parse.js lines 958-969.
func HandleBraceTraversal(s *ParseState, value string) error {
	output := value
	if brace, ok := s.BraceStack.Peek(); ok {
		if top, okStack := s.Stack.Peek(); okStack && top == ParserContextBraces {
			brace.Comma = true
			if brace.Token != nil {
				brace.Token.Comma = true
			}
			output = "|"
		}
	}

	s.PushToken(NewParseToken(TokenTypeComma, value, output))
	return nil
}

// HandleUnclosedBraces checks open brace balances at EOF and escapes trailing unclosed braces.
// Matches original picomatch/lib/parse.js lines 1298-1302.
func HandleUnclosedBraces(s *ParseState) error {
	for s.Braces > 0 {
		if s.Opts != nil && s.Opts.StrictBrackets {
			return SyntaxError("closing", "}")
		}
		s.Output = EscapeLast(s.Output, '{', len(s.Output)-1)
		s.Decrement(ParserContextBraces)
		s.BraceStack.Pop()
	}
	return nil
}
