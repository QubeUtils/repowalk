package aggregator_test

import (
	"strings"
	"testing"

	"github.com/QubeUtils/repowalk/internal/aggregator"
	"github.com/QubeUtils/repowalk/internal/walker"
)

func TestGenerateMarkdown(t *testing.T) {
	nodes := []walker.FileNode{
		{
			Path:    "main.go",
			Content: []byte("package main\n\nfunc main() {}"),
		},
		{
			Path:     "binary.png",
			IsBinary: true,
			Content:  []byte{0x00},
		},
	}

	opts := aggregator.DumpOptions{
		TreeString: ".\n└── main.go",
	}

	result, err := aggregator.GenerateMarkdown(nodes, opts)
	if err != nil {
		t.Fatalf("Failed to generate markdown: %v", err)
	}

	// Should contain the tree string
	if !strings.Contains(result, opts.TreeString) {
		t.Errorf("Result is missing tree string")
	}

	// Should contain main.go content
	if !strings.Contains(result, "package main") {
		t.Errorf("Result is missing main.go content")
	}

	// Should exclude binary.png entirely
	if strings.Contains(result, "binary.png") {
		t.Errorf("Result erroneously includes binary file")
	}
}

func TestGenerateMarkdown_CustomTemplate(t *testing.T) {
	nodes := []walker.FileNode{
		{
			Path:    "test.txt",
			Content: []byte("hello world"),
		},
	}

	opts := aggregator.DumpOptions{
		TemplateString: `CUSTOM: {{ range .Files }}{{ .Path }} -> {{ .ContentString }}{{ end }}`,
	}

	result, err := aggregator.GenerateMarkdown(nodes, opts)
	if err != nil {
		t.Fatalf("Failed to generate markdown: %v", err)
	}

	expected := "CUSTOM: test.txt -> hello world"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}
