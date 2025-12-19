package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/JNZader/ai-toolkit/goreview/internal/config"
)

// OllamaProvider implementa Provider para Ollama
type OllamaProvider struct {
	baseURL     string
	model       string
	timeout     time.Duration
	maxTokens   int
	temperature float64
	client      *http.Client
}

// ollamaRequest estructura de request para Ollama
type ollamaRequest struct {
	Model    string                 `json:"model"`
	Prompt   string                 `json:"prompt"`
	Stream   bool                   `json:"stream"`
	Options  map[string]interface{} `json:"options,omitempty"`
	Format   string                 `json:"format,omitempty"`
}

// ollamaResponse estructura de response de Ollama
type ollamaResponse struct {
	Model           string `json:"model"`
	Response        string `json:"response"`
	Done            bool   `json:"done"`
	Context         []int  `json:"context,omitempty"`
	TotalDuration   int64  `json:"total_duration,omitempty"`
	LoadDuration    int64  `json:"load_duration,omitempty"`
	PromptEvalCount int    `json:"prompt_eval_count,omitempty"`
	EvalCount       int    `json:"eval_count,omitempty"`
}

// NewOllamaProvider crea un nuevo provider de Ollama
func NewOllamaProvider(cfg *config.ProviderConfig) (*OllamaProvider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	model := cfg.Model
	if model == "" {
		model = "qwen2.5-coder:7b"
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	return &OllamaProvider{
		baseURL:     strings.TrimSuffix(baseURL, "/"),
		model:       model,
		timeout:     timeout,
		maxTokens:   cfg.MaxTokens,
		temperature: cfg.Temperature,
		client: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// Name retorna el nombre del provider
func (p *OllamaProvider) Name() string {
	return "ollama"
}

// Review realiza un code review
func (p *OllamaProvider) Review(ctx context.Context, request *ReviewRequest) (*ReviewResponse, error) {
	prompt := p.buildPrompt(request)

	reqBody := ollamaRequest{
		Model:  p.model,
		Prompt: prompt,
		Stream: false,
		Options: map[string]interface{}{
			"temperature": p.temperature,
			"num_predict": p.maxTokens,
		},
		Format: "json", // Forzamos JSON para parsear la respuesta
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/generate", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama API error (status %d): %s", resp.StatusCode, string(body))
	}

	var ollamaResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	duration := time.Since(start).Milliseconds()

	// Parsear la respuesta JSON del LLM
	var issues []Issue
	// Intentamos limpiar la respuesta por si el modelo incluyo markdown ```json ... ```
	cleanResp := strings.TrimSpace(ollamaResp.Response)
	cleanResp = strings.TrimPrefix(cleanResp, "```json")
	cleanResp = strings.TrimPrefix(cleanResp, "```")
	cleanResp = strings.TrimSuffix(cleanResp, "```")
	
	// Intentar parsear como lista []Issue
	if err := json.Unmarshal([]byte(cleanResp), &issues); err != nil {
		// Si falla, intentar como objeto simple Issue
		var singleIssue Issue
		if errSingle := json.Unmarshal([]byte(cleanResp), &singleIssue); errSingle == nil {
			issues = []Issue{singleIssue}
		} else {
			// Si falla, intentar como objeto wrapper { "issues": [...] }
			var wrapper struct {
				Issues []Issue `json:"issues"`
			}
			if errWrapper := json.Unmarshal([]byte(cleanResp), &wrapper); errWrapper == nil {
				issues = wrapper.Issues
			} else {
				// Si todo falla, retornar error original con la respuesta para debug
				return nil, fmt.Errorf("failed to parse LLM response as JSON: %w\nResponse was: %s", err, cleanResp)
			}
		}
	}

	return &ReviewResponse{
		Issues:         issues,
		Summary:        fmt.Sprintf("Review completed using %s. Found %d issues.", p.model, len(issues)),
		Score:          calculateScore(issues), // TODO: Implementar logica de score real
		TokensUsed:     ollamaResp.EvalCount + ollamaResp.PromptEvalCount,
		ProcessingTime: duration,
	}, nil
}

// HealthCheck verifica que el provider esta disponible
func (p *OllamaProvider) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/api/tags", nil)
	if err != nil {
		return err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("ollama health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama health check status: %d", resp.StatusCode)
	}

	return nil
}

// Close cierra conexiones (no-op para http client)
func (p *OllamaProvider) Close() error {
	return nil
}

// buildPrompt construye el prompt para el LLM
func (p *OllamaProvider) buildPrompt(req *ReviewRequest) string {
	var sb strings.Builder

	sb.WriteString("You are an expert code reviewer and security auditor. ")
	sb.WriteString("Your task is to analyze the following code changes and identify bugs, security vulnerabilities, and quality issues.\n\n")
	
	if req.Language != "" {
		sb.WriteString(fmt.Sprintf("Language: %s\n", req.Language))
	}
	if req.FilePath != "" {
		sb.WriteString(fmt.Sprintf("File: %s\n", req.FilePath))
	}
	if req.Context != "" {
		sb.WriteString(fmt.Sprintf("Context: %s\n", req.Context))
	}

	sb.WriteString("\nAnalyze the following git diff:\n\n")
	sb.WriteString(req.Diff)
	sb.WriteString("\n\n")

	sb.WriteString("Instructions:\n")
	sb.WriteString("1. Identify potential bugs, security risks, and performance issues.\n")
	sb.WriteString("2. Focus on the changed lines (additions/modifications) but use context to understand impact.\n")
	sb.WriteString("3. Ignore minor formatting issues unless they affect readability significantly.\n")
	sb.WriteString("4. Provide concrete suggestions for fixing each issue.\n")
	sb.WriteString("5. Return the result strictly as a JSON array of issue objects.\n")
	
	sb.WriteString("\nResponse Format (JSON Array):\n")
	sb.WriteString(`[
  {
    "id": "unique_id",
    "type": "bug|security|performance|style|best_practice",
    "severity": "info|warning|error|critical",
    "message": "Concise description of the issue",
    "suggestion": "How to fix it",
    "location": {
      "file": "filename",
      "start_line": 10
    },
    "fixed_code": "optional corrected code snippet"
  }
]`) // Corrected: Removed unnecessary backslashes before newlines and quotes within the JSON string literal.

	return sb.String()
}

func calculateScore(issues []Issue) int {
	score := 100
	for _, issue := range issues {
		switch issue.Severity {
		case SeverityCritical:
			score -= 20
		case SeverityError:
			score -= 10
		case SeverityWarning:
			score -= 5
		case SeverityInfo:
			score -= 1
		}
	}
	if score < 0 {
		return 0
	}
	return score
}
