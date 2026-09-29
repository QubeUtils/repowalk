package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a project-local .repowalk.json configuration file",
	Long: `Creates a .repowalk.json file in your current directory. 
This local configuration will override your global settings and can be committed to your 
VCS so your entire team shares the same repowalk configuration (e.g. ignored extensions).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := os.Stat(".repowalk.json"); err == nil {
			fmt.Println("❌ .repowalk.json already exists in this directory.")
			return nil
		}

		// Write a clean default configuration
		cfg := Config{
			IgnoreExts:        DefaultConfig.IgnoreExts,
			DefaultOutputPath: DefaultConfig.DefaultOutputPath,
			Theme:             DefaultConfig.Theme,
			CostPer1MTokens:   DefaultConfig.CostPer1MTokens,
		}

		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to generate JSON: %w", err)
		}

		if err := os.WriteFile(".repowalk.json", data, 0644); err != nil {
			return fmt.Errorf("failed to write .repowalk.json: %w", err)
		}

		fmt.Println("✅ Successfully created .repowalk.json!")
		fmt.Println("You can now customize ignore extensions, default paths, and UI themes for this project.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
