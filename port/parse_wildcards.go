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
		return nil
	}

	chars := GetGlobChars(s.Opts != nil && s.Opts.Windows)
	tok := NewParseToken(TokenTypeSlash, value, chars.SlashLiteral)
	tok.OutputSet = true
	s.PushToken(tok)
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

	// parse.js:1008-1011: Dots outside braces/parens not following BOS or slash are treated as plain text literal dots.
	// The output must be the RE2-escaped literal "\." rather than the raw "." character, which would act as a
	// regex wildcard matching any character. Node.js picomatch generates "\.\." for "*..*"; Go must do the same.
	if (s.Braces+s.Parens) == 0 && prev != nil && prev.Type != TokenTypeBos && prev.Type != TokenTypeSlash {
		tok := NewParseToken(TokenTypeText, value, `\.`)
		tok.OutputSet = true
		s.PushToken(tok)
		return nil
	}

	chars := GetGlobChars(s.Opts != nil && s.Opts.Windows)
	tok := NewParseToken(TokenTypeDot, value, chars.DotLiteral)
	tok.OutputSet = true
	s.PushToken(tok)
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

	chars := GetGlobChars(s.Opts != nil && s.Opts.Windows)
	// parse.js:1040-1043: If dot option is not set and preceded by slash or BOS, apply QMARK_NO_DOT anchor
	if (s.Opts == nil || !s.Opts.Dot) && prev != nil && (prev.Type == TokenTypeSlash || prev.Type == TokenTypeBos) {
		tok := NewParseToken(TokenTypeQmark, value, chars.QmarkNoDot)
		tok.OutputSet = true
		s.PushToken(tok)
		return nil
	}

	tok := NewParseToken(TokenTypeQmark, value, chars.Qmark)
	tok.OutputSet = true
	s.PushToken(tok)
	return nil
}

// HandleStar evaluates star wildcard evaluation, globstar structural collapsing, and consecutive "/**/" stripping.
// Matches original picomatch/lib/parse.js lines 1128-1284.
func HandleStar(s *ParseState, value string) error {
	prev := s.CurrentToken()
	chars := GetGlobChars(s.Opts != nil && s.Opts.Windows)

	// parse.js:1128-1137: Collapse consecutive stars or stars following a globstar (e.g., ***)
	if prev != nil && (prev.Type == TokenTypeGlobstar || prev.Star) {
		star := chars.Star
		if s.Opts != nil && s.Opts.Bash {
			star = Globstar(s.Opts, chars)
		}
		if s.Opts != nil && s.Opts.Capture {
			star = "(" + star + ")"
		}

		if len(s.Output) >= len(prev.Output) {
			s.Output = s.Output[:len(s.Output)-len(prev.Output)]
		} else if len(s.Output) >= len(prev.Value) {
			s.Output = s.Output[:len(s.Output)-len(prev.Value)]
		}

		prev.Type = TokenTypeStar
		prev.Star = true
		prev.Value += value
		prev.Output = star
		prev.OutputSet = true

		s.Backtrack = true
		s.Globstar = true
		s.Output += prev.Output
		s.Consume(value, 0)
		return nil
	}

	// parse.js:1145-1244: Second consecutive star evaluating whether to upgrade to globstar (**)
	if prev != nil && prev.Type == TokenTypeStar {
		if s.Opts != nil && s.Opts.NoGlobstar {
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
			prev.Output = Globstar(s.Opts, chars)
			prev.OutputSet = true
			s.Globstar = true
			s.Output = prev.Output
			s.Consume(value, 0)
			return nil
		}

		// Case 2: parse.js:1188-1199: Trailing slash globstar at EOS (not preceded by BOS or another star)
		if prior != nil && prior.Type == TokenTypeSlash && (prior.Prev == nil || prior.Prev.Type != TokenTypeBos) && !afterStar && s.EOS() {
			if len(s.Output) >= len(prior.Output+prev.Output) {
				s.Output = s.Output[:len(s.Output)-len(prior.Output+prev.Output)]
			} else if len(s.Output) >= len(prior.Value+prev.Value) {
				s.Output = s.Output[:len(s.Output)-len(prior.Value+prev.Value)]
			}
			prior.Output = "(?:" + prior.Output
			prior.OutputSet = true

			prev.Type = TokenTypeGlobstar
			prev.Value += value
			suffix := "|$)"
			if s.Opts != nil && s.Opts.StrictSlashes {
				suffix = ")"
			}
			prev.Output = Globstar(s.Opts, chars) + suffix
			prev.OutputSet = true
			s.Globstar = true
			s.Output += prior.Output + prev.Output
			s.Consume(value, 0)
			return nil
		}

		// Case 3: parse.js:1201-1218: Mid-pattern slash globstar followed by slash (a/**/b)
		if prior != nil && prior.Type == TokenTypeSlash && (prior.Prev == nil || prior.Prev.Type != TokenTypeBos) && len(rest) > 0 && rest[0] == '/' {
			end := ""
			if len(rest) > 1 {
				end = "|$"
			}
			if len(s.Output) >= len(prior.Output+prev.Output) {
				s.Output = s.Output[:len(s.Output)-len(prior.Output+prev.Output)]
			} else if len(s.Output) >= len(prior.Value+prev.Value) {
				s.Output = s.Output[:len(s.Output)-len(prior.Value+prev.Value)]
			}
			prior.Output = "(?:" + prior.Output
			prior.OutputSet = true

			prev.Type = TokenTypeGlobstar
			prev.Value += value
			prev.Output = Globstar(s.Opts, chars) + chars.SlashLiteral + "|" + chars.SlashLiteral + end + ")"
			prev.OutputSet = true
			s.Output += prior.Output + prev.Output
			s.Globstar = true

			nextCh := s.Advance()
			s.Consume(value+string([]byte{nextCh}), 0)

			slashTok := NewParseToken(TokenTypeSlash, "/", "")
			slashTok.OutputSet = true
			s.PushToken(slashTok)
			return nil
		}

		// Case 4: parse.js:1220-1229: Leading BOS globstar followed by slash (**/a)
		if prior != nil && prior.Type == TokenTypeBos && len(rest) > 0 && rest[0] == '/' {
			prev.Type = TokenTypeGlobstar
			prev.Value += value
			prev.Output = "(?:^|" + chars.SlashLiteral + "|" + Globstar(s.Opts, chars) + chars.SlashLiteral + ")"
			prev.OutputSet = true
			s.Output = prev.Output
			s.Globstar = true

			nextCh := s.Advance()
			s.Consume(value+string([]byte{nextCh}), 0)

			slashTok := NewParseToken(TokenTypeSlash, "/", "")
			slashTok.OutputSet = true
			s.PushToken(slashTok)
			return nil
		}

		// Case 5: parse.js:1231-1243: Default fallback globstar upgrade
		if len(s.Output) >= len(prev.Output) {
			s.Output = s.Output[:len(s.Output)-len(prev.Output)]
		} else if len(s.Output) >= len(prev.Value) {
			s.Output = s.Output[:len(s.Output)-len(prev.Value)]
		}
		prev.Type = TokenTypeGlobstar
		prev.Value += value
		prev.Output = Globstar(s.Opts, chars)
		prev.OutputSet = true
		s.Globstar = true
		s.Output += prev.Output
		s.Consume(value, 0)
		return nil
	}

	// parse.js:1246-1284: Single star wildcard evaluation
	star := chars.Star
	if s.Opts != nil && s.Opts.Bash {
		star = Globstar(s.Opts, chars)
	}
	if s.Opts != nil && s.Opts.Capture {
		star = "(" + star + ")"
	}

	tok := NewParseToken(TokenTypeStar, value, star)
	tok.OutputSet = true

	if s.Opts != nil && s.Opts.Bash {
		tok.Output = ".*?"
		if prev != nil && (prev.Type == TokenTypeBos || prev.Type == TokenTypeSlash) {
			nodot := chars.NoDot
			if s.Opts != nil && s.Opts.Dot {
				nodot = ""
			}
			tok.Output = nodot + tok.Output
		}
		s.PushToken(tok)
		return nil
	}

	if prev != nil && (prev.Type == TokenTypeBracket || prev.Type == TokenTypeParen) && s.Opts != nil && s.Opts.Regex {
		tok.Output = value
		s.PushToken(tok)
		return nil
	}

	if s.Index == s.Start || (prev != nil && (prev.Type == TokenTypeSlash || prev.Type == TokenTypeDot)) {
		if prev != nil && prev.Type == TokenTypeDot {
			s.Output += chars.NoDotSlash
			if prev != nil {
				prev.Output += chars.NoDotSlash
				prev.OutputSet = true
			}
		} else if s.Opts != nil && s.Opts.Dot {
			s.Output += chars.NoDotsSlash
			if prev != nil {
				prev.Output += chars.NoDotsSlash
				prev.OutputSet = true
			}
		} else {
			s.Output += chars.NoDot
			if prev != nil {
				prev.Output += chars.NoDot
				prev.OutputSet = true
			}
		}

		if s.Peek(1) != '*' {
			s.Output += chars.OneChar
			if prev != nil {
				prev.Output += chars.OneChar
				prev.OutputSet = true
			}
		}
	}

	s.PushToken(tok)
	return nil
}
