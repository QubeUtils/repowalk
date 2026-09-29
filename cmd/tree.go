package cmd

import (
	"context"
	"fmt"

	"github.com/QubeUtils/repowalk/internal/ignore"
	"github.com/QubeUtils/repowalk/internal/tree"
	"github.com/QubeUtils/repowalk/internal/walker"
	"github.com/spf13/cobra"
)

var treeCmd = &cobra.Command{
	Use:   "tree [path]",
	Short: "Generate a visual tree representation of the repository",
	Long: `Prints a colored, visual tree structure of the repository, respecting .gitignore.
Useful for previewing what will be included in the context dump.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		targetPath := "."
		if len(args) > 0 {
			targetPath = args[0]
		}

		level, _ := cmd.Flags().GetInt("level")

		fs := walker.OSFileSystem{}

		var matchers []ignore.Matcher
		matchers = append(matchers, ignore.DefaultGlobalIgnores())
		if localIgnore, err := ignore.NewGitIgnoreMatcherFromFile(".gitignore"); err == nil {
			matchers = append(matchers, localIgnore)
		}
		multiMatcher := ignore.NewMultiMatcher(matchers...)

		opts := walker.WalkOptions{
			RootPath: targetPath,
			MaxSize:  0, // Set max size to 0 so we skip reading file contents (drastically speeds up tree command)
		}

		w := walker.NewWalker(fs, multiMatcher, opts)
		nodes, err := w.Walk(context.Background())
		if err != nil {
			return fmt.Errorf("failed to walk repository: %w", err)
		}

		treeStr := tree.GenerateColorTree(nodes, level)
		fmt.Print(treeStr)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(treeCmd)

	treeCmd.Flags().IntP("level", "L", 0, "Descend only level directories deep")
	treeCmd.Flags().Bool("compact", false, "Print a compact representation of the tree")
}
