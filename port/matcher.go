package picomatch

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// Matcher encapsulates a compiled regular expression and state metadata needed to perform pattern matching on input strings.
type Matcher struct {
	Pattern        string
	Regexp         *regexp.Regexp
	Opts           *ParseOptions
	State          *ParseState
	Negated        bool
	IgnoreMatchers []*Matcher
	IsExtglobNeg   bool
	patSegments    []string
}

type cacheKeyStruct struct {
	pattern             string
	nilOpts             bool
	windows             bool
	maxLength           int
	prepend             string
	capture             bool
	dot                 bool
	bash                bool
	noExt               bool
	noExtglob           bool
	fastpaths           bool
	unescape            bool
	contains            bool
	keepQuotes          bool
	strictBrackets      bool
	noBracket           bool
	posix               bool
	literalBrackets     bool
	noBrace             bool
	noGlobstar          bool
	strictSlashes       bool
	regex               bool
	noNegate            bool
	maxExtglobRecursion int
	noExtglobRecursion  bool
	matchBase           bool
	basename            bool
	ignore              string
	nocase              bool
	debug               bool
}

var (
	cacheMu      sync.RWMutex
	cacheNilOpts = make(map[string]*Matcher)
	cache        = make(map[cacheKeyStruct]*Matcher)
	maxCacheSize = 1024
)

// cacheKey generates a unique structural lookup key for compiled matcher persistence.
func cacheKey(pattern string, opts *ParseOptions) cacheKeyStruct {
	if opts == nil {
		return cacheKeyStruct{pattern: pattern, nilOpts: true}
	}
	var ignore string
	if len(opts.Ignore) > 0 {
		ignore = strings.Join(opts.Ignore, "\x00")
	}
	return cacheKeyStruct{
		pattern:             pattern,
		nilOpts:             false,
		windows:             opts.Windows,
		maxLength:           opts.MaxLength,
		prepend:             opts.Prepend,
		capture:             opts.Capture,
		dot:                 opts.Dot,
		bash:                opts.Bash,
		noExt:               opts.NoExt,
		noExtglob:           opts.NoExtglob,
		fastpaths:           opts.Fastpaths,
		unescape:            opts.Unescape,
		contains:            opts.Contains,
		keepQuotes:          opts.KeepQuotes,
		strictBrackets:      opts.StrictBrackets,
		noBracket:           opts.NoBracket,
		posix:               opts.Posix,
		literalBrackets:     opts.LiteralBrackets,
		noBrace:             opts.NoBrace,
		noGlobstar:          opts.NoGlobstar,
		strictSlashes:       opts.StrictSlashes,
		regex:               opts.Regex,
		noNegate:            opts.NoNegate,
		maxExtglobRecursion: opts.MaxExtglobRecursion,
		noExtglobRecursion:  opts.NoExtglobRecursion,
		matchBase:           opts.MatchBase,
		basename:            opts.Basename,
		ignore:              ignore,
		nocase:              opts.Nocase,
		debug:               opts.Debug,
	}
}

// clearCache resets the compiled matcher storage when upper bounds are exceeded.
func clearCache() {
	cacheNilOpts = make(map[string]*Matcher)
	cache = make(map[cacheKeyStruct]*Matcher)
}

// Compile constructs an executable pattern matcher from a glob expression and runtime configuration.
func Compile(pattern string, opts *ParseOptions) (*Matcher, error) {
	if pattern == "" {
		return nil, errors.New("expected pattern to be a non-empty string")
	}

	cacheMu.RLock()
	if opts == nil {
		if cached, found := cacheNilOpts[pattern]; found {
			cacheMu.RUnlock()
			return cached, nil
		}
	} else {
		key := cacheKey(pattern, opts)
		if cached, found := cache[key]; found {
			cacheMu.RUnlock()
			return cached, nil
		}
	}
	cacheMu.RUnlock()

	state, err := Parse(pattern, opts)
	if err != nil {
		return nil, err
	}

	matcher := &Matcher{
		Pattern: pattern,
		Opts:    opts,
		State:   state,
		Negated: state.Negated && (opts == nil || !opts.NoNegate),
	}

	patternNorm := pattern
	if strings.Contains(patternNorm, "\\") {
		patternNorm = strings.ReplaceAll(patternNorm, "\\", "/")
	}
	if matcher.Negated && strings.HasPrefix(patternNorm, "!") {
		patternNorm = strings.TrimPrefix(patternNorm, "!")
	}
	matcher.patSegments = strings.Split(patternNorm, "/")

	if opts != nil && len(opts.Ignore) > 0 {
		ignoreOpts := *opts
		ignoreOpts.Ignore = nil
		for _, ignorePat := range opts.Ignore {
			if igMatcher, igErr := Compile(ignorePat, &ignoreOpts); igErr == nil {
				matcher.IgnoreMatchers = append(matcher.IgnoreMatchers, igMatcher)
			}
		}
	}

	chars := GetGlobChars(opts != nil && opts.Windows)
	prepend := "^"
	appendStr := "$"
	if opts != nil && opts.Contains {
		prepend = ""
		appendStr = ""
	}

	rawOutput := state.Output
	source := fmt.Sprintf("%s(?:%s)%s", prepend, rawOutput, appendStr)
	re2Source := toRE2(source, opts, chars)

	if opts != nil && opts.Nocase {
		re2Source = "(?i)" + re2Source
	}

	re, err := regexp.Compile(re2Source)
	if err != nil {
		// Detect irreconcilable lookahead semantics (e.g. complex extglob negation)
		if strings.Contains(rawOutput, "(?:(?!(?:") {
			matcher.IsExtglobNeg = true
		} else {
			return nil, fmt.Errorf("failed to compile regular expression %q: %w", re2Source, err)
		}
	}
	matcher.Regexp = re

	cacheMu.Lock()
	if len(cacheNilOpts)+len(cache) >= maxCacheSize {
		clearCache()
	}
	if opts == nil {
		cacheNilOpts[pattern] = matcher
	} else {
		cache[cacheKey(pattern, opts)] = matcher
	}
	cacheMu.Unlock()

	return matcher, nil
}

// toRE2 converts JavaScript regular expression lookaround assertions generated by the parser into RE2-compatible syntax.
func toRE2(source string, opts *ParseOptions, chars *GlobChars) string {
	// 1. Remove Dot/Dir exclusion lookaheads
	source = strings.ReplaceAll(source, chars.NoDot, "")
	source = strings.ReplaceAll(source, chars.NoDots, "")
	source = strings.ReplaceAll(source, chars.NoDotSlash, "")
	source = strings.ReplaceAll(source, chars.NoDotsSlash, "")

	source = strings.ReplaceAll(source, `(?!\.)`, "")
	source = strings.ReplaceAll(source, `(?!(?:^|\/)\.{1,2}(?:\/|$))`, "")
	source = strings.ReplaceAll(source, `(?!\.{0,1}(?:\/|$))`, "")
	source = strings.ReplaceAll(source, `(?!\.{1,2}(?:\/|$))`, "")
	source = strings.ReplaceAll(source, `(?!(?:^|[\/])\.{1,2}(?:[\/]|$))`, "")
	source = strings.ReplaceAll(source, `(?!\.{0,1}(?:[\/]|$))`, "")
	source = strings.ReplaceAll(source, `(?!\.{1,2}(?:[\/]|$))`, "")
	source = strings.ReplaceAll(source, `(?!(?:^|[\\/])\.{1,2}(?:[\\/]|$))`, "")
	source = strings.ReplaceAll(source, `(?!\.{0,1}(?:[\\/]|$))`, "")
	source = strings.ReplaceAll(source, `(?!\.{1,2}(?:[\\/]|$))`, "")
	source = strings.ReplaceAll(source, `(?!(?:^|\\/)\.{1,2}(?:\\/|$))`, "")
	source = strings.ReplaceAll(source, `(?!\.{0,1}(?:\\/|$))`, "")
	source = strings.ReplaceAll(source, `(?!\.{1,2}(?:\\/|$))`, "")

	// 2. Simplify Globstar lookaround loops to RE2 non-greedy match
	source = strings.ReplaceAll(source, `(?:(?!(?:^|\/)\.).)*?`, `.*?`)
	source = strings.ReplaceAll(source, `(?:(?!(?:^|\/)\.{1,2}(?:\/|$)).)*?`, `.*?`)
	source = strings.ReplaceAll(source, `(?:(?!(?:^|[\/])\.).)*?`, `.*?`)
	source = strings.ReplaceAll(source, `(?:(?!(?:^|[\/])\.{1,2}(?:[\/]|$)).)*?`, `.*?`)
	source = strings.ReplaceAll(source, `(?:(?!(?:^|[\\/])\.).)*?`, `.*?`)
	source = strings.ReplaceAll(source, `(?:(?!(?:^|[\\/])\.{1,2}(?:[\\/]|$)).)*?`, `.*?`)
	source = strings.ReplaceAll(source, `(?:(?!(?:^|\\/)\.).)*?`, `.*?`)
	source = strings.ReplaceAll(source, `(?:(?!(?:^|\\/)\.{1,2}(?:\\/|$)).)*?`, `.*?`)

	// 3. Handle OneChar (?=.) assertion
	source = strings.ReplaceAll(source, `(?=.)[^/]*?)$`, `[^/]+?)$`)
	source = strings.ReplaceAll(source, `(?=.)[^/]*?$`, `[^/]+?$`)
	source = strings.ReplaceAll(source, `(?=.)[^\\/]*?)$`, `[^\\/]+?)$`)
	source = strings.ReplaceAll(source, `(?=.)[^\\/]*?$`, `[^\\/]+?$`)
	source = strings.ReplaceAll(source, `(?=.).*?)$`, `.+?)$`)
	source = strings.ReplaceAll(source, `(?=.).*?$`, `.+?$`)
	source = strings.ReplaceAll(source, `(?=.)`, "")

	return source
}

// basename extracts the trailing filename component from a path string across platform separator boundaries.
func basename(input string, windows bool) string {
	if windows || strings.Contains(input, "\\") {
		input = strings.ReplaceAll(input, "\\", "/")
	}
	idx := strings.LastIndex(input, "/")
	if idx >= 0 {
		return input[idx+1:]
	}
	return input
}

// Match evaluates an input string against a glob pattern using runtime matching rules and option constraints.
func Match(pattern, input string, opts *ParseOptions) (bool, error) {
	matcher, err := Compile(pattern, opts)
	if err != nil {
		return false, err
	}
	return matcher.Match(input), nil
}

// Match tests an input string against the compiled Matcher instance.
func (m *Matcher) Match(input string) bool {
	if input == "" {
		return false
	}

	output := input
	if m.Opts != nil {
		if m.Opts.Format != nil {
			output = m.Opts.Format(input)
		} else if m.Opts.Posix || m.Opts.Windows {
			output = strings.ReplaceAll(input, "\\", "/")
		}
	}

	if input == m.Pattern || output == m.Pattern {
		return !m.Negated
	}

	target := output
	if m.Opts != nil && (m.Opts.MatchBase || m.Opts.Basename) {
		target = basename(output, m.Opts.Windows)
	}

	// Runtime verification of dotfile and special directory constraints
	if !m.validateDotAndSpecialDirs(target) {
		return false
	}

	var isMatch bool
	if m.Regexp != nil {
		isMatch = m.Regexp.MatchString(target)
	} else if m.IsExtglobNeg {
		// Structural fallback evaluation for complex negated extglob syntax
		isMatch = m.evaluateExtglobNegation(target)
	}

	if m.Negated {
		isMatch = !isMatch
	}

	if isMatch && len(m.IgnoreMatchers) > 0 {
		for _, ig := range m.IgnoreMatchers {
			if ig.Match(input) {
				return false
			}
		}
	}

	return isMatch
}

// validateDotAndSpecialDirs verifies that wildcard pattern segments do not inappropriately match hidden dotfiles or special directory identifiers.
func (m *Matcher) validateDotAndSpecialDirs(path string) bool {
	norm := path
	if strings.Contains(norm, "\\") {
		norm = strings.ReplaceAll(norm, "\\", "/")
	}
	allowDot := m.Opts != nil && m.Opts.Dot

	i := 0
	rem := norm
	for {
		idx := strings.IndexByte(rem, '/')
		var seg string
		if idx == -1 {
			seg = rem
		} else {
			seg = rem[:idx]
		}

		if seg == "." || seg == ".." {
			// Special directory components are forbidden unless explicitly matched by pattern literals
			if i < len(m.patSegments) && m.patSegments[i] == seg {
				// Continue to next segment
			} else {
				return false
			}
		} else if strings.HasPrefix(seg, ".") && !allowDot {
			// Dotfile segment must be matched by an explicit leading dot in the corresponding pattern segment
			isDotFilePat := func(s string) bool {
				return s != "." && s != ".." && (strings.HasPrefix(s, ".") || strings.HasPrefix(s, "\\."))
			}
			matchedDot := false
			if i < len(m.patSegments) {
				if isDotFilePat(m.patSegments[i]) {
					matchedDot = true
				}
			}
			if !matchedDot && idx == -1 && len(m.patSegments) > 0 {
				if isDotFilePat(m.patSegments[len(m.patSegments)-1]) {
					matchedDot = true
				}
			}
			if !matchedDot {
				seenGlobstar := false
				for _, pSeg := range m.patSegments {
					if pSeg == "**" {
						seenGlobstar = true
					} else if seenGlobstar && isDotFilePat(pSeg) {
						matchedDot = true
						break
					}
				}
			}
			if !matchedDot && !strings.HasPrefix(m.Pattern, ".*") && !strings.HasPrefix(m.Pattern, "**/.*") {
				return false
			}
		}

		i++
		if idx == -1 {
			break
		}
		rem = rem[idx+1:]
	}
	return true
}

// evaluateExtglobNegation provides fallback structural pattern evaluation when lookahead assertions cannot be compiled by RE2.
// It resolves composite and nested negated extglob expressions via formal set differentiation: !(X) ≡ * \ @(X).
func (m *Matcher) evaluateExtglobNegation(input string) bool {
	pat := m.Pattern
	if m.Negated && strings.HasPrefix(pat, "!") && !strings.HasPrefix(pat, "!(") {
		pat = strings.TrimPrefix(pat, "!")
	}
	return m.evalExtglobPattern(pat, input)
}

func (m *Matcher) evalExtglobPattern(pat, target string) bool {
	if branches := splitPositiveExtglob(pat); len(branches) > 1 {
		for _, branch := range branches {
			if m.evalExtglobPattern(branch, target) {
				return true
			}
		}
		return false
	}
	start, end := findNegatedExtglob(pat)
	if start == -1 {
		sub, err := Compile(pat, m.Opts)
		if err != nil || sub.Regexp == nil {
			return false
		}
		return sub.Regexp.MatchString(target)
	}
	patWildcard := pat[:start] + "*" + pat[end:]
	patPositive := pat[:start] + "@(" + pat[start+2:end] + pat[end:]
	return m.evalExtglobPattern(patWildcard, target) && !m.evalExtglobPattern(patPositive, target)
}

func splitPositiveExtglob(pat string) []string {
	for i := 0; i < len(pat)-1; i++ {
		if pat[i] == '\\' {
			i++
			continue
		}
		ch := pat[i]
		if (ch == '@' || ch == '?' || ch == '+' || ch == '*') && pat[i+1] == '(' {
			depth := 1
			var j int
			var brIdx []int
			for j = i + 2; j < len(pat); j++ {
				if pat[j] == '\\' {
					j++
					continue
				}
				if pat[j] == '(' {
					depth++
				} else if pat[j] == ')' {
					depth--
					if depth == 0 {
						break
					}
				} else if pat[j] == '|' && depth == 1 {
					brIdx = append(brIdx, j)
				}
			}
			if len(brIdx) > 0 && j < len(pat) {
				inner := pat[i+2 : j]
				var branches []string
				last := 0
				for _, idx := range brIdx {
					rel := idx - (i + 2)
					branches = append(branches, inner[last:rel])
					last = rel + 1
				}
				branches = append(branches, inner[last:])

				result := make([]string, len(branches))
				for k, br := range branches {
					result[k] = pat[:i] + string(ch) + "(" + br + ")" + pat[j+1:]
				}
				return result
			}
		}
	}
	return nil
}

func findNegatedExtglob(pat string) (int, int) {
	for i := 0; i < len(pat); i++ {
		if pat[i] == '\\' {
			i++
			continue
		}
		if pat[i] == '!' && i+1 < len(pat) && pat[i+1] == '(' {
			depth := 1
			for j := i + 2; j < len(pat); j++ {
				if pat[j] == '\\' {
					j++
					continue
				}
				if pat[j] == '(' {
					depth++
				} else if pat[j] == ')' {
					depth--
					if depth == 0 {
						return i, j + 1
					}
				}
			}
			return -1, -1
		}
	}
	return -1, -1
}
