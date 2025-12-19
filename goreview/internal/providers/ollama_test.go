package providers

import (
	"strings"
	"testing"
	"time"

	"github.com/JNZader/ai-toolkit/goreview/internal/config"
)

func TestNewOllamaProvider(t *testing.T) {
	cfg := &config.ProviderConfig{
		BaseURL: "http://test-url",
		Model:   "test-model",
		Timeout: 10 * time.Second,
	}

	provider, err := NewOllamaProvider(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if provider.baseURL != "http://test-url" {
		t.Errorf("expected url http://test-url, got %s", provider.baseURL)
	}

	if provider.model != "test-model" {
		t.Errorf("expected model test-model, got %s", provider.model)
	}
}

func TestOllamaProvider_Name(t *testing.T) {
	p := &OllamaProvider{}
	if p.Name() != "ollama" {
		t.Errorf("expected name ollama, got %s", p.Name())
	}
}

func TestBuildPrompt(t *testing.T) {
	p := &OllamaProvider{}
	req := &ReviewRequest{
		Diff:     "diff content",
		Language: "go",
		FilePath: "main.go",
	}

	prompt := p.buildPrompt(req)

	if !strings.Contains(prompt, "diff content") {
		t.Error("prompt should contain diff")
	}
	if !strings.Contains(prompt, "Language: go") {
		t.Error("prompt should contain language")
	}
	if !strings.Contains(prompt, "File: main.go") {
		t.Error("prompt should contain filepath")
	}
	if !strings.Contains(prompt, "JSON Array") {
		t.Error("prompt should request JSON format")
	}
}

func TestFactory_Create(t *testing.T) {
	cfg := &config.ProviderConfig{
		Name: "ollama",
	}
	factory := NewFactory(cfg)

	provider, err := factory.Create()
	if err != nil {
		t.Fatalf("factory failed to create ollama provider: %v", err)
	}

	if provider.Name() != "ollama" {
		t.Errorf("expected ollama provider, got %s", provider.Name())
	}
}
