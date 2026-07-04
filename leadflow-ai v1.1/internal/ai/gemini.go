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

type GeminiProvider struct {
	apiKey string
	model  string
	client *http.Client
}

func NewGeminiProvider(cfg config.AIConfig) *GeminiProvider {
	model := cfg.Model
	if model == "" {
		model = "gemini-2.0-flash"
	}
	return &GeminiProvider{
		apiKey: cfg.APIKey,
		model:  model,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *GeminiProvider) Name() string { return "gemini" }

func (p *GeminiProvider) Analyze(ctx context.Context, in AnalysisInput) (*AnalysisResult, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("gemini: api key not configured")
	}

	body := map[string]interface{}{
		"contents": []map[string]interface{}{
			{"parts": []map[string]string{{"text": buildPrompt(in)}}},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("gemini: marshal request: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", p.model, p.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("gemini: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini: request failed: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gemini: read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini: status %d: %s", resp.StatusCode, string(rawBody))
	}

	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(rawBody, &parsed); err != nil {
		return nil, fmt.Errorf("gemini: parse response: %w", err)
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("gemini: empty response")
	}

	return parseAIResponse(parsed.Candidates[0].Content.Parts[0].Text)
}
