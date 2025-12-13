package updates

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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

// Service управляет обновлениями приложения
type Service struct {
	githubClient *GitHubClient
	httpClient   *http.Client
}

// NewService создает новый сервис обновлений
func NewService(apiURL string) *Service {
	return &Service{
		githubClient: NewGitHubClient(apiURL),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CheckForUpdates проверяет наличие обновлений
func (s *Service) CheckForUpdates(currentVersion string) (*VersionInfo, error) {
	slog.Debug("Проверка обновлений", "current_version", currentVersion)

	release, err := s.githubClient.GetLatestRelease()
	if err != nil {
		return nil, err
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

// DownloadLatestRelease загружает последний релиз
func (s *Service) DownloadLatestRelease(currentVersion string, downloadPath string) (*DownloadResult, error) {
	slog.Debug("Начало загрузки обновления", "current_version", currentVersion, "download_path", downloadPath)

	release, err := s.githubClient.GetLatestRelease()
	if err != nil {
		return nil, err
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
	if err := s.downloadFile(downloadURL, filePath); err != nil {
		return nil, err
	}

	slog.Debug("Файл успешно загружен", "path", filePath)

	return &DownloadResult{
		FilePath: filePath,
		FileName: fileName,
	}, nil
}

// downloadFile загружает файл по URL
func (s *Service) downloadFile(url, filePath string) error {
	downloadResp, err := s.httpClient.Get(url)
	if err != nil {
		slog.Error("Ошибка загрузки архива", "error", err)
		return fmt.Errorf("ошибка загрузки архива: %w", err)
	}
	defer downloadResp.Body.Close()

	if downloadResp.StatusCode != http.StatusOK {
		slog.Error("Загрузка архива вернула ошибку", "status", downloadResp.StatusCode)
		return fmt.Errorf("загрузка архива вернула статус %d", downloadResp.StatusCode)
	}

	out, err := os.Create(filePath)
	if err != nil {
		slog.Error("Ошибка создания файла", "path", filePath, "error", err)
		return fmt.Errorf("ошибка создания файла: %w", err)
	}
	defer out.Close()

	// Копировать данные
	written, err := io.Copy(out, downloadResp.Body)
	if err != nil {
		slog.Error("Ошибка записи файла", "error", err)
		return fmt.Errorf("ошибка записи файла: %w", err)
	}

	if err := out.Sync(); err != nil {
		slog.Error("Ошибка синхронизации файла", "error", err)
		return fmt.Errorf("ошибка синхронизации файла: %w", err)
	}

	// Проверить размер файла
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		slog.Error("Ошибка получения информации о файле", "error", err)
		return fmt.Errorf("ошибка получения информации о файле: %w", err)
	}

	if fileInfo.Size() == 0 {
		slog.Error("Загруженный файл пустой")
		return fmt.Errorf("загруженный файл пустой")
	}

	slog.Debug("Файл успешно загружен", "size", written, "file_size", fileInfo.Size())
	return nil
}

// isVersionNewer сравнивает две версии
func (s *Service) isVersionNewer(currentVersion, newVersion string) bool {
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
