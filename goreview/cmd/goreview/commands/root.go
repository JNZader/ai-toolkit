package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/JNZader/ai-toolkit/goreview/internal/config"
)

var (
	cfgFile string
	cfg     *config.Config
	verbose bool
)

// rootCmd representa el comando base
var rootCmd = &cobra.Command{
	Use:   "goreview",
	Short: "AI-powered code review tool",
	Long: `GoReview is a CLI tool that performs intelligent code reviews
using AI models. It analyzes your code changes and provides
actionable feedback on bugs, security issues, and best practices.

Examples:
  # Review staged changes
  goreview review --staged

  # Review a specific commit
  goreview review --commit abc123

  # Review changes between branches
  goreview review --base main

  # Review specific files
  goreview review file1.go file2.go`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initConfig(cmd)
	},
}

// Execute ejecuta el comando root
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Flags globales
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ./config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Provider flags
	rootCmd.PersistentFlags().String("provider", "", "AI provider (ollama, claude, openai)")
	rootCmd.PersistentFlags().String("model", "", "model to use")
	rootCmd.PersistentFlags().String("api-key", "", "API key for provider")

	// Output flags
	rootCmd.PersistentFlags().StringP("output", "o", "", "output file (default: stdout)")
	rootCmd.PersistentFlags().StringP("format", "f", "", "output format (markdown, json, sarif, html)")
}

func initConfig(cmd *cobra.Command) error {
	loader := config.NewLoader()

	// Si se especifico archivo de config, usarlo
	if cfgFile != "" {
		var err error
		cfg, err = loader.LoadFromFile(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	} else {
		var err error
		cfg, err = loader.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	// Override con flags de linea de comando
	// Usamos cmd.Flags() en lugar de rootCmd.Flags() para evitar ciclo,
	// o mejor aun, usamos las variables globales si estan vinculadas,
	// pero como usamos StringP sin variable global para provider/model, accedemos via cmd.Flag
	
	if provider, _ := cmd.Flags().GetString("provider"); provider != "" {
		cfg.Provider.Name = provider
	}
	if model, _ := cmd.Flags().GetString("model"); model != "" {
		cfg.Provider.Model = model
	}
	if apiKey, _ := cmd.Flags().GetString("api-key"); apiKey != "" {
		cfg.Provider.APIKey = apiKey
	}
	if format, _ := cmd.Flags().GetString("format"); format != "" {
		cfg.Output.Format = format
	}
	if output, _ := cmd.Flags().GetString("output"); output != "" {
		cfg.Output.File = output
	}
	if verbose {
		cfg.Output.Verbose = true
		cfg.LogLevel = "debug"
	}

	return nil
}

// GetConfig retorna la configuracion actual
func GetConfig() *config.Config {
	return cfg
}
