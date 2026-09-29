package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/QubeUtils/repowalk/internal/aggregator"
	"github.com/QubeUtils/repowalk/internal/tree"
	"github.com/QubeUtils/repowalk/internal/walker"
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Generate context based on git diff",
	Long: `Creates an LLM context dump containing only the files 
that have changed according to the git state (untracked, unstaged, staged).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		maxSize, _ := cmd.Flags().GetInt64("max-size")
		templateStr, _ := cmd.Flags().GetString("template")
		persona, _ := cmd.Flags().GetString("persona")
		isJSON, _ := cmd.Flags().GetBool("json")
		isYAML, _ := cmd.Flags().GetBool("yaml")
		noRedact, _ := cmd.Flags().GetBool("no-redact")

		// Parse git status --porcelain to find all changed files reliably
		gitCmd := exec.Command("git", "status", "--porcelain")
		out, err := gitCmd.Output()
		if err != nil {
			return fmt.Errorf("failed to run git status: %w", err)
		}

		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		allowedPaths := make(map[string]bool)
		for _, line := range lines {
			if len(line) < 4 {
				continue
			}

			// Extract path logic handling renames
			parts := strings.Split(line[3:], " -> ")
			filePath := parts[len(parts)-1]
			filePath = strings.Trim(filePath, `"`)

			allowedPaths[filepath.ToSlash(filePath)] = true
		}

		if len(allowedPaths) == 0 {
			fmt.Println("No changed files found.")
			return nil
		}

		fs := walker.OSFileSystem{}
		diffMatcher := &DiffMatcher{allowed: allowedPaths}

		opts := walker.WalkOptions{
			RootPath:      ".",
			MaxSize:       maxSize,
			RedactSecrets: !noRedact,
		}

		w := walker.NewWalker(fs, diffMatcher, opts)
		nodes, err := w.Walk(context.Background())
		if err != nil {
			return fmt.Errorf("failed to walk repository: %w", err)
		}

		treeStr := tree.GenerateTree(nodes, 0)

		dumpOpts := aggregator.DumpOptions{
			TemplateString: templateStr,
			TreeString:     treeStr,
			Persona:        persona,
		}

		var output string
		if isJSON {
			output, err = aggregator.GenerateJSON(nodes, dumpOpts)
		} else if isYAML {
			output, err = aggregator.GenerateYAML(nodes, dumpOpts)
		} else {
			output, err = aggregator.GenerateMarkdown(nodes, dumpOpts)
		}

		if err != nil {
			return fmt.Errorf("failed to generate output: %w", err)
		}

		fmt.Println(output)
		return nil
	},
}

// DiffMatcher implements ignore.Matcher but acts as an allowlist
type DiffMatcher struct {
	allowed map[string]bool
}

func (m *DiffMatcher) MatchesPath(path string) bool {
	// If it's a directory, do not ignore if any allowed file is inside it
	pathWithSlash := path + "/"
	for allowedPath := range m.allowed {
		if allowedPath == path || strings.HasPrefix(allowedPath, pathWithSlash) {
			return false // Allow
		}
	}
	return true // Ignore
}

func init() {
	rootCmd.AddCommand(diffCmd)
}
