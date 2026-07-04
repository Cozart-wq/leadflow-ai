package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/leadflow-ai/leadflow-ai/internal/config"
)

type OpenAIProvider struct {
	apiKey string
	model  string
	client *http.Client
}

func NewOpenAIProvider(cfg config.AIConfig) *OpenAIProvider {
	model := cfg.Model
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &OpenAIProvider{
		apiKey: cfg.APIKey,
		model:  model,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *OpenAIProvider) Name() string { return "openai" }

func (p *OpenAIProvider) Analyze(ctx context.Context, in AnalysisInput) (*AnalysisResult, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("openai: api key not configured")
	}

	body := map[string]interface{}{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "user", "content": buildPrompt(in)},
		},
		"temperature": 0.4,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("openai: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.openai.com/v1/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("openai: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai: request failed: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("openai: read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai: status %d: %s", resp.StatusCode, string(rawBody))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(rawBody, &parsed); err != nil {
		return nil, fmt.Errorf("openai: parse response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("openai: empty response")
	}

	return parseAIResponse(parsed.Choices[0].Message.Content)
}
