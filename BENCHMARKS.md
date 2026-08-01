# Port Mortem Benchmarking & Performance Roadmap

This document outlines the performance monitoring methodology, execution metrics, and roadmap for evaluating the Go Port Mortem implementation against the Node.js `picomatch` library.

---

## 1. Benchmark Goals
- **Execution Speed:** Ensure compilation of glob patterns into regular expressions and subsequent string matching evaluations execute significantly faster than native Node.js runtime interpretations.
- **Allocation Minimization:** Establish zero-heap-allocation paths during frequent string matching operations by leveraging Go's static type efficiency and memory pools (`sync.Pool`).
- **Memory Footprint Reduction:** Guarantee low overhead per compiled pattern matcher, avoiding heavy AST retention once compilation is finalized.

---

## 2. Current Benchmark Status
- **Status:** **PREREQUISITES SATISFIED — READY FOR EMPIRICAL EVALUATION (Post-Sprint 11)**
- **Empirical Numbers:** With runtime matcher integration completed in Sprint 11 ([port/matcher.go](file:///C:/Users/rajpu/Desktop/PortMortem/port/matcher.go)), the foundational execution engines required for valid runtime timing are fully operational. Formal quantitative benchmark suites are scheduled for implementation and execution in Planned Phase C. In accordance with strict engineering standards, zero speculative timing figures are published prior to actual test harness execution.

---

## 3. Why Matcher Integration Enables Meaningful Benchmarking
Prior to Sprint 11, evaluating micro-benchmarks on intermediate syntax scanning or standalone AST string compilation produced distorted, non-representative performance data. With the completion of runtime matcher integration—which wraps single-pass parser pattern string accumulation (`state.Output`), RE2 regex automata compilation, structural pattern segment caching (`patSegments`), and algorithmic set-difference pattern decomposition into cohesive evaluation primitives (`Compile()` and `Match()`)—the project now possesses an authoritative end-to-end matching pipeline. Benchmarking execution across these finalized primitives guarantees representative real-world performance evaluation against native Node.js `picomatch`.

---

## 4. Satisfied Prerequisites & Readiness Milestone Matrix
Every structural and computational prerequisite required for valid benchmarking has been successfully accomplished:
1. **Brace & Extglob Syntax Engine (Sprints 7–8):** Complete grammar traversal across nested branch alternations, option toggles, and ReDoS analysis (`AnalyzeRepeatedExtglob`). *(Satisfied)*
2. **Wildcard & POSIX Translation (Sprints 9–10):** Full evaluation of stars, directory globstars, question marks, and static POSIX ASCII mapping tables. *(Satisfied)*
3. **Regex Pattern Synthesis (Sprint 10):** Direct single-pass synthesis of valid JavaScript regular expression output pattern strings (`state.Output`). *(Satisfied)*
4. **Runtime Matcher Integration & Caching API (Sprint 11):** Implementation of exported user-facing evaluation wrappers (`Compile()`, `Match()`), thread-safe pattern compilation caches (`sync.RWMutex`), and zero-allocation runtime path segment checks (`validateDotAndSpecialDirs`). *(Satisfied)*

---

## 5. Benchmark Methodology
Formal empirical benchmark evaluation will execute via Go's standard `testing.B` execution framework in parallel with structured Node.js V8 runtime timer scripts across identical hardware and kernel configurations:
- **Test Harness Setup:** Automated table-driven benchmarking suites (`BenchmarkCompile_Simple`, `BenchmarkCompile_ExtGlob`, `BenchmarkMatch_DeepHierarchy`, `BenchmarkMatch_NegatedExtglob`) executing across standardized filesystem target datasets (`testdata/fixtures.json`).
- **Concurrent Execution:** Stress evaluation using multi-threaded goroutine test harnesses (`testing.B.RunParallel`) to evaluate lock-free reader scaling across global pattern caches.
- **Execution Environment Documentation:** Complete architectural reporting of CPU frequency/topology, RAM architecture, operating system kernel, Go compiler toolchain version (`go version`), and Node.js execution runtime build numbers.

---

## 6. Target Benchmark Metrics
Benchmark reporting tables will record four quantitative runtime execution indicators:
- **`ns/op` (Nanoseconds per operation):** Total CPU wall-clock elapsed during pattern compilation or target path matching evaluation.
- **`B/op` (Bytes allocated per operation):** Total dynamic heap memory consumption allocated per function invocation.
- **`allocs/op` (Allocations per operation):** Distinct heap object allocations triggered during execution (targeted at precisely `0` for repeated evaluations of pre-compiled `Matcher` structs and fast-path exact equals evaluations).
- **`MB/s` (Throughput rate):** Target evaluation throughput speed when scanning extensive simulated directory trees.

---

## 7. Cross-Library Comparison Plan
To establish empirical runtime competitiveness, benchmarking evaluation will compare Port Mortem across three primary reference targets:
1. **Upstream Node.js Picomatch:** Direct cross-language performance comparisons against native JavaScript runtime execution over simulated filesystem directory traversals and ReDoS evaluation strings.
2. **Go Standard Library:** Baseline speed and memory allocation verification against native standard library `path/filepath.Match` and `io/fs.Glob`.
3. **Leading Go Glob Libraries:** Competitive benchmarking against established open-source Go matching repositories, including `github.com/bmatcuk/doublestar` and `github.com/gobwas/glob`.

---

## 8. Remaining Blockers & Performance Roadmap
With syntax and matcher prerequisites satisfied, the benchmark engineering roadmap is sequenced across three empirical execution stages:
- **Stage 1 (Planned Phase C):** Construct the table-driven `testing.B` evaluation harnesses across compilation and matching primitives, populating empirical performance tables in this document.
- **Stage 2 (Planned Phase D):** Conduct concurrent profiling (`go test -cpuprofile` and `-memprofile`) to analyze evaluation hot loops, lock contention, and cache eviction overhead under high-density multi-goroutine workloads.
- **Stage 3 (Pre-v1.0 Optimization):** Implement advanced structural memory object pools (`sync.Pool`) to eliminate transient token slices and string conversions during complex pattern evaluations, assuring optimal memory behavior for v1.0 production release.
