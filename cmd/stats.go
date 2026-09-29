package cmd

import (
	"context"
	"fmt"

	"github.com/QubeUtils/repowalk/internal/ignore"
	"github.com/QubeUtils/repowalk/internal/walker"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats [path]",
	Short: "Show statistics about the repository context",
	Long: `Calculates and displays statistics such as total files, 
total size, and estimated LLM token count for the current repository state.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		targetPath := "."
		if len(args) > 0 {
			targetPath = args[0]
		}

		fs := walker.OSFileSystem{}

		var matchers []ignore.Matcher
		matchers = append(matchers, ignore.DefaultGlobalIgnores())
		if localIgnore, err := ignore.NewGitIgnoreMatcherFromFile(".gitignore"); err == nil {
			matchers = append(matchers, localIgnore)
		}
		multiMatcher := ignore.NewMultiMatcher(matchers...)

		ignoreExts, _ := cmd.Flags().GetStringSlice("ignore-exts")
		if len(ignoreExts) == 0 {
			cfg := LoadConfig()
			ignoreExts = cfg.IgnoreExts
		}

		opts := walker.WalkOptions{
			RootPath:   targetPath,
			MaxSize:    1048576, // 1MB default
			IgnoreExts: ignoreExts,
		}

		w := walker.NewWalker(fs, multiMatcher, opts)
		nodes, err := w.Walk(context.Background())
		if err != nil {
			return fmt.Errorf("failed to walk repository: %w", err)
		}

		var totalFiles int
		var totalSize int64
		var totalBinary int
		var totalTokens int

		for _, n := range nodes {
			totalFiles++
			totalSize += n.Size
			totalTokens += n.Tokens
			if n.IsBinary {
				totalBinary++
			}
		}

		titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
		statNameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
		statValueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
		borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

		fmt.Println(titleStyle.Render(fmt.Sprintf("Repository Statistics for: %s", targetPath)))
		fmt.Println(borderStyle.Render("----------------------------------------"))
		fmt.Printf("%-24s %s\n", statNameStyle.Render("Total Files Tracked:"), statValueStyle.Render(fmt.Sprintf("%d", totalFiles)))
		fmt.Printf("%-24s %s\n", statNameStyle.Render("Binary/Skipped Files:"), statValueStyle.Render(fmt.Sprintf("%d", totalBinary)))
		fmt.Printf("%-24s %s\n", statNameStyle.Render("Total Context Size:"), statValueStyle.Render(fmt.Sprintf("%d bytes", totalSize)))
		fmt.Printf("%-24s %s\n", statNameStyle.Render("Accurate LLM Tokens:"), statValueStyle.Render(fmt.Sprintf("%d tokens (cl100k_base)", totalTokens)))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statsCmd)
}
