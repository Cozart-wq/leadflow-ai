package analyzer

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/leadflow-ai/leadflow-ai/internal/ai"
)

type Agent struct {
	provider ai.Provider
}

func NewAgent(provider ai.Provider) *Agent {
	return &Agent{provider: provider}
}

var tagRe = regexp.MustCompile(`<[^>]+>`)
var spaceRe = regexp.MustCompile(`\s+`)

// ExtractText грубо превращает HTML в читаемый текст (убирает теги),
// чтобы передать AI-провайдеру контент страницы вместо разметки.
func ExtractText(html string) string {
	text := tagRe.ReplaceAllString(html, " ")
	text = spaceRe.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

func (a *Agent) Analyze(ctx context.Context, in ai.AnalysisInput) (*ai.AnalysisResult, error) {
	result, err := a.provider.Analyze(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("analyzer: %s provider failed: %w", a.provider.Name(), err)
	}
	return result, nil
}
