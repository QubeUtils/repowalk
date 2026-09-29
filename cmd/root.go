package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/QubeUtils/repowalk/internal/aggregator"
	"github.com/QubeUtils/repowalk/internal/ignore"
	"github.com/QubeUtils/repowalk/internal/tree"
	"github.com/QubeUtils/repowalk/internal/walker"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "repowalk [path]",
	Short: "A high-performance repository context aggregator for LLMs",
	Long: `RepoWalk is a CLI tool that rapidly traverses your codebase, 
respecting .gitignore, and aggregates your files into a single context file 
optimized for Large Language Models.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		targetPath := "."
		if len(args) > 0 {
			targetPath = args[0]
		}

		maxSize, _ := cmd.Flags().GetInt64("max-size")
		microservices, _ := cmd.Flags().GetStringSlice("microservice")
		templateStr, _ := cmd.Flags().GetString("template")
		persona, _ := cmd.Flags().GetString("persona")
		isJSON, _ := cmd.Flags().GetBool("json")
		isYAML, _ := cmd.Flags().GetBool("yaml")
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
			Microservices: microservices,
			RedactSecrets: !noRedact,
			IgnoreExts:    ignoreExts,
		}

		w := walker.NewWalker(fs, multiMatcher, opts)

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

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().String("template", "", "Custom template for the LLM dump")
	rootCmd.PersistentFlags().Int64("max-size", 1048576, "Maximum file size to include (in bytes)")
	rootCmd.PersistentFlags().StringSliceP("microservice", "m", []string{}, "Selectively walk specific subdirectories (e.g., -m auth,payment)")
	rootCmd.PersistentFlags().Bool("json", false, "Export context as JSON")
	rootCmd.PersistentFlags().Bool("yaml", false, "Export context as YAML")
	rootCmd.PersistentFlags().Bool("no-redact", false, "Disable automatic secret redaction")
	rootCmd.PersistentFlags().String("persona", "", "Wrap context in a preset persona prompt (security, refactor, review)")
	rootCmd.PersistentFlags().StringSlice("ignore-exts", []string{}, "Override extensions to ignore (comma-separated). Overrides ~/.repowalk/config.json")
}
