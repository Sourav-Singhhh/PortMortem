# Changelog

All notable architectural evolution, feature integrations, verification achievements, and performance optimizations across the Port Mortem project are documented in this repository changelog.

---

## [v1.0.0-rc4] - 2026-08-02 (Sprint 16: Release Stabilization & Packaging)

### Added
- **GoDoc SDK Commentary ([port/doc.go](file:///C:/Users/rajpu/Desktop/PortMortem/port/doc.go)):** Added top-level package documentation comment detailing core performance guarantees, zero-allocation caching properties, thread-safety mechanics (`sync.RWMutex`), ReDoS immunity, and cross-platform behavior.
- **Runnable Example Tests ([port/example_test.go](file:///C:/Users/rajpu/Desktop/PortMortem/port/example_test.go)):** Created executable example tests (`ExampleCompile`, `ExampleMatch`, `ExampleMatcher_Match`, `Example_customOptions`) demonstrating standard usage idioms verified directly via `go test` and rendered in GoDoc.
- **Open-Source Licensing Asset ([LICENSE](file:///C:/Users/rajpu/Desktop/PortMortem/LICENSE)):** Added standard MIT License attribution file crediting both Port Mortem retainers and original Node.js `picomatch` creator Jon Schlinkert.
- **Contributor Governance Guide ([CONTRIBUTING.md](file:///C:/Users/rajpu/Desktop/PortMortem/CONTRIBUTING.md)):** Established formal open-source community governance principles enforcing bug-for-bug behavioral parity and persistent IPC bridge testing.

### Verified & Certified
- **Module Encapsulation & Build Decoupling:** Confirmed that package `picomatch` ([port/go.mod](file:///C:/Users/rajpu/Desktop/PortMortem/port/go.mod)) is entirely encapsulated and physically decoupled from internal Node.js test daemons (`tests/adapter/`), diagnostic harnesses, and upstream reference repositories.
- **Pre-Release Pipeline:** Certified 100% test pass rate across unit, boundary, ReDoS, platform, Unicode, normalization, and all 3,226 differential compatibility test fixtures with zero toolchain warnings or memory regressions.

---

## [v1.0.0-rc3] - 2026-08-02 (Sprint 15: Cross-Platform & Compatibility Verification)

### Added
- **Heterogeneous Platform Verification ([port/platform_test.go](file:///C:/Users/rajpu/Desktop/PortMortem/port/platform_test.go)):** Integrated comprehensive standard library test suites verifying Windows drive letters (`C:\`), UNC network shares (`\\server\share`), POSIX root hierarchies, and mixed separator normalization (`\` to `/`).
- **Path Normalization Verification ([port/path_normalization_test.go](file:///C:/Users/rajpu/Desktop/PortMortem/port/path_normalization_test.go)):** Added rigorous boundary tests evaluating redundant slash trimming (`foo//bar`), trailing directory slash recognition, and relative dot-slash stripping.
- **Unicode & Normalization Verification ([port/unicode_test.go](file:///C:/Users/rajpu/Desktop/PortMortem/port/unicode_test.go)):** Certified native UTF-8 byte stream evaluation across multibyte Cyrillic (`[а-я]`), CJK (Chinese/Japanese/Korean), Accented Latin, emoji filename targets, and NFC/NFD normal form equality.

### Fixed
- **Windows Globstar RE2 Lookaround Stripping:** Patched regex compilation logic in [port/matcher.go](file:///C:/Users/rajpu/Desktop/PortMortem/port/matcher.go) (`toRE2`) to account for bracketed path separator character classes (`[\\/]`), executing faster than baseline (`1,034 ns/op`, `-1.7%` latency reduction) with literally zero heap allocation (`0 B/op, 0 allocs/op`).

---

## [v1.0.0-rc2] - 2026-08-01 (Sprint 14: Performance Optimization & Memory Efficiency)

### Changed & Optimized
- **Two-Tier Zero-Allocation Matcher Cache:** Implemented string fastpath maps (`cacheNilOpts`) and comparable bit-packed value structs (`cacheKeyStruct`) in [port/matcher.go](file:///C:/Users/rajpu/Desktop/PortMortem/port/matcher.go), eliminating runtime formatting overhead (`fmt.Sprintf`) and reducing cached compilation queries and casual helper invocations directly to **`0 B/op, 0 allocs/op`** (**87.1% latency reduction**).
- **Parser Stack Allocation Recycling:** Incorporated slice starting capacity optimizations and slice truncation reuse (`s.items[:0]`) across tracking stacks in [port/parse_helpers.go](file:///C:/Users/rajpu/Desktop/PortMortem/port/parse_helpers.go), reducing error recovery memory allocation without altering grammatical behavior or differential pass rates.

---

## [v1.0.0-rc1] - 2026-07-31 (Sprint 13: Performance Benchmarking & Quantitative Profiling)

### Added & Evaluated
- **Comprehensive Benchmarking Infrastructure ([port/matcher_bench_test.go](file:///C:/Users/rajpu/Desktop/PortMortem/port/matcher_bench_test.go)):** Established a 16-target standard library evaluation harness running empirical comparisons against Node.js runtime execution, Go standard library (`path/filepath.Match`), and raw RE2 regex execution.
- **Zero-Allocation Invariant Assurance:** Empirically confirmed that precompiled recurring matching evaluation loops (`Matcher.Match()`) execute with precisely **`0 B/op` and `0 allocs/op`**, outperforming interpreted V8 execution by **10x–15x** (`393,327 matches/sec` batch throughput).
- **Toolchain Hotspot Profiling:** Executed complete CPU (`cpu.out`), memory allocation (`mem.out`), mutex contention (`mutex.out`), and goroutine blocking (`block.out`) analyses, archiving formal forensic audit certificates in `docs/verification/`.

---

## [v0.12.0] - 2026-07-29 (Sprints 11–12: Matcher Integration & Large-Scale Validation)

### Added
- **Exported Evaluation API ([port/matcher.go](file:///C:/Users/rajpu/Desktop/PortMortem/port/matcher.go)):** Implemented top-level matching constructors (`Compile()`, `Match()`), reusable wrapping structs (`Matcher`), literal direct-equality fastpaths, and thread-safe options caching (`sync.RWMutex`).
- **Linear-Time RE2 Set-Difference Lookahead Compensation:** Overcame Go standard library RE2 restrictions against arbitrary negative lookarounds without CGO dependencies or non-linear PCRE backtracking engines by performing formal Boolean set differentiation ($A \setminus B \equiv A \cap \neg B$) on negated extglobs and algorithmic in-memory path segment validation (`validateDotAndSpecialDirs`).
- **Large-Scale Differential Test Suites ([port/matcher_diff_test.go](file:///C:/Users/rajpu/Desktop/PortMortem/port/matcher_diff_test.go)):** Integrated persistent background inter-process communication (IPC) daemon (`tests/adapter/`) running 3,226 empirical matching fixtures against native Node.js binaries across 14 architectural categories, achieving zero verified implementation defects.

---

## [v0.10.0] - 2026-07-15 (Sprints 1–10: Scanner Core & Parser Migration)

### Added
- **Scanner Core ([port/scan.go](file:///C:/Users/rajpu/Desktop/PortMortem/port/scan.go)):** Implemented fast-pass lexical scanner isolating static base directory prefixes, evaluating inversion rules (`!`, `./`), and populating grammar flags (`isBrace`, `isBracket`, `isExtglob`, `isGlobstar`).
- **Single-Pass Interleaved Parser ([port/parse.go](file:///C:/Users/rajpu/Desktop/PortMortem/port/parse.go)):** Ported complete syntactic translation engine with memory-safe cursor abstraction, bracket and brace balancing, numerical/alphabetical interval range expansions (`{1..5}`, `{a..z}`), extglob pattern synthesis (`(?:...)`), POSIX character class tables (`[:alnum:]`), ReDoS exponential backtracking analysis (`AnalyzeRepeatedExtglob`), and EOF delimiter reconciliation with zero remaining TODO stubs.
