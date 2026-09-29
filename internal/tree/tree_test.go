package tree_test

import (
	"strings"
	"testing"

	"github.com/QubeUtils/repowalk/internal/tree"
	"github.com/QubeUtils/repowalk/internal/walker"
)

func TestGenerateTree(t *testing.T) {
	nodes := []walker.FileNode{
		{Path: "main.go"},
		{Path: "internal/walker/walker.go"},
		{Path: "internal/tree/tree.go"},
		{Path: "README.md"},
	}

	result := tree.GenerateTree(nodes, 0)

	expected := `.
├── README.md
├── internal
│   ├── tree
│   │   └── tree.go
│   └── walker
│       └── walker.go
└── main.go
`

	if strings.TrimSpace(result) != strings.TrimSpace(expected) {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

func TestGenerateTreeWithMaxLevel(t *testing.T) {
	nodes := []walker.FileNode{
		{Path: "main.go"},
		{Path: "internal/walker/walker.go"},
		{Path: "internal/tree/tree.go"},
	}

	// Max level 1 should only show internal and main.go
	result := tree.GenerateTree(nodes, 1)

	expected := `.
├── internal
└── main.go
`

	if strings.TrimSpace(result) != strings.TrimSpace(expected) {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}
