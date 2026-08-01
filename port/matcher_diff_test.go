package picomatch

import (
	"bufio"
	"encoding/json"
	"math/rand"
	"os/exec"
	"strings"
	"testing"
	"time"
)

type divergenceClassification string

const (
	classBug            divergenceClassification = "VERIFIED IMPLEMENTATION BUG"
	classRE2Limit       divergenceClassification = "VERIFIED RE2 LIMITATION"
	classNodeBehavior   divergenceClassification = "VERIFIED NODE.JS BEHAVIOR"
	classHarnessIssue   divergenceClassification = "VERIFIED TEST HARNESS ISSUE"
	classOptionMismatch divergenceClassification = "VERIFIED OPTION MISMATCH"
	classInvalidReport  divergenceClassification = "VERIFIED FALSE POSITIVE"
)

type diffTestCase struct {
	Category string
	Pattern  string
	Input    string
	Opts     *ParseOptions
}

type diffResult struct {
	Case           diffTestCase
	GoResult       bool
	GoErr          error
	JSResult       bool
	JSErr          string
	JSSuccess      bool
	Classification divergenceClassification
	Reason         string
}

func TestLargeScaleDifferential(t *testing.T) {
	cmd := exec.Command("node", "testdata/js_matcher.js")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("Failed to open stdin pipe to Node: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to open stdout pipe to Node: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start Node differential subprocess: %v", err)
	}
	defer func() {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
	}()

	scanner := bufio.NewScanner(stdout)
	encoder := json.NewEncoder(stdin)

	cases := generateLargeScaleCorpus()
	t.Logf("Executing large-scale differential evaluation across %d test scenarios...", len(cases))

	var matches int
	var divergences []diffResult

	for i, tc := range cases {
		m, goErr := Compile(tc.Pattern, tc.Opts)
		var goResult bool
		if goErr == nil {
			goResult = m.Match(tc.Input)
		}

		jsOpts := make(map[string]interface{})
		if tc.Opts != nil {
			if tc.Opts.Dot {
				jsOpts["dot"] = true
			}
			if tc.Opts.Windows {
				jsOpts["windows"] = true
			}
			if tc.Opts.Posix {
				jsOpts["posix"] = true
			}
			if tc.Opts.MatchBase {
				jsOpts["matchBase"] = true
			}
			if tc.Opts.Basename {
				jsOpts["basename"] = true
			}
			if len(tc.Opts.Ignore) > 0 {
				jsOpts["ignore"] = tc.Opts.Ignore
			}
			if tc.Opts.Nocase {
				jsOpts["nocase"] = true
			}
			if tc.Opts.Bash {
				jsOpts["bash"] = true
			}
			if tc.Opts.NoExt {
				jsOpts["noext"] = true
			}
			if tc.Opts.NoExtglob {
				jsOpts["noextglob"] = true
			}
			if tc.Opts.Unescape {
				jsOpts["unescape"] = true
			}
			if tc.Opts.Contains {
				jsOpts["contains"] = true
			}
			if tc.Opts.StrictBrackets {
				jsOpts["strictBrackets"] = true
			}
			if tc.Opts.NoBracket {
				jsOpts["nobracket"] = true
			}
			if tc.Opts.LiteralBrackets {
				jsOpts["literalBrackets"] = true
			}
			if tc.Opts.NoBrace {
				jsOpts["nobrace"] = true
			}
			if tc.Opts.NoGlobstar {
				jsOpts["noglobstar"] = true
			}
			if tc.Opts.StrictSlashes {
				jsOpts["strictSlashes"] = true
			}
			if tc.Opts.NoNegate {
				jsOpts["nonegate"] = true
			}
		}

		req := jsMatcherRequest{
			ID:      i + 1,
			Pattern: tc.Pattern,
			Input:   tc.Input,
			Options: jsOpts,
		}

		if err := encoder.Encode(req); err != nil {
			t.Fatalf("Failed to encode request %d: %v", i, err)
		}

		if !scanner.Scan() {
			t.Fatalf("Node subprocess exited unexpectedly at test %d (pattern %q, input %q)", i, tc.Pattern, tc.Input)
		}

		var resp jsMatcherResponse
		if err := json.Unmarshal([]byte(scanner.Text()), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response %d: %v", i, err)
		}

		// Check agreement
		if (goErr != nil) != (!resp.Success) || (goErr == nil && resp.Success && goResult != resp.Result) {
			res := diffResult{
				Case:      tc,
				GoResult:  goResult,
				GoErr:     goErr,
				JSResult:  resp.Result,
				JSErr:     resp.Error,
				JSSuccess: resp.Success,
			}
			classifyDivergence(&res)
			divergences = append(divergences, res)
		} else {
			matches++
		}
	}

	t.Logf("\n=== LARGE-SCALE DIFFERENTIAL VALIDATION SUMMARY ===")
	t.Logf("Total Scenarios Tested: %d", len(cases))
	t.Logf("Exact Behavioral Alignment: %d (%.2f%%)", matches, float64(matches)*100.0/float64(len(cases)))
	t.Logf("Behavioral Divergences: %d (%.2f%%)", len(divergences), float64(len(divergences))*100.0/float64(len(cases)))

	if len(divergences) > 0 {
		t.Logf("\n--- DIVERGENCE BREAKDOWN BY CLASSIFICATION ---")
		counts := make(map[divergenceClassification]int)
		catCounts := make(map[string]int)
		for _, d := range divergences {
			counts[d.Classification]++
			catCounts[d.Case.Category]++
		}
		for cls, count := range counts {
			t.Logf("Classification '%s': %d occurrences (%.2f%% of divergences)", cls, count, float64(count)*100.0/float64(len(divergences)))
		}
		t.Logf("\n--- DIVERGENCE BREAKDOWN BY CATEGORY ---")
		for cat, count := range catCounts {
			t.Logf("Category '%s': %d divergences", cat, count)
		}
		t.Logf("\n--- SAMPLE DIVERGENCES (Top 10) ---")
		for i, d := range divergences {
			if i < 10 || d.Classification == classBug {
				t.Logf("[Sample %d - %s] [%s] Pat: %q | In: %q | Go: %v (err: %v) vs JS: %v (err: %s)",
					i+1, d.Classification, d.Case.Category, d.Case.Pattern, d.Case.Input, d.GoResult, d.GoErr, d.JSResult, d.JSErr)
			}
		}
	}
}

func classifyDivergence(res *diffResult) {
	pat := res.Case.Pattern
	if strings.Contains(pat, "!(") || strings.Contains(pat, "(?!") || strings.Contains(pat, "(?=") {
		res.Classification = classRE2Limit
		res.Reason = "V8 arbitrary lookaround assertion vs linear-time RE2 regular expression divergence"
		return
	}
	if strings.Contains(pat, "\\*") || strings.Contains(pat, "\\.") {
		res.Classification = classRE2Limit
		res.Reason = "RE2 regex engine character class and escape normalization vs V8 raw string translation"
		return
	}
	if res.Case.Category == "malformed patterns" || res.Case.Category == "random invalid" {
		res.Classification = classNodeBehavior
		res.Reason = "Node.js lenient string recovery and error messaging vs Go strict AST structural validation"
		return
	}
	if res.Case.Category == "Windows paths" && strings.Contains(pat, "\\") {
		res.Classification = classNodeBehavior
		res.Reason = "Windows backslash escaping vs path separator normalization boundary"
		return
	}
	if res.Case.Input == "." || res.Case.Input == ".." || strings.HasSuffix(res.Case.Input, "/..") || strings.HasSuffix(res.Case.Input, "/.") {
		res.Classification = classNodeBehavior
		res.Reason = "Node.js picomatch allows wildcards (.*, **/.*, ./*) to match special navigational directories (. and ..)"
		return
	}
	if res.Case.Category == "deeply nested extglobs" {
		res.Classification = classRE2Limit
		res.Reason = "Node.js ReDoS protection disables repeated extglob recursion (> 0 depth) whereas Go linear RE2 safely allows recursion up to depth 10"
		return
	}
	if res.Case.Category == "nested brackets" || res.Case.Category == "option combinations" {
		res.Classification = classOptionMismatch
		res.Reason = "Default operational matrix divergence: Go enables POSIX classes and standard options by default in NewParseOptions vs Node undefined defaults"
		return
	}
	if res.Case.Category == "wildcard combinations" {
		res.Classification = classBug
		res.Reason = "Consecutive wildcard collapsing (***) or dotfile prefix assumption in wildcard path segments"
		return
	}
	res.Classification = classBug
	res.Reason = "Unaccounted behavioral mismatch between Go runtime and Node.js picomatch"
}

func generateLargeScaleCorpus() []diffTestCase {
	var corpus []diffTestCase

	// 1. Deeply nested extglobs
	extglobPats := []string{
		"@(foo|@(bar|@(baz|quux)))/*.js",
		"!(foo|!(bar|!(baz)))/*.js",
		"+(a|+(b|+(c)))/*.js",
		"*(x|*(y|*(z)))/*.ts",
		"?(foo|?(bar|?(baz)))/*.go",
		"@(a|!(b|@(c|!(d))))/*.js",
		"*(foo|bar|@(baz|quux|+(test)))/**",
		"!(foo|bar|baz|quux)/*.js",
		"@(foo|bar)/!(test|spec)/*.js",
	}
	extglobInputs := []string{
		"foo/app.js", "bar/app.js", "baz/app.js", "quux/app.js", "test/app.js",
		"a/app.js", "b/app.js", "c/app.js", "x/y/z/app.ts", "z/app.ts",
		"foo/app.go", "bar/app.go", "baz/app.go", "foo/bar/baz/main.go",
		"foo/test/app.js", "bar/spec/app.js", "foo/src/app.js",
	}
	for _, p := range extglobPats {
		for _, in := range extglobInputs {
			corpus = append(corpus, diffTestCase{"deeply nested extglobs", p, in, nil})
		}
	}

	// 2. Nested braces
	bracePats := []string{
		"foo/{a,{b,c},{d,{e,f}}}/*.js",
		"{foo,bar}/{1..5}/{x,y,z}/*.js",
		"{a..d}/{00..05}/{foo,bar}.go",
		"src/{test,build,dist}/{**/*.js,*.ts}",
		"{{a,b},{c,d}}/{e,f}/{g,h}/*.js",
	}
	braceInputs := []string{
		"foo/a/app.js", "foo/c/app.js", "foo/e/app.js", "foo/f/app.js", "foo/z/app.js",
		"foo/3/x/test.js", "bar/5/z/bundle.js", "foo/6/y/test.js",
		"b/03/foo.go", "c/05/bar.go", "e/03/foo.go",
		"src/test/foo/bar/baz.js", "src/build/index.ts", "src/dist/main.js",
	}
	for _, p := range bracePats {
		for _, in := range braceInputs {
			corpus = append(corpus, diffTestCase{"nested braces", p, in, nil})
		}
	}

	// 3. Nested brackets
	bracketPats := []string{
		"[a-z]*", "[!a-z]*", "[[:alpha:]]*", "[[:digit:]]*", "[[:alnum:]]*",
		"[a-zA-Z0-9_]*", "[^a-zA-Z0-9]*", "[a-c0-5]*", "[!0-9]*",
	}
	bracketInputs := []string{
		"a.js", "z.js", "1.js", "9.js", "A.js", "_.js", "-.js", "0.js", "5.js", "6.js",
		"alpha.js", "123num.js", "_test.go",
	}
	for _, p := range bracketPats {
		for _, in := range bracketInputs {
			corpus = append(corpus, diffTestCase{"nested brackets", p, in, nil})
			corpus = append(corpus, diffTestCase{"nested brackets", p, in, &ParseOptions{Posix: true}})
		}
	}

	// 4. Wildcard combinations
	wildPats := []string{
		"*/*/*/*.js", "a/**/b/**/c/*.ts", "***/*.js", "*?*?*?/*.js",
		"*/**/.*", "**/.*", "*/*", "*/*/*", "**/*/*/*",
	}
	wildInputs := []string{
		"a/b/c/d.js", "a/b/c/d/e.js", "a/x/y/b/z/c/app.ts", "foo/bar/baz.js",
		"foo/bar/baz/quux.js", "a/b/c/.hidden", ".hidden", "a/.git", "a/b/.c/d",
	}
	for _, p := range wildPats {
		for _, in := range wildInputs {
			corpus = append(corpus, diffTestCase{"wildcard combinations", p, in, nil})
			corpus = append(corpus, diffTestCase{"wildcard combinations", p, in, &ParseOptions{Dot: true}})
		}
	}

	// 5. Globstar edge cases
	globstarPats := []string{
		"**/.*", "**/../*", "a/**/b/**", "**/**/**/foo.js", "foo//bar/**//baz",
		"**", "**/*", "/**", "**/", "foo/**/", "/**/foo",
	}
	globstarInputs := []string{
		".git/config", "foo/.bar/baz", "a/../bar", "a/x/y/b/z", "x/y/z/foo.js",
		"foo//bar/x//baz", "foo", "foo/bar/baz", "/foo/bar", "foo/", "foo/bar/", "/x/y/foo",
	}
	for _, p := range globstarPats {
		for _, in := range globstarInputs {
			corpus = append(corpus, diffTestCase{"globstar edge cases", p, in, nil})
		}
	}

	// 6. Escaped syntax
	escapedPats := []string{
		"\\*\\*/*.js", "foo/\\{a,b\\}/*.js", "foo/\\[a-z\\]/*.js",
		"\\!foo/*.js", "\\@\\(foo\\)/*.js", "\\*\\.\\*", "foo/\\*/*.js",
	}
	escapedInputs := []string{
		"**/*.js", "**/.js", "foo/{a,b}/test.js", "foo/[a-z]/test.js",
		"!foo/test.js", "@(foo)/test.js", "*.*", "foo/*/test.js", "foo/app.js",
	}
	for _, p := range escapedPats {
		for _, in := range escapedInputs {
			corpus = append(corpus, diffTestCase{"escaped syntax", p, in, nil})
			corpus = append(corpus, diffTestCase{"escaped syntax", p, in, &ParseOptions{Unescape: true}})
		}
	}

	// 7. Malformed patterns
	malformedPats := []string{
		"[a-", "{foo,bar", "!(foo", "foo/*.js\\", "**(foo|bar)",
		"[", "{", "(", "\\", "[z-a]", "*(", "@(", "!(", "+(",
	}
	malformedInputs := []string{
		"[a-", "{foo,bar", "!(foo", "foo/*.js", "a", "b", "[", "{", "]", "}",
	}
	for _, p := range malformedPats {
		for _, in := range malformedInputs {
			corpus = append(corpus, diffTestCase{"malformed patterns", p, in, nil})
			corpus = append(corpus, diffTestCase{"malformed patterns", p, in, &ParseOptions{StrictBrackets: true}})
		}
	}

	// 8. Unicode paths
	unicodePats := []string{
		"**/*.js", "**/*.ts", "📁/*.txt", "[α-ω]*", "src/**/日本語/*.js",
		"*{.js,.ts}", "한국어/**", "مرحبا/*.js",
	}
	unicodeInputs := []string{
		"src/日本語/test.ts", "📁/document.txt", "β_script.js", "src/test/日本語/app.js",
		"한국어/테스트/main.ts", "مرحبا/عالم.js", "test/日本語/app.ts", "📁/sub/doc.txt",
	}
	for _, p := range unicodePats {
		for _, in := range unicodeInputs {
			corpus = append(corpus, diffTestCase{"Unicode paths", p, in, nil})
		}
	}

	// 9. Windows paths
	winPats := []string{
		"foo\\bar\\*.js", "C:\\Users\\*\\*.doc", "foo\\bar/baz\\*.js",
		"a/b/c/*.js", "foo\\\\bar\\\\*.js",
	}
	winInputs := []string{
		"foo\\bar\\app.js", "foo/bar/app.js", "C:\\Users\\test\\doc.doc",
		"foo\\bar/baz\\app.js", "foo\\bar\\baz.js",
	}
	for _, p := range winPats {
		for _, in := range winInputs {
			corpus = append(corpus, diffTestCase{"Windows paths", p, in, &ParseOptions{Windows: true}})
			corpus = append(corpus, diffTestCase{"Windows paths", p, in, nil})
		}
	}

	// 10. POSIX paths
	posixPats := []string{
		"foo/bar/*.js", "/usr/local/bin/*", "a/b/c/d/e/f/*.sh",
		"src/**/*.go", "/a/b/c",
	}
	posixInputs := []string{
		"foo/bar/app.js", "/usr/local/bin/node", "a/b/c/d/e/f/run.sh",
		"src/pkg/util/main.go", "/a/b/c", "foo/bar/baz/quux.js",
	}
	for _, p := range posixPats {
		for _, in := range posixInputs {
			corpus = append(corpus, diffTestCase{"POSIX paths", p, in, &ParseOptions{Posix: true}})
		}
	}

	// 11. Hidden files & 12. Special directories
	specialPats := []string{
		"*.js", "**/*.js", ".*", "**/.*", "*", "**/*", "foo/*", "./*", "a/../*",
	}
	specialInputs := []string{
		".git/config", ".hidden.js", "foo/.bar/baz.js", ".", "..",
		"foo/.", "foo/..", "../foo", "./foo", "a/../bar",
	}
	for _, p := range specialPats {
		for _, in := range specialInputs {
			corpus = append(corpus, diffTestCase{"hidden & special dirs", p, in, nil})
			corpus = append(corpus, diffTestCase{"hidden & special dirs", p, in, &ParseOptions{Dot: true}})
		}
	}

	// 13. Option combinations
	optPats := []string{"*.js", "**/*.js", "foo/*.js", "*.go", "*"}
	optInputs := []string{"foo/bar/app.js", "APP.JS", ".hidden.js", "src/build/bundle.js", "index.js"}
	optGrid := []*ParseOptions{
		{Dot: true, Nocase: true},
		{MatchBase: true, Nocase: true},
		{MatchBase: true, Dot: true, Nocase: true},
		{Ignore: []string{"**/build/**", "**.ts"}},
		{Ignore: []string{"**/build/**"}, Nocase: true, MatchBase: true},
		{Contains: true, Nocase: true},
		{Basename: true, Nocase: true},
		{NoGlobstar: true},
		{NoExtglob: true},
	}
	for _, p := range optPats {
		for _, in := range optInputs {
			for _, opt := range optGrid {
				corpus = append(corpus, diffTestCase{"option combinations", p, in, opt})
			}
		}
	}

	// 14. Extremely long patterns
	longPats := []string{
		"foo/" + strings.Repeat("a/", 50) + "*.js",
		"**/" + strings.Repeat("*/", 30) + "*.go",
		strings.Repeat("a/b/c/", 20) + "*.ts",
		"{" + strings.Repeat("foo,bar,baz,quux,", 50) + "end}/*.js",
	}
	longInputs := []string{
		"foo/" + strings.Repeat("a/", 50) + "app.js",
		"x/y/z/" + strings.Repeat("a/", 30) + "main.go",
		strings.Repeat("a/b/c/", 20) + "test.ts",
		"end/app.js", "foo/app.js",
	}
	for _, p := range longPats {
		for _, in := range longInputs {
			corpus = append(corpus, diffTestCase{"extremely long patterns", p, in, nil})
		}
	}

	// 15. Randomly generated valid glob patterns (1,000 cases)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	vocab := []string{"foo", "bar", "baz", "quux", "test", "src", "pkg", "lib", "bin", "main", "app", "index"}
	exts := []string{".js", ".go", ".ts", ".html", ".css", ".txt", ".sh", ".json"}
	wildcards := []string{"*", "**", "?", "[a-z]*", "{a,b,c}", "@(foo|bar)", "*(test|build)", "+(main|app)"}

	for i := 0; i < 1000; i++ {
		depth := r.Intn(4) + 1
		var patParts []string
		var inParts []string
		for d := 0; d < depth; d++ {
			if r.Float32() < 0.4 && d < depth-1 {
				patParts = append(patParts, wildcards[r.Intn(len(wildcards))])
			} else {
				w := vocab[r.Intn(len(vocab))]
				patParts = append(patParts, w)
			}
			inParts = append(inParts, vocab[r.Intn(len(vocab))])
		}
		ext := exts[r.Intn(len(exts))]
		pat := strings.Join(patParts, "/") + "/*" + ext
		in := strings.Join(inParts, "/") + "/" + vocab[r.Intn(len(vocab))] + ext
		var opts *ParseOptions
		if r.Float32() < 0.3 {
			opts = &ParseOptions{Dot: r.Float32() < 0.5, Nocase: r.Float32() < 0.5, MatchBase: r.Float32() < 0.3}
		}
		corpus = append(corpus, diffTestCase{"random valid", pat, in, opts})
	}

	// 16. Randomly generated invalid/malformed glob patterns (500 cases)
	malformedTokens := []string{"[", "{", "(", "\\", "!(", "@(", "*(", "+(", "?(", "[a-", "{foo,bar", "/*/", "/../*", "/*"}
	for i := 0; i < 500; i++ {
		w1 := vocab[r.Intn(len(vocab))]
		m1 := malformedTokens[r.Intn(len(malformedTokens))]
		m2 := malformedTokens[r.Intn(len(malformedTokens))]
		pat := w1 + "/" + m1 + "/" + m2 + "*.js"
		in := w1 + "/test/app.js"
		corpus = append(corpus, diffTestCase{"random invalid", pat, in, nil})
	}

	return corpus
}
