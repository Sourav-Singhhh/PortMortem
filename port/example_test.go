package picomatch_test

import (
	"fmt"
	"log"

	picomatch "github.com/Sourav-Singhhh/PortMortem/port"
)

// ExampleMatch demonstrates casual one-off pattern evaluation against a single filesystem path.
func ExampleMatch() {
	matched, err := picomatch.Match("*.{go,js}", "main.go", nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(matched)
	// Output: true
}

// ExampleCompile demonstrates precompiling a recursive glob pattern for zero-allocation reuse across multiple target paths.
func ExampleCompile() {
	matcher, err := picomatch.Compile("foo/**/*.go", nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(matcher.Match("foo/bar/baz/utils.go"))
	fmt.Println(matcher.Match("foo/app.js"))
	// Output:
	// true
	// false
}

// ExampleMatcher_Match demonstrates evaluating negated extglob patterns using linear-time set-difference decomposition.
func ExampleMatcher_Match() {
	matcher, err := picomatch.Compile("!(test)*.js", nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(matcher.Match("app.js"))
	fmt.Println(matcher.Match("test.js"))
	// Output:
	// true
	// false
}

// Example_customOptions demonstrates configuring custom option toggles such as dotfile matching and case insensitivity.
func Example_customOptions() {
	opts := picomatch.NewParseOptions()
	opts.Dot = true
	opts.Nocase = true

	matched, err := picomatch.Match("foo/*", "FOO/.gitignore", opts)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(matched)
	// Output: true
}
