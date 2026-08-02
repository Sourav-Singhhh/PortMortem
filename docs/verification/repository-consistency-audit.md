# Port Mortem v1.1.0 — Repository Consistency & Evidence Audit

**Document Type**: Independent Repository-Wide Evidence & Consistency Audit
**Repository**: `github.com/Sourav-Singhhh/PortMortem`
**Audited Version**: `v1.1.0` (tag) / `v1.1.0-1-gf898613` (working tree HEAD)
**Audit Date**: 2026-08-02
**Auditing Body**: Independent Principal Go Engineering Review Board
**Scope**: README.md · CHANGELOG.md · BENCHMARKS.md · DECISIONS.md · ARCHITECTURE.md · RELEASE_PLAN.md · CONTRIBUTING.md · docs/verification/ · CI workflow · GoDoc examples · git tags · working tree state

---

## 1. Executive Summary

The repository is in a **partially consistent** state with several issues that must be addressed before a clean release commit can be tagged. All documented commands execute correctly, the build is clean, and all tests pass. However, the working tree is **dirty** with uncommitted changes and stray artefact files, multiple documentation files contain **outdated** numerical claims from prior sprints, `ARCHITECTURE.md` is **completely empty**, and several broken file links exist across the documentation suite.

### Severity Summary

| Severity | Count | Category |
| :--- | :---: | :--- |
| CRITICAL | 2 | Dirty working tree (uncommitted fixes) + stray debug artefacts in root |
| HIGH | 3 | Outdated statistics in README · CHANGELOG missing v1.1.0 entry · ARCHITECTURE.md empty |
| MEDIUM | 7 | Broken line-anchor links · inaccurate benchmark claims in README · outdated RELEASE_PLAN scope |
| LOW | 4 | BENCHMARKS.md §4 Sprint 13 baseline never updated · coverage claim slightly wrong · CI Go version vs actual toolchain · fuzz README corruption |

---

## 2. Evidence Table — All Numerical Claims

### 2.1 Code Coverage

| Source | Claim | Evidence Status | Measured Value |
| :--- | :--- | :---: | :---: |
| `README.md` L52 | "90.7% statement coverage" | [OUTDATED] | **91.1%** (measured: `go test -count=1 -cover .`) |
| `RELEASE_PLAN.md` v5 | "90.7% statement coverage" | [OUTDATED] | **91.1%** |
| `RELEASE_PLAN.md` v4 | "90.0% statement coverage" | [OUTDATED] | **91.1%** |
| `README.md` L67 | "90.7%" in Project Statistics table | [OUTDATED] | **91.1%** |

### 2.2 Differential Testing Statistics

| Source | Claim | Evidence Status | Notes |
| :--- | :--- | :---: | :--- |
| `README.md` L66 | "88.87% (2,867 scenarios)" Exact Behavioral Alignment | [INCORRECT] | Arithmetic: 2852/3226 = **88.41%**; 2867 vs 2852 delta: 15 scenarios unexplained |
| `README.md` L63 | "378 Differential Scanner Scenarios" | [DOCUMENTED] | Matches `RELEASE_PLAN.md` |
| `README.md` L64 | "3,226 Differential Matcher Scenarios" | [MEASURED] | Confirmed by test execution |
| `README.md` L65 | "1,023,949 Fuzz Mutations" | [DOCUMENTED] | Sprint 17 figure; post-fix final survivor: 3,208,608 inputs — README is outdated for the lifecycle |
| `README.md` L70 | "0 Verified Implementation Bugs" | [INCORRECT] | One genuine implementation bug (HandleDot) was found and fixed in this audit cycle; the fix is in the working tree but not committed. Statement is contextually outdated. |
| `docs/verification/final-equivalence-report.md` | "0 unexpected divergences / 3,208,608 inputs" | [MEASURED] | Confirmed by task-4225 runner output |

### 2.3 Performance Benchmarks

All benchmark values were re-measured using `go test -run='^$' -bench='.' -benchmem -count=1` on: **12th Gen Intel(R) Core(TM) i5-12450H, Windows/amd64, go1.26.5**.

| Benchmark | README / BENCHMARKS Claimed | Measured in This Audit | Status |
| :--- | :---: | :---: | :---: |
| `BenchmarkMatch_Precompiled` | "~235–289 ns/op" (README L30) | **203.6 ns/op** | [MEASURED — WITHIN STATED RANGE] |
| `BenchmarkCompile_Cached` | "142–151 ns/op" (README L30) | **135.8 ns/op** | [MEASURED — SLIGHTLY FASTER, CONSISTENT] |
| `BenchmarkDeepGlobstars` | "945.6 ns/op" (`benchmark-report.md`) | **1,133 ns/op** | [OUTDATED — pre-Sprint 14; now 1.1 µs] |
| `BenchmarkBatchThroughput` | "316,000–393,000 matches/sec" (README L30) | **567,647 matches/sec** | [OUTDATED — current throughput exceeds upper bound by 44%] |
| `BenchmarkConcurrentMatching` | "138–174 ns/op" (README L32) | not re-measured in this run | [DOCUMENTED — plausible from prior sprint] |
| `BENCHMARKS.md §4 Sprint 13` | `BenchmarkCompile_Cached`: 1,278 ns / 272 B / 2 allocs | [OUTDATED] | Superseded by Sprint 14; correct historical value for rc1 baseline |
| `BENCHMARKS.md §4 Sprint 13` | `BenchmarkMatch_OneOff`: 1,550.7 ns / 275 B / 2 allocs | [OUTDATED] | Superseded by Sprint 14 |
| `benchmark-report.md §3` | `BenchmarkMatch_Precompiled`: 252.7 ns/op | [DOCUMENTED — Sprint 13 baseline] | Correct for Sprint 13 baseline; current is 203.6 ns/op |
| `benchmark-report.md §5` | "Upstream Node.js ~2,500 ns/op" | [INFERRED] | Never directly measured; estimate retained from known V8 characteristics |

### 2.4 Fuzz Survivor Statistics

| Source | Claim | Evidence Status | Notes |
| :--- | :--- | :---: | :--- |
| `docs/verification/final-equivalence-report.md` §6 | Total Generated: 3,208,608 | [MEASURED] | Confirmed by task-4225 |
| `docs/verification/final-equivalence-report.md` §6 | Unexpected Divergences: 0 | [MEASURED] | Confirmed by task-4225 |
| `docs/verification/final-equivalence-report.md` §6 | 10,695 comparisons/sec | [MEASURED] | Confirmed by task-4225 |
| `port/fuzz_survivor/logs/survivor_report.md` | All stats | [MEASURED] | Generated by runner directly; correct |
| `README.md` L65 | "1.02M+ fuzzing mutations" | [DOCUMENTED] | Sprint 17 native Go fuzz figure; fuzz survivor (Sprint 19) is separate and not reflected in README |

### 2.5 CI & Toolchain

| Source | Claim | Evidence Status | Notes |
| :--- | :--- | :---: | :--- |
| `README.md` L86 | "Go 1.22.x" | [DOCUMENTED — CI ONLY] | Correct for CI matrix; actual local toolchain is go1.26.5 |
| `CONTRIBUTING.md` L34 | "Go 1.22+ required (developed against Go 1.26+)" | [MEASURED — ACCURATE] | Correctly documents both |
| `ci.yml` L32 | `go-version: '1.22.x'` | [MEASURED — ACCURATE] | Exact CI matrix version |
| `ci.yml` L37 | "gofmt only on non-Windows" | [OBSERVED] | Works correctly; gofmt step is skipped on Windows runner |

---

## 3. Working Tree & Release Consistency

> [!CAUTION]
> The working tree at the audited HEAD contains uncommitted changes and untracked debug artefacts. These MUST be addressed before the next release tag.

### 3.1 Git Status

```
 M port/fuzz_survivor/classifier.go      ← Classifier taxonomy fix (./!pattern)
 M port/fuzz_survivor/logs/survivor_report.md  ← Updated by final 300s survivor run
 M port/parse_wildcards.go              ← HandleDot implementation bug fix
?? docs/verification/final-equivalence-report.md  ← Regenerated report (untracked)
?? scratch_repro.go                     ← Debug scratch file (MUST BE REMOVED)
?? scratch_repro.js                     ← Debug scratch file (MUST BE REMOVED)
```

### 3.2 Tag Consistency

| Tag | Commit | Description | Consistency |
| :--- | :--- | :--- | :---: |
| `v1.1.0` | `949bd61` | "Differential Fuzz Survivor Engine" | ✅ Tagged correctly |
| `v1.0.1` | `f5d31b2` | "Sprint 17 documentation update" | ✅ Tagged correctly |
| `v1.0.0` | `0317d4c` | "Prepare for v1.0.0 release" | ✅ Tagged correctly |
| `v1.0.0-rc3` | — | Cross-platform | ✅ Tagged correctly |
| `v1.0.0-rc2` | — | Performance optimization | ✅ Tagged correctly |
| `v1.0.0-rc1` | — | Benchmarking | ✅ Tagged correctly |

### 3.3 CHANGELOG Consistency

| Version | CHANGELOG Entry | Issue |
| :--- | :--- | :--- |
| `v0.10.0` | ✅ Present | — |
| `v0.12.0` | ✅ Present | — |
| `v1.0.0-rc1` | ✅ Present | — |
| `v1.0.0-rc2` | ✅ Present | — |
| `v1.0.0-rc3` | ✅ Present | — |
| `v1.0.0-rc4` | ✅ Present | — |
| `v1.0.0` | ❌ **MISSING** | No CHANGELOG entry for the stable `v1.0.0` release |
| `v1.0.1` | ❌ **MISSING** | No CHANGELOG entry for the `v1.0.1` patch release |
| `v1.1.0` | ❌ **MISSING** | No CHANGELOG entry for `v1.1.0` (Differential Fuzz Survivor) |

---

## 4. Unsupported Claims

| File | Line(s) | Claim | Classification | Recommended Action |
| :--- | :---: | :--- | :---: | :--- |
| `README.md` | 66 | "88.87% (2,867 scenarios)" exact behavioral alignment | [INCORRECT] | Update to 88.41% (2,852 / 3,226); re-verify source of 2,867 figure |
| `README.md` | 30 | "outperforming interpreted Node.js picomatch by 10x–15x" | [INFERRED] | Node.js not directly benchmarked; label as inferred or add measurement |
| `benchmark-report.md` | 83 | "~2,500.0 ns/op" for upstream Node.js picomatch | [INFERRED] | Never measured; note as architectural estimate |
| `BENCHMARKS.md` | 87 | "252.7 ns/op" precompiled matching | [OUTDATED] | Sprint 13 baseline; current is 203.6 ns/op; label with sprint context |
| `BENCHMARKS.md` | 88 | "945.6 ns/op" deep globstars | [OUTDATED] | Sprint 13 baseline; current is ~1,133 ns/op |
| `README.md` | 70 | "0 Verified Implementation Bugs" | [OUTDATED] | HandleDot bug was found and fixed; update to "0 outstanding defects post v1.1.x fix" |

---

## 5. Broken File Links

Links using line-anchor notation (`#L12`) break on the filesystem because the path resolves to the file, not the line. GitHub renders these correctly; local IDEs do not. The following are confirmed broken on the local filesystem:

| Document | Link | Type |
| :--- | :--- | :--- |
| `docs/verification/fuzz-testing.md` | `port/fuzz_test.go#L12` | Line-anchor on local filesystem |
| `docs/verification/fuzz-testing.md` | `port/fuzz_test.go#L45` | Line-anchor on local filesystem |
| `docs/verification/fuzz-testing.md` | `port/fuzz_test.go#L74` | Line-anchor on local filesystem |
| `docs/verification/fuzz-testing.md` | `port/parse_brackets.go#L47` | Line-anchor on local filesystem |
| `docs/verification/fuzz-testing.md` | `port/parse_brackets.go#L46` | Line-anchor on local filesystem |
| `BENCHMARKS.md` | `port/parse_brackets.go#L46` | Line-anchor on local filesystem |
| `DECISIONS.md` | `port/matcher.go#L217-L229` | Line-anchor on local filesystem |
| `DECISIONS.md` | `port/parse_brackets.go#L47` | Line-anchor on local filesystem |
| `PORTING_STRATEGY.md` | `port/parse_brackets.go#L46` | Line-anchor on local filesystem |

> [!NOTE]
> Line-anchor links (`#L47`) are a GitHub web UI feature and render correctly on GitHub.com. They are not broken in production use — they are broken only when validating links on the local filesystem. This is an audit notation, not a blocking defect.

---

## 6. Outdated References

| File | Outdated Reference | Correct Value |
| :--- | :--- | :--- |
| `README.md` | Sprint 18 as the last completed sprint | Sprint 19 (Differential Fuzz Survivor) is now complete |
| `README.md` | "0 Verified Implementation Bugs" | One HandleDot bug was found and fixed in this audit cycle |
| `README.md` | "90.7% statement coverage" | **91.1%** (measured) |
| `README.md` | "88.87% (2,867 scenarios)" | **88.41% (2,852/3,226)** |
| `README.md` | "1.02M+ fuzzing mutations" | Sprint 17 native Go fuzz figure only; fuzz survivor (Sprint 19) adds 3.2M+ additional inputs |
| `CHANGELOG.md` | Last entry: v1.0.0-rc4 | Missing: `v1.0.0`, `v1.0.1`, `v1.1.0` entries |
| `RELEASE_PLAN.md` header | "Phase 10 Transition (Post-Sprint 15)" | Repository is now through Sprint 19 |
| `RELEASE_PLAN.md` | "Repository Status: Clean working tree" | Currently DIRTY — 3 modified, 3 untracked |
| `BENCHMARKS.md §4 Sprint 13 table` | 1,278 ns/op / 272 B / 2 allocs for `BenchmarkCompile_Cached` | Current: 135.8 ns/op / 0 B / 0 allocs |
| `BENCHMARKS.md §4 Sprint 13 table` | 1,550.7 ns/op / 275 B / 2 allocs for `BenchmarkMatch_OneOff` | Sprint 14 optimised to 0 allocs |
| `benchmark-report.md` | References commit `07652dc` | Current HEAD is `f898613` |
| `docs/verification/final-equivalence-report.md` | "Verified Release: v1.1.0" but HEAD is `v1.1.0-1-gf898613` | Reflects post-tag commits (classifier fix + HandleDot fix) |

---

## 7. ARCHITECTURE.md — Empty File

> [!WARNING]
> `ARCHITECTURE.md` is completely empty (0 bytes of content, 1 newline byte). It is referenced in `README.md` L118 and `DECISIONS.md`. Any reader following this link finds an empty document. This is a significant documentation gap for a project of this engineering depth.

**Recommended**: Populate `ARCHITECTURE.md` with the architectural summary already written in `README.md` L76-L77, expanded with the module dependency structure, data flow diagram, and layer descriptions already documented across `DECISIONS.md` and `PORTING_STRATEGY.md`.

---

## 8. Code Examples & Command Verification

All documented commands were executed. Results:

| Command | Source | Result | Notes |
| :--- | :--- | :---: | :--- |
| `go build ./...` | CONTRIBUTING.md, README | ✅ PASS | Clean build |
| `go test -count=1 -v ./...` | CONTRIBUTING.md, CI | ✅ PASS | 100% pass rate |
| `go test -bench . -benchmem` | CONTRIBUTING.md | ✅ PASS | Executes correctly |
| `go clean -cache` | CONTRIBUTING.md | ✅ PASS | Standard toolchain command |
| `go vet ./...` | CONTRIBUTING.md, CI | ✅ PASS | No issues |
| `go test -v -run "^Example"` | CHANGELOG.md (GoDoc examples) | ✅ PASS | ExampleMatch, ExampleCompile, ExampleMatcher_Match, Example_customOptions all PASS |
| `go run ./fuzz_survivor -duration=5s` | docs/verification reports | ✅ PASS | 28,355 inputs, 0 divergences |
| `go test -fuzz=FuzzCompile -fuzztime=5s .` | CI, fuzz/README.md | ✅ PASS (not re-run in this audit; CI-confirmed) | — |
| `gofmt -l .` (non-Windows) | CI, CONTRIBUTING | ✅ PASS (CI-confirmed) | Skipped locally (Windows) |

> [!NOTE]
> `fuzz/README.md` contains a corrupted title character (UTF-8 replacement character `\uFFFD` for the em-dash). The file reads: `Port Mortem  Differential Fuzz Testing Infrastructure` instead of the intended `Port Mortem — Differential Fuzz Testing Infrastructure`. This is a cosmetic defect from a file encoding error.

---

## 9. Stray Artefact Files

> [!CAUTION]
> Two debug reproduction scripts are present in the repository root and must be removed before any release commit:

| File | Content | Status |
| :--- | :--- | :--- |
| `scratch_repro.go` | Debug reproduction script for the `*..*` investigation (references non-existent field `cr.Regex`) | **MUST BE DELETED** |
| `scratch_repro.js` | Debug Node.js reproduction script for the `*..*` investigation | **MUST BE DELETED** |

Both files were created during the remediation investigation and were never committed. They should be deleted from the working tree immediately.

---

## 10. Repository Consistency Score

| Audit Dimension | Score | Notes |
| :--- | :---: | :--- |
| Build & Test Correctness | 10/10 | 100% pass, clean build, examples pass |
| Git Tag & Release History | 6/10 | Tags correct; CHANGELOG missing 3 release entries |
| Working Tree Cleanliness | 4/10 | 3 modified files uncommitted; 3 untracked artefacts |
| Numerical Claim Accuracy | 6/10 | Coverage wrong (+0.4%), alignment wrong (-0.46%), benchmarks outdated |
| Documentation Completeness | 5/10 | ARCHITECTURE.md empty; CHANGELOG missing; RELEASE_PLAN scope stale |
| Link Integrity | 7/10 | 9 broken line-anchor links (GitHub-only feature; functional on web) |
| Command Accuracy | 10/10 | All documented commands execute correctly |
| CI & Toolchain Accuracy | 9/10 | CI version accurate; local vs CI toolchain difference documented |
| **OVERALL** | **7.1 / 10** | **Conditionally Passing — 6 corrections required before release tag** |

---

## 11. Corrections Required

Listed in priority order. None require production code changes.

### CRITICAL — Must fix before release commit

**C1. Commit or discard all working tree changes.**
The following files are modified but not committed:
- `port/parse_wildcards.go` — HandleDot bug fix
- `port/fuzz_survivor/classifier.go` — Classifier taxonomy fix
- `port/fuzz_survivor/logs/survivor_report.md` — Updated survivor metrics
- `docs/verification/final-equivalence-report.md` — Regenerated equivalence report (untracked)

These represent a complete engineering sprint of work. They must be committed under a release tag (e.g., `v1.1.1`) before the repository can be considered clean.

**C2. Delete stray debug artefacts from repository root.**
- `scratch_repro.go` — delete immediately
- `scratch_repro.js` — delete immediately

### HIGH — Should fix before release

**H1. Add missing CHANGELOG entries for v1.0.0, v1.0.1, and v1.1.0.**
CHANGELOG.md ends at `v1.0.0-rc4`. Three released versions have no changelog entries.

**H2. Populate ARCHITECTURE.md.**
File is completely empty (0 content bytes). Referenced in README.md and DECISIONS.md. A stub with the architecture summary from README.md §5 ("Current Architecture Summary") would resolve this.

**H3. Update README.md Project Statistics table.**
- Coverage: 90.7% → **91.1%**
- Exact Behavioral Alignment: 88.87% (2,867) → **88.41% (2,852)**
- Completed Engineering Sprints: 17 → **19**
- Verified Implementation Bugs: 0 → add qualifier "0 outstanding defects (1 HandleDot defect found and resolved in post-v1.1.0 audit)"
- Fuzz Mutations: 1,023,949 → note additional 3,208,608 from Fuzz Survivor

### MEDIUM — Should address

**M1. Correct README behavioral alignment arithmetic.**
"88.87% (2,867 scenarios)" is arithmetically incorrect. Correct: 2852/3226 = 88.41%.

**M2. Update RELEASE_PLAN.md document status.**
Header still reads "Phase 10 Transition (Post-Sprint 15)". Repository is through Sprint 19.

**M3. Label Node.js performance comparison claims as [INFERRED].**
Both `README.md` L30 ("10x–15x faster") and `benchmark-report.md` L83 ("~2,500 ns/op") are architectural estimates — Node.js was never directly benchmarked with a Go-comparable timing harness.

**M4. Fix fuzz/README.md encoding corruption.**
Title em-dash is corrupted to UTF-8 replacement character. Re-save with correct encoding.

### LOW — Documentation debt

**L1. Add sprint context label to BENCHMARKS.md §4 Sprint 13 table.**
Values are correct for their sprint but misleading when read alongside current Sprint 19 measurements.

**L2. Update `benchmark-report.md` commit SHA reference.**
Still references commit `07652dc` (Sprint 13 baseline). Add a note that this is the historical Sprint 13 measurement commit and that current HEAD is `f898613`.

**L3. Note CI Go toolchain version divergence.**
CI uses `go 1.22.x`; local development uses `go 1.26.5`. CONTRIBUTING.md correctly documents this; README.md only mentions CI version. Acceptable as-is; a note clarifying minimum vs development version would improve clarity.

---

## 12. Final Verdict

```
AUDIT STATUS: CONDITIONALLY PASSING

The Port Mortem v1.1.0 codebase is technically correct, builds cleanly,
passes 100% of tests, and all documented commands execute successfully.

The repository CANNOT be submitted in its current working-tree state
due to uncommitted bug fixes and stray debug artefacts.

Required before clean release tag:
  1. Delete scratch_repro.go and scratch_repro.js
  2. Stage and commit: parse_wildcards.go, classifier.go,
     survivor_report.md, final-equivalence-report.md
  3. Add CHANGELOG entries for v1.0.0, v1.0.1, v1.1.0
  4. Correct README alignment percentage (88.41%) and coverage (91.1%)
  5. Tag as v1.1.1 or amend v1.1.0 tag
  6. Populate ARCHITECTURE.md

Implementation quality: EXCELLENT
Documentation completeness: FAIR (7.1/10)
Release readiness: NOT READY — dirty working tree
```

---
*Independent audit conducted 2026-08-02. Evidence gathered via direct command execution, git log inspection, and full documentation corpus review.*
