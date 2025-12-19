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

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	tokens   chan struct{}
	interval time.Duration
	stopCh   chan struct{}
	started  bool
}

// NewRateLimiter creates a rate limiter with the specified requests per second
func NewRateLimiter(rps int) *RateLimiter {
	if rps <= 0 {
		return nil // No rate limiting
	}

	rl := &RateLimiter{
		tokens:   make(chan struct{}, rps),
		interval: time.Second / time.Duration(rps),
		stopCh:   make(chan struct{}),
	}

	// Pre-fill tokens
	for i := 0; i < rps; i++ {
		rl.tokens <- struct{}{}
	}

	// Start refill goroutine
	go rl.refill()
	rl.started = true

	return rl
}

// refill periodically adds tokens back to the bucket
func (rl *RateLimiter) refill() {
	ticker := time.NewTicker(rl.interval)
	defer ticker.Stop()

	for {
		select {
		case <-rl.stopCh:
			return
		case <-ticker.C:
			select {
			case rl.tokens <- struct{}{}:
			default:
				// Bucket full, discard token
			}
		}
	}
}

// Wait blocks until a token is available or context is cancelled
func (rl *RateLimiter) Wait(ctx context.Context) error {
	if rl == nil {
		return nil // No rate limiting
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-rl.tokens:
		return nil
	}
}

// Close stops the rate limiter
func (rl *RateLimiter) Close() {
	if rl != nil && rl.started {
		close(rl.stopCh)
	}
}

// OllamaProvider implementa Provider para Ollama
type OllamaProvider struct {
	baseURL     string
	model       string
	timeout     time.Duration
	maxTokens   int
	temperature float64
	client      *http.Client
	rateLimiter *RateLimiter
}

// ollamaRequest estructura de request para Ollama
type ollamaRequest struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	Stream  bool                   `json:"stream"`
	Options map[string]interface{} `json:"options,omitempty"`
	Format  string                 `json:"format,omitempty"`
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

	// PERF-002: Create rate limiter if configured
	var rateLimiter *RateLimiter
	if cfg.RateLimitRPS > 0 {
		rateLimiter = NewRateLimiter(cfg.RateLimitRPS)
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
		rateLimiter: rateLimiter,
	}, nil
}

// Name retorna el nombre del provider
func (p *OllamaProvider) Name() string {
	return "ollama"
}

// Review realiza un code review
func (p *OllamaProvider) Review(ctx context.Context, request *ReviewRequest) (*ReviewResponse, error) {
	// PERF-002: Wait for rate limiter before making request
	if err := p.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter wait cancelled: %w", err)
	}

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

// GenerateDocumentation genera documentacion de los cambios
func (p *OllamaProvider) GenerateDocumentation(ctx context.Context, diff string, context string) (string, error) {
	var sb strings.Builder
	sb.WriteString("You are a technical writer. Summarize the following code changes into a concise, professional changelog format.\n")
	sb.WriteString("Use bullet points. Focus on the 'what' and 'why'. Do not describe line numbers, describe features/fixes.\n\n")
	
	if context != "" {
		sb.WriteString("Context: " + context + "\n\n")
	}
	
	sb.WriteString("Changes:\n")
	sb.WriteString(diff)

	reqBody := ollamaRequest{
		Model:  p.model,
		Prompt: sb.String(),
		Stream: false,
		Options: map[string]interface{}{
			"temperature": 0.3, // Lower temp for more deterministic docs
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/generate", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	var ollamaResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return strings.TrimSpace(ollamaResp.Response), nil
}

// GenerateCommitMessage genera un mensaje de commit
func (p *OllamaProvider) GenerateCommitMessage(ctx context.Context, diff string) (string, error) {
	prompt := "You are an expert developer. Generate a concise Conventional Commit message for the following diff. Only return the message, no explanation.\n\nDiff:\n" + diff

	reqBody := ollamaRequest{
		Model:  p.model,
		Prompt: prompt,
		Stream: false,
		Options: map[string]interface{}{
			"temperature": 0.2,
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/generate", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var ollamaResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", err
	}

	return strings.TrimSpace(ollamaResp.Response), nil
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

// Close cierra conexiones y limpia recursos
func (p *OllamaProvider) Close() error {
	// PERF-002: Clean up rate limiter
	if p.rateLimiter != nil {
		p.rateLimiter.Close()
	}
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
