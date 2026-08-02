# Port Mortem — Differential Fuzz Testing Infrastructure

Sprint 17 introduces native Go fuzz testing (`testing.F`) to validate parser stability, memory safety, and behavioral equivalence against upstream Node.js `picomatch` v3.0.1.

## Fuzz Targets Summary

| Target Name | Location | Scope | Verification Type |
| :--- | :--- | :--- | :--- |
| `FuzzCompile` | [port/fuzz_test.go](file:///C:/Users/rajpu/Desktop/PortMortem/port/fuzz_test.go) | Pattern parsing, AST construction, & RE2 regex synthesis | Panic & crash resilience |
| `FuzzMatch` | [port/fuzz_test.go](file:///C:/Users/rajpu/Desktop/PortMortem/port/fuzz_test.go) | Compiled matcher execution on arbitrary string inputs | Memory boundary & panic safety |
| `FuzzDifferentialMatcher` | [port/fuzz_test.go](file:///C:/Users/rajpu/Desktop/PortMortem/port/fuzz_test.go) | Live stdio IPC streaming comparison vs Node.js picomatch | Behavioral consensus verification |

## Execution Commands

### 1. Execute Compilation Fuzzing
```bash
go test -fuzz=FuzzCompile -fuzztime=30s ./port
```

### 2. Execute Matching Resilience Fuzzing
```bash
go test -fuzz=FuzzMatch -fuzztime=30s ./port
```

### 3. Execute Live Differential Fuzzing against Node.js
```bash
go test -fuzz=FuzzDifferentialMatcher -fuzztime=30s ./port
```

## Architecture & Corpus Management
- **Seed Corpus:** Seed inputs covering wildcards, globstars, extglobs, ranged braces, character classes, and escaped delimiters are defined directly in `port/fuzz_test.go`.
- **Node.js IPC Integration:** Differential fuzzing connects to `port/testdata/js_matcher.js` over stdin/stdout JSON pipes, evaluating real-time Node.js execution against Go output.
- **Divergence Filtering:** Known architectural adaptations (RE2 non-linear lookaround limits and path traversal blocking on `.` and `..`) are programmatically handled to isolate genuine bugs.
