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

type OpenAIProvider struct {
	apiKey      string
	baseURL     string
	model       string
	timeout     time.Duration
	maxTokens   int
	temperature float64
	client      *http.Client
}

type openaiRequest struct {
	Model       string    `json:"model"`
	Messages    []message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiResponse struct {
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

func NewOpenAIProvider(cfg *config.ProviderConfig) (*OpenAIProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("api_key is required for openai provider")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 2 * time.Minute
	}

	return &OpenAIProvider{
		apiKey:      cfg.APIKey,
		baseURL:     baseURL,
		model:       cfg.Model,
		timeout:     timeout,
		maxTokens:   cfg.MaxTokens,
		temperature: cfg.Temperature,
		client: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func (p *OpenAIProvider) Name() string {
	return "openai"
}

func (p *OpenAIProvider) Review(ctx context.Context, request *ReviewRequest) (*ReviewResponse, error) {
	prompt := p.buildPrompt(request)

	reqBody := openaiRequest{
		Model: p.model,
		Messages: []message{
			{Role: "system", Content: "You are an expert code reviewer. Return only JSON."},
			{Role: "user", Content: prompt},
		},
		Temperature: p.temperature,
		MaxTokens:   p.maxTokens,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	start := time.Now()
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai API error (status %d): %s", resp.StatusCode, string(body))
	}

	var openaiResp openaiResponse
	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(openaiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in openai response")
	}

	content := openaiResp.Choices[0].Message.Content
	duration := time.Since(start).Milliseconds()

	// Parse JSON output (reuse robust logic)
	// TODO: Share parsing logic with Ollama provider
	var issues []Issue
	
	// Clean markdown blocks
	cleanResp := content
	if len(cleanResp) > 3 && cleanResp[:3] == "```" {
		// Remove first line (```json) and last line (```)
		lines := strings.Split(cleanResp, "\n")
		if len(lines) > 2 {
			cleanResp = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	if err := json.Unmarshal([]byte(cleanResp), &issues); err != nil {
		// Fallback for wrapped objects or single objects (simplified here)
		return nil, fmt.Errorf("failed to parse openai response: %w", err)
	}

		return &ReviewResponse{

			Issues:         issues,

			Summary:        fmt.Sprintf("Review completed using %s. Found %d issues.", p.model, len(issues)),

			Score:          100, // Placeholder

			TokensUsed:     openaiResp.Usage.TotalTokens,

			ProcessingTime: duration,

		}, nil

	}

	

	// GenerateDocumentation genera documentacion

	func (p *OpenAIProvider) GenerateDocumentation(ctx context.Context, diff string, context string) (string, error) {

		prompt := fmt.Sprintf("You are a technical writer. Summarize the following code changes into a concise changelog.\n\nContext: %s\n\nChanges:\n%s", context, diff)

	

		reqBody := openaiRequest{

			Model: p.model,

			Messages: []message{

				{Role: "system", Content: "You are a technical writer."},

				{Role: "user", Content: prompt},

			},

			Temperature: 0.3,

			MaxTokens:   p.maxTokens,

		}

	

		jsonBody, err := json.Marshal(reqBody)

		if err != nil {

			return "", err

		}

	

		req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewBuffer(jsonBody))

		if err != nil {

			return "", err

		}

		req.Header.Set("Content-Type", "application/json")

		req.Header.Set("Authorization", "Bearer "+p.apiKey)

	

		resp, err := p.client.Do(req)

		if err != nil {

			return "", err

		}

		defer resp.Body.Close()

	

		var openaiResp openaiResponse

		if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {

			return "", err

		}

	

		if len(openaiResp.Choices) == 0 {

			return "", fmt.Errorf("no choices in openai response")

		}

	

			return openaiResp.Choices[0].Message.Content, nil

	

		}

	

		

	

		// GenerateCommitMessage genera un mensaje de commit

	

		func (p *OpenAIProvider) GenerateCommitMessage(ctx context.Context, diff string) (string, error) {

	

			prompt := "Generate a concise Conventional Commit message for this diff. Output only the message."

	

			

	

			reqBody := openaiRequest{

	

				Model: p.model,

	

				Messages: []message{

	

					{Role: "system", Content: prompt},

	

					{Role: "user", Content: diff},

	

				},

	

				Temperature: 0.2,

	

				MaxTokens:   100,

	

			}

	

		

	

			jsonBody, err := json.Marshal(reqBody)

	

			if err != nil {

	

				return "", err

	

			}

	

		

	

			req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewBuffer(jsonBody))

	

			if err != nil {

	

				return "", err

	

			}

	

			req.Header.Set("Content-Type", "application/json")

	

			req.Header.Set("Authorization", "Bearer "+p.apiKey)

	

		

	

			resp, err := p.client.Do(req)

	

			if err != nil {

	

				return "", err

	

			}

	

			defer resp.Body.Close()

	

		

	

			var openaiResp openaiResponse

	

			if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {

	

				return "", err

	

			}

	

		

	

			if len(openaiResp.Choices) == 0 {

	

				return "", fmt.Errorf("no choices in openai response")

	

			}

	

		

	

			return openaiResp.Choices[0].Message.Content, nil

	

		}

	

		

	

		func (p *OpenAIProvider) HealthCheck(ctx context.Context) error {
	// Simple models check
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check status: %d", resp.StatusCode)
	}
	return nil
}

func (p *OpenAIProvider) Close() error {
	return nil
}

func (p *OpenAIProvider) buildPrompt(req *ReviewRequest) string {
	// Reuse Ollama prompt logic or customize
	return fmt.Sprintf("Analyze this code:\n%s\nRules: %v", req.Diff, req.Rules)
}
