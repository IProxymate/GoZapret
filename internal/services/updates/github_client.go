package updates

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// GitHubRelease представляет релиз на GitHub
type GitHubRelease struct {
	TagName    string `json:"tag_name"`
	Name       string `json:"name"`
	HTMLURL    string `json:"html_url"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
	Assets     []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

const (
	githubUserAgent = "GoZapret-Updater/1.0"
	maxRetries      = 2
)

// GitHubClient работает с GitHub API
type GitHubClient struct {
	apiURL         string
	client         *http.Client
	insecureClient *http.Client // клиент без проверки TLS для обхода корпоративных прокси
}

// NewGitHubClient создает новый GitHub клиент
func NewGitHubClient(apiURL string) *GitHubClient {
	// Стандартный клиент с проверкой TLS
	normalClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Клиент без проверки TLS (для корпоративных прокси с MITM)
	insecureTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	insecureClient := &http.Client{
		Timeout:   30 * time.Second,
		Transport: insecureTransport,
	}

	return &GitHubClient{
		apiURL:         apiURL,
		client:         normalClient,
		insecureClient: insecureClient,
	}
}

// GetLatestRelease получает последний релиз.
// Если /releases/latest возвращает 404 (например, все релизы — pre-release),
// автоматически делает fallback на /releases и выбирает первый подходящий.
func (g *GitHubClient) GetLatestRelease() (*GitHubRelease, error) {
	slog.Debug("Запрос последнего релиза", "api_url", g.apiURL)

	// Пробуем /releases/latest
	release, err := g.fetchRelease(g.apiURL)
	if err == nil {
		return release, nil
	}

	slog.Warn("Не удалось получить /releases/latest, пробуем fallback на /releases", "error", err)

	// Fallback: если /releases/latest вернул ошибку (404 для pre-release, или DPI/прокси),
	// пробуем получить список всех релизов
	fallbackURL := g.buildFallbackURL()
	if fallbackURL == "" {
		return nil, err // не удалось построить fallback URL
	}

	release, fallbackErr := g.fetchLatestFromList(fallbackURL)
	if fallbackErr != nil {
		slog.Error("Fallback на /releases тоже не удался", "error", fallbackErr)
		// Возвращаем оригинальную ошибку — она более информативна
		return nil, fmt.Errorf("%w (fallback тоже не удался: %v)", err, fallbackErr)
	}

	return release, nil
}

// fetchRelease выполняет запрос к конкретному URL и декодирует один релиз
func (g *GitHubClient) fetchRelease(url string) (*GitHubRelease, error) {
	body, err := g.doRequestWithRetry(url)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var release GitHubRelease
	if err := json.NewDecoder(body).Decode(&release); err != nil {
		slog.Error("Ошибка парсинга ответа GitHub API", "error", err)
		return nil, fmt.Errorf("ошибка парсинга ответа GitHub API: %w", err)
	}

	if release.TagName == "" {
		return nil, fmt.Errorf("GitHub API вернул релиз без версии (tag_name пустой)")
	}

	return &release, nil
}

// fetchLatestFromList получает список релизов и выбирает первый подходящий (не draft, не prerelease)
func (g *GitHubClient) fetchLatestFromList(url string) (*GitHubRelease, error) {
	body, err := g.doRequestWithRetry(url)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var releases []GitHubRelease
	if err := json.NewDecoder(body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("ошибка парсинга списка релизов: %w", err)
	}

	if len(releases) == 0 {
		return nil, fmt.Errorf("список релизов пуст")
	}

	// Ищем первый не-draft, не-prerelease
	for i := range releases {
		if !releases[i].Draft && !releases[i].Prerelease {
			slog.Debug("Найден стабильный релиз через fallback", "tag", releases[i].TagName)
			return &releases[i], nil
		}
	}

	// Если стабильных нет, берём первый не-draft (pre-release)
	for i := range releases {
		if !releases[i].Draft {
			slog.Debug("Стабильных релизов нет, используем pre-release", "tag", releases[i].TagName)
			return &releases[i], nil
		}
	}

	// Крайний случай — берём первый вообще
	slog.Warn("Нет ни одного опубликованного релиза, используем первый доступный", "tag", releases[0].TagName)
	return &releases[0], nil
}

// doRequestWithRetry выполняет HTTP-запрос с ретраями и TLS-fallback.
// Возвращает тело ответа (обязательно закрыть вызывающей стороной).
func (g *GitHubClient) doRequestWithRetry(url string) (io.ReadCloser, error) {
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Пробуем обычный клиент
		body, err := g.doRequest(g.client, url)
		if err == nil {
			return body, nil
		}

		slog.Warn("Ошибка запроса к GitHub (попытка с TLS)",
			"attempt", attempt, "url", url, "error", err)

		// Пробуем без проверки TLS (корпоративный прокси / DPI)
		body, err = g.doRequest(g.insecureClient, url)
		if err == nil {
			slog.Debug("Запрос успешен без проверки TLS")
			return body, nil
		}

		lastErr = err
		slog.Warn("Ошибка запроса к GitHub (попытка без TLS)",
			"attempt", attempt, "url", url, "error", err)

		// Пауза перед ретраем (кроме последней попытки)
		if attempt < maxRetries {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}

	return nil, fmt.Errorf("не удалось получить данные от GitHub после %d попыток: %w", maxRetries, lastErr)
}

// doRequest выполняет единичный HTTP-запрос с правильными заголовками
func (g *GitHubClient) doRequest(client *http.Client, url string) (io.ReadCloser, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	// GitHub API требует User-Agent и рекомендует Accept
	req.Header.Set("User-Agent", githubUserAgent)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка сетевого запроса: %w", err)
	}

	if resp.StatusCode == http.StatusForbidden {
		resp.Body.Close()
		return nil, fmt.Errorf("GitHub API вернул 403 (возможно превышен лимит запросов — 60 в час для анонимных пользователей)")
	}

	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, fmt.Errorf("GitHub API вернул 404 (ресурс не найден или временно недоступен)")
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("GitHub API вернул статус %d", resp.StatusCode)
	}

	return resp.Body, nil
}

// buildFallbackURL строит URL для списка всех релизов из URL /releases/latest
func (g *GitHubClient) buildFallbackURL() string {
	// apiURL: https://api.github.com/repos/{owner}/{repo}/releases/latest
	// fallback: https://api.github.com/repos/{owner}/{repo}/releases?per_page=5
	if strings.HasSuffix(g.apiURL, "/latest") {
		base := strings.TrimSuffix(g.apiURL, "/latest")
		return base + "?per_page=5"
	}
	return ""
}
