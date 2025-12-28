package updates

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"github.com/IProxymate/GoZapret/internal/config"
	"github.com/fynelabs/fyneselfupdate"
	"github.com/fynelabs/selfupdate"
)

var extCfg = config.GetExternalConfig()

// SelfUpdater управляет обновлениями самого приложения GoZapret
type SelfUpdater struct {
	currentVersion string
	httpClient     *http.Client
	insecureClient *http.Client // клиент без проверки TLS для обхода корпоративных прокси
	app            fyne.App
	window         fyne.Window
	config         *selfupdate.Config
}

// NewSelfUpdater создаёт новый сервис самообновления
func NewSelfUpdater(currentVersion string) *SelfUpdater {
	// Стандартный клиент с проверкой TLS
	normalClient := &http.Client{
		Timeout: 60 * time.Second,
	}

	// Клиент без проверки TLS (для корпоративных прокси с MITM)
	insecureTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	insecureClient := &http.Client{
		Timeout:   60 * time.Second,
		Transport: insecureTransport,
	}

	return &SelfUpdater{
		currentVersion: strings.TrimPrefix(currentVersion, "v"),
		httpClient:     normalClient,
		insecureClient: insecureClient,
	}
}

// SetupWithFyne настраивает self-updater с Fyne приложением
// Должен вызываться после создания окна приложения
func (s *SelfUpdater) SetupWithFyne(app fyne.App, window fyne.Window) error {
	s.app = app
	s.window = window

	// Создаём HTTP источник для загрузки обновлений
	// {{.Version}}, {{.OS}}, {{.Arch}}, {{.Ext}} будут заменены автоматически
	httpSource := selfupdate.NewHTTPSource(s.httpClient, extCfg.GoZapretReleasesURL())

	// Создаём конфигурацию без автоматической проверки
	// Проверка будет только по нажатию кнопки
	s.config = fyneselfupdate.NewConfigWithTimeout(
		app,
		window,
		2*time.Minute, // таймаут на загрузку
		httpSource,
		selfupdate.Schedule{
			FetchOnStart: false, // НЕ проверять при запуске автоматически
			Interval:     0,     // НЕ проверять периодически
		},
		nil, // без подписи (можно добавить позже)
	)

	// Устанавливаем текущую версию
	s.config.Current = &selfupdate.Version{
		Number: s.currentVersion,
	}

	slog.Debug("SelfUpdater настроен",
		"version", s.currentVersion,
		"os", runtime.GOOS,
		"arch", runtime.GOARCH,
		"url", extCfg.GoZapretReleasesURL())

	return nil
}

// CheckForUpdates проверяет наличие обновлений и показывает диалог если есть новая версия
func (s *SelfUpdater) CheckForUpdates() error {
	if s.config == nil {
		return fmt.Errorf("self-updater не настроен, вызовите SetupWithFyne сначала")
	}

	slog.Info("Проверка обновлений GoZapret", "current_version", s.currentVersion)

	// Используем Manage для проверки и обновления
	// fyneselfupdate автоматически покажет диалоги
	_, err := selfupdate.Manage(s.config)
	if err != nil {
		slog.Error("Ошибка проверки обновлений", "error", err)
		return fmt.Errorf("ошибка проверки обновлений: %w", err)
	}

	return nil
}

// GetCurrentVersion возвращает текущую версию приложения
func (s *SelfUpdater) GetCurrentVersion() string {
	return s.currentVersion
}

// CheckForUpdatesManual выполняет ручную проверку обновлений с callback'ами
func (s *SelfUpdater) CheckForUpdatesManual(onUpdateAvailable func(newVersion string), onNoUpdate func(), onError func(error)) {
	if s.config == nil {
		onError(fmt.Errorf("self-updater не настроен"))
		return
	}

	go func() {
		slog.Debug("Начинаем ручную проверку обновлений")

		// Получаем информацию о последней версии через GitHub API
		client := NewGitHubClient(extCfg.GoZapretAPIURL())
		release, err := client.GetLatestRelease()
		if err != nil {
			slog.Error("Ошибка получения информации о релизе", "error", err)
			fyne.Do(func() {
				onError(fmt.Errorf("не удалось проверить обновления: %w", err))
			})
			return
		}

		latestVersion := strings.TrimPrefix(release.TagName, "v")
		currentVersion := strings.TrimPrefix(s.currentVersion, "v")

		slog.Debug("Сравнение версий", "current", currentVersion, "latest", latestVersion)

		if s.isNewerVersion(currentVersion, latestVersion) {
			slog.Info("Доступна новая версия", "new_version", latestVersion)
			fyne.Do(func() {
				onUpdateAvailable(latestVersion)
			})
		} else {
			slog.Info("Установлена последняя версия", "version", currentVersion)
			fyne.Do(func() {
				onNoUpdate()
			})
		}
	}()
}

// isNewerVersion сравнивает версии (простое сравнение строк)
func (s *SelfUpdater) isNewerVersion(current, latest string) bool {
	if current == "" {
		return true
	}
	if current == latest {
		return false
	}
	// Простое сравнение - если версии разные, считаем что latest новее
	// Для более точного сравнения можно использовать semver
	return latest > current
}

// PerformUpdateAsync запускает процесс обновления асинхронно с callback'ами
func (s *SelfUpdater) PerformUpdateAsync(latestVersion string, onProgress func(status string), onSuccess func(), onError func(error)) {
	go func() {
		slog.Info("Запуск процесса обновления", "target_version", latestVersion)

		fyne.Do(func() {
			onProgress("Загрузка обновления...")
		})

		// Формируем URL для скачивания
		downloadURL := extCfg.DownloadURL(latestVersion)

		slog.Debug("URL для скачивания", "url", downloadURL)

		// Скачиваем новую версию (сначала пробуем обычный клиент, потом без проверки TLS)
		resp, err := s.httpClient.Get(downloadURL)
		if err != nil {
			slog.Warn("Ошибка скачивания с проверкой TLS, пробуем без проверки", "error", err)
			resp, err = s.insecureClient.Get(downloadURL)
			if err != nil {
				slog.Error("Ошибка скачивания", "error", err)
				fyne.Do(func() {
					onError(fmt.Errorf("ошибка скачивания: %w", err))
				})
				return
			}
			slog.Debug("Скачивание успешно без проверки TLS")
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			slog.Error("Сервер вернул ошибку", "status", resp.StatusCode)
			fyne.Do(func() {
				onError(fmt.Errorf("сервер вернул статус %d", resp.StatusCode))
			})
			return
		}

		fyne.Do(func() {
			onProgress("Сохранение файла...")
		})

		// Получаем путь к текущему exe
		currentExe, err := os.Executable()
		if err != nil {
			fyne.Do(func() {
				onError(fmt.Errorf("не удалось определить путь к exe: %w", err))
			})
			return
		}
		currentExe, _ = filepath.EvalSymlinks(currentExe)

		// Сохраняем новую версию рядом с текущей
		newExePath := currentExe + ".new"
		newFile, err := os.Create(newExePath)
		if err != nil {
			fyne.Do(func() {
				onError(fmt.Errorf("не удалось создать файл: %w", err))
			})
			return
		}

		_, err = newFile.ReadFrom(resp.Body)
		newFile.Close()
		if err != nil {
			os.Remove(newExePath)
			fyne.Do(func() {
				onError(fmt.Errorf("ошибка записи файла: %w", err))
			})
			return
		}

		fyne.Do(func() {
			onProgress("Подготовка к замене...")
		})

		// Создаём bat-скрипт для замены exe после закрытия приложения
		batPath := filepath.Join(filepath.Dir(currentExe), "update.bat")
		batContent := fmt.Sprintf(`@echo off
chcp 65001 >nul
echo Ожидание закрытия приложения...
ping -n 2 127.0.0.1 >nul
:retry
del "%s" >nul 2>&1
if exist "%s" (
    ping -n 1 127.0.0.1 >nul
    goto retry
)
move "%s" "%s"
start "" "%s"
del "%%~f0"
`, currentExe, currentExe, newExePath, currentExe, currentExe)

		err = os.WriteFile(batPath, []byte(batContent), 0644)
		if err != nil {
			os.Remove(newExePath)
			fyne.Do(func() {
				onError(fmt.Errorf("ошибка создания скрипта обновления: %w", err))
			})
			return
		}

		slog.Info("Обновление подготовлено, запуск скрипта замены")

		// Запускаем bat-скрипт
		cmd := exec.Command("cmd", "/C", "start", "/min", "", batPath)
		cmd.Start()

		fyne.Do(func() {
			onSuccess()
		})
	}()
}
