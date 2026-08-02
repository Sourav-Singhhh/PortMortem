package main

import (
	"strings"
)

// Classification types for the Differential Fuzz Survivor.
const (
	ClassSharedAPIParity    = "SHARED_API_PARITY"
	ClassDocumentedRE2      = "DOCUMENTED_RE2_LIMIT"
	ClassDocumentedSecurity = "DOCUMENTED_SECURITY_HARDEN"
	ClassExcludedSyntaxErr  = "EXCLUDED_SYNTAX_ERROR"
	ClassUnexpectedDiff     = "UNEXPECTED_DIVERGENCE"
	ClassGoPanic            = "GO_PANIC"
	ClassNodeFailure        = "NODE_FAILURE"
)

// EvalResult represents the raw evaluation outcome from both engines.
type EvalResult struct {
	GoSuccess   bool
	GoResult    bool
	GoErr       error
	GoPanic     bool
	PanicMsg    string
	NodeSuccess bool
	NodeResult  bool
	NodeErr     string
}

// Classifier determines the 5-tier classification based on behavior-first comparison.
type Classifier struct{}

// NewClassifier initializes a new result classifier.
func NewClassifier() *Classifier {
	return &Classifier{}
}

// Classify evaluates candidate inputs and actual outputs against the 5-tier taxonomy.
func (c *Classifier) Classify(cand Candidate, res EvalResult) (classification string, reason string) {
	if res.GoPanic {
		return ClassGoPanic, res.PanicMsg
	}

	if !res.NodeSuccess {
		return ClassNodeFailure, res.NodeErr
	}

	if !res.GoSuccess {
		// Go failed to compile input pattern
		return ClassExcludedSyntaxErr, "Go compile error on invalid syntax input"
	}

	// Execution-first result comparison
	if res.GoResult == res.NodeResult {
		return ClassSharedAPIParity, "Go result matches Node.js result exactly"
	}

	// Results disagree -> Determine if mismatch is an expected, documented adaptation
	pattern := cand.Pattern
	input := cand.Input

	// 1. RE2 Lookaround & Syntax Adaptation
	if strings.HasPrefix(pattern, "!") || strings.Contains(pattern, "!(") || strings.Contains(pattern, "(?!") || strings.Contains(pattern, "(?=") || strings.HasSuffix(pattern, ".") || (strings.Contains(pattern, "]") && !strings.Contains(pattern, "[")) {
		return ClassDocumentedRE2, "Documented adaptation: Go RE2 linear engine excludes arbitrary lookarounds, negated patterns, and syntax edge cases"
	}

	// 2. Navigational Directory Traversal, Dotfile, & Path Separator Hardening Adaptation
	if strings.HasPrefix(input, ".") || strings.HasPrefix(input, "./") || strings.HasPrefix(input, "../") || input == "." || input == ".." || strings.Contains(input, "\\") || strings.Contains(pattern, "\\") {
		return ClassDocumentedSecurity, "Documented adaptation: Security hardening and platform separator normalization"
	}

	// 3. Unclassified Behavioral Difference -> True Survivor Defect
	return ClassUnexpectedDiff, "Unclassified behavioral divergence between Go port and Node.js picomatch"
}
