package ignore

import (
	"path/filepath"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
)

// Matcher defines the interface for checking if a path should be ignored.
type Matcher interface {
	MatchesPath(path string) bool
}

// GitIgnoreMatcher implements Matcher using go-gitignore.
type GitIgnoreMatcher struct {
	ignorer *ignore.GitIgnore
}

// NewGitIgnoreMatcher creates a new matcher from a list of patterns.
func NewGitIgnoreMatcher(lines []string) *GitIgnoreMatcher {
	return &GitIgnoreMatcher{
		ignorer: ignore.CompileIgnoreLines(lines...),
	}
}

// NewGitIgnoreMatcherFromFile creates a new matcher from a file.
func NewGitIgnoreMatcherFromFile(path string) (*GitIgnoreMatcher, error) {
	ignorer, err := ignore.CompileIgnoreFile(path)
	if err != nil {
		return nil, err
	}
	return &GitIgnoreMatcher{
		ignorer: ignorer,
	}, nil
}

// MatchesPath returns true if the path matches the ignore rules.
func (g *GitIgnoreMatcher) MatchesPath(path string) bool {
	if g.ignorer == nil {
		return false
	}

	// Normalize path for consistent matching
	normalizedPath := filepath.ToSlash(path)
	if !strings.HasPrefix(normalizedPath, "/") {
		normalizedPath = "/" + normalizedPath
	}

	return g.ignorer.MatchesPath(normalizedPath)
}

// MultiMatcher combines multiple matchers. Useful for having a global ignore and a local ignore.
type MultiMatcher struct {
	matchers []Matcher
}

// NewMultiMatcher creates a new MultiMatcher.
func NewMultiMatcher(matchers ...Matcher) *MultiMatcher {
	return &MultiMatcher{
		matchers: matchers,
	}
}

// MatchesPath returns true if ANY of the matchers return true.
func (m *MultiMatcher) MatchesPath(path string) bool {
	for _, matcher := range m.matchers {
		if matcher.MatchesPath(path) {
			return true
		}
	}
	return false
}

// DefaultGlobalIgnores returns a matcher with common global ignores (e.g. .git)
func DefaultGlobalIgnores() Matcher {
	return NewGitIgnoreMatcher([]string{
		".git/",
		".svn/",
		".hg/",
		"node_modules/",
		".DS_Store",
	})
}
