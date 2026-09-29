package walker_test

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QubeUtils/repowalk/internal/ignore"
	"github.com/QubeUtils/repowalk/internal/walker"
)

// MockFileInfo implements fs.FileInfo
type MockFileInfo struct {
	name  string
	size  int64
	isDir bool
}

func (m *MockFileInfo) Name() string       { return m.name }
func (m *MockFileInfo) Size() int64        { return m.size }
func (m *MockFileInfo) Mode() fs.FileMode  { return 0 }
func (m *MockFileInfo) ModTime() time.Time { return time.Time{} }
func (m *MockFileInfo) IsDir() bool        { return m.isDir }
func (m *MockFileInfo) Sys() interface{}   { return nil }

// MockDirEntry implements fs.DirEntry
type MockDirEntry struct {
	info *MockFileInfo
}

func (m *MockDirEntry) Name() string               { return m.info.name }
func (m *MockDirEntry) IsDir() bool                { return m.info.isDir }
func (m *MockDirEntry) Type() fs.FileMode          { return 0 }
func (m *MockDirEntry) Info() (fs.FileInfo, error) { return m.info, nil }

// MockFileSystem implements FileSystem for testing.
type MockFileSystem struct {
	Files map[string][]byte
}

func (m *MockFileSystem) WalkDir(root string, fn fs.WalkDirFunc) error {
	fn(root, &MockDirEntry{&MockFileInfo{name: filepath.Base(root), isDir: true}}, nil)

	for path, content := range m.Files {
		// Simulate files inside root
		fullPath := filepath.Join(root, path)

		// Convert mock path to OS specific separator just in case
		fullPath = filepath.Clean(fullPath)

		err := fn(fullPath, &MockDirEntry{&MockFileInfo{
			name:  filepath.Base(fullPath),
			size:  int64(len(content)),
			isDir: false,
		}}, nil)
		if err != nil && err != filepath.SkipDir {
			return err
		}
	}
	return nil
}

func (m *MockFileSystem) ReadFile(name string) ([]byte, error) {
	// Strip root to get mock path
	parts := strings.Split(name, string(filepath.Separator))
	rel := filepath.Join(parts[1:]...)
	// convert back to forward slash for map lookup
	rel = filepath.ToSlash(rel)

	content, ok := m.Files[rel]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return content, nil
}

func (m *MockFileSystem) Stat(name string) (fs.FileInfo, error) {
	parts := strings.Split(name, string(filepath.Separator))
	rel := filepath.Join(parts[1:]...)
	rel = filepath.ToSlash(rel)

	content, ok := m.Files[rel]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return &MockFileInfo{name: filepath.Base(name), size: int64(len(content))}, nil
}

func TestWalker_Walk(t *testing.T) {
	mockFS := &MockFileSystem{
		Files: map[string][]byte{
			"main.go":           []byte("package main"),
			"ignored.log":       []byte("some log data"),
			"binary.bin":        {0, 1, 2, 3},
			"large.txt":         []byte(strings.Repeat("a", 2048)),
			"auth/service.go":   []byte("package auth"),
			"billing/charge.go": []byte("package billing"),
		},
	}

	matcher := ignore.NewGitIgnoreMatcher([]string{"*.log"})

	opts := walker.WalkOptions{
		RootPath:   "root",
		MaxSize:    1024,
		MaxWorkers: 2,
	}

	w := walker.NewWalker(mockFS, matcher, opts)
	nodes, err := w.Walk(context.Background())
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	var validNames []string
	for _, n := range nodes {
		if !n.IsDir && !n.IsIgnored {
			validNames = append(validNames, n.Name)
		}
		if n.Name == "ignored.log" && !n.IsIgnored {
			t.Errorf("Walk should have marked ignored.log as IsIgnored=true")
		}
		if n.Name == "binary.bin" && !n.IsBinary {
			t.Errorf("Expected binary.bin to be identified as binary")
		}
		if n.Name == "large.txt" && !n.IsBinary {
			t.Errorf("Expected large.txt to be marked as binary due to size limit")
		}
		if n.Name == "main.go" && n.IsBinary {
			t.Errorf("Expected main.go to be text")
		}
	}

	if len(validNames) != 5 {
		t.Errorf("Expected 5 valid files, got %d: %v", len(validNames), validNames)
	}
}

func TestWalker_WalkMicroservice(t *testing.T) {
	mockFS := &MockFileSystem{
		Files: map[string][]byte{
			"main.go":           []byte("package main"),
			"auth/service.go":   []byte("package auth"),
			"billing/charge.go": []byte("package billing"),
		},
	}

	opts := walker.WalkOptions{
		RootPath:      "root",
		MaxSize:       1024,
		Microservices: []string{"auth"},
		MaxWorkers:    2,
	}

	w := walker.NewWalker(mockFS, nil, opts)
	nodes, err := w.Walk(context.Background())
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	var validNames []string
	for _, n := range nodes {
		if !n.IsDir && !n.IsIgnored {
			validNames = append(validNames, n.Name)
		}
	}
	
	if len(validNames) != 1 || validNames[0] != "service.go" {
		t.Errorf("Expected only service.go, got: %v", validNames)
	}
}
