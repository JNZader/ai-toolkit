package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Loader carga la configuracion desde multiples fuentes
type Loader struct {
	viper *viper.Viper
}

// NewLoader crea un nuevo loader
func NewLoader() *Loader {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	// Buscar config en multiples ubicaciones
	v.AddConfigPath(".")
	v.AddConfigPath("$HOME/.config/goreview")
	v.AddConfigPath("/etc/goreview")

	// Environment variables
	v.SetEnvPrefix("GOREVIEW")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	return &Loader{viper: v}
}

// Load carga la configuracion
func (l *Loader) Load() (*Config, error) {
	// Empezar con defaults
	cfg := DefaultConfig()

	// Intentar leer archivo de config
	if err := l.viper.ReadInConfig(); err != nil {
		// Es OK si no hay archivo de config
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config: %w", err)
		}
	}

	// Unmarshal a struct
	if err := l.viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}

	// Validar
	if err := l.validate(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

// LoadFromFile carga config desde un archivo especifico
func (l *Loader) LoadFromFile(path string) (*Config, error) {
	l.viper.SetConfigFile(path)
	return l.Load()
}

// validate valida la configuracion
func (l *Loader) validate(cfg *Config) error {
	// Validar provider
	validProviders := map[string]bool{
		"ollama": true,
		"claude": true,
		"openai": true,
	}
	if !validProviders[cfg.Provider.Name] {
		return fmt.Errorf("invalid provider: %s", cfg.Provider.Name)
	}

	// Validar que API key este presente para providers que lo requieren
	if cfg.Provider.Name != "ollama" && cfg.Provider.APIKey == "" {
		return fmt.Errorf("api_key required for provider %s", cfg.Provider.Name)
	}

	// Validar formato de output
	validFormats := map[string]bool{
		"markdown": true,
		"json":     true,
		"sarif":    true,
		"html":     true,
	}
	if !validFormats[cfg.Output.Format] {
		return fmt.Errorf("invalid output format: %s", cfg.Output.Format)
	}

	// Validar severidad
	validSeverities := map[string]bool{
		"info":     true,
		"warning":  true,
		"error":    true,
		"critical": true,
	}
	if !validSeverities[cfg.Review.MinSeverity] {
		return fmt.Errorf("invalid min_severity: %s", cfg.Review.MinSeverity)
	}

	return nil
}

// Set establece un valor de configuracion
func (l *Loader) Set(key string, value interface{}) {
	l.viper.Set(key, value)
}

// GetConfigFile retorna el archivo de config usado
func (l *Loader) GetConfigFile() string {
	return l.viper.ConfigFileUsed()
}

// WriteConfig escribe la configuracion actual a archivo
func (l *Loader) WriteConfig(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	l.viper.SetConfigFile(path)
	return l.viper.WriteConfig()
}
