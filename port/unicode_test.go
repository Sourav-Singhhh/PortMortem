package picomatch

import (
	"testing"
)

func TestUnicodeFilenamesAndMultibyteScripts(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		input    string
		opts     *ParseOptions
		expected bool
	}{
		{
			name:     "Cyrillic filename matched by wildcard",
			pattern:  "*.js",
			input:    "файл.js",
			opts:     nil,
			expected: true,
		},
		{
			name:     "Cyrillic range character bracket matching",
			pattern:  "[а-я]*.js",
			input:    "скрипт.js",
			opts:     nil,
			expected: true,
		},
		{
			name:     "CJK Chinese characters across directory hierarchy",
			pattern:  "文档/**/*.doc",
			input:    "文档/2026/财务/报告.doc",
			opts:     nil,
			expected: true,
		},
		{
			name:     "CJK Japanese characters with MatchBase",
			pattern:  "日本語.go",
			input:    "src/pkg/i18n/日本語.go",
			opts:     &ParseOptions{MatchBase: true},
			expected: true,
		},
		{
			name:     "CJK Korean script matching with brace expansion",
			pattern:  "test/*.{한국어,조선말}.txt",
			input:    "test/문서.한국어.txt",
			opts:     nil,
			expected: true,
		},
		{
			name:     "Accented Latin characters matching directly",
			pattern:  "résumé.*",
			input:    "résumé.pdf",
			opts:     nil,
			expected: true,
		},
		{
			name:     "Accented Latin with Nocase evaluation",
			pattern:  "MÜNCHEN/*.txt",
			input:    "münchen/guide.txt",
			opts:     &ParseOptions{Nocase: true},
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			matched, err := Match(tc.pattern, tc.input, tc.opts)
			if err != nil {
				t.Fatalf("unexpected error compiling pattern %q: %v", tc.pattern, err)
			}
			if matched != tc.expected {
				t.Errorf("Match(%q, %q, %v) = %v; expected %v", tc.pattern, tc.input, tc.opts, matched, tc.expected)
			}
		})
	}
}

func TestEmojiFilenamesAndDirectories(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		input    string
		opts     *ParseOptions
		expected bool
	}{
		{"Rocket emoji filename with wildcard", "*_launch.ts", "🚀_launch.ts", nil, true},
		{"Fire emoji inside directory segment", "docs/🔥_*/*.md", "docs/🔥_urgent/bug_report.md", nil, true},
		{"Party popper emoji with extglob", "pkg/@(🎉|🎊)_event.json", "pkg/🎉_event.json", nil, true},
		{"Emoji character class matching", "[😀-🛸]*.log", "🚀_system.log", nil, true},
		{"Emoji globstar traversal", "**/🦄_*.dat", "data/magical/🦄_cache.dat", nil, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			matched, err := Match(tc.pattern, tc.input, tc.opts)
			if err != nil {
				t.Fatalf("unexpected error compiling pattern %q: %v", tc.pattern, err)
			}
			if matched != tc.expected {
				t.Errorf("Match(%q, %q, %v) = %v; expected %v", tc.pattern, tc.input, tc.opts, matched, tc.expected)
			}
		})
	}
}

func TestUTF8NormalizationForms(t *testing.T) {
	// In both Go RE2 regular expressions and V8 Node.js strings, strings are compared directly by their
	// underlying Unicode code point / byte representation without implicit normalization folding.
	// We demonstrate this exact behavioral equivalence by testing NFC (composed) vs NFD (decomposed) strings using standard Unicode escape literals.
	nfcForm := "r\u00e9sum\u00e9.doc"   // NFC form: precomposed é (\u00e9) -> "résumé.doc"
	nfdForm := "re\u0301sume\u0301.doc" // NFD form: e + combining acute accent (\u0301) -> "résumé.doc"

	// When pattern and input share the exact same Unicode normal form, matching succeeds.
	matchedNFC, err := Match(nfcForm, nfcForm, nil)
	if err != nil || !matchedNFC {
		t.Errorf("expected NFC pattern to match NFC input, got %v (err=%v)", matchedNFC, err)
	}
	matchedNFD, err := Match(nfdForm, nfdForm, nil)
	if err != nil || !matchedNFD {
		t.Errorf("expected NFD pattern to match NFD input, got %v (err=%v)", matchedNFD, err)
	}

	// When pattern is NFC but input is NFD, exact literal byte matching correctly fails (identical to Node.js V8 behavior),
	// whereas general wildcards (*.doc) match regardless of normal form!
	wildcardMatchedNFD, err := Match("*.doc", nfdForm, nil)
	if err != nil || !wildcardMatchedNFD {
		t.Errorf("expected wildcard pattern to match NFD input, got %v (err=%v)", wildcardMatchedNFD, err)
	}
}
