package tree

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/QubeUtils/repowalk/internal/walker"
	"github.com/charmbracelet/lipgloss"
)

// node represents a generic node in our tree (file or inferred directory)
type node struct {
	Name     string
	IsDir    bool
	Children map[string]*node
	FileNode *walker.FileNode // nil for inferred directories
}

var (
	colorDir    = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true) // Light Blue
	colorFile   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))           // Light Gray
	colorBranch = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))           // Dark Gray
	colorRoot   = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true) // Pink
)

func buildTree(files []walker.FileNode) *node {
	if len(files) == 0 {
		return nil
	}

	// Build a trie to represent the directory structure
	root := &node{
		Name:     ".",
		IsDir:    true,
		Children: make(map[string]*node),
	}

	for i := range files {
		f := files[i]
		// Use forward slashes for consistent paths
		parts := strings.Split(filepath.ToSlash(f.Path), "/")
		current := root

		for j, part := range parts {
			if existing, exists := current.Children[part]; !exists {
				isDir := j < len(parts)-1 || (j == len(parts)-1 && f.IsDir)
				current.Children[part] = &node{
					Name:     part,
					IsDir:    isDir,
					Children: make(map[string]*node),
				}
			} else {
				if j < len(parts)-1 || f.IsDir {
					existing.IsDir = true
				}
			}
			current = current.Children[part]
		}
		current.FileNode = &f
	}
	return root
}

// GenerateTree takes a flat list of FileNodes and generates an ASCII tree string.
func GenerateTree(files []walker.FileNode, maxLevel int) string {
	root := buildTree(files)
	if root == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(".\n")

	printNode(&sb, root, "", 1, maxLevel, false)
	return sb.String()
}

// GenerateColorTree generates a colored ASCII tree string using lipgloss.
func GenerateColorTree(files []walker.FileNode, maxLevel int) string {
	root := buildTree(files)
	if root == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(colorRoot.Render(".") + "\n")

	printNode(&sb, root, "", 1, maxLevel, true)
	return sb.String()
}

func printNode(sb *strings.Builder, n *node, prefix string, currentLevel, maxLevel int, useColor bool) {
	if maxLevel > 0 && currentLevel > maxLevel {
		return
	}

	// Sort children alphabetically for consistent output
	var names []string
	for k := range n.Children {
		names = append(names, k)
	}
	sort.Strings(names)

	for i, name := range names {
		child := n.Children[name]
		isLast := i == len(names)-1

		connector := "├── "
		childPrefix := prefix + "│   "
		if isLast {
			connector = "└── "
			childPrefix = prefix + "    "
		}

		if useColor {
			sb.WriteString(colorBranch.Render(prefix+connector))
			if child.IsDir {
				sb.WriteString(colorDir.Render(name) + "\n")
			} else {
				sb.WriteString(colorFile.Render(name) + "\n")
			}
		} else {
			sb.WriteString(prefix + connector + name + "\n")
		}

		if child.IsDir {
			printNode(sb, child, childPrefix, currentLevel+1, maxLevel, useColor)
		}
	}
}
