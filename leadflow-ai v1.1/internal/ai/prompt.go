package ai

import "fmt"

// buildPrompt формирует единый промпт для анализа лида, переиспользуемый
// всеми провайдерами. Ожидается ответ строго в JSON-формате.
func buildPrompt(in AnalysisInput) string {
	return fmt.Sprintf(`Ты — ассистент по анализу потенциальных клиентов (лидов) для B2B лидогенерации.

Данные о компании:
Название: %s
Сайт: %s
Email: %s
Телефон: %s
Соцсети: %v

Текст главной страницы сайта (обрезан):
%s

Оцени качество этого лида и верни ТОЛЬКО JSON без markdown-разметки, строго в формате:
{"quality_score": <int 0-100>, "recommendation": "<краткая рекомендация на русском, стоит ли связываться и почему>", "outreach_message": "<короткое персонализированное сообщение для первого контакта на русском>"}`,
		in.CompanyName, in.Website, in.Email, in.Phone, in.Socials, truncate(in.PageText, 3000))
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
