package config

import "time"

// Config representa la configuracion completa de GoReview
type Config struct {
	// Provider settings
	Provider ProviderConfig `yaml:"provider" mapstructure:"provider"`

	// Git settings
	Git GitConfig `yaml:"git" mapstructure:"git"`

	// Review settings
	Review ReviewConfig `yaml:"review" mapstructure:"review"`

	// Output settings
	Output OutputConfig `yaml:"output" mapstructure:"output"`

	// Cache settings
	Cache CacheConfig `yaml:"cache" mapstructure:"cache"`

	// Rules settings
	Rules RulesConfig `yaml:"rules" mapstructure:"rules"`

	// Logging
	LogLevel string `yaml:"log_level" mapstructure:"log_level"`
}

// ProviderConfig configura el provider de IA
type ProviderConfig struct {
	// Name del provider: ollama, claude, openai
	Name string `yaml:"name" mapstructure:"name"`

	// Model a usar
	Model string `yaml:"model" mapstructure:"model"`

	// API Key (para Claude/OpenAI)
	APIKey string `yaml:"api_key" mapstructure:"api_key"`

	// Base URL (para Ollama o custom endpoints)
	BaseURL string `yaml:"base_url" mapstructure:"base_url"`

	// Timeout para requests
	Timeout time.Duration `yaml:"timeout" mapstructure:"timeout"`

	// Max tokens de respuesta
	MaxTokens int `yaml:"max_tokens" mapstructure:"max_tokens"`

	// Temperature (0.0 - 1.0)
	Temperature float64 `yaml:"temperature" mapstructure:"temperature"`
}

// GitConfig configura la integracion con Git
type GitConfig struct {
	// Path al repositorio (default: directorio actual)
	RepoPath string `yaml:"repo_path" mapstructure:"repo_path"`

	// Branch base para comparacion
	BaseBranch string `yaml:"base_branch" mapstructure:"base_branch"`

	// Incluir archivos no trackeados
	IncludeUntracked bool `yaml:"include_untracked" mapstructure:"include_untracked"`

	// Patrones a ignorar
	IgnorePatterns []string `yaml:"ignore_patterns" mapstructure:"ignore_patterns"`
}

// ReviewConfig configura el proceso de review
type ReviewConfig struct {
	// Modo de review: staged, commit, branch, file
	Mode string `yaml:"mode" mapstructure:"mode"`

	// Commit especifico a revisar
	Commit string `yaml:"commit" mapstructure:"commit"`

	// Archivos especificos a revisar
	Files []string `yaml:"files" mapstructure:"files"`

	// Severidad minima a reportar: info, warning, error, critical
	MinSeverity string `yaml:"min_severity" mapstructure:"min_severity"`

	// Maximo de issues a reportar
	MaxIssues int `yaml:"max_issues" mapstructure:"max_issues"`

	// Contexto adicional para el review
	Context string `yaml:"context" mapstructure:"context"`
}

// OutputConfig configura la salida
type OutputConfig struct {
	// Formato: markdown, json, sarif, html
	Format string `yaml:"format" mapstructure:"format"`

	// Archivo de salida (vacio = stdout)
	File string `yaml:"file" mapstructure:"file"`

	// Incluir codigo en el reporte
	IncludeCode bool `yaml:"include_code" mapstructure:"include_code"`

	// Colorear output (solo para terminal)
	Color bool `yaml:"color" mapstructure:"color"`

	// Verbose output
	Verbose bool `yaml:"verbose" mapstructure:"verbose"`
}

// CacheConfig configura el cache
type CacheConfig struct {
	// Habilitar cache
	Enabled bool `yaml:"enabled" mapstructure:"enabled"`

	// Directorio de cache
	Dir string `yaml:"dir" mapstructure:"dir"`

	// TTL del cache
	TTL time.Duration `yaml:"ttl" mapstructure:"ttl"`

	// Maximo tamano del cache en MB
	MaxSizeMB int `yaml:"max_size_mb" mapstructure:"max_size_mb"`
}

// RulesConfig configura las reglas
type RulesConfig struct {
	// Preset a usar: minimal, standard, strict
	Preset string `yaml:"preset" mapstructure:"preset"`

	// Archivos de reglas adicionales
	Files []string `yaml:"files" mapstructure:"files"`

	// Reglas deshabilitadas
	Disabled []string `yaml:"disabled" mapstructure:"disabled"`

	// Reglas habilitadas (override de disabled)
	Enabled []string `yaml:"enabled" mapstructure:"enabled"`
}
