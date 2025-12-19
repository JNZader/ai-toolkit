package config

import (
	"os"
	"path/filepath"
	"time"
)

// DefaultConfig retorna la configuracion por defecto
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	cacheDir := filepath.Join(homeDir, ".cache", "goreview")

	return &Config{
		Provider: ProviderConfig{
			Name:        "ollama",
			Model:       "qwen2.5-coder:7b",
			BaseURL:     "http://localhost:11434",
			Timeout:     5 * time.Minute,
			MaxTokens:   4096,
			Temperature: 0.1,
		},
		Git: GitConfig{
			RepoPath:         ".",
			BaseBranch:       "main",
			IncludeUntracked: false,
			IgnorePatterns: []string{
				"*.md",
				"*.txt",
				"*.json",
				"*.yaml",
				"*.yml",
				"go.sum",
				"package-lock.json",
				"pnpm-lock.yaml",
				"yarn.lock",
				"vendor/*",
				"node_modules/*",
				".git/*",
			},
		},
		Review: ReviewConfig{
			Mode:        "staged",
			MinSeverity: "warning",
			MaxIssues:   50,
		},
		Output: OutputConfig{
			Format:      "markdown",
			IncludeCode: true,
			Color:       true,
			Verbose:     false,
		},
		Cache: CacheConfig{
			Enabled:   true,
			Dir:       cacheDir,
			TTL:       24 * time.Hour,
			MaxSizeMB: 100,
		},
		Rules: RulesConfig{
			Preset: "standard",
		},
		LogLevel: "info",
	}
}
