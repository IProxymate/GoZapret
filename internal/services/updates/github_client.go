package updates

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// GitHubRelease представляет релиз на GitHub
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	HTMLURL string `json:"html_url"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// GitHubClient работает с GitHub API
type GitHubClient struct {
	apiURL string
	client *http.Client
}

// NewGitHubClient создает новый GitHub клиент
func NewGitHubClient(apiURL string) *GitHubClient {
	return &GitHubClient{
		apiURL: apiURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetLatestRelease получает последний релиз
func (g *GitHubClient) GetLatestRelease() (*GitHubRelease, error) {
	slog.Debug("Запрос последнего релиза", "api_url", g.apiURL)

	resp, err := g.client.Get(g.apiURL)
	if err != nil {
		slog.Error("Ошибка при запросе к GitHub API", "error", err)
		return nil, fmt.Errorf("ошибка при запросе к GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("GitHub API вернул ошибку", "status", resp.StatusCode)
		return nil, fmt.Errorf("GitHub API вернул статус %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		slog.Error("Ошибка парсинга ответа GitHub API", "error", err)
		return nil, fmt.Errorf("ошибка парсинга ответа GitHub API: %w", err)
	}

	return &release, nil
}
