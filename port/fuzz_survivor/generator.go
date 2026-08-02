package main

import (
	"fmt"
	"math/rand"
	"strings"
)

// Candidate represents a single fuzzing input triplet.
type Candidate struct {
	Pattern string
	Input   string
	Options map[string]interface{}
}

// Generator generates fuzzing candidates using grammar templates, mutations, and regression seeds.
type Generator struct {
	rnd   *rand.Rand
	seeds []Candidate
}

// NewGenerator initializes a candidate generator with a pseudorandom seed.
func NewGenerator(seed int64) *Generator {
	g := &Generator{
		rnd: rand.New(rand.NewSource(seed)),
	}
	g.initSeeds()
	return g
}

func (g *Generator) initSeeds() {
	g.seeds = []Candidate{
		{Pattern: "*.js", Input: "index.js"},
		{Pattern: "src/**/*.go", Input: "src/cmd/main.go"},
		{Pattern: "foo/{bar,baz}.js", Input: "foo/bar.js"},
		{Pattern: "[a-z]*", Input: "alpha.txt"},
		{Pattern: "@(foo|bar)/*.ts", Input: "foo/app.ts"},
		{Pattern: "!(test).go", Input: "main.go"},
		{Pattern: "file?.txt", Input: "file1.txt"},
		{Pattern: "**/*.json", Input: "package.json"},
		{Pattern: "[а-я]*.txt", Input: "привет.txt"},
		{Pattern: ".env*", Input: ".env.local"},
		{Pattern: "C:\\\\foo\\\\bar\\\\*.js", Input: "C:\\foo\\bar\\index.js"},
		{Pattern: "🚀/*.ts", Input: "🚀/app.ts"},
		{Pattern: "[[:alnum:]_]*", Input: "var_123"},
		{Pattern: "src/{a..z}/{1..100}/*.ts", Input: "src/b/42/app.ts"},
		{Pattern: "[:[:]", Input: "[:[:]"}, // Saved Sprint 17 fuzz regression input
	}
}

// Next Candidate generates the next fuzzing candidate using hybrid strategy.
func (g *Generator) Next() Candidate {
	r := g.rnd.Intn(100)
	if r < 30 && len(g.seeds) > 0 {
		// Replay/mutate seed candidate
		seed := g.seeds[g.rnd.Intn(len(g.seeds))]
		return g.mutateCandidate(seed)
	} else if r < 70 {
		// Grammar-aware generation
		return g.generateGrammarCandidate()
	} else {
		// Mutation-based random generation
		return g.generateMutatedCandidate()
	}
}

func (g *Generator) generateGrammarCandidate() Candidate {
	prefixes := []string{"", "src/", "build/", "a/b/c/", "./", "!"}
	literals := []string{"foo", "bar", "test", "app", "file", "index", "🚀", "привет"}
	extensions := []string{".js", ".ts", ".go", ".json", ".txt", ".md"}

	grammarTypes := []int{0, 1, 2, 3, 4, 5, 6}
	gt := grammarTypes[g.rnd.Intn(len(grammarTypes))]

	var pattern string
	switch gt {
	case 0: // Simple wildcard
		pattern = fmt.Sprintf("%s*%s", prefixes[g.rnd.Intn(len(prefixes))], extensions[g.rnd.Intn(len(extensions))])
	case 1: // Globstar
		pattern = fmt.Sprintf("%s**/*%s", prefixes[g.rnd.Intn(len(prefixes))], extensions[g.rnd.Intn(len(extensions))])
	case 2: // Brace expansion
		l1 := literals[g.rnd.Intn(len(literals))]
		l2 := literals[g.rnd.Intn(len(literals))]
		pattern = fmt.Sprintf("%s{%s,%s}%s", prefixes[g.rnd.Intn(len(prefixes))], l1, l2, extensions[g.rnd.Intn(len(extensions))])
	case 3: // Extglob
		l1 := literals[g.rnd.Intn(len(literals))]
		l2 := literals[g.rnd.Intn(len(literals))]
		extOps := []string{"@", "+", "*", "?", "!"}
		op := extOps[g.rnd.Intn(len(extOps))]
		pattern = fmt.Sprintf("%s%s(%s|%s)/*%s", prefixes[g.rnd.Intn(len(prefixes))], op, l1, l2, extensions[g.rnd.Intn(len(extensions))])
	case 4: // POSIX class
		classes := []string{"alnum", "alpha", "digit", "lower", "upper"}
		cls := classes[g.rnd.Intn(len(classes))]
		pattern = fmt.Sprintf("%s[[:%s:]]*%s", prefixes[g.rnd.Intn(len(prefixes))], cls, extensions[g.rnd.Intn(len(extensions))])
	case 5: // Range expansion
		pattern = fmt.Sprintf("%s{1..50}/*%s", prefixes[g.rnd.Intn(len(prefixes))], extensions[g.rnd.Intn(len(extensions))])
	case 6: // Windows path
		pattern = fmt.Sprintf("C:\\\\%s\\\\*%s", literals[g.rnd.Intn(len(literals))], extensions[g.rnd.Intn(len(extensions))])
	}

	input := g.generateMatchingOrNonMatchingInput(pattern)
	return Candidate{
		Pattern: pattern,
		Input:   input,
		Options: nil,
	}
}

func (g *Generator) generateMutatedCandidate() Candidate {
	chars := "abcdefghijklmnopqrstuvwxyz0123456789/*?[]{}()!._-\\"
	patLen := g.rnd.Intn(20) + 1
	var sb strings.Builder
	for i := 0; i < patLen; i++ {
		sb.WriteByte(chars[g.rnd.Intn(len(chars))])
	}

	inputLen := g.rnd.Intn(15) + 1
	var ib strings.Builder
	for i := 0; i < inputLen; i++ {
		ib.WriteByte(chars[g.rnd.Intn(len(chars))])
	}

	return Candidate{
		Pattern: sb.String(),
		Input:   ib.String(),
		Options: nil,
	}
}

func (g *Generator) mutateCandidate(c Candidate) Candidate {
	mut := c
	if g.rnd.Intn(2) == 0 {
		mut.Pattern += string(byte('a' + g.rnd.Intn(26)))
	} else {
		mut.Input += string(byte('a' + g.rnd.Intn(26)))
	}
	return mut
}

func (g *Generator) generateMatchingOrNonMatchingInput(pattern string) string {
	inputs := []string{
		"index.js", "src/app.ts", "foo/bar.js", "main.go", "file1.txt",
		"alpha.txt", "package.json", "привет.txt", ".env.local", "C:\\foo\\bar\\index.js",
		"🚀/app.ts", "var_123", "src/b/42/app.ts", ".", "..", "./foo", "../bar",
	}
	return inputs[g.rnd.Intn(len(inputs))]
}
