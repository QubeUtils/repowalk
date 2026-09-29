package cmd

import (
	"context"
	"fmt"

	"github.com/QubeUtils/repowalk/internal/ignore"
	"github.com/QubeUtils/repowalk/internal/tui"
	"github.com/QubeUtils/repowalk/internal/walker"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var uiCmd = &cobra.Command{
	Use:   "ui [path]",
	Short: "Launch the interactive Terminal User Interface",
	Long: `Launches an interactive TUI to visually browse the repository,
select specific files/folders, and generate an LLM context dump.

Keybindings:
  ↑/↓ or k/j: Navigate tree
  Space: Toggle file inclusion
  e: Edit Mode (preview and strip lines with Space)
  s: Save As... (interactive export prompt)
  Enter: Quick Save to default output
  /: Search and filter
  c: Copy to clipboard`,
	RunE: func(cmd *cobra.Command, args []string) error {
		targetPath := "."
		if len(args) > 0 {
			targetPath = args[0]
		}

		maxSize, err := cmd.Flags().GetInt64("max-size")
		if err != nil || maxSize == 0 {
			maxSize = 1048576 // 1MB default
		}
		templateStr, _ := cmd.Flags().GetString("template")
		persona, _ := cmd.Flags().GetString("persona")
		noRedact, _ := cmd.Flags().GetBool("no-redact")

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
			RootPath:      targetPath,
			MaxSize:       maxSize,
			Microservices: nil, // We'll let the user filter visually
			RedactSecrets: !noRedact,
			IgnoreExts:    ignoreExts,
		}

		w := walker.NewWalker(fs, multiMatcher, opts)
		nodes, err := w.Walk(context.Background())
		if err != nil {
			return fmt.Errorf("failed to walk repository: %w", err)
		}

		cfg := LoadConfig() // Need this to pass theme and cost if missing

		m := tui.NewModel(nodes, templateStr, persona, cfg.Theme, cfg.DefaultOutputPath, cfg.CostPer1MTokens)
		p := tea.NewProgram(m, tea.WithAltScreen())

		if _, err := p.Run(); err != nil {
			return fmt.Errorf("error running TUI: %w", err)
		}

		// The TUI will handle copying to clipboard or saving to file if needed.
		// We could also return the generated string from the model and print it here
		// if we wanted to pipe it, but the TUI takes over the screen.

		return nil
	},
}

func init() {
	rootCmd.AddCommand(uiCmd)
}
