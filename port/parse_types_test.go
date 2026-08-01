package picomatch

import (
	"reflect"
	"testing"
)

func TestParseConstructorsAndDefaults(t *testing.T) {
	// 1. Verify NewParseOptions defaults
	opts := NewParseOptions()
	if !opts.Fastpaths || !opts.Posix || opts.MaxLength != ParserMaxInputLength || opts.MaxExtglobRecursion != ParserDefaultMaxExtglobRecursion {
		t.Errorf("NewParseOptions() returned unexpected default values: %+v", opts)
	}
	if opts.Prepend != DefaultPrepend {
		t.Errorf("expected Prepend default %q, got %q", DefaultPrepend, opts.Prepend)
	}

	// 2. Verify token and state element constructors
	tok := NewParseToken(TokenTypeSlash, "/", "\\/")
	if tok.Type != TokenTypeSlash || tok.Value != "/" || tok.Output != "\\/" || !tok.OutputSet {
		t.Errorf("NewParseToken() assignment failure: %+v", tok)
	}

	unSetTok := NewParseToken(TokenTypeText, "a", "")
	if unSetTok.OutputSet {
		t.Errorf("NewParseToken() should not set OutputSet when output is empty string")
	}

	braceState := NewBraceState(tok, "{", "(", 0, 1)
	if braceState.Token != tok || braceState.Value != "{" || braceState.Output != "(" || braceState.OutputIndex != 0 || braceState.TokensIndex != 1 {
		t.Errorf("NewBraceState() assignment failure: %+v", braceState)
	}

	extState := NewExtglobState(tok, TokenTypePlus, "+", 0, 2, 3)
	if extState.Token != tok || extState.Type != TokenTypePlus || extState.Value != "+" || extState.Parens != 0 || extState.StartIndex != 2 || extState.TokensIndex != 3 {
		t.Errorf("NewExtglobState() assignment failure: %+v", extState)
	}

	// 3. Verify NewParseState invariants (nil opts vs customized opts)
	stateNilOpts := NewParseState("foo", nil)
	if stateNilOpts.Input != "foo" || stateNilOpts.Index != -1 || stateNilOpts.Start != 0 {
		t.Errorf("NewParseState() with nil opts failed initialization invariants")
	}
	if len(stateNilOpts.Tokens) != 1 || stateNilOpts.Tokens[0].Type != TokenTypeBos {
		t.Errorf("NewParseState() must initialize with immutable BOS token at index 0")
	}
	if err := stateNilOpts.ValidateStacks(); err != nil {
		t.Errorf("ValidateStacks() failed on newly instantiated state: %v", err)
	}
	if err := stateNilOpts.ValidateDepthCounters(); err != nil {
		t.Errorf("ValidateDepthCounters() failed on newly instantiated state: %v", err)
	}
	if err := stateNilOpts.ValidateTokenLinks(); err != nil {
		t.Errorf("ValidateTokenLinks() failed on newly instantiated state: %v", err)
	}

	customOpts := &ParseOptions{Prepend: "^", Dot: true}
	stateCustom := NewParseState("bar", customOpts)
	if !stateCustom.Dot {
		t.Errorf("NewParseState() failed to capture custom Dot toggle")
	}
	if !stateCustom.Tokens[0].OutputSet || stateCustom.Tokens[0].Output != "^" {
		t.Errorf("NewParseState() failed to apply custom Prepend string to opening BOS token")
	}
}

func TestParserStackOperations(t *testing.T) {
	// 1. ParserStack (Delimiter categories)
	stack := NewParserStack()
	if !stack.IsEmpty() || stack.Len() != 0 {
		t.Errorf("NewParserStack() must begin completely empty")
	}
	if val, ok := stack.Pop(); ok || val != "" {
		t.Errorf("Pop() on empty ParserStack should return empty and false")
	}
	if val, ok := stack.Peek(); ok || val != "" {
		t.Errorf("Peek() on empty ParserStack should return empty and false")
	}
	stack.Push(ParserContextBrackets)
	stack.Push(ParserContextBraces)
	if stack.Len() != 2 || stack.IsEmpty() {
		t.Errorf("ParserStack length mismatch after Push")
	}
	if val, ok := stack.Peek(); !ok || val != ParserContextBraces {
		t.Errorf("Peek() returned unexpected value %q", val)
	}
	if val, ok := stack.Pop(); !ok || val != ParserContextBraces {
		t.Errorf("Pop() returned unexpected value %q", val)
	}
	stack.Clear()
	if !stack.IsEmpty() {
		t.Errorf("ParserStack not empty after Clear()")
	}

	// 2. BraceStack
	bStack := NewBraceStack()
	if !bStack.IsEmpty() || bStack.Len() != 0 {
		t.Errorf("NewBraceStack() must begin completely empty")
	}
	if _, ok := bStack.Pop(); ok {
		t.Errorf("Pop() on empty BraceStack should return false")
	}
	if _, ok := bStack.Peek(); ok {
		t.Errorf("Peek() on empty BraceStack should return false")
	}
	bItem := NewBraceState(nil, "{", "(", 0, 0)
	bStack.Push(bItem)
	if bStack.Len() != 1 {
		t.Errorf("BraceStack len mismatch")
	}
	if peeked, ok := bStack.Peek(); !ok || peeked != bItem {
		t.Errorf("BraceStack Peek() failure")
	}
	if popped, ok := bStack.Pop(); !ok || popped != bItem {
		t.Errorf("BraceStack Pop() failure")
	}
	bStack.Push(bItem)
	bStack.Clear()
	if !bStack.IsEmpty() {
		t.Errorf("BraceStack not empty after Clear()")
	}

	// 3. ExtglobStack
	eStack := NewExtglobStack()
	if !eStack.IsEmpty() || eStack.Len() != 0 {
		t.Errorf("NewExtglobStack() must begin empty")
	}
	if _, ok := eStack.Pop(); ok {
		t.Errorf("Pop() on empty ExtglobStack should return false")
	}
	if _, ok := eStack.Peek(); ok {
		t.Errorf("Peek() on empty ExtglobStack should return false")
	}
	eItem := NewExtglobState(nil, TokenTypeStar, "*", 0, 0, 0)
	eStack.Push(eItem)
	if eStack.Len() != 1 {
		t.Errorf("ExtglobStack len mismatch")
	}
	if peeked, ok := eStack.Peek(); !ok || peeked != eItem {
		t.Errorf("ExtglobStack Peek() failure")
	}
	if popped, ok := eStack.Pop(); !ok || popped != eItem {
		t.Errorf("ExtglobStack Pop() failure")
	}
	eStack.Push(eItem)
	eStack.Clear()
	if !eStack.IsEmpty() {
		t.Errorf("ExtglobStack not empty after Clear()")
	}
}

func TestParseStateDepthCounters(t *testing.T) {
	s := NewParseState("", nil)

	// Increment categories
	s.Increment(ParserContextBrackets)
	s.Increment(ParserContextBraces)
	s.Increment(ParserContextParens)
	s.Increment("other")

	if s.Brackets != 1 || s.Braces != 1 || s.Parens != 1 || s.Stack.Len() != 4 {
		t.Errorf("Increment() failed to update depth counters or delimiter stack")
	}
	if err := s.ValidateDepthCounters(); err != nil {
		t.Errorf("ValidateDepthCounters() failed on positive counters: %v", err)
	}

	// Decrement categories
	s.Decrement(ParserContextBrackets)
	s.Decrement(ParserContextBraces)
	s.Decrement(ParserContextParens)
	s.Decrement("other")
	if s.Brackets != 0 || s.Braces != 0 || s.Parens != 0 || !s.Stack.IsEmpty() {
		t.Errorf("Decrement() failed to decrement counters or pop stack")
	}

	// Decrement on zero counters should guard against negative depth
	s.Decrement(ParserContextBrackets)
	s.Decrement(ParserContextBraces)
	s.Decrement(ParserContextParens)
	if s.Brackets != 0 || s.Braces != 0 || s.Parens != 0 {
		t.Errorf("Decrement() allowed negative depth counter underflow")
	}
}

func TestParseLexingHelpers(t *testing.T) {
	s := NewParseState("ab", nil)

	if !s.EOS() && s.Index != -1 {
		t.Errorf("initial index should be -1 and not EOS")
	}
	if s.Peek(1) != 'a' || s.Peek(2) != 'b' || s.Peek(3) != 0 || s.Peek(-1) != 0 {
		t.Errorf("Peek() boundary inspection failed")
	}

	if s.Remaining() != "ab" {
		t.Errorf("Remaining() from initial index -1 should return full string")
	}

	if ch := s.Advance(); ch != 'a' || s.EOS() {
		t.Errorf("Advance() failed on first character")
	}
	if s.Remaining() != "b" {
		t.Errorf("Remaining() failed after first advance")
	}

	if ch := s.Advance(); ch != 'b' || !s.EOS() {
		t.Errorf("Advance() failed on second character or EOS detection")
	}

	if ch := s.Advance(); ch != 0 {
		t.Errorf("Advance() past end of string should safely return 0")
	}
	if s.Remaining() != "" {
		t.Errorf("Remaining() past end of string should return empty string")
	}

	// Out of bounds negative index remaining test
	s.Index = -5
	if s.Remaining() != "" || s.Peek(1) != 0 {
		t.Errorf("Remaining() and Peek() should return empty/0 on negative out of bounds index")
	}
	s.Advance() // check negative out-of-bounds advance returns 0
	if s.Index < -1 && s.Advance() != 0 {
		t.Errorf("Advance() should return 0 when index remains negative out-of-bounds")
	}

	// Consume test
	s.Index = 0
	s.Consume("test", 5)
	if s.Consumed != "test" || s.Index != 5 {
		t.Errorf("Consume() failed to append accumulator or advance index")
	}
}

func TestParseTokenPushAndMerge(t *testing.T) {
	// 1. Test standard token append and linking via CurrentToken()
	s := NewParseState("test", nil)
	if tok := s.CurrentToken(); tok == nil || tok.Type != TokenTypeBos {
		t.Errorf("CurrentToken() failed to return initial BOS token")
	}

	tok1 := NewParseToken(TokenTypeSlash, "/", "\\/")
	s.PushToken(tok1)

	if len(s.Tokens) != 2 || s.CurrentToken() != tok1 || tok1.Prev != s.Tokens[0] {
		t.Errorf("PushToken() failed to establish double-linked Prev back-linkage")
	}
	if s.Output != "\\/" || s.Consumed != "/" {
		t.Errorf("PushToken() failed to append Output or Consumed")
	}
	if err := s.ValidateTokenLinks(); err != nil {
		t.Errorf("ValidateTokenLinks() failed on valid AST: %v", err)
	}

	// 2. Test plain text merging without OutputSet
	t1 := NewParseToken(TokenTypeText, "f", "")
	s.PushToken(t1)
	t2 := NewParseToken(TokenTypeText, "o", "")
	s.PushToken(t2)

	if len(s.Tokens) != 3 {
		t.Errorf("PushToken() should merge consecutive text tokens in place without appending new nodes; total len = %d", len(s.Tokens))
	}
	if s.CurrentToken().Value != "fo" || s.CurrentToken().Output != "" {
		t.Errorf("Text token merge failed: value=%q, output=%q", s.CurrentToken().Value, s.CurrentToken().Output)
	}

	// 3. Test text merging when incoming token has explicit OutputSet
	t3 := NewParseToken(TokenTypeText, "o", "escaped-o")
	s.PushToken(t3)
	if len(s.Tokens) != 3 || s.CurrentToken().Value != "foo" || s.CurrentToken().Output != "foescaped-o" || !s.CurrentToken().OutputSet {
		t.Errorf("Text merge with explicit OutputSet failed: value=%q, output=%q", s.CurrentToken().Value, s.CurrentToken().Output)
	}

	// 4. Test text merging when previous token already had explicit OutputSet and incoming does not
	t4 := NewParseToken(TokenTypeText, "x", "")
	s.PushToken(t4)
	if s.CurrentToken().Value != "foox" || s.CurrentToken().Output != "foescaped-ox" {
		t.Errorf("Text merge onto explicit OutputSet failed: value=%q, output=%q", s.CurrentToken().Value, s.CurrentToken().Output)
	}
	if err := s.ValidateTokenLinks(); err != nil {
		t.Errorf("ValidateTokenLinks() failed on merged text AST: %v", err)
	}

	// 5. Test push token on totally empty tokens slice
	sEmpty := &ParseState{Tokens: []*ParseToken{}}
	if sEmpty.CurrentToken() != nil {
		t.Errorf("CurrentToken() on empty slice must return nil")
	}
	tokSolo := NewParseToken(TokenTypeBos, "", "^")
	sEmpty.PushToken(tokSolo)
	if len(sEmpty.Tokens) != 1 || sEmpty.CurrentToken().Prev != nil {
		t.Errorf("PushToken() on empty slice failed")
	}

	// 6. Test Append directly with OutputSet vs non-OutputSet
	sApp := &ParseState{}
	sApp.Append(NewParseToken(TokenTypeStar, "*", ".*"))
	sApp.Append(NewParseToken(TokenTypeSlash, "/", ""))
	if sApp.Output != ".*/" || sApp.Consumed != "*/" {
		t.Errorf("Append() failed output and consumed concatenation: output=%q", sApp.Output)
	}
}

func TestInvariantValidationErrors(t *testing.T) {
	s := NewParseState("", nil)

	// Test ValidateStacks error conditions
	s.Stack = nil
	if err := s.ValidateStacks(); err == nil {
		t.Errorf("ValidateStacks() should error when Stack is nil")
	}
	s = NewParseState("", nil)
	s.BraceStack = nil
	if err := s.ValidateStacks(); err == nil {
		t.Errorf("ValidateStacks() should error when BraceStack is nil")
	}
	s = NewParseState("", nil)
	s.ExtglobStack = nil
	if err := s.ValidateStacks(); err == nil {
		t.Errorf("ValidateStacks() should error when ExtglobStack is nil")
	}

	// Test ValidateDepthCounters error conditions
	s = NewParseState("", nil)
	s.Brackets = -1
	if err := s.ValidateDepthCounters(); err == nil {
		t.Errorf("ValidateDepthCounters() should error when Brackets < 0")
	}
	s.Brackets = 0
	s.Braces = -1
	if err := s.ValidateDepthCounters(); err == nil {
		t.Errorf("ValidateDepthCounters() should error when Braces < 0")
	}
	s.Braces = 0
	s.Parens = -1
	if err := s.ValidateDepthCounters(); err == nil {
		t.Errorf("ValidateDepthCounters() should error when Parens < 0")
	}
	s.Parens = 0
	s.Quotes = -1
	if err := s.ValidateDepthCounters(); err == nil {
		t.Errorf("ValidateDepthCounters() should error when Quotes < 0")
	}

	// Test ValidateTokenLinks error conditions
	s = NewParseState("", nil)
	s.Tokens = nil
	if err := s.ValidateTokenLinks(); err == nil {
		t.Errorf("ValidateTokenLinks() should error when Tokens is empty")
	}

	s.Tokens = []*ParseToken{NewParseToken(TokenTypeSlash, "/", "")}
	if err := s.ValidateTokenLinks(); err == nil {
		t.Errorf("ValidateTokenLinks() should error when index 0 is not BOS")
	}

	// Test broken Prev link
	bos := NewParseToken(TokenTypeBos, "", "")
	brokenTok := NewParseToken(TokenTypeSlash, "/", "")
	brokenTok.Prev = nil // broken linkage
	s.Tokens = []*ParseToken{bos, brokenTok}
	if err := s.ValidateTokenLinks(); err == nil {
		t.Errorf("ValidateTokenLinks() should error on broken Prev linkage")
	}

	// Test adjacent un-merged text tokens
	t1 := NewParseToken(TokenTypeText, "a", "")
	t1.Prev = bos
	t2 := NewParseToken(TokenTypeText, "b", "")
	t2.Prev = t1
	s.Tokens = []*ParseToken{bos, t1, t2}
	if err := s.ValidateTokenLinks(); err == nil {
		t.Errorf("ValidateTokenLinks() should error when adjacent unmerged text tokens exist")
	}
}

func TestConstantInvariants(t *testing.T) {
	// Verify formatting constants compile and can be referenced
	if reflect.TypeOf(ErrMissingOpening).Kind() != reflect.String ||
		reflect.TypeOf(ErrMissingClosing).Kind() != reflect.String ||
		reflect.TypeOf(ErrInputExceeds).Kind() != reflect.String ||
		reflect.TypeOf(ErrExpectedString).Kind() != reflect.String {
		t.Errorf("Error template constant type mismatch")
	}
}
