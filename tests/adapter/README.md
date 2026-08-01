# Testing Bridge (Production Hardened Refactor)

This directory contains the production-ready testing bridge linking the JavaScript test suite to the Go implementation via a persistent subprocess.

## File Tree

```
tests/adapter/
├── bridge.js          # JS proxy managing the Go subprocess, timeouts, limits, and state
├── bridge.test.js     # Comprehensive stress and unit tests (1000 sequential & concurrent requests)
├── main.go            # Persistent Go process evaluating JSON lines with graceful return
└── README.md          # This documentation
```

## Protocol Specification

Communication occurs over standard input (`stdin`) and standard output (`stdout`) using newline-delimited JSON.

### Maximum Message Size
The maximum supported message size is **1 MB** (`1,048,576 bytes`). Both the JavaScript sender (`GoBridge`) and the Go receiver (`bufio.Scanner` buffer) strictly enforce this limit to prevent buffer overflow attacks or resource exhaustion.

### Message Schema & Validation
Every message exchanged across the bridge must contain a valid `version` and `type` field. If either is missing or invalid, an explicit error response is dispatched and the message is rejected.

| Field | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `version` | string | **Yes** | Protocol version (must strictly equal `"1.0"`). |
| `id` | string | No | Unique message identifier for correlating requests to responses. Required for requests and delays. |
| `type` | string | **Yes** | Message type corresponding to shared protocol constants. |
| `payload` | object | No | The dynamic data payload carrying parameters or results. |
| `error` | string | No | Present only if `type` is `"error"`, containing the descriptive failure reason. |

### Shared Message Type Constants

Both environments use identical, frozen string constants for routing traffic:

- `handshake` (`MSG_TYPES.HANDSHAKE`): Initiates the session. JS sends a handshake and expects a `{ "payload": { "status": "ready" } }` reply before marking the bridge ready.
- `request` (`MSG_TYPES.REQUEST`): Standard execution request. JS passes parameters in the `payload`. Go replies with `"type": "response"` and the result in `payload`.
- `simulate_delay` (`MSG_TYPES.SIMULATE_DELAY`): A testing command instructing Go to sleep for a specified duration in `payload.ms`. Used to test timeout determinism.
- `shutdown` (`MSG_TYPES.SHUTDOWN`): Tells Go to cleanly exit. Go replies with `{ "payload": { "status": "closing" } }` and performs a graceful return from `main()`.
- `response` / `error`: Used by Go to communicate successes or explicit protocol/syntax validation errors back to JS.

## Setup and Testing Instructions

1. **Build the Go Binary**: The JS bridge executes a pre-compiled binary (`bridge.exe` on Windows, `bridge` on POSIX).
   ```bash
   cd tests/adapter
   go build -o bridge.exe main.go
   ```

2. **Run the Unit & Stress Tests**:
   ```bash
   node bridge.test.js
   ```

## Production Enhancements & Safeguards Met

- **Awaiting Process Exit**: `bridge.close()` actively resolves only after the Go subprocess emits its terminal `close` event, preventing orphaned file descriptors.
- **Graceful Return over `os.Exit(0)`**: Go terminates by breaking out of the main scanner loop and returning naturally from `main()`, allowing deferred handlers to execute if added later.
- **Race Condition Mitigations**: During shutdown, a strict boolean guard (`isClosing`) prevents any new requests from being scheduled while pending traffic is drained or rejected.
- **High-Throughput Stress Testing**: Verified against **1000 sequential** and **1000 concurrent** simultaneous promises to guarantee zero message drop or correlation mismatch under heavy load.
- **Strict Input & Field Validation**: Detects malformed JSON, missing required schema fields, and excessive message payloads on both ends of the bridge.
