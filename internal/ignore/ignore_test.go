package ignore_test

import (
	"testing"

	"github.com/QubeUtils/repowalk/internal/ignore"
)

func TestGitIgnoreMatcher_MatchesPath(t *testing.T) {
	patterns := []string{
		"*.log",
		"build/",
		"secret.txt",
		"!important.log",
	}

	matcher := ignore.NewGitIgnoreMatcher(patterns)

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"Matches extension", "test.log", true},
		{"Matches directory", "build/output.bin", true},
		{"Matches exact file", "secret.txt", true},
		{"Matches nested exact file", "folder/secret.txt", true},
		{"Matches negation", "important.log", false},
		{"Does not match unrelated", "main.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.MatchesPath(tt.path)
			if result != tt.expected {
				t.Errorf("MatchesPath(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestMultiMatcher_MatchesPath(t *testing.T) {
	matcher1 := ignore.NewGitIgnoreMatcher([]string{"*.tmp"})
	matcher2 := ignore.NewGitIgnoreMatcher([]string{"*.bak"})

	multi := ignore.NewMultiMatcher(matcher1, matcher2)

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"Matches first", "file.tmp", true},
		{"Matches second", "file.bak", true},
		{"Matches neither", "file.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := multi.MatchesPath(tt.path)
			if result != tt.expected {
				t.Errorf("MatchesPath(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestDefaultGlobalIgnores(t *testing.T) {
	matcher := ignore.DefaultGlobalIgnores()

	if !matcher.MatchesPath(".git/config") {
		t.Errorf("Expected DefaultGlobalIgnores to match .git/config")
	}
	if !matcher.MatchesPath("node_modules/express/index.js") {
		t.Errorf("Expected DefaultGlobalIgnores to match node_modules")
	}
	if matcher.MatchesPath("main.go") {
		t.Errorf("Expected DefaultGlobalIgnores to not match main.go")
	}
}
