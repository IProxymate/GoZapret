package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/IProxymate/GoZapret/internal/domain"
	"github.com/IProxymate/GoZapret/internal/services/updates"
	"github.com/IProxymate/GoZapret/internal/utils"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// HelpView представляет функционал меню помощи
type HelpView struct {
	app *App
}

// NewHelpView создает новый HelpView
func NewHelpView(app *App) *HelpView {
	return &HelpView{
		app: app,
	}
}

// ShowAbout показывает окно "О программе"
func (v *HelpView) ShowAbout() {
	cfg := v.app.configManager.GetConfig()
	aboutText := fmt.Sprintf(`**Zapret GUI**

Графический интерфейс для winws (zapret)

**Версия:** %s

**Описание:**
Zapret GUI предоставляет удобный графический интерфейс для управления zapret - инструментом для обхода блокировок интернет-ресурсов.

**Возможности:**
• Запуск и остановка стратегий zapret
• Диагностика системы
• Очистка кэша Discord
• Автозапуск при старте системы
• Управление режимами IPset
• Автоматическое обновление

**Репозиторий:** github.com/Flowseal/zapret-discord-youtube

© 2024 Zapret GUI`, cfg.Version)

	aboutLabel := widget.NewRichTextFromMarkdown(aboutText)
	scroll := container.NewScroll(aboutLabel)
	scroll.SetMinSize(fyne.NewSize(500, 400))

	dialog.ShowCustom("О программе", "Закрыть", scroll, v.app.window)
}

// CheckForUpdates проверяет наличие обновлений
func (v *HelpView) CheckForUpdates() {
	progressDialog := v.createProgressDialog("Проверка обновлений", "Проверка обновлений...")
	progressDialog.Show()

	go func() {
		versionInfo, err := v.checkVersion()

		fyne.Do(func() {
			progressDialog.Hide()
			v.handleVersionCheckResult(versionInfo, err)
		})
	}()
}

// checkVersion проверяет версию на GitHub
func (v *HelpView) checkVersion() (*updates.VersionInfo, error) {
	cfg := v.app.configManager.GetConfig()
	return v.app.updateService.CheckForUpdates(cfg.Version)
}

// handleVersionCheckResult обрабатывает результат проверки версии
func (v *HelpView) handleVersionCheckResult(versionInfo *updates.VersionInfo, err error) {
	if err != nil {
		dialog.ShowError(fmt.Errorf("ошибка проверки обновлений:\n%w", err), v.app.window)
		return
	}

	if !versionInfo.IsNewer {
		dialog.ShowInformation("Обновления",
			fmt.Sprintf("У вас установлена последняя версия: %s", versionInfo.Current),
			v.app.window)
		return
	}

	v.showUpdateDialog(versionInfo)
}

// showUpdateDialog показывает диалог с вариантами обновления
func (v *HelpView) showUpdateDialog(versionInfo *updates.VersionInfo) {
	message := fmt.Sprintf(`Доступна новая версия!

Текущая версия: %s
Новая версия: %s

Выберите действие:`, versionInfo.Current, versionInfo.Latest)

	messageLabel := widget.NewLabel(message)
	messageLabel.Wrapping = fyne.TextWrapWord

	content := container.NewVBox(messageLabel)
	customDialog := dialog.NewCustom("Доступно обновление", "Отмена", content, v.app.window)

	// Создаем кнопки
	updateButton := widget.NewButton("Обновить текущую версию", func() {
		customDialog.Hide()
		v.updateCurrentVersion(versionInfo)
	})
	updateButton.Importance = widget.HighImportance

	downloadButton := widget.NewButton("Скачать отдельно", func() {
		customDialog.Hide()
		v.downloadSeparately(versionInfo)
	})

	cancelButton := widget.NewButton("Отмена", func() {
		customDialog.Hide()
	})

	buttonContainer := container.NewGridWithColumns(3, updateButton, downloadButton, cancelButton)
	content.Add(widget.NewLabel(""))
	content.Add(buttonContainer)

	customDialog.Show()
}

// updateCurrentVersion обновляет текущую версию
func (v *HelpView) updateCurrentVersion(versionInfo *updates.VersionInfo) {
	assetsPath := v.app.configManager.GetAssetsPath()
	if assetsPath == "" {
		dialog.ShowError(fmt.Errorf("путь к ресурсам не установлен"), v.app.window)
		return
	}

	// Запоминаем состояние перед обновлением
	updateState := v.captureCurrentState()

	// Создаем прогресс диалог
	progressBar := widget.NewProgressBarInfinite()
	progressBar.Resize(fyne.NewSize(300, 30))
	statusLabel := widget.NewLabel("Загрузка обновления...")

	dialogContent := container.NewVBox(
		statusLabel,
		widget.NewLabel("Это может занять некоторое время."),
		progressBar,
	)

	progressDialog := dialog.NewCustomWithoutButtons("Обновление", dialogContent, v.app.window)
	progressDialog.Show()

	// Выполняем обновление в горутине
	go v.performUpdate(versionInfo, updateState, statusLabel, progressDialog)
}

// updateState хранит состояние приложения перед обновлением
type updateState struct {
	wasRunning       bool
	lastStrategyName domain.StrategyName
	gameFilter       bool
}

// captureCurrentState сохраняет текущее состояние приложения
func (v *HelpView) captureCurrentState() *updateState {
	state := &updateState{
		wasRunning: v.app.processManager.IsRunning(),
	}

	if state.wasRunning {
		processInfo := v.app.processManager.GetCurrentProcess()
		if processInfo != nil {
			state.lastStrategyName = processInfo.Strategy
		}
		if state.lastStrategyName == "" {
			state.lastStrategyName = v.app.configManager.GetLastStrategyName()
		}
	}

	state.gameFilter, _ = v.app.gameFilter.Get()
	return state
}

// performUpdate выполняет процесс обновления
func (v *HelpView) performUpdate(versionInfo *updates.VersionInfo, state *updateState, statusLabel *widget.Label, progressDialog dialog.Dialog) {
	// Создаем временную директорию
	tempDir, err := os.MkdirTemp("", "zapret_update_*")
	if err != nil {
		v.showUpdateError(progressDialog, "ошибка создания временной директории", err)
		return
	}
	defer os.RemoveAll(tempDir)

	// Загружаем архив
	v.updateStatus(statusLabel, "Загрузка архива...")
	result, err := v.app.updateService.DownloadLatestRelease(versionInfo.Current, tempDir)
	if err != nil {
		v.showUpdateError(progressDialog, "ошибка загрузки обновления", err)
		return
	}

	// Останавливаем процесс если был запущен
	if state.wasRunning {
		v.updateStatus(statusLabel, "Остановка процесса...")
		if err := v.app.processManager.StopProcess(); err != nil {
			v.showUpdateError(progressDialog, "ошибка остановки процесса", err)
			return
		}
	}

	// Распаковываем архив
	v.updateStatus(statusLabel, "Распаковка архива...")
	extractedPath, err := v.extractArchive(result)
	if err != nil {
		v.showUpdateError(progressDialog, "ошибка распаковки архива", err)
		return
	}

	// Обновляем конфигурацию
	v.updateStatus(statusLabel, "Обновление конфигурации...")
	newAssetsPath := domain.AssetsPath(extractedPath)
	if err := v.app.configManager.SetAssetsPath(newAssetsPath); err != nil {
		v.showUpdateError(progressDialog, "ошибка обновления пути к ресурсам", err)
		return
	}

	// Подготавливаем рабочую директорию
	v.updateStatus(statusLabel, "Подготовка рабочей директории...")
	if err := v.app.configManager.PrepareWorkingDirectory(); err != nil {
		v.showUpdateError(progressDialog, "ошибка подготовки рабочей директории", err)
		return
	}

	// Перезагружаем стратегии
	v.updateStatus(statusLabel, "Загрузка стратегий...")
	fyne.Do(func() {
		v.app.loadStrategies(newAssetsPath)
	})

	// Запускаем стратегию если была запущена
	if state.wasRunning && state.lastStrategyName != "" {
		if err := v.restartStrategy(state, newAssetsPath, statusLabel); err != nil {
			v.showUpdateError(progressDialog, "ошибка запуска стратегии", err)
			return
		}
	}

	// Показываем успешное завершение
	v.showUpdateSuccess(progressDialog, versionInfo, newAssetsPath, state)
}

// extractArchive распаковывает архив
func (v *HelpView) extractArchive(result *updates.DownloadResult) (string, error) {
	assetsPath := v.app.configManager.GetAssetsPath()
	parentDir := filepath.Dir(assetsPath.String())

	versionDirName := strings.TrimSuffix(result.FileName, filepath.Ext(result.FileName))
	versionDir := filepath.Join(parentDir, versionDirName)

	if strings.HasSuffix(result.FileName, ".rar") {
		return utils.ExtractRar(result.FilePath, versionDir)
	} else if strings.HasSuffix(result.FileName, ".zip") {
		return utils.ExtractZip(result.FilePath, versionDir)
	}

	return "", fmt.Errorf("неподдерживаемый формат архива: %s", result.FileName)
}

// restartStrategy перезапускает стратегию после обновления
func (v *HelpView) restartStrategy(state *updateState, newAssetsPath domain.AssetsPath, statusLabel *widget.Label) error {
	v.updateStatus(statusLabel, "Запуск стратегии...")

	strategy, err := v.app.strategyService.GetByName(state.lastStrategyName)
	if err != nil {
		return fmt.Errorf("ошибка получения стратегии '%s': %w", state.lastStrategyName, err)
	}

	if err := v.app.processManager.StartStrategy(strategy, newAssetsPath, state.gameFilter); err != nil {
		return fmt.Errorf("ошибка запуска стратегии '%s': %w", state.lastStrategyName, err)
	}

	fyne.Do(func() {
		v.app.updateStatus()
	})

	return nil
}

// updateStatus обновляет статус в UI
func (v *HelpView) updateStatus(statusLabel *widget.Label, message string) {
	fyne.Do(func() {
		statusLabel.SetText(message)
	})
}

// showUpdateError показывает ошибку обновления
func (v *HelpView) showUpdateError(progressDialog dialog.Dialog, context string, err error) {
	fyne.Do(func() {
		progressDialog.Hide()
		dialog.ShowError(fmt.Errorf("%s:\n%w", context, err), v.app.window)
	})
}

// showUpdateSuccess показывает успешное завершение обновления
func (v *HelpView) showUpdateSuccess(progressDialog dialog.Dialog, versionInfo *updates.VersionInfo, newAssetsPath domain.AssetsPath, state *updateState) {
	fyne.Do(func() {
		progressDialog.Hide()

		var successMessage string
		if state.wasRunning && state.lastStrategyName != "" {
			successMessage = fmt.Sprintf(`Обновление успешно установлено!

Новая версия: %s
Путь: %s

Стратегия '%s' автоматически запущена.`, versionInfo.Latest, newAssetsPath, state.lastStrategyName)
		} else {
			successMessage = fmt.Sprintf(`Обновление успешно установлено!

Новая версия: %s
Путь: %s

Приложение готово к работе с новой версией.`, versionInfo.Latest, newAssetsPath)
		}

		dialog.ShowInformation("Обновление завершено", successMessage, v.app.window)
	})
}

// downloadSeparately скачивает обновление в выбранную папку
func (v *HelpView) downloadSeparately(versionInfo *updates.VersionInfo) {
	fileDialog := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil || uri == nil {
			return
		}
		v.performDownload(versionInfo, uri.Path())
	}, v.app.window)

	fileDialog.Resize(fyne.NewSize(800, 600))
	fileDialog.Show()
}

// performDownload выполняет загрузку в указанную директорию
func (v *HelpView) performDownload(versionInfo *updates.VersionInfo, downloadPath string) {
	progressDialog := v.createProgressDialog("Загрузка", "Загрузка обновления...")
	progressDialog.Show()

	go func() {
		result, err := v.app.updateService.DownloadLatestRelease(versionInfo.Current, downloadPath)

		fyne.Do(func() {
			progressDialog.Hide()

			if err != nil {
				dialog.ShowError(fmt.Errorf("ошибка загрузки обновления:\n%w", err), v.app.window)
				return
			}

			successMessage := fmt.Sprintf(`Обновление успешно загружено!

Файл: %s
Путь: %s

Распакуйте архив и используйте новую версию.`, result.FileName, result.FilePath)

			dialog.ShowInformation("Загрузка завершена", successMessage, v.app.window)
		})
	}()
}

// createProgressDialog создает диалог с прогресс-баром
func (v *HelpView) createProgressDialog(title, message string) dialog.Dialog {
	progressBar := widget.NewProgressBarInfinite()
	progressBar.Resize(fyne.NewSize(300, 30))

	dialogContent := container.NewVBox(
		widget.NewLabel(message),
		widget.NewLabel("Это может занять некоторое время."),
		progressBar,
	)

	return dialog.NewCustomWithoutButtons(title, dialogContent, v.app.window)
}
