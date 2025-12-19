package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  "View, edit, and manage GoReview configuration",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := GetConfig()
		if cfg == nil {
			return fmt.Errorf("no configuration loaded")
		}

		data, err := yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}

		fmt.Println(string(data))
		return nil
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a new configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, _ := cmd.Flags().GetString("path")
		if path == "" {
			path = ".goreview.yaml"
		}

		// Verificar si ya existe
		if _, err := os.Stat(path); err == nil {
			force, _ := cmd.Flags().GetBool("force")
			if !force {
				return fmt.Errorf("config file already exists: %s (use --force to overwrite)", path)
			}
		}

		// Crear config por defecto
		cfg := GetConfig()
		data, err := yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}

		// Crear directorio si no existe
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		// Escribir archivo
		if err := os.WriteFile(path, data, 0644); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}

		fmt.Printf("Configuration file created: %s\n", path)
		return nil
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show configuration file path",
	Run: func(cmd *cobra.Command, args []string) {
		// Mostrar posibles ubicaciones
		fmt.Println("Configuration file locations (in order of precedence):")
		fmt.Println("  1. --config flag")
		fmt.Println("  2. ./config.yaml")
		fmt.Println("  3. ./.goreview.yaml")
		fmt.Println("  4. $HOME/.config/goreview/config.yaml")
		fmt.Println("  5. /etc/goreview/config.yaml")
		fmt.Println()
		fmt.Println("Environment variables:")
		fmt.Println("  GOREVIEW_PROVIDER_NAME     - AI provider")
		fmt.Println("  GOREVIEW_PROVIDER_MODEL    - Model name")
		fmt.Println("  GOREVIEW_PROVIDER_API_KEY  - API key")
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configPathCmd)

	configInitCmd.Flags().String("path", ".goreview.yaml", "path to create config file")
	configInitCmd.Flags().Bool("force", false, "overwrite existing config")
}
