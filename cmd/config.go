package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	IgnoreExts        []string `json:"ignore_extensions"`
	Theme             string   `json:"theme"`
	CostPer1MTokens   float64  `json:"cost_per_1m_tokens"`
	DefaultOutputPath string   `json:"default_output_path"`
}

var DefaultConfig = Config{
	IgnoreExts: []string{
		".jpg", ".jpeg", ".png", ".gif", ".mp4", ".avi", ".mov", ".mkv",
		".mp3", ".wav", ".ogg", ".flac", ".pdf", ".zip", ".tar", ".gz",
		".rar", ".7z", ".ttf", ".otf", ".woff", ".woff2", ".eot", ".ico",
		".svg", ".webp",
	},
	Theme:             "dracula",
	CostPer1MTokens:   3.0, // Defaults to Claude 3.5 Sonnet / GPT-4o input cost
	DefaultOutputPath: "repowalk_context.md",
}

func LoadConfig() Config {
	home, err := os.UserHomeDir()
	if err != nil {
		return DefaultConfig
	}

	configDir := filepath.Join(home, ".repowalk")
	configPath := filepath.Join(configDir, "config.json")

	// If the config doesn't exist, create it with the default values
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0755); err == nil {
			if data, err := json.MarshalIndent(DefaultConfig, "", "  "); err == nil {
				_ = os.WriteFile(configPath, data, 0644)
			}
		}
		return DefaultConfig
	}

	cfg := DefaultConfig

	// Read global config
	if data, err := os.ReadFile(configPath); err == nil {
		_ = json.Unmarshal(data, &cfg)
	}

	// Read local config (.repowalk.json) and merge it over
	if localData, err := os.ReadFile(".repowalk.json"); err == nil {
		_ = json.Unmarshal(localData, &cfg)
	}

	// Ensure there are some defaults if the JSON is somehow empty
	if len(cfg.IgnoreExts) == 0 {
		cfg.IgnoreExts = DefaultConfig.IgnoreExts
	}
	if cfg.Theme == "" {
		cfg.Theme = DefaultConfig.Theme
	}
	if cfg.CostPer1MTokens == 0 {
		cfg.CostPer1MTokens = DefaultConfig.CostPer1MTokens
	}
	if cfg.DefaultOutputPath == "" {
		cfg.DefaultOutputPath = DefaultConfig.DefaultOutputPath
	}

	return cfg
}
