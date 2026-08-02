# Port Mortem — Architecture Reference

**Repository**: `github.com/Sourav-Singhhh/PortMortem`
**Release**: v1.1.0 (post-Sprint 20 verified)
**Source of Content**: Consolidated from `README.md`, `DECISIONS.md`, `PORTING_STRATEGY.md`, and `RELEASE_PLAN.md` — no invented architecture.

---

## 1. Overview

Port Mortem is a high-performance, memory-safe, behaviorally equivalent Go port of the Node.js `picomatch` glob matching library. The implementation strictly targets **bug-for-bug behavioral parity** with the original JavaScript library across all option toggles, boundary edge cases, unclosed delimiter recovery rules, and syntax parsing decisions.

The library exposes a minimal public API:
- `Compile(pattern string, opts *ParseOptions) (*Matcher, error)` — parse and cache a glob pattern
- `Match(pattern, input string, opts *ParseOptions) (bool, error)` — single-call evaluation
- `(*Matcher).Match(input string) bool` — hot-path evaluation against a pre-compiled pattern

---

## 2. High-Level Architecture

```
 ┌───────────────────────────────────────────────────────────────────────┐
 │                          Public API Layer                             │
 │             Compile()  ·  Match()  ·  Matcher.Match()               │
 └─────────────────────────────┬─────────────────────────────────────────┘
                               │
 ┌─────────────────────────────▼─────────────────────────────────────────┐
 │                    Runtime Matcher  (matcher.go)                      │
 │  Two-tier zero-allocation cache  ·  Thread-safe sync.RWMutex         │
 │  Literal fastpath  ·  patSegments  ·  RE2 set-difference resolution  │
 └───────────────┬───────────────────────────────┬───────────────────────┘
                 │                               │
 ┌───────────────▼──────────────┐ ┌─────────────▼────────────────────────┐
 │     Scanner  (scan.go)       │ │  Parser  (parse.go + parse_*.go)     │
 │  Single-pass prefix analysis │ │  Single-pass interleaved evaluation  │
 │  Base directory extraction   │ │  Token tree builder (ParseToken)     │
 │  Grammar flag population     │ │  RE2 synthesis  ·  Stack management  │
 └──────────────────────────────┘ └──────────────────────────────────────┘
```

---

## 3. Module Structure

The production implementation lives entirely within the `port/` subdirectory, which is an independent Go module (`github.com/Sourav-Singhhh/PortMortem/port`).

```
port/
├── go.mod                    # Independent Go module
├── doc.go                    # Package GoDoc commentary
├── scan.go                   # Lexical scanner
├── parse.go                  # Main parse loop (character dispatch)
├── parse_literals.go         # Literal character handling
├── parse_brackets.go         # Bracket expression parser [a-z]
├── parse_braces.go           # Brace expansion parser {a,b,c} / {1..5}
├── parse_extglobs.go         # Extended glob parser ?(x) *(x) +(x) @(x) !(x)
├── parse_wildcards.go        # Wildcard and dot handler * ** .
├── parse_regex.go            # GlobChars constants, platform-specific RE2 strings
├── parse_helpers.go          # Stack data structures, token constructors
├── matcher.go                # Compile(), Match(), Matcher struct, cache
├── matcher_diff_test.go      # 3,226 live differential test scenarios
├── matcher_bench_test.go     # 16-target performance benchmark suite
├── fuzz_test.go              # Native Go fuzz targets (testing.F)
├── example_test.go           # GoDoc runnable examples
└── fuzz_survivor/
    ├── main.go               # Fuzz Survivor entry point
    ├── classifier.go         # Divergence taxonomy classifier
    ├── survivor_test.go      # go test integration harnesses
    └── logs/
        ├── survivor_report.md    # Latest 300s run report (generated)
        └── survivor_log.jsonl    # JSONL divergence log (generated)
```

**Separated infrastructure** (not part of the importable package):
```
tests/adapter/        # Node.js IPC bridge daemon for differential testing
original-picomatch/   # Reference Node.js picomatch source (unmodified)
fuzz/README.md        # Fuzz testing documentation
docs/verification/    # Sprint audit certificates and verification reports
```

---

## 4. Component Responsibilities

### 4.1 Scanner (`scan.go`)
Performs a **single-pass fast scan** of a raw glob pattern string to:
- Extract the static base directory prefix (everything before the first glob character)
- Evaluate prefix grammar: leading `!` (negation), `./` (relative path stripping)
- Populate grammar flags: `isBrace`, `isBracket`, `isExtglob`, `isGlobstar`, `isGlob`
- Determine `slashes` count and `maxDepth` for directory segment validation

The scanner does **not** modify patterns; it only classifies them for the parser and matcher.

### 4.2 Parser (`parse.go` + `parse_*.go`)
Implements a **single-pass interleaved character evaluation loop** that simultaneously performs:
- Lexical tokenisation into `ParseToken` linked-tree structures
- Syntactic transformation into RE2 regex string fragments
- Stack-based delimiter balancing: `BraceStack`, `ExtglobStack`, `ParserStack`
- POSIX character class translation (`[:alnum:]` → `[a-zA-Z0-9]`)
- Numerical and alphabetical brace interval expansion (`{1..5}`, `{a..z}`)
- Extglob synthesis: `?(x)` → `(?:x)?`, `*(x)` → `(?:x)*`, `!(x)` → set-difference
- ReDoS exponential backtracking detection and mitigation (`AnalyzeRepeatedExtglob`)
- EOF delimiter reconciliation for unclosed brackets and braces (`EscapeLast`)

Parser dispatch is character-driven: each character type routes to a dedicated handler (`HandleStar`, `HandleDot`, `HandleBracket`, `HandleBrace`, `HandleExtglob`, `HandleLiteral`, `HandleSlash`).

### 4.3 Runtime Matcher (`matcher.go`)
Provides the public evaluation API and manages pattern lifecycle:

**Two-Tier Zero-Allocation Compilation Cache:**
- Tier 1: `cacheNilOpts map[string]*Matcher` — string-keyed fastpath for nil-options patterns
- Tier 2: `cache map[cacheKeyStruct]*Matcher` — comparable value-struct keyed cache for custom options
- Both tiers protected by `sync.RWMutex` for goroutine safety
- Result: `0 B/op, 0 allocs/op` on all cache hits

**RE2 Lookaround Resolution:**
Go's RE2 engine prohibits arbitrary lookahead/lookbehind assertions (required for extglob negation `!(x)`). The matcher resolves this without CGO or PCRE via:
- `toRE2()`: strips incompatible lookaround sequences from picomatch-generated regex
- Set-difference decomposition: `A \ B ≡ A ∩ ¬B` — tests the positive case then the exclusion case separately, combining results with Go boolean logic

**Security Hardening:**
- `validateDotAndSpecialDirs()`: prevents wildcards from matching navigational directories (`.`, `..`) unless explicitly declared in the pattern
- Path separator normalisation: converts backslashes to forward slashes under `Windows: true` or `Posix: true`

---

## 5. Data Flow

```
Input: pattern string + input string + *ParseOptions
         │
         ▼
   [Cache Lookup] ─── HIT ──────────────────────────────────┐
         │                                                    │
        MISS                                                  │
         │                                                    │
         ▼                                                    │
   [scan.go: Scan()] → ScanResult (base, flags, slashes)    │
         │                                                    │
         ▼                                                    │
   [parse.go: Parse()] → ParseState (tokens, output string)  │
         │                                                    │
         ▼                                                    │
   [matcher.go: toRE2()] → RE2-compatible regex string       │
         │                                                    │
         ▼                                                    │
   [regexp.Compile()] → *regexp.Regexp                       │
         │                                                    │
         ▼                                                    │
   [Matcher struct stored in cache] ───────────────────────►─┘
         │
         ▼
   [Matcher.Match(input)]
     ├── Literal fastpath (no regex needed)
     ├── validateDotAndSpecialDirs() security check
     ├── Path separator normalisation
     ├── regexp.MatchString() or Regexp.MatchString()
     └── Set-difference for negated extglobs
         │
         ▼
   Output: bool (matched)
```

---

## 6. Differential Testing & Verification Pipeline

```
Node.js picomatch v3.0.1 (reference)
        │  stdin/stdout JSON IPC
        ▼
tests/adapter/main.go (Go IPC bridge daemon)
        │
        ├── port/matcher_diff_test.go (3,226 structured scenarios)
        │       └── go test -count=1 -v ./...
        │
        └── port/fuzz_survivor/main.go (adversarial continuous testing)
                ├── Random pattern + input generation
                ├── Go evaluation (port/matcher.go)
                ├── Node.js evaluation (IPC)
                ├── Classifier (classifier.go)
                └── JSONL log + Markdown report
```

### Differential Test Categories (3,226 scenarios across 14 categories)
Wildcards, globstars, extglobs, brace expansions, POSIX classes, bracket expressions, negation patterns, dot-file semantics, path traversal, Windows paths, UNC paths, Unicode scripts, emoji filenames, malformed patterns.

### Fuzz Survivor Statistics (Sprint 19 — 300s certified run)
- Total inputs generated: **3,208,608**
- Exact Go/Node.js agreement: **2,925,773**
- Documented RE2 adaptations: **55,038**
- Documented security adaptations: **1,324**
- Unexpected divergences: **0**
- Go panics: **0**
- Throughput: **10,695 comparisons/sec**

---

## 7. Continuous Integration (`.github/workflows/ci.yml`)

Multi-platform GitHub Actions matrix across `ubuntu-latest`, `windows-latest`, `macos-latest`:

1. **Source Formatting** — `gofmt -l .` (Linux/macOS only)
2. **Static Analysis** — `go vet ./...`
3. **Build** — `go build -v ./...`
4. **Test Suite** — `go test -v -count=1 ./...`
5. **Fuzz Smoke** — `go test -fuzz=FuzzCompile -fuzztime=5s .`

**Go Toolchain**: `1.22.x` (CI minimum); developed and tested locally against `go1.26.5`.

---

## 8. RE2 Adaptation Strategy

Go's RE2 engine guarantees **linear-time O(N) matching** by prohibiting backtracking constructs, including lookahead (`(?=...)`) and lookbehind (`(?<!...)`). picomatch generates these constructs for several features. Port Mortem's adaptation strategy:

| Feature | picomatch Output | Port Mortem Adaptation |
| :--- | :--- | :--- |
| Dot-file protection | `(?!\.)` negative lookahead | `validateDotAndSpecialDirs()` pre-check |
| Extglob negation `!(x)` | `(?!(?:x))` negative lookahead | Set-difference: match A, subtract @(x) matches |
| Globstar dot protection | `(?!(?:^|\/)\.{1,2}(?:\/|$))` | Programmatic path segment validation |
| Windows separator classes | `[\\\/]` in lookaround | Extended `toRE2()` stripping for bracket classes |

All RE2 adaptations are **verified** to produce identical match/no-match results against Node.js for all standard inputs. Divergences exist only for adversarially malformed patterns (documented in `DECISIONS.md`).

---

## 9. Performance Profile

All values [MEASURED] on 12th Gen Intel(R) Core(TM) i5-12450H, Windows/amd64, go1.26.5:

| Operation | ns/op | B/op | allocs/op |
| :--- | :---: | :---: | :---: |
| Precompiled matching (`Matcher.Match`) | ~203–289 | 0 | 0 |
| Cached compilation (`Compile` cache hit) | ~116–151 | 0 | 0 |
| One-off matching (`Match`) | ~329–370 | 0 | 0 |
| Concurrent matching (12 goroutines) | ~138–148 | 0 | 0 |
| Batch throughput | — | 0 | 0 |
| Deep globstar (`foo/**/bar/**/baz/**/*.js`) | ~945–1,133 | 0 | 0 |
| Cold compile (uncached, full AST build) | ~3,600–5,000 | ~3,770 | ~53 |

**Batch throughput**: 375,000–567,000 matches/sec [MEASURED across Sprint 13–20 benchmark rounds].

---

*Architecture documentation consolidated from existing repository sources. Last synchronized: Sprint 20 (2026-08-02).*
