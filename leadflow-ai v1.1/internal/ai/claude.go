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

type ClaudeProvider struct {
	apiKey string
	model  string
	client *http.Client
}

func NewClaudeProvider(cfg config.AIConfig) *ClaudeProvider {
	model := cfg.Model
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	return &ClaudeProvider{
		apiKey: cfg.APIKey,
		model:  model,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *ClaudeProvider) Name() string { return "claude" }

func (p *ClaudeProvider) Analyze(ctx context.Context, in AnalysisInput) (*AnalysisResult, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("claude: api key not configured")
	}

	body := map[string]interface{}{
		"model":      p.model,
		"max_tokens": 1024,
		"messages": []map[string]string{
			{"role": "user", "content": buildPrompt(in)},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("claude: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.anthropic.com/v1/messages", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("claude: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("claude: request failed: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("claude: read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("claude: status %d: %s", resp.StatusCode, string(rawBody))
	}

	var parsed struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(rawBody, &parsed); err != nil {
		return nil, fmt.Errorf("claude: parse response: %w", err)
	}
	if len(parsed.Content) == 0 {
		return nil, fmt.Errorf("claude: empty response")
	}

	return parseAIResponse(parsed.Content[0].Text)
}
