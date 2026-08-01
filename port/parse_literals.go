package picomatch

// IsNonSpecialChar determines whether a raw ASCII or UTF-8 byte represents a non-special literal character
// corresponding to REGEX_NON_SPECIAL_CHARS in constants.js:98 (^[^@![\].,$*+?^{}()|\\/]+).
// Null bytes (0) are treated as special to permit immediate filtering in the primary scanning loop.
// All UTF-8 multibyte continuation bytes (>= 128) evaluate to true as grammar delimiters fall exclusively within ASCII (< 128).
func IsNonSpecialChar(b byte) bool {
	switch b {
	case 0, '@', '!', '[', ']', '.', ',', '$', '*', '+', '?', '^', '{', '}', '(', ')', '|', '\\', '/':
		return false
	default:
		return true
	}
}

// HandleEscape executes backslash escape handling, consecutive backslash collapse, and escaped character consumption
// in strict adherence to original parse.js:672-711 control flow.
// It returns the mutated value string along with true if the escaped token was fully handled or discarded, and false if
// execution must fall through to active regex character class evaluation (when state.Brackets > 0).
func HandleEscape(s *ParseState, value string) (string, bool) {
	next := s.Peek(1)

	// In standard glob evaluation, escaping path slashes (/), dots (.), and semicolons (;) discards the backslash (parse.js:675-681)
	if next == '/' && (s.Opts == nil || !s.Opts.Bash) {
		return value, true
	}

	if next == '.' || next == ';' {
		return value, true
	}

	// Trailing backslash terminating the string emits as a literal escaped backslash "\\" (parse.js:683-687)
	if s.EOS() {
		value += "\\"
		s.PushToken(NewParseToken(TokenTypeText, value, ""))
		return value, true
	}

	// Collapse consecutive backslashes to reduce potential for denial of service exploits (parse.js:689-699)
	rem := s.Remaining()
	slashes := 0
	for i := 0; i < len(rem) && rem[i] == '\\'; i++ {
		slashes++
	}

	if slashes > 2 {
		s.Index += slashes
		if slashes%2 != 0 {
			value += "\\"
		}
	}

	// Consume the escaped byte without re-encoding to preserve exact UTF-8 character boundaries (parse.js:701-705)
	var escapedChar string
	if !s.EOS() {
		escapedChar = string([]byte{s.Advance()})
	} else {
		s.Advance() // Advance index past EOF without generating null-byte strings
	}

	if s.Opts != nil && s.Opts.Unescape {
		value = escapedChar
	} else {
		value += escapedChar
	}

	// Outside regex character classes, emit the text token and return true (parse.js:707-710)
	if s.Brackets == 0 {
		s.PushToken(NewParseToken(TokenTypeText, value, ""))
		return value, true
	}

	// When state.Brackets > 0, return false to enable fallthrough into character class evaluation in future milestones
	return value, false
}

// HandlePlainText executes plain text literal accumulation, regex anchor character escaping, and fast-forward
// sequential scanning across consecutive non-special characters in accordance with parse.js:1109-1122.
func HandlePlainText(s *ParseState, value string) {
	// Escape standalone regex start ($) and line end (^) anchor symbols (parse.js:1110-1112)
	if value == "$" || value == "^" {
		value = "\\" + value
	}

	// Fast-forward across consecutive non-special characters to minimize token allocation and backtracking (parse.js:1114-1118)
	rem := s.Remaining()
	i := 0
	for i < len(rem) && IsNonSpecialChar(rem[i]) {
		i++
	}

	if i > 0 {
		value += rem[:i]
		s.Index += i
	}

	s.PushToken(NewParseToken(TokenTypeText, value, ""))
}
