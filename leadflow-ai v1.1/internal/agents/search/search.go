package search

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type Company struct {
	Name    string
	Website string
}

type Agent interface {
	Find(ctx context.Context, query string, limit int) ([]Company, error)
}

// DuckDuckGoAgent ищет компании через HTML-версию DuckDuckGo (не требует
// API-ключа). Это дефолтная реализация Search Agent для проекта с открытым
// исходным кодом — её легко заменить на платный Search API (Google Places,
// Bing, Crunchbase и т.д.), реализовав тот же интерфейс Agent.
type DuckDuckGoAgent struct {
	client *http.Client
}

func NewDuckDuckGoAgent() *DuckDuckGoAgent {
	return &DuckDuckGoAgent{client: &http.Client{Timeout: 15 * time.Second}}
}

var resultLinkRe = regexp.MustCompile(`class="result__a"[^>]*href="([^"]+)"[^>]*>([\s\S]*?)</a>`)
var tagRe = regexp.MustCompile(`<[^>]+>`)

func (a *DuckDuckGoAgent) Find(ctx context.Context, query string, limit int) ([]Company, error) {
	endpoint := "https://html.duckduckgo.com/html/?q=" + url.QueryEscape(query)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("search: build request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; LeadFlowAI/1.0)")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, fmt.Errorf("search: read response: %w", err)
	}

	matches := resultLinkRe.FindAllStringSubmatch(string(body), -1)

	seen := map[string]bool{}
	var companies []Company
	for _, m := range matches {
		if len(companies) >= limit {
			break
		}
		rawLink := decodeDuckDuckGoLink(m[1])
		title := strings.TrimSpace(tagRe.ReplaceAllString(m[2], ""))
		if rawLink == "" || title == "" || seen[rawLink] {
			continue
		}
		seen[rawLink] = true
		companies = append(companies, Company{Name: title, Website: rawLink})
	}

	return companies, nil
}

// decodeDuckDuckGoLink извлекает реальный URL из редиректной ссылки вида
// //duckduckgo.com/l/?uddg=<encoded-url>&...
func decodeDuckDuckGoLink(href string) string {
	if strings.Contains(href, "uddg=") {
		parsed, err := url.Parse(href)
		if err == nil {
			if u := parsed.Query().Get("uddg"); u != "" {
				return u
			}
		}
	}
	if strings.HasPrefix(href, "http") {
		return href
	}
	return ""
}
