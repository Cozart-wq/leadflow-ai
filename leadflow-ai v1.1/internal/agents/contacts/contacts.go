package contacts

import (
	"regexp"
	"strings"
)

type ContactInfo struct {
	Email   string
	Phone   string
	Socials map[string]string
}

var (
	emailRe = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
	phoneRe = regexp.MustCompile(`\+?\d[\d\s\-\(\)]{8,16}\d`)
	tagRe   = regexp.MustCompile(`<[^>]+>`)

	socialPatterns = map[string]*regexp.Regexp{
		"facebook":  regexp.MustCompile(`https?://(www\.)?facebook\.com/[^\s"'<>]+`),
		"instagram": regexp.MustCompile(`https?://(www\.)?instagram\.com/[^\s"'<>]+`),
		"linkedin":  regexp.MustCompile(`https?://(www\.)?linkedin\.com/[^\s"'<>]+`),
		"twitter":   regexp.MustCompile(`https?://(www\.)?(twitter|x)\.com/[^\s"'<>]+`),
		"telegram":  regexp.MustCompile(`https?://(www\.)?t\.me/[^\s"'<>]+`),
		"youtube":   regexp.MustCompile(`https?://(www\.)?youtube\.com/[^\s"'<>]+`),
	}

	ignoredEmailSuffixes = []string{".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp"}
)

type Agent struct{}

func NewAgent() *Agent { return &Agent{} }

// Extract парсит "сырой" HTML и вытаскивает email, телефон и ссылки на
// соцсети через регулярные выражения. Полноценный HTML-парсер (goquery)
// здесь избыточен: контакты почти всегда лежат в тексте или href-атрибутах,
// а не требуют понимания DOM-дерева.
func (a *Agent) Extract(html string) ContactInfo {
	info := ContactInfo{Socials: map[string]string{}}

	if m := emailRe.FindString(html); m != "" && isValidEmail(m) {
		info.Email = m
	}

	plainText := tagRe.ReplaceAllString(html, " ")
	if m := phoneRe.FindString(plainText); m != "" {
		info.Phone = strings.TrimSpace(m)
	}

	for name, re := range socialPatterns {
		if m := re.FindString(html); m != "" {
			info.Socials[name] = m
		}
	}

	return info
}

func isValidEmail(email string) bool {
	lower := strings.ToLower(email)
	for _, suffix := range ignoredEmailSuffixes {
		if strings.HasSuffix(lower, suffix) {
			return false
		}
	}
	return true
}
