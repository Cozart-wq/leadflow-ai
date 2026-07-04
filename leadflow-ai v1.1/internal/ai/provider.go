package ai

import (
	"context"
	"fmt"

	"github.com/leadflow-ai/leadflow-ai/internal/config"
)

type AnalysisInput struct {
	CompanyName string
	Website     string
	Email       string
	Phone       string
	Socials     map[string]string
	PageText    string
}

type AnalysisResult struct {
	QualityScore   int    // 0-100
	Recommendation string // текстовая рекомендация: стоит ли связываться
	OutreachMessage string
}

// Provider — единый интерфейс поверх любого LLM-провайдера.
type Provider interface {
	Analyze(ctx context.Context, in AnalysisInput) (*AnalysisResult, error)
	Name() string
}

// New выбирает реализацию Provider по конфигурации.
func New(cfg config.AIConfig) (Provider, error) {
	switch cfg.Provider {
	case "openai":
		return NewOpenAIProvider(cfg), nil
	case "claude", "anthropic":
		return NewClaudeProvider(cfg), nil
	case "gemini":
		return NewGeminiProvider(cfg), nil
	case "mock", "":
		return NewMockProvider(), nil
	default:
		return nil, fmt.Errorf("ai: unknown provider %q", cfg.Provider)
	}
}
