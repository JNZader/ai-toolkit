package providers

import (
	"fmt"

	"github.com/JNZader/ai-toolkit/goreview/internal/config"
)

// ProviderType tipo de provider
type ProviderType string

const (
	ProviderOllama ProviderType = "ollama"
	ProviderClaude ProviderType = "claude"
	ProviderOpenAI ProviderType = "openai"
)

// Factory crea providers
type Factory struct {
	config *config.ProviderConfig
}

// NewFactory crea una nueva factory
func NewFactory(cfg *config.ProviderConfig) *Factory {
	return &Factory{config: cfg}
}

// Create crea un provider segun la configuracion
func (f *Factory) Create() (Provider, error) {
	switch ProviderType(f.config.Name) {
	case ProviderOllama:
		return NewOllamaProvider(f.config)
	// case ProviderClaude:
	// 	return NewClaudeProvider(f.config)
	// case ProviderOpenAI:
	// 	return NewOpenAIProvider(f.config)
	default:
		return nil, fmt.Errorf("unknown provider: %s", f.config.Name)
	}
}

// CreateByName crea un provider por nombre
func (f *Factory) CreateByName(name string, cfg *config.ProviderConfig) (Provider, error) {
	switch ProviderType(name) {
	case ProviderOllama:
		return NewOllamaProvider(cfg)
	// case ProviderClaude:
	// 	return NewClaudeProvider(cfg)
	// case ProviderOpenAI:
	// 	return NewOpenAIProvider(cfg)
	default:
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
}

// AvailableProviders retorna los providers disponibles
func AvailableProviders() []string {
	return []string{
		string(ProviderOllama),
		string(ProviderClaude),
		string(ProviderOpenAI),
	}
}
