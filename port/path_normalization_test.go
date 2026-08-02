package picomatch

import (
	"path"
	"strings"
	"testing"
)

func TestPathSeparatorNormalization(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		input    string
		opts     *ParseOptions
		expected bool
	}{
		{
			name:     "Trailing slash on directory wildcard",
			pattern:  "foo/*/",
			input:    "foo/bar/",
			opts:     nil,
			expected: true,
		},
		{
			name:     "Trailing slash matched by default on directory target",
			pattern:  "foo/*",
			input:    "foo/bar/",
			opts:     nil,
			expected: true,
		},
		{
			name:     "Trailing slash rejection when StrictSlashes=true",
			pattern:  "foo/*",
			input:    "foo/bar/",
			opts:     &ParseOptions{StrictSlashes: true},
			expected: false,
		},
		{
			name:     "Mixed slash normalization under Posix option",
			pattern:  "foo/bar/*.js",
			input:    "foo\\bar/main.js",
			opts:     &ParseOptions{Posix: true},
			expected: true,
		},
		{
			name:     "Mixed slash normalization under Windows option",
			pattern:  "foo/bar/*.js",
			input:    "foo\\bar\\main.js",
			opts:     &ParseOptions{Windows: true},
			expected: true,
		},
		{
			name:     "Cleaned redundant slashes before evaluation",
			pattern:  "foo/bar/*.go",
			input:    path.Clean("foo//bar///app.go"),
			opts:     nil,
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

func TestSpecialDirectoryAndHiddenFileExclusion(t *testing.T) {
	// Port Mortem implements an intentional, verified architectural divergence from Node.js picomatch:
	// Special navigational directories "." and ".." are strictly forbidden from matching wildcards (*, **, .*)
	// unless expressly matched by explicit literal segments in the pattern. This guarantees secure filesystem traversal.
	tests := []struct {
		name     string
		pattern  string
		input    string
		opts     *ParseOptions
		expected bool
	}{
		{"Wildcard dot asterisk against current dir point (.)", ".*", ".", nil, false},
		{"Wildcard dot asterisk against parent dir (..)", ".*", "..", nil, false},
		{"Wildcard dot asterisk against legitimate hidden file", ".*", ".gitignore", nil, true},
		{"Globstar against navigational trailing point (foo/.)", "**/*", "foo/.", nil, false},
		{"Globstar against navigational trailing parent (foo/..)", "**/*", "foo/..", nil, false},
		{"Explicit navigational dot literal pattern matches point", "foo/./*", "foo/./bar.js", nil, true},
		{"Explicit navigational parent literal pattern matches parent", "foo/../*", "foo/../bar.js", nil, true},
		{"Standard wildcard ignores hidden file by default", "*.js", ".hidden.js", nil, false},
		{"Standard wildcard matches hidden file with Dot=true", "*.js", ".hidden.js", &ParseOptions{Dot: true}, true},
		{"Globstar ignores hidden folder without Dot option", "**/*.js", "foo/.config/app.js", nil, false},
		{"Globstar matches hidden folder with Dot option", "**/*.js", "foo/.config/app.js", &ParseOptions{Dot: true}, true},
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

func TestAbsoluteAndRelativePathBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		input    string
		opts     *ParseOptions
		expected bool
	}{
		{"Absolute POSIX pattern matching absolute input", "/usr/local/bin/*", "/usr/local/bin/go", nil, true},
		{"Absolute POSIX pattern rejecting relative input", "/usr/local/bin/*", "usr/local/bin/go", nil, false},
		{"Relative dot slash pattern matching clean relative input", "./src/*.js", "src/app.js", nil, true},
		{"Relative dot slash pattern matching leading dot-slash input via Format option", "./src/*.js", "./src/app.js", &ParseOptions{Format: func(s string) string { return strings.TrimPrefix(s, "./") }}, true},
		{"Relative dot parent pattern matching parent input", "../build/*.js", "../build/output.js", nil, true},
		{"MatchBase extracting filename from absolute POSIX path", "*.log", "/var/log/syslog.log", &ParseOptions{MatchBase: true}, true},
		{"MatchBase extracting filename from relative nested path", "*.log", "./logs/daemon.log", &ParseOptions{MatchBase: true}, true},
		{"Basename evaluation on deeply nested absolute path", "config.yaml", "/opt/service/conf/config.yaml", &ParseOptions{Basename: true}, true},
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
