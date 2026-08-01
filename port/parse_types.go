package picomatch

// TokenType represents a classified structural syntax element within a glob pattern.
type TokenType string

// ParserContext represents a strongly typed syntactic category identifier pushed onto the delimiter stack.
type ParserContext string

// Valid token type enumeration constants representing structural lexical units.
const (
	TokenTypeBos        TokenType = "bos"
	TokenTypeAt         TokenType = "at"
	TokenTypeBrace      TokenType = "brace"
	TokenTypeBracket    TokenType = "bracket"
	TokenTypeComma      TokenType = "comma"
	TokenTypeDot        TokenType = "dot"
	TokenTypeDots       TokenType = "dots"
	TokenTypeGlobstar   TokenType = "globstar"
	TokenTypeMaybeSlash TokenType = "maybe_slash"
	TokenTypeNegate     TokenType = "negate"
	TokenTypeParen      TokenType = "paren"
	TokenTypePipe       TokenType = "pipe"
	TokenTypePlus       TokenType = "plus"
	TokenTypeQmark      TokenType = "qmark"
	TokenTypeSlash      TokenType = "slash"
	TokenTypeStar       TokenType = "star"
	TokenTypeText       TokenType = "text"
)

// ExpandRangeFunc defines the functional signature for custom numeric or alphabetical range expansion callbacks.
type ExpandRangeFunc func(left string, right string, opts *ParseOptions) string

// ParseOptions defines configuration toggles and parameter constraints that customize syntax recognition.
type ParseOptions struct {
	Windows             bool
	MaxLength           int
	Prepend             string
	Capture             bool
	Dot                 bool
	Bash                bool
	NoExt               bool
	NoExtglob           bool
	Fastpaths           bool
	Unescape            bool
	Contains            bool
	KeepQuotes          bool
	StrictBrackets      bool
	NoBracket           bool
	Posix               bool
	LiteralBrackets     bool
	NoBrace             bool
	ExpandRange         ExpandRangeFunc
	NoGlobstar          bool
	StrictSlashes       bool
	Regex               bool
	NoNegate            bool
	MaxExtglobRecursion int
	NoExtglobRecursion  bool
}

// ParseToken represents an atomic lexical unit and syntax tree node generated during parsing.
type ParseToken struct {
	Type        TokenType
	Value       string
	Output      string
	OutputSet   bool
	Prev        *ParseToken
	Suffix      string
	Posix       bool
	Dots        bool
	Comma       bool
	Star        bool
	Extglob     bool
	Conditions  int
	Inner       string
	Parens      int
	StartIndex  int
	TokensIndex int
	Open        string
	Close       string
	OutputIndex int
}

// BraceState maintains positional snapshots and structural flags for an active opening curly brace expression.
type BraceState struct {
	Token       *ParseToken
	Value       string
	Output      string
	OutputIndex int
	TokensIndex int
	Dots        bool
	Comma       bool
}

// ExtglobState tracks accumulating conditions, inner substrings, and structural markers for an active extended glob expression shell.
type ExtglobState struct {
	Token       *ParseToken
	Type        TokenType
	Value       string
	Open        string
	Close       string
	Conditions  int
	Inner       string
	Parens      int
	StartIndex  int
	TokensIndex int
	Output      string
}

// ParserStack is a LIFO data structure for tracking structural delimiter context category names during pattern evaluation.
type ParserStack struct {
	items []ParserContext
}

// BraceStack is a LIFO data structure for managing active opening BraceState entries during parsing.
type BraceStack struct {
	items []*BraceState
}

// ExtglobStack is a LIFO data structure for managing active opening ExtglobState entries during parsing.
type ExtglobStack struct {
	items []*ExtglobState
}

// ParseState encapsulates runtime string scanning trackers, active lexical stacks, and compiled token arrays.
type ParseState struct {
	Input          string
	Index          int
	Start          int
	Dot            bool
	Consumed       string
	Output         string
	Prefix         string
	Backtrack      bool
	Negated        bool
	NegatedExtglob bool
	Brackets       int
	Braces         int
	Parens         int
	Quotes         int
	Globstar       bool
	Tokens         []*ParseToken
	Stack          *ParserStack
	BraceStack     *BraceStack
	ExtglobStack   *ExtglobStack
	Opts           *ParseOptions
}
