package picomatch

import (
	"errors"
	"fmt"
)

// NewParseOptions creates a new ParseOptions struct populated with standard default operational values.
func NewParseOptions() *ParseOptions {
	return &ParseOptions{
		MaxLength:           ParserMaxInputLength,
		Fastpaths:           DefaultFastpaths,
		Posix:               DefaultPosix,
		Prepend:             DefaultPrepend,
		MaxExtglobRecursion: ParserDefaultMaxExtglobRecursion,
	}
}

// NewParserStack instantiates a new empty ParserStack structure for tracking structural delimiter contexts.
func NewParserStack() *ParserStack {
	return &ParserStack{items: make([]ParserContext, 0)}
}

// NewBraceStack instantiates a new empty BraceStack structure for tracking active brace groups.
func NewBraceStack() *BraceStack {
	return &BraceStack{items: make([]*BraceState, 0)}
}

// NewExtglobStack instantiates a new empty ExtglobStack structure for tracking active extglob shells.
func NewExtglobStack() *ExtglobStack {
	return &ExtglobStack{items: make([]*ExtglobState, 0)}
}

// NewParseToken allocates and initializes a new ParseToken structural syntax node.
func NewParseToken(tokenType TokenType, value string, output string) *ParseToken {
	return &ParseToken{
		Type:      tokenType,
		Value:     value,
		Output:    output,
		OutputSet: output != "",
	}
}

// NewBraceState allocates and initializes a tracking structure for an opening curly brace group.
func NewBraceState(token *ParseToken, value string, output string, outIdx int, tokIdx int) *BraceState {
	return &BraceState{
		Token:       token,
		Value:       value,
		Output:      output,
		OutputIndex: outIdx,
		TokensIndex: tokIdx,
	}
}

// NewExtglobState allocates and initializes a tracking structure for an extended glob expression shell.
func NewExtglobState(token *ParseToken, extType TokenType, value string, parens int, startIdx int, tokIdx int) *ExtglobState {
	return &ExtglobState{
		Token:       token,
		Type:        extType,
		Value:       value,
		Parens:      parens,
		StartIndex:  startIdx,
		TokensIndex: tokIdx,
	}
}

// NewParseState allocates and initializes a complete runtime parsing state object with an immutable opening BOS token.
func NewParseState(input string, opts *ParseOptions) *ParseState {
	if opts == nil {
		opts = NewParseOptions()
	}

	bos := NewParseToken(TokenTypeBos, "", opts.Prepend)
	if opts.Prepend != "" {
		bos.OutputSet = true
	}

	return &ParseState{
		Input:        input,
		Index:        -1,
		Start:        0,
		Dot:          opts.Dot,
		Tokens:       []*ParseToken{bos},
		Stack:        NewParserStack(),
		BraceStack:   NewBraceStack(),
		ExtglobStack: NewExtglobStack(),
		Opts:         opts,
	}
}

// Push appends a delimiter context category onto the LIFO ParserStack.
func (s *ParserStack) Push(item ParserContext) {
	s.items = append(s.items, item)
}

// Pop removes and returns the most recently pushed category from the ParserStack.
func (s *ParserStack) Pop() (ParserContext, bool) {
	if len(s.items) == 0 {
		return "", false
	}
	idx := len(s.items) - 1
	val := s.items[idx]
	s.items = s.items[:idx]
	return val, true
}

// Peek retrieves the top category from the ParserStack without removing it.
func (s *ParserStack) Peek() (ParserContext, bool) {
	if len(s.items) == 0 {
		return "", false
	}
	return s.items[len(s.items)-1], true
}

// Len returns the current number of elements currently on the ParserStack.
func (s *ParserStack) Len() int {
	return len(s.items)
}

// IsEmpty evaluates whether the ParserStack contains zero elements.
func (s *ParserStack) IsEmpty() bool {
	return len(s.items) == 0
}

// Clear resets the ParserStack to an empty slice.
func (s *ParserStack) Clear() {
	s.items = make([]ParserContext, 0)
}

// Push appends an active BraceState structure onto the BraceStack.
func (b *BraceStack) Push(item *BraceState) {
	b.items = append(b.items, item)
}

// Pop removes and returns the most recently pushed BraceState from the BraceStack.
func (b *BraceStack) Pop() (*BraceState, bool) {
	if len(b.items) == 0 {
		return nil, false
	}
	idx := len(b.items) - 1
	val := b.items[idx]
	b.items = b.items[:idx]
	return val, true
}

// Peek retrieves the top BraceState from the BraceStack without removing it.
func (b *BraceStack) Peek() (*BraceState, bool) {
	if len(b.items) == 0 {
		return nil, false
	}
	return b.items[len(b.items)-1], true
}

// Len returns the current number of active entries in the BraceStack.
func (b *BraceStack) Len() int {
	return len(b.items)
}

// IsEmpty evaluates whether the BraceStack contains zero entries.
func (b *BraceStack) IsEmpty() bool {
	return len(b.items) == 0
}

// Clear resets the BraceStack to an empty state.
func (b *BraceStack) Clear() {
	b.items = make([]*BraceState, 0)
}

// Push appends an active ExtglobState structure onto the ExtglobStack.
func (e *ExtglobStack) Push(item *ExtglobState) {
	e.items = append(e.items, item)
}

// Pop removes and returns the most recently pushed ExtglobState from the ExtglobStack.
func (e *ExtglobStack) Pop() (*ExtglobState, bool) {
	if len(e.items) == 0 {
		return nil, false
	}
	idx := len(e.items) - 1
	val := e.items[idx]
	e.items = e.items[:idx]
	return val, true
}

// Peek retrieves the top ExtglobState from the ExtglobStack without removing it.
func (e *ExtglobStack) Peek() (*ExtglobState, bool) {
	if len(e.items) == 0 {
		return nil, false
	}
	return e.items[len(e.items)-1], true
}

// Len returns the current number of active entries in the ExtglobStack.
func (e *ExtglobStack) Len() int {
	return len(e.items)
}

// IsEmpty evaluates whether the ExtglobStack contains zero entries.
func (e *ExtglobStack) IsEmpty() bool {
	return len(e.items) == 0
}

// Clear resets the ExtglobStack to an empty state.
func (e *ExtglobStack) Clear() {
	e.items = make([]*ExtglobState, 0)
}

// Increment advances the corresponding structural depth counter and pushes the category onto the state delimiter stack.
func (s *ParseState) Increment(contextType ParserContext) {
	switch contextType {
	case ParserContextBrackets:
		s.Brackets++
	case ParserContextBraces:
		s.Braces++
	case ParserContextParens:
		s.Parens++
	}
	s.Stack.Push(contextType)
}

// Decrement reduces the corresponding structural depth counter and removes the top category from the state delimiter stack.
func (s *ParseState) Decrement(contextType ParserContext) {
	switch contextType {
	case ParserContextBrackets:
		if s.Brackets > 0 {
			s.Brackets--
		}
	case ParserContextBraces:
		if s.Braces > 0 {
			s.Braces--
		}
	case ParserContextParens:
		if s.Parens > 0 {
			s.Parens--
		}
	}
	s.Stack.Pop()
}

// EOS evaluates whether the sequential scanning index has reached or exceeded the end of the input pattern string.
func (s *ParseState) EOS() bool {
	return s.Index >= len(s.Input)-1
}

// Peek inspects the byte located n characters ahead of the current scanning index without modifying state position.
func (s *ParseState) Peek(n int) byte {
	target := s.Index + n
	if target < 0 || target >= len(s.Input) {
		return 0
	}
	return s.Input[target]
}

// Advance increments the sequential scanning index and returns the byte at the new offset.
func (s *ParseState) Advance() byte {
	s.Index++
	if s.Index < 0 || s.Index >= len(s.Input) {
		return 0
	}
	return s.Input[s.Index]
}

// Remaining returns the substring slice from immediately after the current scanning index to the end of input.
func (s *ParseState) Remaining() string {
	start := s.Index + 1
	if start < 0 || start >= len(s.Input) {
		return ""
	}
	return s.Input[start:]
}

// Consume appends a literal sequence to the consumed accumulator and advances the scanning index by num offsets.
func (s *ParseState) Consume(value string, num int) {
	s.Consumed += value
	s.Index += num
}

// Append concatenates a token's output representation (or raw value if output is unset) onto the compiled regex state output.
func (s *ParseState) Append(token *ParseToken) {
	if token.OutputSet || token.Output != "" {
		s.Output += token.Output
	} else {
		s.Output += token.Value
	}
	s.Consume(token.Value, 0)
}

// CurrentToken safely retrieves the most recently appended token at the tail of the AST sequence.
func (s *ParseState) CurrentToken() *ParseToken {
	if len(s.Tokens) == 0 {
		return nil
	}
	return s.Tokens[len(s.Tokens)-1]
}

// PushToken ingests a newly constructed token into the AST slice, merging consecutive plain text tokens to optimize allocation and establishing back-links.
func (s *ParseState) PushToken(token *ParseToken) {
	// Downgrade globstar to single star when followed by non-directory or non-syntax delimiters (parse.js:494-505)
	if prev := s.CurrentToken(); prev != nil && prev.Type == TokenTypeGlobstar {
		isBrace := s.Braces > 0 && (token.Type == TokenTypeComma || token.Type == TokenTypeBrace)
		isExtglob := token.Extglob || (s.ExtglobStack != nil && !s.ExtglobStack.IsEmpty() && (token.Type == TokenTypePipe || token.Type == TokenTypeParen))

		if token.Type != TokenTypeSlash && token.Type != TokenTypeParen && !isBrace && !isExtglob {
			if len(s.Output) >= len(prev.Value) {
				s.Output = s.Output[:len(s.Output)-len(prev.Value)]
			}
			prev.Type = TokenTypeStar
			prev.Value = "*"
			prev.Output = ""
			prev.OutputSet = false
			s.Output += prev.Value
		}
	}

	// Accumulate token values into active extglob expressions (parse.js:507-509)
	if s.ExtglobStack != nil && !s.ExtglobStack.IsEmpty() && token.Type != TokenTypeParen {
		if ext, ok := s.ExtglobStack.Peek(); ok {
			ext.Inner += token.Value
		}
	}

	if token.Value != "" || token.Output != "" || token.OutputSet {
		s.Append(token)
	}

	if prev := s.CurrentToken(); prev != nil {
		if prev.Type == TokenTypeText && token.Type == TokenTypeText {
			if prev.Output != "" || prev.OutputSet || token.Output != "" || token.OutputSet {
				outAdd := token.Output
				if !token.OutputSet && token.Output == "" {
					outAdd = token.Value
				}
				prevOut := prev.Output
				if !prev.OutputSet && prev.Output == "" {
					prevOut = prev.Value
				}
				prev.Output = prevOut + outAdd
				prev.OutputSet = true
			}
			prev.Value += token.Value
			return
		}
		token.Prev = prev
	}
	s.Tokens = append(s.Tokens, token)
}

// ValidateStacks checks that all structural parser stacks are instantiated and valid (test usage only).
func (s *ParseState) ValidateStacks() error {
	if s.Stack == nil || s.BraceStack == nil || s.ExtglobStack == nil {
		return errors.New("one or more parser tracking stacks are uninitialized (nil)")
	}
	return nil
}

// ValidateDepthCounters checks that structural depth counters are non-negative (test usage only).
func (s *ParseState) ValidateDepthCounters() error {
	if s.Brackets < 0 || s.Braces < 0 || s.Parens < 0 || s.Quotes < 0 {
		return errors.New("structural depth counters cannot fall below zero")
	}
	return nil
}

// ValidateTokenLinks audits the token array to ensure BOS immutability, proper Prev back-linkage, and text token merging invariants (test usage only).
func (s *ParseState) ValidateTokenLinks() error {
	if len(s.Tokens) == 0 {
		return errors.New("token stream cannot be empty; missing opening BOS token")
	}
	if s.Tokens[0].Type != TokenTypeBos {
		return errors.New("token stream index 0 must be an immutable BOS token")
	}
	for i := 1; i < len(s.Tokens); i++ {
		curr := s.Tokens[i]
		if curr.Prev != s.Tokens[i-1] {
			return fmt.Errorf("token linkage broken at index %d (%q): Prev does not point to index %d", i, curr.Type, i-1)
		}
		if curr.Type == TokenTypeText && curr.Prev.Type == TokenTypeText {
			return fmt.Errorf("adjacent text tokens found at index %d and %d; text consolidation invariant violated", i-1, i)
		}
	}
	return nil
}
