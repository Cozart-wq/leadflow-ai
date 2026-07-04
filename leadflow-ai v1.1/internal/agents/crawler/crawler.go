package crawler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Agent struct {
	client *http.Client
}

func NewAgent() *Agent {
	return &Agent{client: &http.Client{Timeout: 15 * time.Second}}
}

// Fetch скачивает HTML страницы по URL. Размер ответа ограничен, чтобы
// один аномально большой сайт не исчерпал память процесса.
func (a *Agent) Fetch(ctx context.Context, rawURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("crawler: build request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; LeadFlowAI/1.0; +https://github.com/leadflow-ai)")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("crawler: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("crawler: unexpected status %d for %s", resp.StatusCode, rawURL)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20)) // максимум 5 МБ
	if err != nil {
		return "", fmt.Errorf("crawler: read body: %w", err)
	}

	return string(body), nil
}
