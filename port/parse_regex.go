package picomatch

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// GlobChars holds standard regular expression building blocks and structural character constants
// tailored for either POSIX or Windows operational platforms. Matches constants.js:29-67.
type GlobChars struct {
	DotLiteral   string
	PlusLiteral  string
	QmarkLiteral string
	SlashLiteral string
	OneChar      string
	Qmark        string
	EndAnchor    string
	StartAnchor  string
	DotsSlash    string
	NoDot        string
	NoDots       string
	NoDotSlash   string
	NoDotsSlash  string
	QmarkNoDot   string
	Star         string
	Sep          string
}

// ExtglobCharDef defines the syntax transformations for an extended glob operator symbol.
type ExtglobCharDef struct {
	Type  string
	Open  string
	Close string
}

// GetGlobChars instantiates platform-specific regular expression pattern strings matching constants.js globChars(win32).
func GetGlobChars(windows bool) *GlobChars {
	if windows {
		const winSlash = `\\/`
		const winNoSlash = `[^` + winSlash + `]`
		return &GlobChars{
			DotLiteral:   `\.`,
			PlusLiteral:  `\+`,
			QmarkLiteral: `\?`,
			SlashLiteral: `[` + winSlash + `]`,
			OneChar:      `(?=.)`,
			Qmark:        winNoSlash,
			EndAnchor:    `(?:[` + winSlash + `]|$)`,
			StartAnchor:  `(?:^|[` + winSlash + `])`,
			DotsSlash:    `\.{1,2}(?:[` + winSlash + `]|$)`,
			NoDot:        `(?!\.)`,
			NoDots:       `(?!(?:^|[` + winSlash + `])\.{1,2}(?:[` + winSlash + `]|$))`,
			NoDotSlash:   `(?!\.{0,1}(?:[` + winSlash + `]|$))`,
			NoDotsSlash:  `(?!\.{1,2}(?:[` + winSlash + `]|$))`,
			QmarkNoDot:   `[^.` + winSlash + `]`,
			Star:         winNoSlash + `*?`,
			Sep:          `\`,
		}
	}

	return &GlobChars{
		DotLiteral:   `\.`,
		PlusLiteral:  `\+`,
		QmarkLiteral: `\?`,
		SlashLiteral: `\/`,
		OneChar:      `(?=.)`,
		Qmark:        `[^/]`,
		EndAnchor:    `(?:\/|$)`,
		StartAnchor:  `(?:^|\/)`,
		DotsSlash:    `\.{1,2}(?:\/|$)`,
		NoDot:        `(?!\.)`,
		NoDots:       `(?!(?:^|\/)\.{1,2}(?:\/|$))`,
		NoDotSlash:   `(?!\.{0,1}(?:\/|$))`,
		NoDotsSlash:  `(?!\.{1,2}(?:\/|$))`,
		QmarkNoDot:   `[^.\/]`,
		Star:         `[^/]*?`,
		Sep:          `/`,
	}
}

// GetExtglobCharDefs constructs extglob operator transformations matching constants.js extglobChars(chars).
func GetExtglobCharDefs(chars *GlobChars) map[string]*ExtglobCharDef {
	return map[string]*ExtglobCharDef{
		"!": {Type: "negate", Open: "(?:(?!(?:", Close: "))" + chars.Star + ")"},
		"?": {Type: "qmark", Open: "(?:", Close: ")?"},
		"+": {Type: "plus", Open: "(?:", Close: ")+"},
		"*": {Type: "star", Open: "(?:", Close: ")*"},
		"@": {Type: "at", Open: "(?:", Close: ")"},
	}
}

// GetPosixRegexSource returns the regex character class string for a given POSIX class name.
// Matches original picomatch/lib/constants.js lines 73-89.
func GetPosixRegexSource(name string) (string, bool) {
	switch name {
	case "alnum":
		return "a-zA-Z0-9", true
	case "alpha":
		return "a-zA-Z", true
	case "ascii":
		return `\x00-\x7F`, true
	case "blank":
		return " \t", true
	case "cntrl":
		return `\x00-\x1F\x7F`, true
	case "digit":
		return "0-9", true
	case "graph":
		return `\x21-\x7E`, true
	case "lower":
		return "a-z", true
	case "print":
		return `\x20-\x7E `, true
	case "punct":
		return `\-!"#$%&\'()\*+,./:;<=>?@[\\]^_\` + "`" + `{|}~`, true
	case "space":
		return " \t\r\n\v\f", true
	case "upper":
		return "A-Z", true
	case "word":
		return "A-Za-z0-9_", true
	case "xdigit":
		return "A-Fa-f0-9", true
	default:
		return "", false
	}
}

// HasRegexChars evaluates whether a string contains standard regex structural syntax characters.
// Matches original picomatch/lib/utils.js hasRegexChars (REGEX_SPECIAL_CHARS: /[-*+?.^${}(|)[\]]/).
func HasRegexChars(str string) bool {
	for i := 0; i < len(str); i++ {
		switch str[i] {
		case '-', '*', '+', '?', '.', '^', '$', '{', '}', '(', ')', '|', '[', ']':
			return true
		}
	}
	return false
}

// IsRegexChar returns true if str is exactly one byte long and represents a regex syntax character.
// Matches original picomatch/lib/utils.js isRegexChar.
func IsRegexChar(str string) bool {
	return len(str) == 1 && HasRegexChars(str)
}

// EscapeRegex inserts backslashes before every regex special character in str.
// Matches original picomatch/lib/utils.js escapeRegex (str.replace(REGEX_SPECIAL_CHARS_GLOBAL, '\\$1')).
func EscapeRegex(str string) string {
	var sb strings.Builder
	sb.Grow(len(str) * 2)
	for i := 0; i < len(str); i++ {
		ch := str[i]
		switch ch {
		case '-', '*', '+', '?', '.', '^', '$', '{', '}', '(', ')', '|', '[', ']':
			sb.WriteByte('\\')
		}
		sb.WriteByte(ch)
	}
	return sb.String()
}

// Globstar synthesizes the globstar lookaround regex pattern matching parse.js:395-397.
func Globstar(opts *ParseOptions, chars *GlobChars) string {
	capture := "?:"
	if opts != nil && opts.Capture {
		capture = ""
	}
	dotLit := chars.DotLiteral
	if opts != nil && opts.Dot {
		dotLit = chars.DotsSlash
	}
	return fmt.Sprintf("(%s(?:(?!%s%s).)*?)", capture, chars.StartAnchor, dotLit)
}

// ExpandRange evaluates numerical or alphabetical brace ranges (e.g. {1..5}) and synthesizes a valid character class or escaped string.
// Matches original picomatch/lib/parse.js lines 22-38.
func ExpandRange(args []string, opts *ParseOptions) string {
	if opts != nil && opts.ExpandRange != nil && len(args) >= 2 {
		return opts.ExpandRange(args[0], args[1], opts)
	}

	sorted := make([]string, len(args))
	copy(sorted, args)
	sort.Strings(sorted)

	value := "[" + strings.Join(sorted, "-") + "]"
	if _, err := regexp.Compile(value); err != nil {
		escaped := make([]string, len(args))
		for i, v := range args {
			escaped[i] = EscapeRegex(v)
		}
		return strings.Join(escaped, "..")
	}

	return value
}

// ReDoSAnalysis captures the risk classification and consolidated character class output for repeated extglob structures.
type ReDoSAnalysis struct {
	Risky         bool
	SafeOutput    string
	HasSafeOutput bool
}

// RepeatedExtglobMatch models a recognized outer extglob quantifier expression.
type RepeatedExtglobMatch struct {
	Type byte
	Body string
	End  int
}

// SplitTopLevel divides an extglob expression body across top-level alternation pipe symbols outside brackets and parens.
// Matches original picomatch/lib/parse.js lines 48-98.
func SplitTopLevel(input string) []string {
	var parts []string
	bracket := 0
	paren := 0
	quote := 0
	value := ""
	escaped := false

	for i := 0; i < len(input); i++ {
		ch := input[i]
		if escaped {
			value += string(ch)
			escaped = false
			continue
		}
		if ch == '\\' {
			value += string(ch)
			escaped = true
			continue
		}
		if ch == '"' {
			if quote == 1 {
				quote = 0
			} else {
				quote = 1
			}
			value += string(ch)
			continue
		}
		if quote == 0 {
			if ch == '[' {
				bracket++
			} else if ch == ']' && bracket > 0 {
				bracket--
			} else if bracket == 0 {
				if ch == '(' {
					paren++
				} else if ch == ')' && paren > 0 {
					paren--
				} else if ch == '|' && paren == 0 {
					parts = append(parts, value)
					value = ""
					continue
				}
			}
		}
		value += string(ch)
	}
	parts = append(parts, value)
	return parts
}

// IsPlainBranch evaluates whether a branch consists solely of plain characters without wildcards or quantifiers.
// Matches original picomatch/lib/parse.js lines 100-120.
func IsPlainBranch(branch string) bool {
	escaped := false
	for i := 0; i < len(branch); i++ {
		ch := branch[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if ch == '?' || ch == '*' || ch == '+' || ch == '@' || ch == '!' || ch == '(' || ch == ')' || ch == '[' || ch == ']' || ch == '{' || ch == '}' {
			return false
		}
	}
	return true
}

// NormalizeSimpleBranch unrolls redundant @(literal) shells and unescapes literal branch sequences.
// Matches original picomatch/lib/parse.js lines 122-140.
func NormalizeSimpleBranch(branch string) string {
	value := strings.TrimSpace(branch)
	changed := true
	for changed {
		changed = false
		if len(value) > 3 && value[0] == '@' && value[1] == '(' && value[len(value)-1] == ')' {
			inner := value[2 : len(value)-1]
			hasDisallowed := false
			for i := 0; i < len(inner); i++ {
				ch := inner[i]
				if ch == '\\' || ch == '(' || ch == ')' || ch == '[' || ch == ']' || ch == '{' || ch == '}' || ch == '|' {
					hasDisallowed = true
					break
				}
			}
			if !hasDisallowed {
				value = inner
				changed = true
			}
		}
	}

	if !IsPlainBranch(value) {
		return ""
	}

	var sb strings.Builder
	escaped := false
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if !escaped && ch == '\\' {
			escaped = true
			continue
		}
		sb.WriteByte(ch)
		escaped = false
	}
	return sb.String()
}

// HasRepeatedCharPrefixOverlap detects problematic character repetition overlap across alternation branches prone to ReDoS.
// Matches original picomatch/lib/parse.js lines 142-162.
func HasRepeatedCharPrefixOverlap(branches []string) bool {
	var values []string
	for _, b := range branches {
		norm := NormalizeSimpleBranch(b)
		if norm != "" {
			values = append(values, norm)
		}
	}

	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			a := values[i]
			b := values[j]
			if len(a) == 0 || len(b) == 0 {
				continue
			}
			char := a[0]
			if a != strings.Repeat(string(char), len(a)) || b != strings.Repeat(string(char), len(b)) {
				continue
			}
			if a == b || strings.HasPrefix(a, b) || strings.HasPrefix(b, a) {
				return true
			}
		}
	}
	return false
}

// ParseRepeatedExtglob analyzes pattern substrings for leading repeated extglob quantifiers (+ / *).
// Matches original picomatch/lib/parse.js lines 164-231.
func ParseRepeatedExtglob(pattern string, requireEnd bool) *RepeatedExtglobMatch {
	if len(pattern) < 3 {
		return nil
	}
	if (pattern[0] != '+' && pattern[0] != '*') || pattern[1] != '(' {
		return nil
	}

	bracket := 0
	paren := 0
	quote := 0
	escaped := false

	for i := 1; i < len(pattern); i++ {
		ch := pattern[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if ch == '"' {
			if quote == 1 {
				quote = 0
			} else {
				quote = 1
			}
			continue
		}
		if quote == 1 {
			continue
		}
		if ch == '[' {
			bracket++
			continue
		}
		if ch == ']' && bracket > 0 {
			bracket--
			continue
		}
		if bracket > 0 {
			continue
		}
		if ch == '(' {
			paren++
			continue
		}
		if ch == ')' {
			paren--
			if paren == 0 {
				if requireEnd && i != len(pattern)-1 {
					return nil
				}
				return &RepeatedExtglobMatch{
					Type: pattern[0],
					Body: pattern[2:i],
					End:  i,
				}
			}
		}
	}
	return nil
}

// BuildCharClassStar synthesizes a flat, ReDoS-safe consolidated character class star expression.
// Matches original picomatch/lib/parse.js lines 233-239.
func BuildCharClassStar(chars []string) string {
	if len(chars) == 1 {
		return EscapeRegex(chars[0]) + "*"
	}
	var sb strings.Builder
	for _, ch := range chars {
		sb.WriteString(EscapeRegex(ch))
	}
	return "[" + sb.String() + "]*"
}

// GetStarExtglobSequenceChars extracts uniform literal sequences from star extglob patterns.
// Matches original picomatch/lib/parse.js lines 241-271.
func GetStarExtglobSequenceChars(pattern string) []string {
	index := 0
	var chars []string

	for index < len(pattern) {
		match := ParseRepeatedExtglob(pattern[index:], false)
		if match == nil || match.Type != '*' {
			return nil
		}
		branches := SplitTopLevel(match.Body)
		if len(branches) != 1 {
			return nil
		}
		branch := NormalizeSimpleBranch(branches[0])
		if branch == "" || len(branch) != 1 {
			return nil
		}
		chars = append(chars, branch)
		index += match.End + 1
	}

	if len(chars) < 1 {
		return nil
	}
	return chars
}

// RepeatedExtglobRecursion determines the recursive nesting depth of repeated extglob structures.
// Matches original picomatch/lib/parse.js lines 273-285.
func RepeatedExtglobRecursion(pattern string) int {
	depth := 0
	value := strings.TrimSpace(pattern)
	match := ParseRepeatedExtglob(value, true)
	for match != nil {
		depth++
		value = strings.TrimSpace(match.Body)
		match = ParseRepeatedExtglob(value, true)
	}
	return depth
}

// AnalyzeRepeatedExtglob performs catastrophic backtracking risk analysis across an active extglob body.
// Matches original picomatch/lib/parse.js lines 287-347.
func AnalyzeRepeatedExtglob(body string, opts *ParseOptions) ReDoSAnalysis {
	if opts != nil && opts.NoExtglobRecursion {
		return ReDoSAnalysis{Risky: false}
	}

	max := 0
	if opts != nil && opts.MaxExtglobRecursion > 0 {
		max = opts.MaxExtglobRecursion
	}

	branchesRaw := SplitTopLevel(body)
	var branches []string
	for _, b := range branchesRaw {
		branches = append(branches, strings.TrimSpace(b))
	}

	if len(branches) > 1 {
		hasEmpty := false
		hasOnlyWilds := false
		for _, b := range branches {
			if b == "" {
				hasEmpty = true
			}
			onlyWild := len(b) > 0
			for i := 0; i < len(b); i++ {
				if b[i] != '*' && b[i] != '?' {
					onlyWild = false
					break
				}
			}
			if onlyWild {
				hasOnlyWilds = true
			}
		}
		if hasEmpty || hasOnlyWilds || HasRepeatedCharPrefixOverlap(branches) {
			return ReDoSAnalysis{Risky: true}
		}
	}

	var safeChars []string
	sawStarSequence := false
	combinable := true

	for _, branch := range branches {
		chars := GetStarExtglobSequenceChars(branch)
		if len(chars) > 0 {
			sawStarSequence = true
			safeChars = append(safeChars, chars...)
			continue
		}

		literal := NormalizeSimpleBranch(branch)
		if literal != "" && len(literal) == 1 {
			safeChars = append(safeChars, literal)
			continue
		}

		combinable = false

		if RepeatedExtglobRecursion(branch) > max {
			return ReDoSAnalysis{Risky: true}
		}
	}

	if sawStarSequence {
		if combinable {
			seen := make(map[string]bool)
			var uniq []string
			for _, c := range safeChars {
				if !seen[c] {
					seen[c] = true
					uniq = append(uniq, c)
				}
			}
			return ReDoSAnalysis{
				Risky:         true,
				SafeOutput:    BuildCharClassStar(uniq),
				HasSafeOutput: true,
			}
		}
		return ReDoSAnalysis{Risky: true}
	}

	return ReDoSAnalysis{Risky: false}
}
