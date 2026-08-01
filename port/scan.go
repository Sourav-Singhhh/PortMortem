package picomatch

// ScanOptions defines configuration toggles that modify scanner behavior.
type ScanOptions struct {
	Parts     bool // If true, populates Parts array and sets ScanToEnd = true
	ScanToEnd bool // If true, forces full string traversal after glob detection
	Tokens    bool // If true, generates detailed segmented tokens and depth
	NoExt     bool // If true, ignores extended glob feature syntax (+(@|*))
	NoNegate  bool // If true, ignores leading exclamation marks (!) as negations
	NoParen   bool // If true, ignores plain parentheses groups ((...))
	Unescape  bool // If true, strips backslash escapes from base and glob strings
}

// Token represents metadata for an isolated path segment between slashes.
type Token struct {
	Value       string
	Depth       int  // Matching traversal depth (0 if Infinity == true)
	IsGlob      bool // True if this specific segment contains active glob syntax
	IsPrefix    bool // True if this token represents leading prefix characters (e.g. "./")
	IsBrace     bool // True if this segment contains brace expansion syntax
	IsBracket   bool // True if this segment contains character class brackets
	IsExtglob   bool // True if this segment contains extended glob syntax
	IsGlobstar  bool // True if this segment contains a globstar (**)
	Negated     bool // True if this segment is negated
	Backslashes bool // True if this segment contained backslash escapes
	Infinity    bool // Go-specific flag: true if Depth should be mathematically treated as Infinity
}

// ScanState represents the complete architectural evaluation of a scanned glob string.
type ScanState struct {
	Prefix         string
	Input          string
	Base           string
	Glob           string
	Start          int
	MaxDepth       int
	IsBrace        bool
	IsBracket      bool
	IsGlob         bool
	IsExtglob      bool
	IsGlobstar     bool
	Negated        bool
	NegatedExtglob bool
	MaxInfinity    bool
	Slashes        []int
	Parts          []string
	Tokens         []Token
}

// ASCII character constants matching constants.js in original picomatch.
const (
	charAsterisk           = '*'
	charAt                 = '@'
	charBackwardSlash      = '\\'
	charComma              = ','
	charDot                = '.'
	charExclamationMark    = '!'
	charForwardSlash       = '/'
	charLeftCurlyBrace     = '{'
	charLeftParentheses    = '('
	charLeftSquareBracket  = '['
	charPlus               = '+'
	charQuestionMark       = '?'
	charRightCurlyBrace    = '}'
	charRightParentheses   = ')'
	charRightSquareBracket = ']'
)

// isPathSeparator returns true if byte is a forward or backward slash.
func isPathSeparator(code byte) bool {
	return code == charForwardSlash || code == charBackwardSlash
}

// depth calculates traversal depth of a segmented token, setting Infinity flag if globstar.
func depth(token *Token) {
	if !token.IsPrefix {
		if token.IsGlobstar {
			token.Infinity = true
			token.Depth = 0
		} else {
			token.Depth = 1
			token.Infinity = false
		}
	}
}

// removeBackslashes strips '\' characters exactly matching JS /(?:\[.*?[^\\]\]|\\(?=.))/g
// preserving backslashes inside valid POSIX bracket character classes with zero allocations if no slash exists.
func removeBackslashes(s string) string {
	hasSlash := false
	for i := 0; i < len(s); i++ {
		if s[i] == charBackwardSlash {
			hasSlash = true
			break
		}
	}
	if !hasSlash {
		return s
	}

	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); {
		if s[i] == charLeftSquareBracket {
			closeIdx := -1
			for j := i + 1; j < len(s); j++ {
				if s[j] == charRightSquareBracket && (j == 0 || s[j-1] != charBackwardSlash) {
					closeIdx = j
					break
				}
			}
			if closeIdx != -1 {
				b = append(b, s[i:closeIdx+1]...)
				i = closeIdx + 1
				continue
			}
		}
		if s[i] == charBackwardSlash && i+1 < len(s) {
			i++ // Skip backslash, preserve trailing character
			continue
		}
		b = append(b, s[i])
		i++
	}
	return string(b)
}

// Scan quickly analyzes a glob pattern, extracting useful structural properties
// such as Base path, active Glob expression, negation status, and segment tokens.
func Scan(input string, opts *ScanOptions) *ScanState {
	if opts == nil {
		opts = &ScanOptions{}
	}

	length := len(input) - 1
	scanToEnd := opts.Parts || opts.ScanToEnd
	slashes := make([]int, 0, 4)
	tokens := make([]Token, 0, 4)
	parts := make([]string, 0, 4)

	str := input
	index := -1
	start := 0
	lastIndex := 0
	isBrace := false
	isBracket := false
	isGlob := false
	isExtglob := false
	isGlobstar := false
	braceEscaped := false
	backslashes := false
	negated := false
	negatedExtglob := false
	finished := false
	braces := 0
	var prev byte
	var code byte
	token := Token{}

	eos := func() bool {
		return index >= length
	}
	peek := func() byte {
		if index+1 < len(str) {
			return str[index+1]
		}
		return 0
	}
	advance := func() byte {
		prev = code
		index++
		if index < len(str) {
			code = str[index]
			return code
		}
		code = 0
		return code
	}

	for index < length {
		code = advance()
		var next byte

		if code == charBackwardSlash {
			backslashes = true
			token.Backslashes = true
			code = advance()

			if code == charLeftCurlyBrace {
				braceEscaped = true
			}
			continue
		}

		if braceEscaped || code == charLeftCurlyBrace {
			braces++

			for !eos() {
				code = advance()
				if code == 0 {
					break
				}
				if code == charBackwardSlash {
					backslashes = true
					token.Backslashes = true
					advance()
					continue
				}

				if code == charLeftCurlyBrace {
					braces++
					continue
				}

				if !braceEscaped && code == charDot {
					next = advance()
					if next == charDot {
						isBrace = true
						token.IsBrace = true
						isGlob = true
						token.IsGlob = true
						finished = true

						if scanToEnd {
							continue
						}
						break
					}
					code = next
				}

				if !braceEscaped && code == charComma {
					isBrace = true
					token.IsBrace = true
					isGlob = true
					token.IsGlob = true
					finished = true

					if scanToEnd {
						continue
					}
					break
				}

				if code == charRightCurlyBrace {
					braces--
					if braces == 0 {
						braceEscaped = false
						isBrace = true
						token.IsBrace = true
						finished = true
						break
					}
				}
			}

			if scanToEnd {
				continue
			}
			break
		}

		if code == charForwardSlash {
			slashes = append(slashes, index)
			tokens = append(tokens, token)
			token = Token{}

			if finished {
				continue
			}
			if prev == charDot && index == (start+1) {
				start += 2
				continue
			}

			lastIndex = index + 1
			continue
		}

		if !opts.NoExt {
			isExtglobChar := code == charPlus || code == charAt || code == charAsterisk ||
				code == charQuestionMark || code == charExclamationMark

			if isExtglobChar && peek() == charLeftParentheses {
				isGlob = true
				token.IsGlob = true
				isExtglob = true
				token.IsExtglob = true
				finished = true
				if code == charExclamationMark && index == start {
					negatedExtglob = true
				}

				if scanToEnd {
					for !eos() {
						code = advance()
						if code == 0 {
							break
						}
						if code == charBackwardSlash {
							backslashes = true
							token.Backslashes = true
							advance()
							continue
						}
						if code == charRightParentheses {
							isGlob = true
							token.IsGlob = true
							finished = true
							break
						}
					}
					continue
				}
				break
			}
		}

		if code == charAsterisk {
			if prev == charAsterisk {
				isGlobstar = true
				token.IsGlobstar = true
			}
			isGlob = true
			token.IsGlob = true
			finished = true

			if scanToEnd {
				continue
			}
			break
		}

		if code == charQuestionMark {
			isGlob = true
			token.IsGlob = true
			finished = true

			if scanToEnd {
				continue
			}
			break
		}

		if code == charLeftSquareBracket {
			for !eos() {
				next = advance()
				if next == 0 {
					break
				}
				if next == charBackwardSlash {
					backslashes = true
					token.Backslashes = true
					advance()
					continue
				}
				if next == charRightSquareBracket {
					isBracket = true
					token.IsBracket = true
					isGlob = true
					token.IsGlob = true
					finished = true
					break
				}
			}

			if scanToEnd {
				continue
			}
			break
		}

		if !opts.NoNegate && code == charExclamationMark && index == start {
			negated = true
			token.Negated = true
			start++
			continue
		}

		if !opts.NoParen && code == charLeftParentheses {
			isGlob = true
			token.IsGlob = true

			if scanToEnd {
				for !eos() {
					code = advance()
					if code == 0 {
						break
					}
					// Preserve exact JS line 261 check where only CHAR_LEFT_PARENTHESES sets backslashes
					if code == charLeftParentheses {
						backslashes = true
						token.Backslashes = true
						code = advance()
						continue
					}
					if code == charRightParentheses {
						finished = true
						break
					}
				}
				continue
			}
			break
		}

		if isGlob {
			finished = true
			if scanToEnd {
				continue
			}
			break
		}
	}

	if opts.NoExt {
		isExtglob = false
		isGlob = false
	}

	base := str
	prefix := ""
	glob := ""

	if start > 0 && start <= len(str) {
		prefix = str[:start]
		str = str[start:]
		lastIndex -= start
	}

	if len(base) > 0 && isGlob && lastIndex > 0 && lastIndex <= len(str) {
		base = str[:lastIndex]
		glob = str[lastIndex:]
	} else if isGlob {
		base = ""
		glob = str
	} else {
		base = str
	}

	if len(base) > 0 && base != "" && base != "/" && base != str {
		if isPathSeparator(base[len(base)-1]) {
			base = base[:len(base)-1]
		}
	}

	if opts.Unescape {
		if len(glob) > 0 {
			glob = removeBackslashes(glob)
		}
		if len(base) > 0 && backslashes {
			base = removeBackslashes(base)
		}
	}

	state := &ScanState{
		Prefix:         prefix,
		Input:          input,
		Start:          start,
		Base:           base,
		Glob:           glob,
		IsBrace:        isBrace,
		IsBracket:      isBracket,
		IsGlob:         isGlob,
		IsExtglob:      isExtglob,
		IsGlobstar:     isGlobstar,
		Negated:        negated,
		NegatedExtglob: negatedExtglob,
		Slashes:        []int{},
		Parts:          []string{},
		Tokens:         []Token{},
	}

	if opts.Tokens {
		state.MaxDepth = 0
		if len(input) == 0 || !isPathSeparator(code) {
			tokens = append(tokens, token)
		}
		state.Tokens = tokens
	}

	if opts.Parts || opts.Tokens {
		// In JS, prevIndex starts undefined (falsy) and index 0 is also falsy. We use 0 as falsy check.
		prevIndex := 0
		state.Slashes = slashes
		state.Parts = parts

		for idx := 0; idx < len(slashes); idx++ {
			n := start
			if prevIndex != 0 {
				n = prevIndex + 1
			}
			i := slashes[idx]
			var value string
			if n <= i && i <= len(input) && n >= 0 {
				value = input[n:i]
			}
			if opts.Tokens && idx < len(tokens) {
				if idx == 0 && start != 0 {
					tokens[idx].IsPrefix = true
					tokens[idx].Value = prefix
				} else {
					tokens[idx].Value = value
				}
				depth(&tokens[idx])
				if tokens[idx].Infinity {
					state.MaxInfinity = true
				} else {
					state.MaxDepth += tokens[idx].Depth
				}
			}
			if idx != 0 || value != "" {
				parts = append(parts, value)
			}
			prevIndex = i
		}

		if prevIndex != 0 && prevIndex+1 < len(input) {
			value := input[prevIndex+1:]
			parts = append(parts, value)

			if opts.Tokens && len(tokens) > 0 {
				lastTokIdx := len(tokens) - 1
				tokens[lastTokIdx].Value = value
				depth(&tokens[lastTokIdx])
				if tokens[lastTokIdx].Infinity {
					state.MaxInfinity = true
				} else {
					state.MaxDepth += tokens[lastTokIdx].Depth
				}
			}
		}

		state.Parts = parts
		if opts.Tokens {
			state.Tokens = tokens
		}
	}

	return state
}
