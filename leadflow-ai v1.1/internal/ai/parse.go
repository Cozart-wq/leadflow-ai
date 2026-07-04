package ai

import (
	"encoding/json"
	"fmt"
	"regexp"
)

var jsonBlockRe = regexp.MustCompile(`\{[\s\S]*\}`)

type aiResponse struct {
	QualityScore    int    `json:"quality_score"`
	Recommendation  string `json:"recommendation"`
	OutreachMessage string `json:"outreach_message"`
}

// parseAIResponse извлекает JSON-объект из ответа модели (даже если модель
// обернула его в текст или markdown-код) и приводит к AnalysisResult.
func parseAIResponse(raw string) (*AnalysisResult, error) {
	match := jsonBlockRe.FindString(raw)
	if match == "" {
		return nil, fmt.Errorf("ai: no JSON object found in response")
	}

	var resp aiResponse
	if err := json.Unmarshal([]byte(match), &resp); err != nil {
		return nil, fmt.Errorf("ai: failed to parse response JSON: %w", err)
	}

	return &AnalysisResult{
		QualityScore:     resp.QualityScore,
		Recommendation:   resp.Recommendation,
		OutreachMessage:  resp.OutreachMessage,
	}, nil
}
