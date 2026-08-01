# Port Mortem Benchmarking & Performance Roadmap

This document outlines the performance monitoring methodology, execution metrics, and roadmap for evaluating the Go Port Mortem implementation against the Node.js `picomatch` library.

---

## 1. Benchmark Goals
- **Execution Speed:** Ensure compilation of glob patterns into regular expressions and subsequent string matching evaluations execute significantly faster than native Node.js runtime interpretations.
- **Allocation Minimization:** Establish zero-heap-allocation paths during frequent string matching operations by leveraging Go's static type efficiency and memory pools (`sync.Pool`).
- **Memory Footprint Reduction:** Guarantee low overhead per compiled pattern matcher, avoiding heavy AST retention once compilation is finalized.

---

## 2. Current Benchmark Status
- **Status:** **BENCHMARKS POSTPONED (Sprint 6)**
- **Empirical Numbers:** No performance metrics or benchmark timing numbers currently exist in this repository. In accordance with strict architectural reporting guidelines, zero speculative benchmark figures are invented or published prior to actual compilation and matcher implementation.

---

## 3. Why Benchmarks Are Postponed
Micro-benchmarking intermediate syntactic parsing stages (such as standalone character scanning or partial bracket foundation traversal in Sprint 6) generates distorted and non-representative performance data. In real-world operation, parser traversal occurs within a tightly bound pipeline terminating in regex compilation and string matcher invocation. Evaluating execution runtime before full grammar synthesis exists would provide misleading baseline comparisons against Node.js `picomatch`.

---

## 4. Required Milestones Before Meaningful Benchmarks
Formal benchmarking execution will commence only after the completion of the following prerequisites:
1. **Sprint 7 (Brace Expansion & Extglob Parsing):** Comprehensive handling of nested pattern branching and complex grammar traversal.
2. **Sprint 8 (Wildcard Semantics & POSIX Tables):** Full AST synthesis for stars, globstars, question marks, and character ranges.
3. **Sprint 9 (Regex Compiler Core):** Translation of AST structures and unclosed delimiter strings into valid compiled regular expression sources (`compiler.go`).
4. **Sprint 10 (Public Matcher API):** Integration of user-facing evaluation functions (`picomatch.Match()`, compiled `Matcher` structs, and caching wrappers).

---

## 5. Planned Benchmark Methodology
When compiler and matcher foundations are complete, benchmarks will be executed using Go's standard `testing.B` harness in parallel with structured Node.js runtime timers across equivalent hardware environments:
- **Harness Setup:** Table-driven benchmarking suites (`BenchmarkCompile_Simple`, `BenchmarkCompile_ExtGlob`, `BenchmarkMatch_DeepHierarchy`) running across standardized fixture datasets (`testdata/fixtures.json`).
- **Execution Environment:** Documentation of CPU topology, memory architecture, operating system kernel, Go compiler toolchain version (`go version`), and Node.js execution runtime version.

---

## 6. Planned Metrics
Benchmark output tables will track three vital quantitative indicators:
- `ns/op` (Nanoseconds per operation): Overall CPU clock elapsed during pattern compilation or string evaluation.
- `B/op` (Bytes allocated per operation): Total dynamic heap memory allocated during invocation.
- `allocs/op` (Allocations per operation): Distinct heap object allocations triggered by pattern processing (targeted at `0` for compiled matcher executions).

---

## 7. Cross-Library Comparison Plan
To establish empirical competitiveness, benchmarks will execute comparative runtime tests against three key categories:
1. **Upstream Node.js Picomatch:** Direct differential performance comparisons against running JavaScript execution environments over large directory scanning simulations.
2. **Go Standard Library:** Baseline speed verification against native `path/filepath.Match`.
3. **Popular Go Glob Libraries:** Competitive benchmarking against recognized third-party Go packages, including `github.com/bmatcuk/doublestar` and `github.com/gobwas/glob`.

---

## 8. Future Benchmark Roadmap
- **Phase 1 (Post-Sprint 9):** Establish baseline compilation benchmarks across basic wildcards, character classes, and negated patterns.
- **Phase 2 (Post-Sprint 10):** Execute concurrent throughput matching evaluations using Go goroutine stress tests (`RunParallel`).
- **Phase 3 (Optimization Sprints):** Perform CPU and memory profiling (`go test -cpuprofile` / `-memprofile`) to identify hot loops, eliminate redundant string conversions, and implement pattern compilation caching pools.
