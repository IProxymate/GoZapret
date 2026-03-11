package ui

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/IProxymate/GoZapret/internal/app"
	"github.com/IProxymate/GoZapret/internal/config"
	"github.com/IProxymate/GoZapret/internal/domain"
	"github.com/IProxymate/GoZapret/internal/services/updates"
	"github.com/IProxymate/GoZapret/internal/utils"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

var extCfg = config.GetExternalConfig()

// NewGitHubClientForSelfUpdate создаёт клиент GitHub для проверки обновлений GoZapret
func NewGitHubClientForSelfUpdate() *updates.GitHubClient {
	return updates.NewGitHubClient(extCfg.GoZapretAPIURL())
}

// HelpView представляет функционал меню помощи
type HelpView struct {
	app *app.App
}

// NewHelpView создает новый HelpView
func NewHelpView(a *app.App) *HelpView {
	return &HelpView{
		app: a,
	}
}

// ShowAbout показывает окно "О программе"
func (v *HelpView) ShowAbout() {
	cfg := v.app.Services.Config.GetConfig()

	// Создаём отдельное окно вместо диалога
	aboutWindow := v.app.FyneApp.NewWindow("О программе")
	aboutWindow.Resize(fyne.NewSize(550, 500))
	aboutWindow.CenterOnScreen()

	// Заголовок
	titleLabel := widget.NewLabelWithStyle("GoZapret", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	subtitleLabel := widget.NewLabelWithStyle("Графический интерфейс для winws (zapret)", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})

	// Версия
	versionCard := widget.NewCard("", "", container.NewVBox(
		container.NewHBox(
			widget.NewLabel("Версия приложения:"),
			widget.NewLabelWithStyle(cfg.Version, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		),
		container.NewHBox(
			widget.NewLabel("Путь к ресурсам:"),
			widget.NewLabel(v.truncatePath(cfg.AssetsPath.String(), 40)),
		),
	))

	// Описание
	descriptionText := `GoZapret предоставляет удобный графический интерфейс для управления zapret - инструментом для обхода DPI-блокировок интернет-ресурсов.

Приложение позволяет легко переключаться между стратегиями обхода, настраивать параметры и диагностировать проблемы.`

	descLabel := widget.NewLabel(descriptionText)
	descLabel.Wrapping = fyne.TextWrapWord

	// Возможности
	featuresText := `• Запуск и остановка стратегий zapret
• Выбор стратегии из списка доступных
• Режим Game Filter для игрового трафика
• Управление режимами IPset (any/none/loaded)
• Диагностика системы и проверка конфликтов
• Проверка доступности доменов
• Мониторинг приложения
• Очистка кэша Discord
• Автозапуск при старте системы
• Автоматическое обновление
• Редактирование списков доменов`

	featuresLabel := widget.NewLabel(featuresText)
	featuresCard := widget.NewCard("Возможности", "", featuresLabel)

	// Ссылки
	repoLink := widget.NewHyperlink("GitHub: zapret-discord-youtube", parseURL(extCfg.ZapretResourcesGitHubURL()))
	guiRepoLink := widget.NewHyperlink("GitHub: GoZapret (GUI)", parseURL(extCfg.GoZapretGitHubURL()))
	issuesLink := widget.NewHyperlink("Сообщить о проблеме", parseURL(extCfg.GoZapretIssuesURL()))
	originalLink := widget.NewHyperlink("Оригинальный zapret (bol-van)", parseURL(extCfg.OriginalZapretGitHubURL()))

	linksCard := widget.NewCard("Ссылки", "", container.NewVBox(
		guiRepoLink,
		repoLink,
		issuesLink,
		originalLink,
	))

	// Благодарности
	thanksText := `• bol-van - автор оригинального zapret
• Flowseal - адаптация для Discord/YouTube
• Сообщество - тестирование и обратная связь`
	thanksLabel := widget.NewLabel(thanksText)
	thanksCard := widget.NewCard("Благодарности", "", thanksLabel)

	// Копирайт
	copyrightLabel := widget.NewLabelWithStyle("© 2024-2025 GoZapret", fyne.TextAlignCenter, fyne.TextStyle{})

	// Кнопка закрытия
	closeButton := widget.NewButton("Закрыть", func() {
		aboutWindow.Close()
	})

	// Компоновка
	content := container.NewVBox(
		titleLabel,
		subtitleLabel,
		widget.NewSeparator(),
		versionCard,
		widget.NewSeparator(),
		descLabel,
		widget.NewSeparator(),
		featuresCard,
		linksCard,
		thanksCard,
		widget.NewSeparator(),
		copyrightLabel,
		closeButton,
	)

	scroll := container.NewScroll(content)
	aboutWindow.SetContent(container.NewPadded(scroll))
	aboutWindow.Show()
}

// truncatePath обрезает путь если он слишком длинный
func (v *HelpView) truncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	if path == "" {
		return "Не задан"
	}
	return "..." + path[len(path)-maxLen+3:]
}

// parseURL парсит URL для гиперссылки
func parseURL(urlStr string) *url.URL {
	u, _ := url.Parse(urlStr)
	return u
}

// updateCheckResult хранит результат проверки обновления
type updateCheckResult struct {
	name       string
	hasUpdate  bool
	currentVer string
	latestVer  string
	err        error
	updateFunc func() // функция для выполнения обновления
}

// CheckAllUpdates проверяет все типы обновлений и показывает результат
func (v *HelpView) CheckAllUpdates() {
	progressDialog := v.createProgressDialog("Проверка обновлений", "Проверка всех обновлений...")
	progressDialog.Show()

	go func() {
		results := make([]*updateCheckResult, 3)
		done := make(chan int, 3)

		// Проверяем обновление приложения GoZapret
		go func() {
			result := &updateCheckResult{name: "GoZapret"}
			client := NewGitHubClientForSelfUpdate()
			release, err := client.GetLatestRelease()
			if err != nil {
				result.err = err
			} else {
				currentVersion := v.app.Services.SelfUpdate.GetCurrentVersion()
				latestVersion := strings.TrimPrefix(release.TagName, "v")
				result.currentVer = currentVersion
				result.latestVer = latestVersion
				result.hasUpdate = latestVersion > currentVersion && latestVersion != currentVersion
				if result.hasUpdate {
					result.updateFunc = func() {
						v.showAppUpdateDialog(latestVersion)
					}
				}
			}
			results[0] = result
			done <- 0
		}()

		// Проверяем обновление ресурсов zapret
		go func() {
			result := &updateCheckResult{name: "Ресурсы zapret"}
			versionInfo, err := v.checkVersion()
			if err != nil {
				result.err = err
			} else {
				result.currentVer = versionInfo.Current
				result.latestVer = versionInfo.Latest
				result.hasUpdate = versionInfo.IsNewer
				if result.hasUpdate {
					result.updateFunc = func() {
						v.showUpdateDialog(versionInfo)
					}
				}
			}
			results[1] = result
			done <- 1
		}()

		// IPset всегда можно обновить (нет проверки версии)
		go func() {
			result := &updateCheckResult{
				name:       "Список IPset",
				hasUpdate:  true, // Всегда доступно для обновления
				currentVer: "—",
				latestVer:  "Доступно",
				updateFunc: func() {
					v.UpdateIpsetList()
				},
			}
			results[2] = result
			done <- 2
		}()

		// Ждём завершения всех проверок
		for i := 0; i < 3; i++ {
			<-done
		}

		fyne.Do(func() {
			progressDialog.Hide()
			v.showAllUpdatesResult(results)
		})
	}()
}

// showAllUpdatesResult показывает результаты проверки всех обновлений
func (v *HelpView) showAllUpdatesResult(results []*updateCheckResult) {
	// Создаём окно с результатами
	updatesWindow := v.app.FyneApp.NewWindow("Обновления")
	updatesWindow.Resize(fyne.NewSize(600, 500))
	updatesWindow.CenterOnScreen()

	// Заголовок
	titleLabel := widget.NewLabelWithStyle("Результаты проверки обновлений", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	// Контейнер для карточек
	cardsContainer := container.NewVBox()

	hasAnyUpdate := false
	for _, result := range results {
		if result == nil {
			continue
		}

		var statusText string
		var statusIcon string
		var button *widget.Button

		if result.err != nil {
			statusIcon = "❌"
			statusText = fmt.Sprintf("Ошибка: %v", result.err)
		} else if result.hasUpdate {
			hasAnyUpdate = true
			statusIcon = "🔄"
			if result.name == "Список IPset" {
				statusText = "Можно обновить"
			} else {
				statusText = fmt.Sprintf("%s → %s", result.currentVer, result.latestVer)
			}
			updateFunc := result.updateFunc
			button = widget.NewButton("Обновить", func() {
				updatesWindow.Hide()
				if updateFunc != nil {
					updateFunc()
				}
			})
			button.Importance = widget.HighImportance
		} else {
			statusIcon = "✅"
			statusText = fmt.Sprintf("Актуальная версия: %s", result.currentVer)
		}

		// Создаём строку с информацией
		nameLabel := widget.NewLabelWithStyle(fmt.Sprintf("%s %s", statusIcon, result.name), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		statusLabel := widget.NewLabel(statusText)

		var cardContent fyne.CanvasObject
		if button != nil {
			cardContent = container.NewBorder(nil, nil, nil, button,
				container.NewVBox(nameLabel, statusLabel))
		} else {
			cardContent = container.NewVBox(nameLabel, statusLabel)
		}

		card := widget.NewCard("", "", cardContent)
		cardsContainer.Add(card)
	}

	// Итоговое сообщение
	var summaryText string
	if hasAnyUpdate {
		summaryText = "Доступны обновления. Нажмите кнопку для установки."
	} else {
		summaryText = "Все компоненты актуальны."
	}
	summaryLabel := widget.NewLabel(summaryText)
	summaryLabel.Alignment = fyne.TextAlignCenter

	// Кнопка закрытия
	closeButton := widget.NewButton("Закрыть", func() {
		updatesWindow.Close()
	})

	// Компоновка
	content := container.NewVBox(
		titleLabel,
		widget.NewSeparator(),
		cardsContainer,
		widget.NewSeparator(),
		summaryLabel,
		closeButton,
	)

	scroll := container.NewScroll(content)
	updatesWindow.SetContent(container.NewPadded(scroll))
	updatesWindow.Show()
}

// CheckForUpdates проверяет наличие обновлений ресурсов zapret
func (v *HelpView) CheckForUpdates() {
	progressDialog := v.createProgressDialog("Проверка обновлений", "Проверка обновлений ресурсов zapret...")
	progressDialog.Show()

	go func() {
		versionInfo, err := v.checkVersion()

		fyne.Do(func() {
			progressDialog.Hide()
			v.handleVersionCheckResult(versionInfo, err)
		})
	}()
}

// CheckForAppUpdates проверяет наличие обновлений самого приложения GoZapret
func (v *HelpView) CheckForAppUpdates() {
	progressDialog := v.createProgressDialog("Проверка обновлений", "Проверка обновлений GoZapret...")
	progressDialog.Show()

	v.app.Services.SelfUpdate.CheckForUpdatesManual(
		// onUpdateAvailable
		func(newVersion string) {
			progressDialog.Hide()
			v.showAppUpdateDialog(newVersion)
		},
		// onNoUpdate
		func() {
			progressDialog.Hide()
			currentVersion := v.app.Services.SelfUpdate.GetCurrentVersion()
			dialog.ShowInformation("Обновления",
				fmt.Sprintf("У вас установлена последняя версия GoZapret: %s", currentVersion),
				v.app.Window)
		},
		// onError
		func(err error) {
			progressDialog.Hide()
			dialog.ShowError(fmt.Errorf("ошибка проверки обновлений:\n%w", err), v.app.Window)
		},
	)
}

// showAppUpdateDialog показывает диалог с предложением обновить приложение
func (v *HelpView) showAppUpdateDialog(newVersion string) {
	currentVersion := v.app.Services.SelfUpdate.GetCurrentVersion()

	message := fmt.Sprintf(`Доступна новая версия GoZapret!

Текущая версия: %s
Новая версия: %s

Хотите обновить приложение сейчас?
После обновления приложение будет перезапущено.`, currentVersion, newVersion)

	dialog.ShowConfirm("Доступно обновление", message,
		func(confirmed bool) {
			if confirmed {
				v.performAppUpdate(newVersion)
			}
		},
		v.app.Window)
}

// performAppUpdate выполняет обновление приложения
func (v *HelpView) performAppUpdate(newVersion string) {
	// Останавливаем winws перед обновлением
	if v.app.Services.Process.IsRunning() {
		v.app.Logger.Info("Остановка winws перед обновлением приложения")
		_ = v.app.Services.StrategyController.StopStrategy()
	}

	// Создаём диалог с прогрессом
	progressBar := widget.NewProgressBarInfinite()
	statusLabel := widget.NewLabel("Подготовка к обновлению...")

	dialogContent := container.NewVBox(
		statusLabel,
		progressBar,
	)

	progressDialog := dialog.NewCustomWithoutButtons("Обновление GoZapret", dialogContent, v.app.Window)
	progressDialog.Show()

	// Запускаем обновление асинхронно
	v.app.Services.SelfUpdate.PerformUpdateAsync(
		newVersion,
		// onProgress
		func(status string) {
			statusLabel.SetText(status)
		},
		// onSuccess
		func() {
			progressDialog.Hide()
			// Показываем диалог и закрываем приложение для замены exe
			dialog.ShowInformation("Обновление завершено",
				"Обновление загружено!\n\nПриложение будет перезапущено.",
				v.app.Window)
			// Даём время показать диалог и закрываем
			go func() {
				time.Sleep(2 * time.Second)
				v.app.FyneApp.Quit()
			}()
		},
		// onError
		func(err error) {
			progressDialog.Hide()
			dialog.ShowError(fmt.Errorf("ошибка обновления:\n%w", err), v.app.Window)
		},
	)
}

// checkVersion проверяет версию на GitHub
func (v *HelpView) checkVersion() (*updates.VersionInfo, error) {
	cfg := v.app.Services.Config.GetConfig()
	return v.app.Services.Update.CheckForUpdates(cfg.Version)
}

// handleVersionCheckResult обрабатывает результат проверки версии
func (v *HelpView) handleVersionCheckResult(versionInfo *updates.VersionInfo, err error) {
	if err != nil {
		dialog.ShowError(fmt.Errorf("ошибка проверки обновлений:\n%w", err), v.app.Window)
		return
	}

	if !versionInfo.IsNewer {
		dialog.ShowInformation("Обновления",
			fmt.Sprintf("У вас установлена последняя версия: %s", versionInfo.Current),
			v.app.Window)
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
	customDialog := dialog.NewCustom("Доступно обновление", "Отмена", content, v.app.Window)

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
	assetsPath := v.app.Services.Config.GetAssetsPath()
	if assetsPath == "" {
		dialog.ShowError(fmt.Errorf("путь к ресурсам не установлен"), v.app.Window)
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

	progressDialog := dialog.NewCustomWithoutButtons("Обновление", dialogContent, v.app.Window)
	progressDialog.Show()

	// Выполняем обновление в горутине
	go v.performUpdate(versionInfo, updateState, statusLabel, progressDialog)
}

// updateState хранит состояние приложения перед обновлением
type updateState struct {
	wasRunning       bool
	lastStrategyName domain.StrategyName
	gameFilterMode   string
}

// captureCurrentState сохраняет текущее состояние приложения
func (v *HelpView) captureCurrentState() *updateState {
	controller := v.app.Services.StrategyController

	state := &updateState{
		wasRunning: controller.IsRunning(),
	}

	if state.wasRunning {
		if currentName := controller.GetCurrentStrategyName(); currentName != "" {
			state.lastStrategyName = domain.StrategyName(currentName)
		}
		if state.lastStrategyName == "" {
			state.lastStrategyName = domain.StrategyName(controller.GetLastStrategyName())
		}
	}

	state.gameFilterMode, _ = v.app.State.GameFilterMode.Get()
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
	result, err := v.app.Services.Update.DownloadLatestRelease(versionInfo.Current, tempDir)
	if err != nil {
		v.showUpdateError(progressDialog, "ошибка загрузки обновления", err)
		return
	}

	// Останавливаем процесс - всегда пытаемся остановить, независимо от state.wasRunning
	// (процесс мог быть запущен извне или состояние могло быть неточным)
	v.updateStatus(statusLabel, "Остановка процесса...")

	// Сначала пытаемся остановить через контроллер
	_ = v.app.Services.StrategyController.StopStrategy()

	// Принудительно убиваем все процессы winws
	_ = utils.RunHidden("taskkill", "/F", "/IM", "winws.exe")

	// Ждём завершения процесса
	v.updateStatus(statusLabel, "Ожидание завершения процесса...")
	if err := v.waitForProcessTermination(5 * time.Second); err != nil {
		v.app.Logger.Warn("Процесс не завершился за отведённое время", "error", err)
	}

	// Выгружаем драйвер WinDivert и ждём освобождения файлов
	v.updateStatus(statusLabel, "Освобождение ресурсов...")
	v.unloadWinDivertDriver()

	// Даём больше времени на освобождение файлов драйвером
	time.Sleep(3 * time.Second)

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
	if err := v.app.Services.Config.SetAssetsPath(newAssetsPath); err != nil {
		v.showUpdateError(progressDialog, "ошибка обновления пути к ресурсам", err)
		return
	}

	// Подготавливаем рабочую директорию
	v.updateStatus(statusLabel, "Подготовка рабочей директории...")
	if err := v.app.Services.Config.PrepareWorkingDirectory(); err != nil {
		v.showUpdateError(progressDialog, "ошибка подготовки рабочей директории", err)
		return
	}

	// Обновляем список ipset
	v.updateStatus(statusLabel, "Обновление списка IPset...")
	workingDir := v.app.Services.Config.GetWorkingDir()
	if _, err := v.app.Services.Update.UpdateIpsetList(newAssetsPath.String(), workingDir); err != nil {
		// Не прерываем обновление из-за ошибки ipset, просто логируем
		v.app.Logger.Warn("Не удалось обновить список ipset", "error", err)
	}

	// Перезагружаем стратегии
	v.updateStatus(statusLabel, "Загрузка стратегий...")
	v.app.LoadStrategiesAndRefreshUI(newAssetsPath)

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
	assetsPath := v.app.Services.Config.GetAssetsPath()
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

	// Статус обновится автоматически через EventBus
	err := v.app.Services.StrategyController.StartStrategy(string(state.lastStrategyName), state.gameFilterMode)
	if err != nil {
		return fmt.Errorf("ошибка запуска стратегии '%s': %w", state.lastStrategyName, err)
	}

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
		dialog.ShowError(fmt.Errorf("%s:\n%w", context, err), v.app.Window)
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

		dialog.ShowInformation("Обновление завершено", successMessage, v.app.Window)
	})
}

// downloadSeparately скачивает обновление в выбранную папку
func (v *HelpView) downloadSeparately(versionInfo *updates.VersionInfo) {
	fileDialog := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil || uri == nil {
			return
		}
		v.performDownload(versionInfo, uri.Path())
	}, v.app.Window)

	fileDialog.Resize(fyne.NewSize(800, 600))
	fileDialog.Show()
}

// performDownload выполняет загрузку в указанную директорию
func (v *HelpView) performDownload(versionInfo *updates.VersionInfo, downloadPath string) {
	progressDialog := v.createProgressDialog("Загрузка", "Загрузка обновления...")
	progressDialog.Show()

	go func() {
		result, err := v.app.Services.Update.DownloadLatestRelease(versionInfo.Current, downloadPath)

		fyne.Do(func() {
			progressDialog.Hide()

			if err != nil {
				dialog.ShowError(fmt.Errorf("ошибка загрузки обновления:\n%w", err), v.app.Window)
				return
			}

			successMessage := fmt.Sprintf(`Обновление успешно загружено!

Файл: %s
Путь: %s

Распакуйте архив и используйте новую версию.`, result.FileName, result.FilePath)

			dialog.ShowInformation("Загрузка завершена", successMessage, v.app.Window)
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

	return dialog.NewCustomWithoutButtons(title, dialogContent, v.app.Window)
}

// UpdateIpsetList обновляет список ipset из удалённого источника
func (v *HelpView) UpdateIpsetList() {
	progressDialog := v.createProgressDialog("Обновление IPset", "Загрузка списка IPset...")
	progressDialog.Show()

	go func() {
		assetsPath := v.app.Services.Config.GetAssetsPath()
		workingDir := v.app.Services.Config.GetWorkingDir()

		result, err := v.app.Services.Update.UpdateIpsetList(assetsPath.String(), workingDir)

		fyne.Do(func() {
			progressDialog.Hide()

			if err != nil {
				dialog.ShowError(fmt.Errorf("ошибка обновления списка IPset:\n%w", err), v.app.Window)
				return
			}

			// Формируем итоговый ipset-all.txt с учётом текущего режима и пользовательских подсетей
			mode, _ := v.app.State.IpsetMode.Get()
			if mode == "" {
				mode = "loaded"
			}
			if workingDir != "" {
				v.app.Services.Ipset.UpdateIpsetFile(workingDir, mode)
			}

			// Перезапускаем стратегию если запущена
			v.restartStrategyIfRunning()

			var filesInfo string
			if len(result.UpdatedFiles) > 0 {
				filesInfo = fmt.Sprintf("\n\nОбновлено файлов: %d", len(result.UpdatedFiles))
			}

			dialog.ShowInformation("Обновление IPset",
				fmt.Sprintf("Список IPset успешно обновлён!%s", filesInfo),
				v.app.Window)
		})
	}()
}

// restartStrategyIfRunning перезапускает стратегию если она запущена.
// Делегирует вызов App.RestartCurrentStrategy()
func (v *HelpView) restartStrategyIfRunning() {
	v.app.RestartCurrentStrategy()
}

// waitForProcessTermination ожидает завершения процесса winws
func (v *HelpView) waitForProcessTermination(timeout time.Duration) error {
	checkInterval := 200 * time.Millisecond
	elapsed := time.Duration(0)

	for elapsed < timeout {
		if !v.app.Services.Process.IsWinwsProcessRunning() {
			return nil
		}
		time.Sleep(checkInterval)
		elapsed += checkInterval
	}

	// Ещё одна попытка принудительного завершения
	_ = utils.RunHidden("taskkill", "/F", "/IM", "winws.exe")
	time.Sleep(500 * time.Millisecond)

	if v.app.Services.Process.IsWinwsProcessRunning() {
		return fmt.Errorf("процесс winws всё ещё запущен после %v", timeout)
	}
	return nil
}

// unloadWinDivertDriver пытается выгрузить драйвер WinDivert
func (v *HelpView) unloadWinDivertDriver() {
	// Пытаемся остановить службу WinDivert (если она запущена как служба)
	_ = utils.RunHidden("sc", "stop", "WinDivert")
	_ = utils.RunHidden("sc", "stop", "WinDivert14")

	// Пытаемся выгрузить драйвер через pnputil (может потребовать прав администратора)
	_ = utils.RunHidden("sc", "delete", "WinDivert")
	_ = utils.RunHidden("sc", "delete", "WinDivert14")
}
