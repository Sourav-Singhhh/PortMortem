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

---

## 11. Brace Parsing Foundation Decisions (Sprint 7)
- **Why brace parsing was implemented before brace expansion:** Structural brace parsing—detecting opening `{`, closing `}`, and comma `,` delimiters while tracking nesting depth (`state.Braces`), stack balance (`state.BraceStack`), and malformed EOF closure—is a foundational prerequisite defined in `parse.js:881-969`. Attempting to implement expansion algorithms (`{1..5}`, `{a..z}`) before verifying structural delimiter containment would conflate syntax boundary recognition with recursive pattern synthesis.
- **Why rollback behavior follows `parse.js` exactly:** In upstream `parse.js:925`, when a closing `}` is encountered without an intervening comma `,` or range dots `..` (`!brace.Comma && !brace.Dots`), Bash and glob semantics dictate that solitary braces do not constitute expansion groups (e.g., `{a}` must literally match `{a}`). By replicating the exact backwards output slicing (`state.output.slice(0, brace.outputIndex)`) and token mutations to literal escaped expressions (`\{` and `\}`), our Go implementation preserves bug-for-bug regex pattern compilation without requiring secondary AST rewriting passes.
- **Why brace expansion remains intentionally deferred:** Range enumeration and combinatorial brace expansion generate recursive arrays and potential exponential state growth. Deferring expansion ensures structural grammar scanning remains linear-time, clean, and completely isolated within the foundational single-pass loop.
- **Why comma tracking is stack-context dependent:** In `parse.js:958-965`, a comma `,` is only converted to regex alternation `|` and flags `brace.Comma = true` if the immediate top delimiter on `stack` is `'braces'`. If a comma occurs inside parens or brackets nested inside a brace expression (e.g., `{foo,(a,b)}`), the top of `stack` is `'parens'` or `'brackets'`, leaving the comma literally untouched. This stack context invariance prevents nested grammar constructs from splitting enclosing brace expansion groups.
- **Why single-pass architecture was preserved:** Integrating brace handlers directly into the primary single-pass switch statement (`while (!state.EOS())`) avoids multi-pass preprocessing or preliminary regex scanning, maintaining optimal performance and exact algorithmic parity with `parse.js:661`.
- **Accepted Go refactors:**
  1. *Strongly-typed stack wrappers:* Wrapping untyped JavaScript token references inside a strongly-typed `BraceState` struct pushed onto `BraceStack`, ensuring static Go type safety while preserving shared token mutations with `Tokens`.
  2. *Slice bounds verifications:* Adding strict Go slice bounds verifications before truncating `Output[:outIdx]` and `Tokens[tokIdx:]` during backtracking rollbacks to prevent runtime panics on malformed structures.
  3. *Stack EOF termination:* Invoking `BraceStack.Pop()` alongside `s.Decrement(ParserContextBraces)` during EOF unclosed cleanup (`HandleUnclosedBraces`) to guarantee clean structural state termination.
- **Rejected alternatives:**
  1. *Recursive pre-parsing / expansion:* Pre-expanding brace patterns prior to syntax parsing was rejected because it breaks backslash escaping rules and distorts character indices.
  2. *Secondary regex lookahead scanners:* Using secondary sub-parsers or regular expression lookaheads to evaluate brace validity was rejected as it violates single-pass architectural integrity and diverges from `parse.js`.
- **Deferred implementation decisions:**
  1. *Numeric and alphabetic range enumeration (`{1..5}`, `{a..z}`):* Intentionally deferred to dedicated compiler/expansion milestones via architectural TODO placeholders (`parse.js:907-923`).
  2. *Extglob parenthesis tracking (`!(...)`, `@(...)`, etc.):* Completed in Sprint 8 to isolate brace structural validation from recursive extglob parsing.

---

## 12. Extglob Parsing Foundation Decisions (Sprint 8)
- **Why extglob parsing foundation was implemented before regex synthesis:** Structural extglob parsing—detecting operator prefixes (`?`, `!`, `+`, `@`, `*`) followed by `(`, tracking parenthetical nesting depth (`state.Parens`), managing state checkpoints (`state.ExtglobStack`), counting condition alternations on pipe `|` symbols, accumulating inner text (`state.Inner`), and reconciling unclosed parens at EOF—is defined in `parse.js:507-600` and `1021-1143`. Attempting to generate complex regular expressions before verifying structural nesting and token boundary containment would conflate grammar state management with regex compilation.
- **Why single-pass architecture was preserved:** Embedding extglob prefix handlers (`HandleExtglobPrefix`), parenthesis handlers (`HandleOpenParen`, `HandleCloseParen`), and condition trackers (`HandlePipe`) directly within the primary character switch statement avoids multi-pass preprocessing or preliminary regex scanning, maintaining exact algorithmic parity with `parse.js`.
- **Why `@(...)` does not push onto `ExtglobStack`:** In upstream `parse.js:1096-1099`, `@(...)` expressions do not invoke `extglobOpen` nor push state entries onto the tracking stack. Instead, `@` emits a stripped token marked with `extglob:true`, leaving the subsequent `(...)` block to open and close as a standard parenthetical group. By replicating this behavior identically, our Go implementation preserves bug-for-bug control flow parity.
- **Why regex capture group exclusion was implemented:** When evaluating negation syntax starting with `!(...`, if the characters immediately following the opening parenthesis are lookaround or capture group indicators (`?`, `!`, `=`, `<`, or `:` via `isRegexGroupChar`), `HandleExtglobPrefix` deliberately suppresses extglob initiation (`parse.js:1054-1059`), ensuring regular expression groups (such as `!(?:foo)` or `!(?!bar)`) are preserved without corruption.
- **Why inner string accumulation occurs in `PushToken`:** In upstream `parse.js:507-509`, token insertion via `push(tok)` appends non-paren token values directly into `ext.Inner` for the active topmost extglob state on `extglobs`. Integrating this logic directly into Go's `PushToken` helper ensures inner content aggregation runs synchronously during traversal without requiring AST traversal passes.
- **Accepted Go refactors:**
  1. *Strongly-typed extglob state:* Wrapping untyped JavaScript object dictionaries inside a strongly-typed `ExtglobState` struct pushed onto `ExtglobStack`, ensuring Go static type safety while preserving exact checkpoint field mappings (`Output`, `StartIndex`, `Parens`, `TokensIndex`).
  2. *Defensive nil & bounds checking:* Adding explicit nil checks and stack emptiness assertions when accessing `ExtglobStack.Peek()` in `PushToken` and `HandlePipe`, preventing runtime panics when tests use manually constructed state instances.
- **Rejected alternatives:**
  1. *Regular expression pre-compilation during parsing:* Transforming extglob structures into compiled RE2 regular expression strings during character traversal was rejected because it violates foundational separation of concerns and prematurely ties structural grammar parsing to specific regex engine targets.
  2. *Multi-pass parenthetical lookahead scanning:* Using auxiliary scanners to locate closing parentheses before opening an extglob state was rejected as it violates single-pass linear time invariants and diverges from `parse.js` EOF unclosed reconciliation rules (`EscapeLast`).
- **Deferred implementation decisions:**
  1. *Extglob regex pattern synthesis (`(?:...`, `(?!(?:...`):* Intentionally deferred to subsequent compiler sprints via architectural TODO placeholders (`parse.js:542-591`).
  2. *ReDoS safeguard analysis (`analyzeRepeatedExtglob`):* Repeated extglob optimization analysis is deferred to regex synthesis sprints to keep structural parser foundations isolated and linear-time.
