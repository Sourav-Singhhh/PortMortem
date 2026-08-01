package picomatch

// Immutable parser configuration boundaries and operational limit thresholds.
const (
	// ParserMaxInputLength defines the maximum permitted pattern string byte length before syntax evaluation terminates.
	ParserMaxInputLength = 65536

	// ParserDefaultMaxExtglobRecursion defines the standard maximum nesting depth for repeated extended glob expressions.
	ParserDefaultMaxExtglobRecursion = 10
)

// Structural syntax context identifiers pushed onto the primary parser delimiter stack.
const (
	ParserContextBrackets ParserContext = "brackets"
	ParserContextBraces   ParserContext = "braces"
	ParserContextParens   ParserContext = "parens"
)

// Default configuration settings for new parser instances.
const (
	DefaultFastpaths = true
	DefaultPosix     = true
	DefaultPrepend   = ""
)

// Standard error formatting strings for unbalanced structural syntax delimiters.
const (
	ErrMissingOpening = "Missing opening: %q - use \"\\%s\" to match literal characters"
	ErrMissingClosing = "Missing closing: %q - use \"\\%s\" to match literal characters"
	ErrInputExceeds   = "Input length: %d, exceeds maximum allowed length: %d"
	ErrExpectedString = "Expected a string"
)
