package ai

import (
	"context"
	"fmt"
	"strings"
)

// MockProvider — эвристический анализ без обращения к внешним LLM API.
// Используется по умолчанию (provider=mock или пустой ключ), чтобы
// проект был полностью работоспособен "из коробки" без ключей API.
type MockProvider struct{}

func NewMockProvider() *MockProvider { return &MockProvider{} }

func (p *MockProvider) Name() string { return "mock" }

func (p *MockProvider) Analyze(ctx context.Context, in AnalysisInput) (*AnalysisResult, error) {
	score := 20
	if in.Email != "" {
		score += 25
	}
	if in.Phone != "" {
		score += 20
	}
	if len(in.Socials) > 0 {
		score += 15
	}
	if len(in.PageText) > 500 {
		score += 20
	}
	if score > 100 {
		score = 100
	}

	rec := "Низкий приоритет: недостаточно контактных данных для качественного outreach."
	switch {
	case score >= 70:
		rec = "Высокий приоритет: у компании есть все ключевые контакты, стоит связаться в первую очередь."
	case score >= 40:
		rec = "Средний приоритет: часть контактов отсутствует, стоит проверить вручную перед outreach."
	}

	name := in.CompanyName
	if name == "" {
		name = "команда"
	}

	msg := fmt.Sprintf("Здравствуйте! Меня заинтересовала компания %s — хотел бы обсудить, как мы можем быть полезны в развитии вашего бизнеса. Удобно ли вам созвониться на этой неделе?", strings.TrimSpace(name))

	return &AnalysisResult{
		QualityScore:     score,
		Recommendation:   rec,
		OutreachMessage:  msg,
	}, nil
}
