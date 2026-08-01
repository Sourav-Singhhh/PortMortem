package picomatch

import (
	"path/filepath"
	"regexp"
	"testing"
)

func BenchmarkCompile_Uncached(b *testing.B) {
	b.ReportAllocs()
	opts := &ParseOptions{Dot: true}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Invoke Parse directly to bypass compiled structural caching and measure pure AST parsing + regex synthesis
		_, _ = Parse("foo/**/bar/*.js", opts)
	}
}

func BenchmarkCompile_Cached(b *testing.B) {
	b.ReportAllocs()
	opts := &ParseOptions{Dot: true}
	// Warm up cache
	_, _ = Compile("*.js", opts)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 100% cache hit ratio
		_, _ = Compile("*.js", opts)
	}
}

func BenchmarkMatch_OneOff(b *testing.B) {
	b.ReportAllocs()
	opts := &ParseOptions{Dot: true}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Match("*.js", "index.js", opts)
	}
}

func BenchmarkMatch_Precompiled(b *testing.B) {
	b.ReportAllocs()
	m, _ := Compile("*.js", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.Match("index.js")
	}
}

func BenchmarkLargeDirectoryPatterns(b *testing.B) {
	b.ReportAllocs()
	pat := "foo/bar/baz/quux/test/sub/app/*.js"
	input := "foo/bar/baz/quux/test/sub/app/controller.js"
	m, _ := Compile(pat, nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.Match(input)
	}
}

func BenchmarkDeepGlobstars(b *testing.B) {
	b.ReportAllocs()
	pat := "foo/**/bar/**/baz/**/*.js"
	input := "foo/a/b/c/bar/x/y/z/baz/1/2/3/bundle.js"
	m, _ := Compile(pat, nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.Match(input)
	}
}

func BenchmarkNestedExtglobs(b *testing.B) {
	b.ReportAllocs()
	pat := "@(foo|@(bar|@(baz|quux)))/*.js"
	input := "quux/controller.js"
	m, _ := Compile(pat, nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.Match(input)
	}
}

func BenchmarkBraceExpansion(b *testing.B) {
	b.ReportAllocs()
	pat := "src/{build,test,dist}/{1..10}/{x,y,z}/*.js"
	input := "src/test/7/y/module.js"
	m, _ := Compile(pat, nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.Match(input)
	}
}

func BenchmarkPOSIXClasses(b *testing.B) {
	b.ReportAllocs()
	pat := "[[:alpha:]][[:alnum:]]*.[[:alnum:]]*"
	input := "Controller_V2.js"
	m, _ := Compile(pat, nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.Match(input)
	}
}

// Comparative Benchmarks against Standard Library and RE2 Baselines

func BenchmarkComparison_Picomatch_Wildcard(b *testing.B) {
	b.ReportAllocs()
	m, _ := Compile("foo/bar/*.js", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.Match("foo/bar/bundle.js")
	}
}

func BenchmarkComparison_FilepathMatch_Wildcard(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = filepath.Match("foo/bar/*.js", "foo/bar/bundle.js")
	}
}

func BenchmarkComparison_StandardRegexp_Wildcard(b *testing.B) {
	b.ReportAllocs()
	re := regexp.MustCompile(`^foo/bar/[^/]*?\.js$`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = re.MatchString("foo/bar/bundle.js")
	}
}

// Throughput benchmark over batch filesystem operations
func BenchmarkBatchThroughput(b *testing.B) {
	b.ReportAllocs()
	pat := "**/*.{js,ts,go}"
	m, _ := Compile(pat, nil)
	files := []string{
		"src/index.ts",
		"src/components/app.js",
		"pkg/matcher/matcher.go",
		"test/unit/main.go",
		"README.md",
		"build/output.css",
		"vendor/github.com/pkg/errors/errors.go",
		"docs/verification/parser.md",
	}
	b.ResetTimer()
	var matches int
	for i := 0; i < b.N; i++ {
		matches = 0
		for _, f := range files {
			if m.Match(f) {
				matches++
			}
		}
	}
	b.ReportMetric(float64(matches*b.N)/b.Elapsed().Seconds(), "matches/sec")
}

func BenchmarkMixedComplexExpressions(b *testing.B) {
	b.ReportAllocs()
	pat := "src/{build,test,dist}/**/@(foo|bar)/*.[[:alpha:]]*"
	input := "src/test/x/y/bar/module.ts"
	m, _ := Compile(pat, nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.Match(input)
	}
}

func BenchmarkMalformedPatterns(b *testing.B) {
	b.ReportAllocs()
	pat := "foo/[a-/*(bar|{1..5"
	input := "foo/[a-/bundle.js"
	m, _ := Compile(pat, nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if m != nil {
			_ = m.Match(input)
		} else {
			_, _ = Compile(pat, nil)
		}
	}
}

func BenchmarkConcurrentMatching(b *testing.B) {
	b.ReportAllocs()
	pat := "foo/**/@(bar|baz)/*.js"
	input := "foo/a/b/bar/bundle.js"
	// Ensure pattern is cached and compiled
	m, _ := Compile(pat, nil)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Simultaneous evaluation of cached compilation and pre-compiled structural matching
			cached, _ := Compile(pat, nil)
			_ = cached.Match(input)
			_ = m.Match(input)
		}
	})
}
