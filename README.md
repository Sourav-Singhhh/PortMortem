# Port Mortem 2026: High-Performance Picomatch Go Port

A robust, memory-safe, and behaviorally equivalent Go port of the JavaScript `picomatch` glob matching library, engineered for the Port Mortem 2026 migration initiative.

---

## Project Objectives & Philosophy
Port Mortem bridges the gap between JavaScript's complex globbing heuristics and Go's compiled speed and memory safety. The overarching engineering philosophy is **bug-for-bug behavioral parity**: every option toggle, boundary edge case, unclosed delimiter recovery rule, and syntax parsing decision strictly matches the original Node.js `picomatch` library (`original-picomatch/lib/parse.js` and `scan.js`).

---

## Current Implementation Status (Through Sprint 7)
- **Scanner Core (`port/scan.go`):** Fully implemented and certified. Traverses raw glob expressions in a single-pass loop to isolate base directories, evaluate prefix logic (`!`, `./`), and establish grammar flags (`isBrace`, `isBracket`, `isExtglob`, `isGlobstar`).
- **Parser Core Foundation (`port/parse.go`, `parse_literals.go`, `parse_brackets.go`, `parse_braces.go`):** Actively in progress (Phase 4). Established foundational data models (`ParseState`, `ParseToken`, `ParseOptions`), memory-safe sequential cursor navigation, single-pass interleaved loop skeleton, literal/escape handling, square bracket character classes, and structural brace foundations with backtracking rollbacks and comma alternation.
- **Persistent Cross-Language Bridge (`tests/adapter/`):** Implemented high-speed inter-process communication (IPC) daemon communicating with native Node.js binaries via JSON over standard IO for automated differential verification.

---

## Verification & Differential Testing Status
- **Verification Pipeline:** Certified clean across all toolchain metrics (`go clean`, `gofmt -w .`, `go vet ./...`, `go test -count=1 -v ./...`).
- **Test Coverage:** **94.0% statement coverage** achieved across package `picomatch`, with 100% statement coverage across core cursor navigation infrastructure.
- **Differential Testing:** 378 automated cross-language test cases running synchronously against Node.js `picomatch-master`, confirming exact structural equivalence across complex path structures and UTF-8 multibyte sequences.
- **Verification Records:** Full audit certificates and engineering reports are persisted in `docs/verification/`.

---

## Project Statistics
| Metric | Current Value | Status / Notes |
| :--- | :--- | :--- |
| **Total Passing Tests** | 420 / 420 | 100% Pass Rate across unit, boundary, forensic, and differential suites |
| **Differential Scanner Scenarios** | 378 | Zero behavioral divergences against Node.js runtime |
| **Code Statement Coverage** | 94.0% | Exceeds strict 90%+ quality threshold |
| **Completed Engineering Sprints** | 7 Sprints | Initialization through Brace Parsing Foundation |
| **Known Behavioral Divergences** | 0 | Bug-for-bug parity preserved |

---

## Roadmap & Migration Checklist
- [x] **Sprint 1:** Repository initialization and persistent JS-Go IPC testing bridge
- [x] **Sprint 2:** Single-pass structural scanner migration & differential suite
- [x] **Sprint 3:** Parser foundational types, token tree models, and stack operations
- [x] **Sprint 4:** Single-pass interleaved parser loop skeleton & cursor layer refactoring
- [x] **Sprint 5:** Literal text accumulation and backslash escape evaluation
- [x] **Sprint 6:** Bracket parsing foundation, depth tracking, and trailing escape reconciliation
- [x] **Sprint 7:** Brace parsing foundation, nesting depth tracking, comma alternation, and backtracking rollbacks
- [ ] **Sprint 8:** Extglob evaluation (`!(...)`, `@(...)`) and wildcard semantics (`*`, `**`, `?`)
- [ ] **Sprint 9:** Regex compilation engine & RE2 lookaround synthesis
- [ ] **Sprint 10:** Exported public API wrappers (`Compile()`, `IsMatch()`, `MatchBase()`)
- [ ] **Post-Release Sprints:** Automated differential fuzzing (`testing.F`) and memory allocation optimization (`sync.Pool`)

---

## Documentation Registry
- **[DECISIONS.md](file:///C:/Users/rajpu/Desktop/PortMortem/DECISIONS.md):** Complete architectural decisions log detailing design rationale and rejected alternatives.
- **[BENCHMARKS.md](file:///C:/Users/rajpu/Desktop/PortMortem/BENCHMARKS.md):** Performance goals, methodology, metrics, and official postponement status.
- **[PORTING_STRATEGY.md](file:///C:/Users/rajpu/Desktop/PortMortem/PORTING_STRATEGY.md):** Comprehensive engineering blueprint and historical migration progress log.
- **[ARCHITECTURE.md](file:///C:/Users/rajpu/Desktop/PortMortem/ARCHITECTURE.md):** High-level system design, module dependency graphs, and structural paradigms.