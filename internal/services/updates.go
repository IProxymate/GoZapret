package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// UpdateService управляет обновлениями приложения
type UpdateService struct {
	githubAPI string
	client    *http.Client
}

// NewUpdateService создает новый сервис обновлений
func NewUpdateService() *UpdateService {
	return &UpdateService{
		githubAPI: "https://api.github.com/repos/Flowseal/zapret-discord-youtube/releases/latest",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

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

// VersionInfo содержит информацию о версии
type VersionInfo struct {
	Current string
	Latest  string
	IsNewer bool
	URL     string
}

// DownloadResult содержит результат загрузки
type DownloadResult struct {
	FilePath string
	FileName string
}

// CheckForUpdates проверяет наличие обновлений на GitHub
func (s *UpdateService) CheckForUpdates(currentVersion string) (*VersionInfo, error) {
	slog.Debug("Проверка обновлений", "current_version", currentVersion)

	resp, err := s.client.Get(s.githubAPI)
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

	isNewer := s.isVersionNewer(currentVersion, release.TagName)

	versionInfo := &VersionInfo{
		Current: currentVersion,
		Latest:  release.TagName,
		IsNewer: isNewer,
		URL:     release.HTMLURL,
	}

	slog.Debug("Проверка обновлений завершена",
		"current", currentVersion,
		"latest", release.TagName,
		"is_newer", isNewer)

	return versionInfo, nil
}

// isVersionNewer сравнивает две версии
func (s *UpdateService) isVersionNewer(currentVersion, newVersion string) bool {
	currentVersion = strings.TrimPrefix(currentVersion, "v")
	newVersion = strings.TrimPrefix(newVersion, "v")

	if currentVersion == "" {
		return true
	}

	if currentVersion == newVersion {
		return false
	}

	return true
}

// DownloadLatestRelease загружает последний релиз с GitHub
func (s *UpdateService) DownloadLatestRelease(currentVersion string, downloadPath string) (*DownloadResult, error) {
	slog.Debug("Начало загрузки обновления", "current_version", currentVersion, "download_path", downloadPath)

	resp, err := s.client.Get(s.githubAPI)
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

	isNewer := s.isVersionNewer(currentVersion, release.TagName)
	if !isNewer {
		slog.Warn("Локальная версия актуальна", "current", currentVersion, "latest", release.TagName)
		return nil, fmt.Errorf("локальная версия актуальна - обновление не требуется")
	}

	// Найти архив в релизе
	var downloadURL string
	var fileName string
	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.Name, ".rar") || strings.HasSuffix(asset.Name, ".zip") {
			downloadURL = asset.BrowserDownloadURL
			fileName = asset.Name
			break
		}
	}

	if downloadURL == "" {
		slog.Error("Архив не найден в релизе")
		return nil, fmt.Errorf("не найден .rar или .zip архив в релизе")
	}

	slog.Debug("Найден архив для загрузки", "file", fileName, "url", downloadURL)

	// Создать папку для загрузки
	if err := os.MkdirAll(downloadPath, os.ModePerm); err != nil {
		slog.Error("Ошибка создания папки для загрузки", "path", downloadPath, "error", err)
		return nil, fmt.Errorf("ошибка создания папки для загрузки: %w", err)
	}

	filePath := filepath.Join(downloadPath, fileName)

	// Загрузить файл
	downloadResp, err := s.client.Get(downloadURL)
	if err != nil {
		slog.Error("Ошибка загрузки архива", "error", err)
		return nil, fmt.Errorf("ошибка загрузки архива: %w", err)
	}
	defer downloadResp.Body.Close()

	if downloadResp.StatusCode != http.StatusOK {
		slog.Error("Загрузка архива вернула ошибку", "status", downloadResp.StatusCode)
		return nil, fmt.Errorf("загрузка архива вернула статус %d", downloadResp.StatusCode)
	}

	out, err := os.Create(filePath)
	if err != nil {
		slog.Error("Ошибка создания файла", "path", filePath, "error", err)
		return nil, fmt.Errorf("ошибка создания файла: %w", err)
	}
	defer out.Close()

	// Копировать данные
	written, err := io.Copy(out, downloadResp.Body)
	if err != nil {
		slog.Error("Ошибка записи файла", "error", err)
		return nil, fmt.Errorf("ошибка записи файла: %w", err)
	}

	if err := out.Sync(); err != nil {
		slog.Error("Ошибка синхронизации файла", "error", err)
		return nil, fmt.Errorf("ошибка синхронизации файла: %w", err)
	}

	// Проверить размер файла
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		slog.Error("Ошибка получения информации о файле", "error", err)
		return nil, fmt.Errorf("ошибка получения информации о файле: %w", err)
	}

	if fileInfo.Size() == 0 {
		slog.Error("Загруженный файл пустой")
		return nil, fmt.Errorf("загруженный файл пустой")
	}

	slog.Debug("Файл успешно загружен",
		"path", filePath,
		"size", written,
		"file_size", fileInfo.Size())

	return &DownloadResult{
		FilePath: filePath,
		FileName: fileName,
	}, nil
}
