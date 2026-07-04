package outreach

import (
	"fmt"
	"strings"

	"github.com/leadflow-ai/leadflow-ai/internal/ai"
)

type Agent struct{}

func NewAgent() *Agent { return &Agent{} }

// PrepareMessage возвращает готовое сообщение для первого контакта.
// Если AI-провайдер не вернул сообщение (например, MockProvider с низким
// скорингом или ошибка генерации), используется простой шаблон-заглушка,
// чтобы у лида всегда было хоть какое-то сообщение для outreach.
func (a *Agent) PrepareMessage(result *ai.AnalysisResult, companyName string) string {
	if result != nil && strings.TrimSpace(result.OutreachMessage) != "" {
		return result.OutreachMessage
	}
	name := strings.TrimSpace(companyName)
	if name == "" {
		name = "команда"
	}
	return fmt.Sprintf("Здравствуйте! Хотели бы обсудить возможное сотрудничество с %s. Расскажете подробнее о ваших текущих задачах?", name)
}
