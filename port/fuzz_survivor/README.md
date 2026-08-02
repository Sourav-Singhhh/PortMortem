# Differential Fuzz Survivor Infrastructure

The **Differential Fuzz Survivor Engine** continuously evaluates Port Mortem against the original Node.js `picomatch` reference runtime to verify behavioral equivalence, identify potential defect survivors, and log fully reproducible test artifacts.

---

## Execution Guide

### 1. Execute Continuous 60-Second Differential Fuzz Survivor Run
Run the continuous differential survivor test using `go test`:

```powershell
go test -v ./fuzz_survivor -run TestRunDifferentialSurvivor
```

Alternatively, run the standalone runner CLI directly from package `port`:

```powershell
go run ./fuzz_survivor -duration=60s -seed=1337
```

---

## Output Artifacts

All survivor execution logs and reports are automatically generated in `port/fuzz_survivor/logs/`:

1. **`port/fuzz_survivor/logs/survivor_log.jsonl`**: JSON Lines structured log recording candidate inputs, classifications, environmental metadata (Git SHA, Go version, Node version), and exact CLI reproduction commands.
2. **`port/fuzz_survivor/logs/survivor_report.md`**: Markdown summary report generated upon run completion detailing execution velocity, total comparisons, shared API agreement, documented adaptations, and final survivor status.

---

## Reproducing a Discovered Candidate

To reproduce and debug any candidate pattern logged in `survivor_log.jsonl`:

```powershell
go test -v ./fuzz_survivor -run TestReproduceSurvivor -pattern="foo/[[:alnum:]]/bar" -input="foo/9/bar"
```
