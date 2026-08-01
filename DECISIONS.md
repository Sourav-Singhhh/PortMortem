# Port Mortem Architecture & Engineering Decisions Log

This document records the historical decision architecture of the Port Mortem project from initialization through Sprint 6. All design choices prioritize **bug-for-bug behavioral parity** with the authoritative JavaScript library (`original-picomatch/lib/parse.js` and `scan.js`) over arbitrary simplifications or idiom adaptations.

---

## 1. Why the Scanner Was Ported First
- **Decision:** The structural fast scanner (`scan.js` $\rightarrow$ `scan.go`) was implemented before attempting any parser or compiler migration.
- **Rationale:** In `picomatch`, the scanner acts as a fast-pass analyzer that traverses raw glob strings in a simple loop without building complex Abstract Syntax Trees (ASTs). Its sole objective is isolating the static `base` path, the dynamic `glob` suffix, and setting boolean grammar flags (`isBrace`, `isBracket`, `isExtglob`, `isGlobstar`). Porting the scanner first proved that Go string indexing, boundary checking, and dynamic option flags could achieve exact parity with JavaScript string parsing while validating the cross-language testing framework early.

---

## 2. Why Differential Testing Was Chosen
- **Decision:** Automated differential testing against running Node.js binaries was selected over reliance on hand-written unit test assertions alone.
- **Rationale:** JavaScript glob matching rules feature thousands of obscure boundary behaviors, undocumented option permutations (`nonegate`, `noext`, `literalBrackets`), and malformed input recovery heuristics. A pure Go unit test suite risked testing against assumed behavior rather than empirical reality. Differential testing feeds identical inputs simultaneously to both the Go implementation and the native Node.js library, instantly catching subtle divergences in slice boundaries, output strings, or state flags.

---

## 3. Why the Persistent Node Bridge Exists
- **Decision:** A persistent inter-process communication (IPC) testing bridge (`tests/adapter/`) running over standard inputs/outputs was constructed instead of invoking CLI sub-processes per test.
- **Rationale:** Executing `node -e "..."` in a separate OS sub-process for every individual test case incurs catastrophic operating system fork/exec overhead, increasing test suite runtimes to several minutes. A persistent Go-JS IPC bridge initializes a single background Node.js instance at test suite startup and communicates via serialized JSON over `stdin`/`stdout`, running hundreds of complex differential verification fixtures in under two seconds.

---

## 4. Why the Parser Foundation Was Separated (Sprint 3)
- **Decision:** Before writing any syntactic grammar interpretation, a foundational milestone (Sprint 3) cleanly separated and validated core structures (`ParseState`, `ParseToken`, `ParseOptions`, and stack management helpers like `PushToken()`, `Append()`, `Increment()`, `Decrement()`).
- **Rationale:** Attempting to build grammatical parsing loops simultaneously with core state tracking structures creates untraceable regressions. Isolating foundational data models guaranteed that token merging heuristics (`PushToken` combining sequential plain text items) and state invariants matched `parse.js` exactly before complex switch control flows were introduced.

---

## 5. Why the Standalone Lexer Was Removed & Single-Pass Adopted (Sprint 4 & Refactor)
- **Decision:** An early experimental two-pass standalone lexer (`Lexer`, `Step()`, `Tokenize()`, `TokenTypeEscape`) was completely purged from the codebase during forensic review. The project transitioned exclusively to an **interleaved single-pass parser architecture** emulating `while (!state.eos())` from `parse.js:661`.
- **Rationale:** In JavaScript `picomatch`, lexical token classification relies dynamically on active parser runtime state (e.g., whether the parser is currently inside a character class via `state.brackets > 0`, inside an extglob via `state.parens > 0`, or modified by option flags like `nobracket`). Furthermore, syntactic evaluation routinely mutates previously generated token strings and accumulated compiled output via backwards scanning and slicing (e.g., `escapeLast`). Splitting parsing into a rigid two-pass pipeline (static pre-tokenization followed by AST assembly) made bug-for-bug JavaScript behavioral replication impossible without severe divergence.

---

## 6. Parser Cursor Abstraction Layer
- **Decision:** All sequential scanning operations (`Advance()`, `Peek()`, `EOS()`, `Remaining()`, and `Consume()`) live directly as methods on `ParseState`, mirroring the closure functions defined in `parse.js:443-447`.
- **Rationale:** Keeping cursor manipulation coupled to `ParseState` enables syntactic switch branches to safely perform lookaheads (`Peek(1)`) and substring extraction (`Remaining()`) without duplicate buffer management. To guarantee Go memory safety and prevent out-of-bounds panics, all cursor methods silently return fallback values (byte `0` or empty string slices `""`) when reading past EOF.

---

## 7. Literal & Backslash Handling Decisions (Sprint 5)
- **Decision:** Literal text character accumulation, plain text AST token merging, and backslash escape processing (`HandleEscape`) were embedded directly within the single-pass switch loop in `Parse()`.
- **Rationale:** Upstream `parse.js:672` evaluates backslash escapes immediately prior to character class traversal (`parse.js:718`). Maintaining this strict evaluation order ensures that escaped symbols inside bracket expressions (e.g., `[a\-z]`) consume the literal character before general character class rules execute.

---

## 8. Bracket Parsing Decisions (Sprint 6)
- **Decision:** All square bracket logic (`HandleOpenBracket`, `HandleCloseBracket`, `HandleBracketTraversal`, `HandleUnclosedBrackets`) operates inside the primary single-pass loop without spawning dedicated secondary sub-parsers.
- **Rationale:** Opening `[` tokens increment `state.Brackets` or immediately escape to `\\[` when no matching `]` exists in `Remaining()`. Closing `]` tokens decrement depth and automatically inject path slashes into negated expressions (`[^...]` $\rightarrow$ `[^.../]`) per `parse.js:846`. At EOF, unbalanced open brackets are reconciled by invoking `EscapeLast` on `state.Output`—matching `parse.js:1286` line-for-line without breaking single-pass runtime behavior.

---

## 9. Accepted & Rejected Alternatives

| Proposed Alternative | Status | Reason for Acceptance or Rejection |
| :--- | :---: | :--- |
| **Two-Pass Standalone Lexing Phase** | **REJECTED** | Pre-tokenizing prevents dynamic grammar state from altering token boundaries and breaks backwards string mutation rules like `escapeLast`. |
| **Recursive String Slicing for `EscapeLast`** | **REJECTED** | Upstream `utils.js` uses recursive string slices for `escapeLast`. In Go, this creates heavy memory allocations and recursion depth overhead; an iterative reverse search loop was accepted as an equivalent behavior refactor. |
| **Byte-Level Scanning with UTF-8 String Slicing** | **ACCEPTED** | ASCII syntax markers (`*`, `[`, `{`, `/`) evaluate cleanly via single bytes, while token string accumulation uses byte slices (`string([]byte{ch})`) to ensure valid UTF-8 reconstruction without character corruption. |
| **Typed Error Wrapping (`fmt.Errorf`)** | **ACCEPTED** | Rather than throwing untyped raw string exceptions as in JS (`new SyntaxError`), Go errors use formatted wrapped error types while preserving exact string messaging parity. |

---

## 10. Future Deferred Decisions
The following complex architectural items have been intentionally deferred to subsequent sprints to preserve scope containment:
- **POSIX Character Class Translation Table (`[:alnum:]`, etc.):** Translating POSIX table classes crosses into regular expression pattern generation; deferred to compiler integration phases via explicit TODO markers.
- **Regex Compiler Output & Lookaround Constraints:** Translating intermediate parser state into finalized regular expressions (`compiler.go`) is deferred to resolve Go `regexp` (RE2) vs `regexp2` (PCRE) negative lookaround limitations (`(?!...)`) without polluting foundational parser grammar.
- **Brace Expansion (`{...}`) & Extglob (`!(...)`) Recursion:** Deferred to Sprints 7 and 8 to verify bracket structural foundations in complete isolation before introducing nested branching mechanics.
