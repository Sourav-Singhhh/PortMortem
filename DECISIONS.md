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

---

## 13. Wildcard & Globstar Parsing Foundation Decisions (Sprint 9)
- **Why wildcard parsing was implemented before regex generation:** Structural wildcard parsing—recognizing slashes `/`, dots `.`, question marks `?`, single stars `*`, globstar transitions `**`, and star collapsing `***` while managing state flags (`Globstar`, `Backtrack`) and lookahead/lookbehind syntax exclusions—is defined in `parse.js:970-1284`. Attempting to generate finished regular expression patterns before establishing strict character-by-character token boundaries would conflate syntax recognition with target engine regex synthesis.
- **Why globstar remains structural only:** Recognizing `**` as a standalone directory globstar vs a regular star sequence requires checking boundary conditions (preceding BOS/slash/syntax and subsequent EOS/slash) per `parse.js:1145-1244`. Implementing the actual matching mechanics or regex expansion of `**` across directory hierarchies during parser traversal would violate single-pass constraints; keeping globstar evaluation strictly structural ensures clean separation of grammar recognition from path evaluation.
- **Parser state mutations introduced:**
  - `state.Start`: Advanced when stripping leading `"./"` sequences at BOS (`parse.js:980-987`), resetting `Consumed` and `Output` accumulators to streamline BOS lookbehind assertions.
  - `state.Globstar` & `state.Backtrack`: Flagged to `true` whenever two consecutive stars upgrade to a globstar token or three/more consecutive stars collapse into a star token (`parse.js:1128-1137`).
  - `BraceState.Dots`: Flagged to `true` when consecutive dots form a range (`".."`) inside an active brace group (`parse.js:998-1006`).
- **Rollback bookkeeping decisions:** Rollback checkpoints created by opening extglobs or braces remain completely deterministic when wildcards are encountered. Because wildcard evaluations mutate tokens and `state.Output` inline without leaking unbounded heap structures, executing a rollback during malformed unclosed brace or extglob recovery cleanly restores character indices, output strings, and token sequences to exact pre-wildcard states.
- **Star collapsing rationale (`***`):** In upstream `parse.js:1128-1137`, encountering three or more consecutive stars (e.g., `***` or `****`) collapses the structural representation back into a single `TokenTypeStar` node while retaining the full string in `Value`, marking `prev.Star = true`, and flagging `s.Backtrack = true` and `s.Globstar = true`. This prevents combinatorial exponential expansion and ReDoS vulnerabilities in down-stream compiler evaluation.
- **Globstar promotion/demotion rationale:**
  - *Promotion:* A second consecutive star promoted to `TokenTypeGlobstar` is strictly conditioned upon boundary guards (`parse.js:1178-1244`), respecting `opts.NoGlobstar` (disables promotion) and `opts.Bash` (enforces strict directory separation).
  - *Demotion (`PushToken`):* In `parse.js:494-505`, when a non-slash, non-paren, or non-syntax delimiter is pushed directly after a globstar token (e.g. `**a` or `**.js`), `PushToken` demotes the globstar node back to a single `TokenTypeStar` with value `"*"`, accurately slicing `"**"` off `state.Output` to preserve structural synchronization.
- **Slash normalization decisions:** Slashes emit `TokenTypeSlash` (`/`). To replicate Node.js picomatch lookbehind optimization at the start of patterns, when `/` follows `TokenTypeDot` at `Index == Start+1` (`"./"`), the dot and slash tokens are removed from the AST slice and state accumulators, allowing subsequent tokens to anchor cleanly against BOS.
- **Dot handling rationale:** Dots emit `TokenTypeDot` at directory boundaries (BOS or following slashes). Inside braces, consecutive dots mutate into range tokens (`TokenTypeDots`, `".."`). Outside braces and parentheses not following BOS or slashes, dots are categorized as plain text literals (`TokenTypeText` per `parse.js:1008-1011`), merging directly with adjacent plain text tokens for memory optimization.
- **Interactions with braces, brackets, and extglobs:** Within POSIX brackets (`[*?]`), wildcards are treated as character class literals without invoking wildcard handlers. Within braces (`{*.js,*.ts}`) and extglobs (`*(a*|b?)`), wildcard handlers execute cleanly within parenthetical and brace groups without perturbing nesting depth counters (`Braces`, `Parens`) or extglob condition alternation trackers.
- **Accepted Go adaptations:**
  1. *Mid-pattern globstar anomaly preservation:* In upstream `parse.js:1201-1218`, evaluating mid-pattern globstars (`a/**/b`) executes both `consume(value + advance())` and `append(tok)`, causing `state.consumed` to accumulate a redundant trailing slash (`"/*/**//***//"` for input `"/*/**/***//"`). Rather than applying an unverified fix, our Go implementation retains this exact behavior to guarantee bug-for-bug forensic parity.
  2. *Defensive regex group validation:* Implemented `isValidNamedGroupOrLookbehind` in Go using explicit bounds checking to evaluate lookbehinds (`<=`, `<!`) and named capture groups (`<name>`) after open parentheses without incurring regex compilation overhead during scanning.
- **Rejected alternatives:**
  1. *Premature regex expansion:* Translating `*` to `[^/]*` and `?` to `[^/]` during parser character scanning was rejected as it violates single-pass structural separation and breaks rollback recovery in unclosed groups.
  2. *AST multi-pass rewriting:* Post-processing the AST slice in a secondary traversal to promote or demote globstar symbols was rejected to adhere strictly to single-pass runtime behavior.
- **Future deferred work:**
  1. *Regular expression synthesis for wildcards:* Generating final regex fragments (`SLASH_LITERAL`, `DOT_LITERAL`, `QMARK`, `NO_DOTS_SLASH`) is intentionally deferred to subsequent compiler sprints via explicit architectural TODO markers (`parse.js:989`, `1013`, `1041-1045`, `1189-1223`, `1263-1281`).
  2. *Globstar matching engine logic:* Recursive directory matching mechanics for `**` are deferred to matcher evaluation sprints.

---

## 14. Parser Completion & Regex Synthesis Decisions (Sprint 10)
- **Why regex synthesis remained inside the existing single-pass parser:** In original Node.js `picomatch`, regular expression string patterns are constructed incrementally within `state.output` directly during character scanning in `original-picomatch/lib/parse.js`. Moving regex generation out into a separate post-processing AST compiler phase would sever synchronous interactions between token mutation (such as globstar demotion in `push()`) and output string modification (`state.Output`), breaking bug-for-bug behavioral alignment and degrading single-pass runtime performance.
- **Why no parser redesign was introduced:** Existing structural foundations from Sprints 4 through 9 (cursor navigation, character handlers, and delimiter stacks) proved structurally sound and mathematically sufficient. Maintaining exact architectural parity with `parse.js` avoided unnecessary refactor churn, preserved 100% of historical test suites without alteration, and ensured seamless integration of regex generation code directly into established handler branches.
- **Parser completion rationale:** With the integration of POSIX character tables, numerical brace range expansions, extglob alternation translations, wildcard replacements (`SLASH_LITERAL`, `DOT_LITERAL`, `QMARK`), ReDoS vulnerability analysis, and EOF delimiter reconciliation, every individual syntactic responsibility from `parse.js`, `constants.js`, and `utils.js` has been realized. All historical `TODO` markers have been systematically eradicated, marking the Go parser migration formally complete.
- **Regex synthesis architecture:** A dedicated support module ([port/parse_regex.go](file:///C:/Users/rajpu/Desktop/PortMortem/port/parse_regex.go)) centralizes regex constant definitions, POSIX mappings (`GetPosixRegexSource`), character escaping (`EscapeRegex`), globstar pattern generators (`Globstar`), and ReDoS scanning functions. During normal parser traversal, character handlers in `port/parse_*.go` call these utility constructors to mutate `state.Output` with platform-aware matching expressions (`GetGlobChars(opts.Windows)`).
- **POSIX translation decisions:** In accordance with `parse.js:719-741` and `constants.js:73-89`, bracket classes referencing standard POSIX symbol names (e.g. `[:alnum:]`, `[:digit:]`, `[:punct:]`) are looked up via a statically compiled Go translation map (`GetPosixRegexSource`), injecting ASCII equivalent character classes (`a-zA-Z0-9`, `0-9`, etc.) directly into bracket expression evaluations without external dependencies.
- **Brace range expansion decisions:** Numerical and alphabetical sequence boundaries (such as `{1..5}` or `{a..z}`) identified by consecutive dot tokens inside braces are expanded via `ExpandRange` ([port/parse_regex.go](file:///C:/Users/rajpu/Desktop/PortMortem/port/parse_regex.go)), leveraging dynamic interval string construction (`[1-5]`, `[a-z]`) or user-supplied custom range callback overrides (`opts.ExpandRange`).
- **Wildcard compilation decisions:**
  - *Slashes (`/`):* Emitted as `SLASH_LITERAL` (`\\/` on POSIX, `\\/|\\\\` on Windows per `GetGlobChars`), stripping redundant tokens when trailing `opts.Prepend` at BOS.
  - *Dots (`.`):* Translated to `DOT_LITERAL` at directory boundaries unless `opts.Dot` is true, ensuring leading dotfile patterns require explicit dot matches in target file strings.
  - *Question Marks (`?`) & Stars (`*`):* Single wildcards compile to character exclusion classes (`[^/]` or `[^/]*?`), excluding leading dots (`(?!(?:\\/|^)\\.)`) when occurring at pattern boundaries unless overridden by dotfile or options configuration.
  - *Globstars (`**`):* Emitted via `Globstar(opts, chars)`, wrapping directory-spanning wildcard sequences in lookbehind/lookahead guards (`(?:^(?:\\/|\\.)?|\\/)(?:(?!(?:^|\\/)\\.).)*?(?:\\/|$)`) with non-capturing group suppression when `opts.Capture` is enabled.
- **Extglob synthesis decisions:** Closing an extglob expression (`HandleExtglobClose` per `parse.js:542-591`) evaluates the saved operator (`?`, `!`, `+`, `*`, `@`) stored on `state.ExtglobStack`. Inner string alternatives accumulated across condition pipes (`|`) are wrapped in equivalent regular expression constructs (e.g., `(?:...)` for `@`, `(?:...)*` for `*`, and negative lookahead exclusions `(?!(?:...))` for `!`), executing string slicing on `state.Output` from the recorded extglob starting boundary.
- **ReDoS mitigation strategy:** To prevent catastrophic exponential backtracking during matching evaluation on complex overlapping extglob repetitions (`+(+(*))`, `*(*(a|b))`), `AnalyzeRepeatedExtglob` ported from `original-picomatch/lib/utils.js` scans branch substrings for repeating quantifier overlap and unbounded self-referential prefix recursion. If an extglob sequence is flagged as risky (`res.Risky == true`), `state.Backtrack` is asserted, neutralizing dangerous combinatorial regex expansion.
- **Accepted Go adaptations:**
  1. *Static Regex Helper Tables:* Replaced JavaScript dynamic object property inspections in `constants.js` with type-safe Go struct lookups (`GlobChars`, `ExtglobCharDef`), ensuring thread-safe concurrency and zero memory allocation overhead across successive `Parse()` invocations.
  2. *Defensive ReDoS Fallback Branching:* Retained all defensive error-recovery branches from `utils.js` inside `SplitTopLevel`, `IsPlainBranch`, and `ParseRepeatedExtglob`, ensuring safe termination without runtime panics when inspecting malformed or unbalanced user fuzz patterns.
- **Rejected architectural alternatives:**
  1. *AST-To-Regexp Transduction Pipeline:* Decoupling syntactic character scanning from regular expression compilation into a multi-phase abstract syntax tree transformer was firmly rejected because it violates foundational single-pass invariants and destroys token-to-output memory synchronization required for exact upstream parity.
  2. *Standard `regexp/syntax` Pre-compilation:* Attempting to directly construct Go `regexp/syntax.Prog` bytecode trees during parser scanning was rejected to ensure `state.Output` retains an inspectable, mathematically verifiable JavaScript regular expression pattern string for precise differential testing against Node.js ground truth.
- **Lessons learned from the entire parser migration:**
  1. *Foundational discipline succeeds:* Resisting premature optimization during early scanning and foundational parsing sprints made complex regex compiler synthesis in Sprint 10 exceptionally linear and predictable.
  2. *Bug-for-bug parity requires algorithmic patience:* Preserving apparent upstream anomalies (such as globstar double-slash consumption and trailing `maybe_slash` EOF logic) was critical for passing extensive differential suites without creating divergence cascades.
  3. *Zero TODO policy guarantees real closure:* Systematically resolving every stub and placeholder ensures the parser operates as an authoritative, self-contained engine ready for high-performance matcher execution.

