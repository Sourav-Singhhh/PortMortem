# Contributing to Port Mortem

We welcome open-source contributions, bug reports, and diagnostic verification improvements to the Port Mortem project! As this repository represents an enterprise-grade, memory-safe Go port of the JavaScript `picomatch` glob matching library, contributors must strictly adhere to the project's engineering governance models and verification standards.

---

## Core Engineering Governance Principles

To preserve operational reliability, memory safety, and behavioral predictability, all collaborative maintainers and external contributors must observe the following inviolable design principles:

### 1. Maintain Behavioral Parity Above All Else
Bug-for-bug behavioral equivalence with original Node.js `picomatch` is the foundational mandate of this library. Never implement intentional divergent optimizations, simplify syntax parsing heuristics, or alter boundary evaluation logic without verifying complete empirical consensus against native Node.js runtime outputs.

### 2. Do Not Redesign the Parser or Matcher Architecture
The single-pass interleaved parser loop ([port/parse.go](file:///C:/Users/rajpu/Desktop/PortMortem/port/parse.go)) and runtime matcher evaluation engine ([port/matcher.go](file:///C:/Users/rajpu/Desktop/PortMortem/port/matcher.go)) represent completed, certified historical engineering implementations. Do not introduce multi-pass lexing layers, intermediate Abstract Syntax Tree (AST) transductions, recursive string manipulation routines, or third-party PCRE backtracking bindings (`regexp2`).

### 3. Require Automated Differential Verification
Any proposed additions or modifications to pattern matching rules, options configurations, or cross-platform normalization fastpaths must be validated using automated cross-language differential test suites communicating via the persistent inter-process communication (IPC) bridge (`tests/adapter/`). Never merge architectural modifications based solely upon theoretical assumptions or manual assertions.

### 4. Record Architectural Decisions in [DECISIONS.md](file:///C:/Users/rajpu/Desktop/PortMortem/DECISIONS.md)
Every significant structural design trade-off, RE2 regular expression adaptation, or performance refactor must be formally logged in `DECISIONS.md`. Document the precise architectural rationale, accepted adaptations, rejected alternatives, security considerations (such as ReDoS resistance), and empirical verification metrics.

### 5. Preserve Zero-Allocation Execution Invariants
All precompiled matching evaluations ([Matcher.Match](file:///C:/Users/rajpu/Desktop/PortMortem/port/matcher.go)), cached pattern compilations ([Compile](file:///C:/Users/rajpu/Desktop/PortMortem/port/matcher.go)), and casual helper invocations ([Match](file:///C:/Users/rajpu/Desktop/PortMortem/port/matcher.go)) must maintain literally **`0 B/op, 0 allocs/op`** during runtime execution. Any PR introducing heap allocations to runtime evaluation fastpaths will be rejected.

### 6. Keep Documentation Synchronized with Code
Preserve absolute synchronization between active source code syntax, exported GoDoc comments, root onboarding guides ([README.md](file:///C:/Users/rajpu/Desktop/PortMortem/README.md)), release changelogs ([CHANGELOG.md](file:///C:/Users/rajpu/Desktop/PortMortem/CHANGELOG.md)), and empirical benchmark performance tables ([BENCHMARKS.md](file:///C:/Users/rajpu/Desktop/PortMortem/BENCHMARKS.md)).

---

## Setting Up Your Development Environment

### Prerequisites
- **Go Toolchain:** Go 1.22+ required (developed and certified against Go 1.26+).
- **Node.js:** Node.js v18+ required for cross-language differential verification bridge execution.

### Verification & Testing Pipeline
Before submitting a Pull Request, execute the complete official project verification pipeline inside the `port/` module directory:

```bash
cd port

# 1. Invalidate local test and build toolchain caches
go clean -cache
go clean -testcache

# 2. Enforce standard formatting across all Go files
gofmt -w .

# 3. Perform static toolchain diagnostics
go vet ./...

# 4. Execute unit, platform, Unicode, normalization, ReDoS, and differential test suites
go test -count=1 -v ./...

# 5. Verify computational speed and zero-allocation memory invariants
go test -bench . -benchmem
```

---

## Submitting Pull Requests

1. Fork the repository and create a descriptive, feature-scoped git branch off `develop`.
2. Ensure commit messages adhere to conventional commit formatting (e.g., `fix(matcher): resolve trailing separator evaluation issue`).
3. Verify that your git working tree is 100% clean of temporary build binaries, profiling outputs, coverage logs, or scratch artifacts.
4. Open a Pull Request referencing any related issues and append complete verification pipeline execution output in the PR description.
