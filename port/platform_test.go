package picomatch

import (
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWindowsPathHandling(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		input    string
		opts     *ParseOptions
		expected bool
	}{
		{
			name:     "C drive letter simple wildcard with Windows=true",
			pattern:  "C:/Users/admin/*.doc",
			input:    "C:\\Users\\admin\\report.doc",
			opts:     &ParseOptions{Windows: true},
			expected: true,
		},
		{
			name:     "D drive letter globstar with Windows=true",
			pattern:  "D:/Projects/**/*.js",
			input:    "D:\\Projects\\src\\lib\\index.js",
			opts:     &ParseOptions{Windows: true},
			expected: true,
		},
		{
			name:     "UNC path share network root with Windows=true",
			pattern:  "//server/share/logs/*.log",
			input:    "\\\\server\\share\\logs\\app.log",
			opts:     &ParseOptions{Windows: true},
			expected: true,
		},
		{
			name:     "UNC path globstar across share nodes",
			pattern:  "//nas-storage/**/backup-*.dat",
			input:    "\\\\nas-storage\\backups\\2026\\08\\backup-system.dat",
			opts:     &ParseOptions{Windows: true},
			expected: true,
		},
		{
			name:     "Mixed slash and backslash inputs",
			pattern:  "foo/bar/baz/*.go",
			input:    "foo\\bar/baz\\main.go",
			opts:     &ParseOptions{Windows: true},
			expected: true,
		},
		{
			name:     "Windows=false preserves backslash as non-slash literal or escape",
			pattern:  "foo/bar/*.go",
			input:    "foo\\bar\\main.go",
			opts:     &ParseOptions{Windows: false},
			expected: false,
		},
		{
			name:     "Windows path MatchBase extraction",
			pattern:  "*.js",
			input:    "C:\\Users\\dev\\workspace\\project\\index.js",
			opts:     &ParseOptions{Windows: true, MatchBase: true},
			expected: true,
		},
		{
			name:     "Windows path Basename option",
			pattern:  "config.json",
			input:    "D:\\srv\\app\\config.json",
			opts:     &ParseOptions{Windows: true, Basename: true},
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

func TestLinuxAndMacPOSIXPathHandling(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		input    string
		opts     *ParseOptions
		expected bool
	}{
		{
			name:     "Linux root path direct wildcard",
			pattern:  "/var/log/*",
			input:    "/var/log/syslog",
			opts:     &ParseOptions{Posix: true},
			expected: true,
		},
		{
			name:     "Linux deeply nested hierarchy globstar",
			pattern:  "/usr/**/bin/*",
			input:    "/usr/local/opt/python/bin/python3",
			opts:     &ParseOptions{Posix: true},
			expected: true,
		},
		{
			name:     "macOS Library root evaluation",
			pattern:  "/Users/*/Library/**/*.plist",
			input:    "/Users/devuser/Library/Preferences/com.apple.Terminal.plist",
			opts:     &ParseOptions{Posix: true},
			expected: true,
		},
		{
			name:     "macOS POSIX path Nocase evaluation",
			pattern:  "/volumes/external/**/*.mov",
			input:    "/Volumes/External/Media/Video/holiday.MOV",
			opts:     &ParseOptions{Posix: true, Nocase: true},
			expected: true,
		},
		{
			name:     "POSIX class matching under Linux paths",
			pattern:  "bin/[[:alpha:]]+",
			input:    "bin/bash",
			opts:     &ParseOptions{Posix: true},
			expected: true,
		},
		{
			name:     "Brace expansion under POSIX directory tree",
			pattern:  "/home/user/*.{sh,bash,zsh}",
			input:    "/home/user/deploy.zsh",
			opts:     &ParseOptions{Posix: true},
			expected: true,
		},
		{
			name:     "Extglob matching in POSIX filesystem path",
			pattern:  "/var/www/@(prod|staging)/*.html",
			input:    "/var/www/staging/index.html",
			opts:     &ParseOptions{Posix: true},
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

func TestPathCleanAndFilepathSeparator(t *testing.T) {
	// Verify that path.Clean and filepath.Clean interact predictably with Port Mortem matching engine
	rawInputs := []string{
		"foo//bar/../bar/app.js",
		"a/./b/./c/test.go",
		"/usr/local//bin/node",
	}

	patterns := []string{
		"foo/bar/*.js",
		"a/b/c/*.go",
		"/usr/local/bin/*",
	}

	for i, raw := range rawInputs {
		pat := patterns[i]

		// 1. Using path.Clean (POSIX style slashes guaranteed)
		cleanedPath := path.Clean(raw)
		matched, err := Match(pat, cleanedPath, nil)
		if err != nil || !matched {
			t.Errorf("expected pattern %q to match path.Clean(%q)=%q, got matched=%v, err=%v", pat, raw, cleanedPath, matched, err)
		}

		// 2. Using filepath.Clean (OS-native separators: backslashes on Windows, slashes on POSIX)
		cleanedFilepath := filepath.Clean(raw)
		opts := &ParseOptions{}
		if runtime.GOOS == "windows" || strings.Contains(cleanedFilepath, "\\") {
			opts.Windows = true
		}
		matchedOS, errOS := Match(pat, cleanedFilepath, opts)
		if errOS != nil || !matchedOS {
			t.Errorf("expected pattern %q to match filepath.Clean(%q)=%q with opts=%+v, got matched=%v, err=%v", pat, raw, cleanedFilepath, opts, matchedOS, errOS)
		}
	}
}

func TestCrossPlatformOptionMatrix(t *testing.T) {
	matrix := []struct {
		name     string
		pattern  string
		input    string
		opts     *ParseOptions
		expected bool
	}{
		{"MatchBase on Windows path", "*.ts", "C:\\dev\\src\\components\\Button.ts", &ParseOptions{Windows: true, MatchBase: true}, true},
		{"Basename on mixed delimiters", "app.min.js", "src/build\\assets/app.min.js", &ParseOptions{Windows: true, Basename: true}, true},
		{"Dot file option on Windows hidden path", "**/.*", "C:\\Users\\guest\\.bash_history", &ParseOptions{Windows: true, Dot: true}, true},
		{"Nocase option across drive letters", "c:/users/**/*.txt", "C:\\USERS\\PUBLIC\\DOCS\\NOTES.TXT", &ParseOptions{Windows: true, Nocase: true}, true},
		{"Ignore option rejecting specific folder", "**/*.js", "foo/build/app.js", &ParseOptions{Ignore: []string{"**/build/**"}}, false},
		{"Ignore option passing valid folder", "**/*.js", "foo/src/app.js", &ParseOptions{Ignore: []string{"**/build/**"}}, true},
		{"Extglob combined with Nocase", "*/*.@(JPG|PNG|GIF)", "photos/VACATION.jpg", &ParseOptions{Nocase: true}, true},
	}

	for _, tc := range matrix {
		t.Run(tc.name, func(t *testing.T) {
			res, err := Match(tc.pattern, tc.input, tc.opts)
			if err != nil {
				t.Fatalf("unexpected compile error for %q: %v", tc.pattern, err)
			}
			if res != tc.expected {
				t.Errorf("Match(%q, %q) with opts %+v = %v; expected %v", tc.pattern, tc.input, tc.opts, res, tc.expected)
			}
		})
	}
}
