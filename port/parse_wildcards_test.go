package picomatch

import (
	"testing"
)

func TestParseWildcards_SlashHandling(t *testing.T) {
	t.Run("Leading ./ prefix stripping at BOS", func(t *testing.T) {
		// parse.js:430 & 980-987: Leading "./" is stripped by removePrefix before loop or by HandleSlash after negation
		state, err := Parse("./foo/bar", nil)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if state.Prefix != "./" {
			t.Errorf("Expected state.Prefix == './', got %q", state.Prefix)
		}
		if state.Start != 0 {
			t.Errorf("Expected state.Start == 0 after removePrefix, got %d", state.Start)
		}
		// Expected tokens: BOS, "foo", "/", "bar"
		if len(state.Tokens) != 4 {
			t.Fatalf("Expected 4 tokens after ./ removal, got %d", len(state.Tokens))
		}
		if state.Tokens[0].Type != TokenTypeBos {
			t.Errorf("Expected first token to remain BOS, got %s", state.Tokens[0].Type)
		}
		if state.Tokens[1].Value != "foo" {
			t.Errorf("Expected second token to be 'foo', got %q", state.Tokens[1].Value)
		}
		if state.Tokens[2].Type != TokenTypeSlash || state.Tokens[2].Value != "/" {
			t.Errorf("Expected third token to be slash, got %+v", state.Tokens[2])
		}

		// Verify HandleSlash stripping after negation prefix (parse.js:980-987)
		stateNeg, err := Parse("!./foo/bar", nil)
		if err != nil {
			t.Fatalf("Parse error on negated !./foo/bar: %v", err)
		}
		if stateNeg.Start != 3 {
			t.Errorf("Expected state.Start == 3 after HandleSlash stripping on !./, got %d", stateNeg.Start)
		}
		if !stateNeg.Negated {
			t.Errorf("Expected state.Negated to be true for !./foo/bar")
		}
	})

	t.Run("Normal slashes inside pattern", func(t *testing.T) {
		state, err := Parse("/foo/./bar", nil)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		// Non-leading "./" inside pattern must not be stripped
		foundDot := false
		for _, tok := range state.Tokens {
			if tok.Type == TokenTypeDot && tok.Value == "." {
				foundDot = true
			}
		}
		if !foundDot {
			t.Errorf("Expected internal dot token in /foo/./bar to remain unstripped, got %+v", state.Tokens)
		}
	})
}

func TestParseWildcards_DotHandling(t *testing.T) {
	t.Run("Literal plain text dots outside braces", func(t *testing.T) {
		// parse.js:1008-1011: Dots not after BOS/slash outside braces are plain text
		state, err := Parse("foo.js", nil)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if len(state.Tokens) != 2 {
			t.Fatalf("Expected merged text token for foo.js, got %d tokens", len(state.Tokens))
		}
		if state.Tokens[1].Type != TokenTypeText || state.Tokens[1].Value != "foo.js" {
			t.Errorf("Expected consolidated TokenTypeText 'foo.js', got %+v", state.Tokens[1])
		}
	})

	t.Run("Directory dots after BOS or slash", func(t *testing.T) {
		state, err := Parse(".hidden/.config", nil)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		// At BOS and after slash, dot emits TokenTypeDot
		if state.Tokens[1].Type != TokenTypeDot || state.Tokens[1].Value != "." {
			t.Errorf("Expected leading dot to be TokenTypeDot, got %+v", state.Tokens[1])
		}
	})

	t.Run("Range dots inside braces", func(t *testing.T) {
		// parse.js:998-1006: Consecutive dots inside braces mutate into TokenTypeDots (..) and mark BraceState.Dots
		// Test unclosed brace so range tokens remain un-popped in state.Tokens
		state, err := Parse("{1..5", nil)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		foundDots := false
		for _, tok := range state.Tokens {
			if tok.Type == TokenTypeDots && tok.Value == ".." {
				foundDots = true
			}
		}
		if !foundDots {
			t.Errorf("Expected TokenTypeDots ('..') token inside unclosed braces, got %+v", state.Tokens)
		}

		// When brace is closed, parse.js:907-923 pops inner tokens and produces expanded range output
		closedState, err := Parse("{1..5}", nil)
		if err != nil {
			t.Fatalf("Parse error on closed range: %v", err)
		}
		if closedState.Output != "[1-5]" {
			t.Errorf("Expected expanded range output for {1..5}, got %q", closedState.Output)
		}
	})
}

func TestParseWildcards_QmarkHandling(t *testing.T) {
	t.Run("Standard question mark wildcard", func(t *testing.T) {
		state, err := Parse("foo/?bar", nil)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		foundQ := false
		for _, tok := range state.Tokens {
			if tok.Type == TokenTypeQmark && tok.Value == "?" {
				foundQ = true
			}
		}
		if !foundQ {
			t.Errorf("Expected TokenTypeQmark for question mark after slash, got %+v", state.Tokens)
		}
	})

	t.Run("Question mark immediately following parenthesis", func(t *testing.T) {
		// parse.js:1028-1038: Exclude valid regex capture groups; escape unrecognized symbols
		state, err := Parse("(?abc)", nil)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		// In (?abc), 'a' is not in /[!=<:]/, so '?' is escaped as "\\?"
		for _, tok := range state.Tokens {
			if tok.Value == "?" || (tok.Type == TokenTypeText && len(tok.Output) > 0) {
				if tok.Output != "\\?" && tok.Output != "\\?abc" {
					t.Errorf("Expected escaped question mark output \\? for unrecognized regex group, got %q", tok.Output)
				}
			}
		}
	})

	t.Run("Valid regex group specifiers after paren", func(t *testing.T) {
		state, err := Parse("(?=abc)", nil)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		// '?' before '=' is a valid lookahead specifier; should not be escaped
		for _, tok := range state.Tokens {
			if tok.Output == "\\?" {
				t.Errorf("Did not expect escaped question mark for valid lookahead group (?=abc)")
			}
		}
	})

	t.Run("Lookbehind and named group validation helper", func(t *testing.T) {
		if !isValidNamedGroupOrLookbehind("<=") || !isValidNamedGroupOrLookbehind("<!") {
			t.Errorf("Expected valid lookbehind symbols <= and <!")
		}
		if !isValidNamedGroupOrLookbehind("<my_group>") {
			t.Errorf("Expected valid named capture group <my_group>")
		}
		if isValidNamedGroupOrLookbehind("<invalid") || isValidNamedGroupOrLookbehind("") {
			t.Errorf("Expected invalid syntax to return false")
		}
	})
}

func TestParseWildcards_StarAndGlobstar(t *testing.T) {
	t.Run("Single star wildcard evaluation", func(t *testing.T) {
		state, err := Parse("foo/*.js", nil)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		foundStar := false
		for _, tok := range state.Tokens {
			if tok.Type == TokenTypeStar && tok.Value == "*" {
				foundStar = true
			}
		}
		if !foundStar {
			t.Errorf("Expected single TokenTypeStar in foo/*.js, got %+v", state.Tokens)
		}
	})

	t.Run("Globstar upgrade on second star", func(t *testing.T) {
		// parse.js:1145-1244: Two consecutive stars upgrade to TokenTypeGlobstar
		patterns := []string{"**", "foo/**", "**/bar", "foo/**/bar"}
		for _, pat := range patterns {
			state, err := Parse(pat, nil)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", pat, err)
			}
			if !state.Globstar {
				t.Errorf("Expected state.Globstar == true for %q", pat)
			}
			foundGlobstar := false
			for _, tok := range state.Tokens {
				if tok.Type == TokenTypeGlobstar && tok.Value == "**" {
					foundGlobstar = true
				}
			}
			if !foundGlobstar {
				t.Errorf("Expected TokenTypeGlobstar token in %q, got %+v", pat, state.Tokens)
			}
		}
	})

	t.Run("Consecutive star collapsing and backtrack marking", func(t *testing.T) {
		// parse.js:1128-1137: Three or more stars collapse back to TokenTypeStar with Star=true, Backtrack=true, Globstar=true
		state, err := Parse("foo/***/bar", nil)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if !state.Backtrack || !state.Globstar {
			t.Errorf("Expected state.Backtrack and state.Globstar to be set for ***, got Backtrack=%v, Globstar=%v", state.Backtrack, state.Globstar)
		}
		foundCollapsed := false
		for _, tok := range state.Tokens {
			if tok.Type == TokenTypeStar && tok.Value == "***" && tok.Star {
				foundCollapsed = true
			}
		}
		if !foundCollapsed {
			t.Errorf("Expected collapsed TokenTypeStar with value *** and Star=true, got %+v", state.Tokens)
		}
	})

	t.Run("Stripping consecutive /**/ sequences", func(t *testing.T) {
		// parse.js:1169-1176: Redundant /**/ segments are stripped during evaluation
		state, err := Parse("foo/**/**/**/bar", nil)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if !state.Globstar {
			t.Errorf("Expected state.Globstar == true for consecutive /**/ stripping")
		}
		// Ensure only a single globstar token remains between foo and bar
		globstarCount := 0
		for _, tok := range state.Tokens {
			if tok.Type == TokenTypeGlobstar {
				globstarCount++
			}
		}
		if globstarCount != 1 {
			t.Errorf("Expected exactly 1 globstar token after stripping consecutive /**/ segments, got %d", globstarCount)
		}
	})

	t.Run("Globstar structural downgrade in PushToken", func(t *testing.T) {
		// parse.js:494-505: When followed by a non-directory/non-syntax token, globstar is downgraded to single star
		state, err := Parse("foo/**a", nil)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		for _, tok := range state.Tokens {
			if tok.Type == TokenTypeGlobstar {
				t.Errorf("Did not expect TokenTypeGlobstar in foo/**a after downgrade, got %+v", state.Tokens)
			}
			if tok.Type == TokenTypeStar && tok.Value != "*" {
				t.Errorf("Expected downgraded star token to collapse value to '*', got %q", tok.Value)
			}
		}
	})

	t.Run("NoGlobstar option disables upgrade", func(t *testing.T) {
		state, err := Parse("foo/**/bar", &ParseOptions{NoGlobstar: true})
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if state.Globstar {
			t.Errorf("Did not expect state.Globstar == true when NoGlobstar is true")
		}
		for _, tok := range state.Tokens {
			if tok.Type == TokenTypeGlobstar {
				t.Errorf("Did not expect TokenTypeGlobstar when NoGlobstar is set, got %+v", state.Tokens)
			}
		}
	})

	t.Run("Bash option globstar boundaries", func(t *testing.T) {
		// In bash mode, globstar must stand alone as a directory segment
		state, err := Parse("a**", &ParseOptions{Bash: true})
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		for _, tok := range state.Tokens {
			if tok.Type == TokenTypeGlobstar {
				t.Errorf("Did not expect TokenTypeGlobstar for a** when Bash=true")
			}
		}
	})
}

func TestParseWildcards_EscapedWildcards(t *testing.T) {
	t.Run("Escaped star and question mark", func(t *testing.T) {
		state, err := Parse("foo\\*bar\\?baz", nil)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		// Escaped wildcards discard backslashes and push plain text, merging into a single consolidated token
		for _, tok := range state.Tokens {
			if tok.Type == TokenTypeStar || tok.Type == TokenTypeQmark {
				t.Errorf("Did not expect wildcard tokens for escaped \\* and \\?, got %+v", state.Tokens)
			}
		}
		if len(state.Tokens) != 2 || state.Tokens[1].Value != "foo\\*bar\\?baz" {
			t.Errorf("Expected single merged text token 'foo\\*bar\\?baz', got %+v", state.Tokens)
		}
	})

	t.Run("Escaped globstar", func(t *testing.T) {
		state, err := Parse("foo/\\*\\*/bar", nil)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if state.Globstar {
			t.Errorf("Did not expect state.Globstar == true for escaped \\*\\*")
		}
		for _, tok := range state.Tokens {
			if tok.Type == TokenTypeGlobstar || tok.Type == TokenTypeStar {
				t.Errorf("Did not expect star or globstar token for escaped \\*\\*, got %+v", state.Tokens)
			}
		}
	})
}

func TestParseWildcards_NestedContexts(t *testing.T) {
	t.Run("Wildcards inside brackets", func(t *testing.T) {
		// Inside POSIX brackets, * and ? lose their wildcard meanings and are treated as character class content
		state, err := Parse("[*?]", nil)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		for _, tok := range state.Tokens {
			if tok.Type == TokenTypeStar || tok.Type == TokenTypeQmark {
				t.Errorf("Did not expect TokenTypeStar or TokenTypeQmark inside character class [*?], got %+v", state.Tokens)
			}
		}
		if state.Brackets == 0 && len(state.Tokens) >= 2 {
			if state.Tokens[1].Type != TokenTypeBracket || state.Tokens[1].Value != "[*?]" {
				t.Errorf("Expected consolidated bracket token '[*?]', got %+v", state.Tokens[1])
			}
		}
	})

	t.Run("Wildcards inside braces", func(t *testing.T) {
		// Inside braces, * and ? retain their wildcard structural token types while tracking brace nesting
		state, err := Parse("{*.js,?.ts}", nil)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		foundStar := false
		foundQmark := false
		for _, tok := range state.Tokens {
			if tok.Type == TokenTypeStar && tok.Value == "*" {
				foundStar = true
			}
			if tok.Type == TokenTypeQmark && tok.Value == "?" {
				foundQmark = true
			}
		}
		if !foundStar || !foundQmark {
			t.Errorf("Expected star and question mark tokens inside braces, got %+v", state.Tokens)
		}
	})

	t.Run("Wildcards inside extglobs", func(t *testing.T) {
		// Inside extglobs, wildcards function within parenthetical structures without breaking stack accumulation
		state, err := Parse("*(a*b|c?d)", nil)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		foundStar := false
		foundQmark := false
		for _, tok := range state.Tokens {
			if tok.Type == TokenTypeStar && !tok.Extglob && tok.Value == "*" {
				foundStar = true
			}
			if tok.Type == TokenTypeQmark && tok.Value == "?" {
				foundQmark = true
			}
		}
		if !foundStar || !foundQmark {
			t.Errorf("Expected internal star and qmark tokens inside extglob group, got %+v", state.Tokens)
		}
	})
}

func TestParseWildcards_DotfileInteractions(t *testing.T) {
	t.Run("Dotfile boundaries at BOS and slashes", func(t *testing.T) {
		patterns := []string{".hidden/*", "dir/.*", ".*"}
		for _, pat := range patterns {
			state, err := Parse(pat, nil)
			if err != nil {
				t.Fatalf("Parse(%q) failed: %v", pat, err)
			}
			foundDot := false
			foundStar := false
			for _, tok := range state.Tokens {
				if tok.Type == TokenTypeDot && tok.Value == "." {
					foundDot = true
				}
				if tok.Type == TokenTypeStar && tok.Value == "*" {
					foundStar = true
				}
			}
			if !foundDot || !foundStar {
				t.Errorf("Expected distinct dot and star structural tokens for dotfile pattern %q, got %+v", pat, state.Tokens)
			}
		}
	})
}

func TestParseWildcards_RollbackBookkeeping(t *testing.T) {
	t.Run("Rollback bookkeeping across malformed braces containing wildcards", func(t *testing.T) {
		// When an unclosed brace or extglob is encountered, rollback bookkeeping restores token arrays and output state cleanly without leaking partial wildcard mutations
		state, err := Parse("{a*,b?", nil)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if !state.EOS() {
			t.Errorf("Expected parser to cleanly reach EOS on unclosed brace with wildcards")
		}
		// Confirm tokens array is valid and contains expected wildcard tokens post-reconciliation
		foundStar := false
		foundQmark := false
		for _, tok := range state.Tokens {
			if tok.Type == TokenTypeStar && tok.Value == "*" {
				foundStar = true
			}
			if tok.Type == TokenTypeQmark && tok.Value == "?" {
				foundQmark = true
			}
		}
		if !foundStar || !foundQmark {
			t.Errorf("Expected wildcards to persist in token sequence after unclosed brace reconciliation, got %+v", state.Tokens)
		}
	})
}
