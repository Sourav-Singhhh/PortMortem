package picomatch

// HandleSlash evaluates path separator slash syntax, stripping leading "./" prefixes to optimize lookbehinds.
// Matches original picomatch/lib/parse.js lines 975-991.
func HandleSlash(s *ParseState, value string) error {
	prev := s.CurrentToken()
	// parse.js:980-987: if the beginning of the glob is "./", advance the start to current index,
	// and don't add the "./" characters to state. This greatly simplifies lookbehinds at BOS.
	if prev != nil && prev.Type == TokenTypeDot && s.Index == s.Start+1 {
		s.Start = s.Index + 1
		s.Consumed = ""
		s.Output = ""
		if len(s.Tokens) > 0 {
			s.Tokens = s.Tokens[:len(s.Tokens)-1]
		}
		// Reset "prev" to the first token (BOS), which naturally resides at s.CurrentToken() after pop
		return nil
	}

	// TODO: Implement regex compilation output (SLASH_LITERAL) (parse.js:989)
	s.PushToken(NewParseToken(TokenTypeSlash, value, ""))
	return nil
}

// HandleDot evaluates dot characters, directory dot semantics, and range formation ("..") inside braces.
// Matches original picomatch/lib/parse.js lines 997-1015.
func HandleDot(s *ParseState, value string) error {
	prev := s.CurrentToken()
	if s.Braces > 0 && prev != nil && prev.Type == TokenTypeDot {
		// parse.js:998-1006: Consecutive dots inside braces indicate a numeric or alphabetical range (e.g., {1..5})
		prev.Type = TokenTypeDots
		prev.Value += value
		if prev.OutputSet || prev.Output != "" {
			prev.Output += value
		}
		if s.BraceStack != nil && !s.BraceStack.IsEmpty() {
			if brace, ok := s.BraceStack.Peek(); ok && brace != nil {
				brace.Dots = true
			}
		}
		s.Append(&ParseToken{Value: value})
		return nil
	}

	// parse.js:1008-1011: Dots outside braces/parens not following BOS or slash are treated as plain text literal dots
	if (s.Braces+s.Parens) == 0 && prev != nil && prev.Type != TokenTypeBos && prev.Type != TokenTypeSlash {
		s.PushToken(NewParseToken(TokenTypeText, value, ""))
		return nil
	}

	// TODO: Implement regex pattern generation (DOT_LITERAL / NO_DOTS_SLASH) (parse.js:1013)
	s.PushToken(NewParseToken(TokenTypeDot, value, ""))
	return nil
}

// isValidNamedGroupOrLookbehind evaluates if a string matches /<([!=]|\w+>)/ per parse.js:1032.
func isValidNamedGroupOrLookbehind(rem string) bool {
	if len(rem) < 2 {
		return false
	}
	if rem[0] != '<' {
		return false
	}
	if rem[1] == '!' || rem[1] == '=' {
		return true
	}
	idx := 1
	for idx < len(rem) && (rem[idx] >= 'a' && rem[idx] <= 'z' || rem[idx] >= 'A' && rem[idx] <= 'Z' || rem[idx] >= '0' && rem[idx] <= '9' || rem[idx] == '_') {
		idx++
	}
	return idx > 1 && idx < len(rem) && rem[idx] == '>'
}

// HandleQmark evaluates question mark wildcard syntax, regex group exclusion, and dotfile anchoring rules.
// Matches original picomatch/lib/parse.js lines 1028-1047.
func HandleQmark(s *ParseState, value string) error {
	prev := s.CurrentToken()
	// parse.js:1028-1038: Intercept question marks immediately following parenthesis tokens
	if prev != nil && prev.Type == TokenTypeParen {
		next := s.Peek(1)
		output := value
		// Exclude regex lookahead/capture syntaxes; if unrecognized, escape the question mark
		if (prev.Value == "(" && !isRegexGroupChar(next)) || (next == '<' && !isValidNamedGroupOrLookbehind(s.Remaining())) {
			output = "\\" + value
		}
		tok := NewParseToken(TokenTypeText, value, output)
		if output != value {
			tok.OutputSet = true
		} else {
			tok.Output = "" // Maintain clean structural token when unescaped
		}
		s.PushToken(tok)
		return nil
	}

	// parse.js:1040-1043: If dot option is not set and preceded by slash or BOS, apply QMARK_NO_DOT anchor
	if (s.Opts == nil || !s.Opts.Dot) && prev != nil && (prev.Type == TokenTypeSlash || prev.Type == TokenTypeBos) {
		// TODO: Implement regex compilation anchor QMARK_NO_DOT (parse.js:1041)
		s.PushToken(NewParseToken(TokenTypeQmark, value, ""))
		return nil
	}

	// TODO: Implement regex compilation output QMARK (parse.js:1045)
	s.PushToken(NewParseToken(TokenTypeQmark, value, ""))
	return nil
}

// HandleStar evaluates star wildcard evaluation, globstar structural collapsing, and consecutive "/**/" stripping.
// Matches original picomatch/lib/parse.js lines 1128-1284.
func HandleStar(s *ParseState, value string) error {
	prev := s.CurrentToken()

	// parse.js:1128-1137: Collapse consecutive stars or stars following a globstar (e.g., ***)
	if prev != nil && (prev.Type == TokenTypeGlobstar || prev.Star) {
		prev.Type = TokenTypeStar
		prev.Star = true
		prev.Value += value
		if prev.OutputSet || prev.Output != "" {
			prev.Output += value
		}
		s.Backtrack = true
		s.Globstar = true
		s.Output += value
		s.Consume(value, 0)
		return nil
	}

	// parse.js:1145-1244: Second consecutive star evaluating whether to upgrade to globstar (**)
	if prev != nil && prev.Type == TokenTypeStar {
		if s.Opts != nil && s.Opts.NoGlobstar {
			s.Output += value
			prev.Value += value
			s.Consume(value, 0)
			return nil
		}

		var prior, before *ParseToken
		prior = prev.Prev
		if prior != nil {
			before = prior.Prev
		}

		isStart := prior != nil && (prior.Type == TokenTypeSlash || prior.Type == TokenTypeBos)
		afterStar := before != nil && (before.Type == TokenTypeStar || before.Type == TokenTypeGlobstar)
		rest := s.Remaining()

		// parse.js:1156-1159: Bash option restriction on globstar boundaries
		if s.Opts != nil && s.Opts.Bash && (!isStart || (len(rest) > 0 && rest[0] != '/')) {
			s.PushToken(NewParseToken(TokenTypeStar, value, ""))
			return nil
		}

		isBrace := s.Braces > 0 && prior != nil && (prior.Type == TokenTypeComma || prior.Type == TokenTypeBrace)
		isExtglob := s.ExtglobStack != nil && !s.ExtglobStack.IsEmpty() && prior != nil && (prior.Type == TokenTypePipe || prior.Type == TokenTypeParen)
		if !isStart && (prior == nil || prior.Type != TokenTypeParen) && !isBrace && !isExtglob {
			s.PushToken(NewParseToken(TokenTypeStar, value, ""))
			return nil
		}

		// parse.js:1169-1176: Strip consecutive "/**/" directory sequences
		for len(rest) >= 3 && rest[:3] == "/**" {
			afterIdx := s.Index + 4
			if afterIdx < len(s.Input) && s.Input[afterIdx] != '/' {
				break
			}
			rest = rest[3:]
			s.Consume("/**", 3)
		}

		// Case 1: parse.js:1178-1186: Leading globstar at EOS
		if prior != nil && prior.Type == TokenTypeBos && s.EOS() {
			prev.Type = TokenTypeGlobstar
			prev.Value += value
			if prev.OutputSet || prev.Output != "" {
				prev.Output += value
			}
			s.Globstar = true
			s.Output += value
			s.Consume(value, 0)
			return nil
		}

		// Case 2: parse.js:1188-1199: Trailing slash globstar at EOS (not preceded by BOS or another star)
		if prior != nil && prior.Type == TokenTypeSlash && (prior.Prev == nil || prior.Prev.Type != TokenTypeBos) && !afterStar && s.EOS() {
			// TODO: Implement regex synthesis for trailing slash globstar boundary (parse.js:1189-1196)
			prev.Type = TokenTypeGlobstar
			prev.Value += value
			if prev.OutputSet || prev.Output != "" {
				prev.Output += value
			}
			s.Globstar = true
			s.Output += value
			s.Consume(value, 0)
			return nil
		}

		// Case 3: parse.js:1201-1218: Mid-pattern slash globstar followed by slash (a/**/b)
		if prior != nil && prior.Type == TokenTypeSlash && (prior.Prev == nil || prior.Prev.Type != TokenTypeBos) && len(rest) > 0 && rest[0] == '/' {
			// TODO: Implement regex synthesis for mid-pattern slash globstar (parse.js:1204-1208)
			prev.Type = TokenTypeGlobstar
			prev.Value += value
			if prev.OutputSet || prev.Output != "" {
				prev.Output += value
			}
			s.Globstar = true
			s.Output += value
			nextCh := s.Advance()
			s.Consume(value+string([]byte{nextCh}), 0)
			s.PushToken(NewParseToken(TokenTypeSlash, "/", ""))
			return nil
		}

		// Case 4: parse.js:1220-1229: Leading BOS globstar followed by slash (**/a)
		if prior != nil && prior.Type == TokenTypeBos && len(rest) > 0 && rest[0] == '/' {
			// TODO: Implement regex synthesis for leading BOS globstar (parse.js:1223)
			prev.Type = TokenTypeGlobstar
			prev.Value += value
			if prev.OutputSet || prev.Output != "" {
				prev.Output += value
			}
			s.Globstar = true
			s.Output += value
			nextCh := s.Advance()
			s.Consume(value+string([]byte{nextCh}), 0)
			s.PushToken(NewParseToken(TokenTypeSlash, "/", ""))
			return nil
		}

		// Case 5: parse.js:1231-1243: Default fallback globstar upgrade
		prev.Type = TokenTypeGlobstar
		prev.Value += value
		if prev.OutputSet || prev.Output != "" {
			prev.Output += value
		}
		s.Globstar = true
		s.Output += value
		s.Consume(value, 0)
		return nil
	}

	// parse.js:1246-1284: Single star wildcard evaluation
	// TODO: Implement regex pattern generation, dotfile anchoring (NO_DOT/NO_DOTS_SLASH), and bash/regex option expressions
	s.PushToken(NewParseToken(TokenTypeStar, value, ""))
	return nil
}
