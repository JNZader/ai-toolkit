package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Provider.Name != "ollama" {
		t.Errorf("expected provider ollama, got %s", cfg.Provider.Name)
	}

	if cfg.Output.Format != "markdown" {
		t.Errorf("expected format markdown, got %s", cfg.Output.Format)
	}

	if !cfg.Cache.Enabled {
		t.Error("expected cache to be enabled by default")
	}
}

func TestLoader_Load(t *testing.T) {
	loader := NewLoader()
	cfg, err := loader.Load()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg == nil {
		t.Fatal("config should not be nil")
	}
}

func TestLoader_LoadFromFile(t *testing.T) {
	// Crear archivo temporal de config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := []byte(`
provider:
  name: claude
  model: claude-3-opus
  api_key: test-key
output:
  format: json
`)

	if err := os.WriteFile(configPath, content, 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	loader := NewLoader()
	cfg, err := loader.LoadFromFile(configPath)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Provider.Name != "claude" {
		t.Errorf("expected provider claude, got %s", cfg.Provider.Name)
	}

	if cfg.Output.Format != "json" {
		t.Errorf("expected format json, got %s", cfg.Output.Format)
	}
}

func TestLoader_ValidateInvalidProvider(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := []byte(`
provider:
  name: invalid-provider
`)

	if err := os.WriteFile(configPath, content, 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	loader := NewLoader()
	_, err := loader.LoadFromFile(configPath)

	if err == nil {
		t.Error("expected error for invalid provider")
	}
}
