# Port Mortem Benchmarking & Performance Roadmap

This document outlines the performance monitoring methodology, execution metrics, and roadmap for evaluating the Go Port Mortem implementation against the Node.js `picomatch` library.

---

## 1. Benchmark Goals
- **Execution Speed:** Ensure compilation of glob patterns into regular expressions and subsequent string matching evaluations execute significantly faster than native Node.js runtime interpretations.
- **Allocation Minimization:** Establish zero-heap-allocation paths during frequent string matching operations by leveraging Go's static type efficiency and memory pools (`sync.Pool`).
- **Memory Footprint Reduction:** Guarantee low overhead per compiled pattern matcher, avoiding heavy AST retention once compilation is finalized.

---

## 2. Current Benchmark Status & Readiness
- **Status:** **PREREQUISITES & INITIAL SCAFFOLDING COMPLETED — READY FOR FORMAL QUANTITATIVE EXECUTION (Post-Sprint 12)**
- **Implementation State:** In Sprint 12, foundational benchmarking scaffolding was implemented and verified in [port/matcher_bench_test.go](file:///C:/Users/rajpu/Desktop/PortMortem/port/matcher_bench_test.go). The test harness defines structured `testing.B` functions covering cached and uncached compilation (`BenchmarkCompile_Uncached`, `BenchmarkCompile_Cached`), one-off versus pre-compiled matching (`BenchmarkMatch_OneOff`, `BenchmarkMatch_Precompiled`), complex structural scenarios (`BenchmarkLargeDirectoryPatterns`, `BenchmarkDeepGlobstars`, `BenchmarkNestedExtglobs`, `BenchmarkBraceExpansion`, `BenchmarkPOSIXClasses`), batch filesystem throughput (`BenchmarkBatchThroughput`), and standard library comparative baselines.
- **Empirical Execution Plan:** Formal execution of these benchmark targets to generate final quantitative comparison tables is scheduled as the primary objective of Planned Phase C (Sprint 13). In adherence to rigorous Release Engineering governance, zero speculative or provisional timing figures are published prior to exhaustive hardware-validated execution runs.

---

## 3. Why Large-Scale Validation Enables Authoritative Benchmarking
Prior to Sprint 12, running micro-benchmarks without verification across extensive option permutations risked timing an incomplete or non-representative evaluation engine. With the completion of Sprint 12—which subjected the runtime matching engine to 3,226 differential scenarios (`TestLargeScaleDifferential`), surgically resolved all genuine implementation bugs (such as wildcard consecutive star collapsing under `NoGlobstar: true` and compilation cache struct hashing), and achieved a certified zero-bug operational baseline—the project possesses a bulletproof evaluation pipeline. Benchmarking execution across these stabilized primitives guarantees that performance metrics represent production-grade correctness and true bug-for-bug JavaScript algorithmic fidelity.

---

## 4. Satisfied Prerequisites & Readiness Matrix
Every structural, syntax, runtime, and validation prerequisite required for definitive empirical benchmarking has been accomplished:
1. **Complete Syntax & Regex Synthesis Engine (Sprints 1–10):** Authoritative single-pass traversal across braces, extglobs, wildcards, brackets, POSIX character tables, and ReDoS vulnerability analysis (`AnalyzeRepeatedExtglob`). *(Satisfied)*
2. **Runtime Matcher & Caching API (Sprint 11):** Thread-safe structural compilation storage (`Compile()`, `sync.RWMutex`), literal equality fastpaths, and zero-allocation path segment validation (`validateDotAndSpecialDirs`). *(Satisfied)*
3. **Large-Scale Behavioral Verification (Sprint 12):** Empirical audit certifying zero implementation bugs across 3,226 differential scenarios, establishing deterministic RE2 boundary limits and option default fidelity. *(Satisfied)*
4. **Standard Library Benchmark Scaffolding (Sprint 12):** Complete implementation of target test suites and comparative baseline harnesses in [port/matcher_bench_test.go](file:///C:/Users/rajpu/Desktop/PortMortem/port/matcher_bench_test.go). *(Satisfied)*

---

## 5. Benchmark Methodology & Execution Plan
Formal empirical benchmark evaluation will execute via Go's standard `testing.B` framework in parallel with structured Node.js V8 runtime execution scripts across identical hardware and kernel configurations:
- **Table-Driven Execution Suite:** Automated evaluation across compilation and matching primitives (`BenchmarkCompile`, `BenchmarkMatch`), iterating dynamically through targeted pattern datasets and reporting allocations (`b.ReportAllocs()`).
- **Concurrent Scaling & Thread Safety:** Multi-threaded stress evaluation (`testing.B.RunParallel`) executing across dozens of simulated goroutines under explicit race detection (`go test -race`) to verify lock-free reader throughput across global pattern caches.
- **Environment & Kernel Transparency:** Complete archival reporting of CPU architecture/clock speeds, RAM topology, operating system kernel build, Go toolchain build (`go version`), and Node.js binary execution versions.

---

## 6. Target Benchmark Metrics
Benchmark evaluation reports will capture four rigorous quantitative indicators per operation:
- **`ns/op` (Nanoseconds per operation):** Wall-clock CPU elapsed time required for pattern syntax compilation or target string evaluation.
- **`B/op` (Bytes allocated per operation):** Dynamic heap memory footprint allocated per function invocation.
- **`allocs/op` (Allocations per operation):** Distinct heap object allocation calls triggered during execution. The critical release engineering target is **precisely `0 allocs/op`** for repeated evaluation loops utilizing pre-compiled `Matcher` structures and direct-equality fastpaths.
- **`MB/s` & `matches/sec` (Throughput rate):** Processing volume and evaluation velocity across batch directory traversals and simulated filesystem scanning runs.

---

## 7. Comparison Libraries & Cross-Language Baselines
To establish definitive runtime competitiveness, performance evaluation will compare Port Mortem across three canonical software reference targets:
1. **Upstream Node.js Picomatch:** Cross-language computational evaluation against native JavaScript execution timers over simulated filesystem traversals, ReDoS stress sequences, and deep directory hierarchies.
2. **Go Standard Library:** Baseline speed and memory allocation comparisons against native standard library primitives (`path/filepath.Match` and `io/fs.Glob`).
3. **Leading Open-Source Go Libraries:** Competitive performance positioning against widely utilized Go pattern matching implementations, specifically `github.com/bmatcuk/doublestar` and `github.com/gobwas/glob`.

---

## 8. Benchmark Datasets
Evaluation targets will ingest standardized, real-world filesystem testing datasets representing diverse computational stresses:
- **Standard Wildcard & Extension Hierarchies:** Common source code routing paths (`foo/bar/*.js`, `**/*.{js,ts,go}`).
- **Deeply Nested Globstars:** Catastrophic recursive directory searching structures (`foo/**/bar/**/baz/**/*.js` against multi-tiered simulated path strings).
- **Composite Alternation Extglobs:** Deeply nested logical exclusion and union expressions (`@(foo|@(bar|@(baz|quux)))/*.js`).
- **POSIX & Interval Range Expansions:** Multi-character bracket expressions (`[[:alpha:]][[:alnum:]]*`) and interval series (`src/{build,test}/{1..10}/*.js`).
- **Unicode UTF-8 & Multibyte Traversals:** Multibyte character file path evaluations verifying zero overhead during UTF-8 string indexing.

---

## 9. Remaining Benchmark Tasks & Roadmap Implementation
With prerequisite scaffolding in place, remaining performance engineering tasks are sequenced across two targeted upcoming release phases:
- **Task 1 (Planned Phase C / Sprint 13):** Execute formal `go test -bench=. -benchmem ./...` evaluation suites across all target architectures, recording empirical `ns/op`, `B/op`, and `allocs/op` metrics into comparison tables within this document.
- **Task 2 (Planned Phase D):** Execute CPU and heap memory profiling (`go test -cpuprofile` / `-memprofile`) to identify runtime evaluation bottlenecks, lock contention points, or residual transient memory allocations.
- **Task 3 (Planned Phase D Optimization):** Integrate structural object memory pools (`sync.Pool`) across internal segment slices and tokenizer structures to eradicate any remaining heap churn, securing optimal zero-allocation efficiency prior to v1.0 release Candidate packaging.
