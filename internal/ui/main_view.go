package ui

import (
	"fmt"

	"github.com/IProxymate/GoZapret/internal/domain"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// MainView представляет главный вид приложения
type MainView struct {
	app      *App
	helpView *HelpView

	// Флаг завершения инициализации UI
	uiInitialized bool

	// Виджет выбора стратегии для обновления извне
	strategySelect *widget.Select
}

// NewMainView создает новый главный вид
func NewMainView(app *App) *MainView {
	return &MainView{
		app:           app,
		helpView:      NewHelpView(app),
		uiInitialized: false,
	}
}

// Build строит UI главного вида
func (v *MainView) Build() fyne.CanvasObject {
	// Создаем меню
	mainMenu := v.buildMenu()
	v.app.window.SetMainMenu(mainMenu)

	// Статус
	statusLabel := widget.NewLabelWithData(v.app.statusText)
	statusLabel.Importance = widget.MediumImportance
	statusCard := widget.NewCard("Статус", "", statusLabel)

	// Управление
	controlCard := v.buildControlCard()

	// Дополнительно
	additionalCard := v.buildAdditionalCard()

	// Информация
	infoCard := v.buildInfoCard()

	// Главный контейнер
	content := container.NewVBox(
		statusCard,
		controlCard,
		additionalCard,
		infoCard,
	)

	// Устанавливаем флаг после полной инициализации UI
	// чтобы дать время всем биндингам стабилизироваться
	go func() {
		fyne.Do(func() {
			v.uiInitialized = true
		})
	}()

	return container.NewPadded(content)
}

// buildMenu создает главное меню
func (v *MainView) buildMenu() *fyne.MainMenu {
	// Меню "Файл"
	fileMenu := fyne.NewMenu("Файл",
		fyne.NewMenuItem("Настройки", v.showSettings),
	)

	// Меню "Списки"
	listsMenu := fyne.NewMenu("Списки",
		fyne.NewMenuItem("Включенные домены", v.showIncludedDomains),
		fyne.NewMenuItem("Исключенные домены", v.showExcludedDomains),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Пользовательские подсети (IPset)", v.showCustomIpset),
	)

	// Меню "Инструменты"
	toolsMenu := fyne.NewMenu("Инструменты",
		fyne.NewMenuItem("Проверить домен", v.showDomainCheck),
		fyne.NewMenuItem("Мониторинг приложения", v.showAppMonitor),
	)

	// Меню "Помощь"
	helpMenu := fyne.NewMenu("Помощь",
		fyne.NewMenuItem("Проверить обновления", v.helpView.CheckForUpdates),
		fyne.NewMenuItem("Обновить список IPset", v.helpView.UpdateIpsetList),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("О программе", v.helpView.ShowAbout),
	)

	return fyne.NewMainMenu(fileMenu, listsMenu, toolsMenu, helpMenu)
}

// buildControlCard создает карточку управления
func (v *MainView) buildControlCard() *widget.Card {
	// Создаем компоненты
	strategySelect := v.createStrategySelect()
	gameFilterCheck := v.createGameFilterCheck()
	ipsetSelect := v.createIpsetSelect()
	buttonContainer := v.createControlButtons()

	// Контейнер для фильтров
	filterContainer := container.NewCenter(
		container.NewHBox(
			gameFilterCheck,
			widget.NewLabel("Режим IPset:"),
			ipsetSelect,
		),
	)

	// Основной контент карточки
	content := container.NewVBox(
		strategySelect,
		filterContainer, // centered filter container with padding
		widget.NewLabel(""),
		buttonContainer,
	)

	return widget.NewCard("Управление", "", content)
}

// createStrategySelect создает выпадающий список для выбора стратегии
func (v *MainView) createStrategySelect() *widget.Select {
	// Получаем начальный список стратегий
	items, _ := v.app.strategies.Get()
	if items == nil {
		items = []string{}
	}

	v.strategySelect = widget.NewSelect(items, func(s string) {
		// Игнорируем изменения до завершения инициализации UI
		if !v.uiInitialized {
			return
		}

		v.app.selectedStrategy.Set(s)
		v.app.configManager.SetLastStrategyName(domain.StrategyName(s))

		// Перезапускаем стратегию только если процесс уже запущен
		if !v.app.processManager.IsRunning() && !v.app.processManager.IsWinwsProcessRunning() {
			return
		}

		// Получаем новую стратегию
		strategy, err := v.app.strategyService.GetByName(domain.StrategyName(s))
		if err != nil {
			return
		}

		assetsPath := v.app.configManager.GetAssetsPath()
		if assetsPath == "" {
			return
		}

		// Получаем текущее состояние GameFilter
		gameFilter, _ := v.app.gameFilter.Get()

		// Перезапускаем стратегию в фоне
		go func() {
			err := v.app.processManager.RestartStrategy(strategy, assetsPath, gameFilter)
			if err != nil {
				fyne.Do(func() {
					dialog.ShowError(fmt.Errorf("ошибка перезапуска стратегии: %v", err), v.app.window)
				})
			}
			// Обновляем статус после перезапуска
			v.app.updateStatus()
		}()
	})

	// Привязываем выбранную стратегию
	v.app.selectedStrategy.AddListener(binding.NewDataListener(func() {
		selected, _ := v.app.selectedStrategy.Get()
		if v.strategySelect != nil {
			v.strategySelect.SetSelected(selected)
		}
	}))

	// Устанавливаем начальные значения
	selected, _ := v.app.selectedStrategy.Get()
	if selected != "" {
		v.strategySelect.SetSelected(selected)
	}
	return v.strategySelect
}

// UpdateStrategyOptions обновляет список стратегий в виджете (вызывается из App)
func (v *MainView) UpdateStrategyOptions(items []string, selectedStrategy string) {
	if v.strategySelect == nil {
		return
	}

	v.app.logger.Debug("UpdateStrategyOptions: обновление виджета", "count", len(items), "selected", selectedStrategy)

	v.strategySelect.Options = items

	// Проверяем, существует ли выбранная стратегия в новом списке
	found := false
	for _, item := range items {
		if item == selectedStrategy {
			found = true
			break
		}
	}

	if found && selectedStrategy != "" {
		v.strategySelect.SetSelected(selectedStrategy)
	} else if len(items) > 0 {
		v.strategySelect.SetSelected(items[0])
	}

	v.strategySelect.Refresh()
}

// createGameFilterCheck создает чекбокс для Game Filter
func (v *MainView) createGameFilterCheck() *widget.Check {
	// Получаем начальное значение из конфига
	initialValue, _ := v.app.gameFilter.Get()

	// Создаем обычный чекбокс без привязки данных
	gameFilterCheck := widget.NewCheck("Game Filter", func(checked bool) {
		// Игнорируем изменения до завершения инициализации UI
		if !v.uiInitialized {
			return
		}

		// Сохраняем новое значение в конфиг
		v.app.configManager.SetGameFilter(checked)
		v.app.gameFilter.Set(checked)

		// Перезапускаем стратегию только если процесс уже запущен
		if !v.app.processManager.IsRunning() && !v.app.processManager.IsWinwsProcessRunning() {
			return
		}

		// Получаем текущую стратегию
		strategyName, _ := v.app.selectedStrategy.Get()
		if strategyName == "" {
			return
		}

		strategy, err := v.app.strategyService.GetByName(domain.StrategyName(strategyName))
		if err != nil {
			return
		}

		assetsPath := v.app.configManager.GetAssetsPath()
		if assetsPath == "" {
			return
		}

		// Перезапускаем стратегию в фоне
		go func() {
			err := v.app.processManager.RestartStrategy(strategy, assetsPath, checked)
			if err != nil {
				fyne.Do(func() {
					dialog.ShowError(fmt.Errorf("ошибка перезапуска стратегии: %v", err), v.app.window)
				})
			}
			// Обновляем статус после перезапуска
			v.app.updateStatus()
		}()
	})

	// Устанавливаем начальное значение
	gameFilterCheck.Checked = initialValue

	return gameFilterCheck
}

// createIpsetSelect создает выпадающий список для выбора режима IPset
func (v *MainView) createIpsetSelect() *widget.Select {
	// Создаем Select с начальными опциями
	options := []string{"any", "none", "loaded"}
	ipsetSelect := widget.NewSelect(options, func(mode string) {
		// Игнорируем изменения до завершения инициализации UI
		if !v.uiInitialized {
			return
		}

		// Сохраняем новое значение в конфиг
		v.app.ipsetMode.Set(mode)
		v.app.configManager.SetIpsetMode(mode)

		// Обновляем файл ipset при изменении режима
		workingDir := v.app.configManager.GetWorkingDir()
		if workingDir != "" {
			v.app.ipsetService.UpdateIpsetFile(workingDir, mode)
		}

		// Перезапускаем стратегию только если процесс уже запущен
		if !v.app.processManager.IsRunning() && !v.app.processManager.IsWinwsProcessRunning() {
			return
		}

		// Получаем текущую стратегию
		strategyName, _ := v.app.selectedStrategy.Get()
		if strategyName == "" {
			return
		}

		strategy, err := v.app.strategyService.GetByName(domain.StrategyName(strategyName))
		if err != nil {
			return
		}

		assetsPath := v.app.configManager.GetAssetsPath()
		if assetsPath == "" {
			return
		}

		// Получаем текущее состояние GameFilter
		gameFilter, _ := v.app.gameFilter.Get()

		// Перезапускаем стратегию в фоне
		go func() {
			err := v.app.processManager.RestartStrategy(strategy, assetsPath, gameFilter)
			if err != nil {
				fyne.Do(func() {
					dialog.ShowError(fmt.Errorf("ошибка перезапуска стратегии: %v", err), v.app.window)
				})
			}
			// Обновляем статус после перезапуска
			v.app.updateStatus()
		}()
	})

	// Привязываем режим ipset
	v.app.ipsetMode.AddListener(binding.NewDataListener(func() {
		mode, _ := v.app.ipsetMode.Get()
		ipsetSelect.SetSelected(mode)
	}))

	// Устанавливаем начальное значение
	mode, _ := v.app.ipsetMode.Get()
	if mode != "" {
		ipsetSelect.SetSelected(mode)
	}

	return ipsetSelect
}

// createControlButtons создает кнопки управления
func (v *MainView) createControlButtons() fyne.CanvasObject {
	startButton := widget.NewButton("Запустить", v.handleStart)
	startButton.Importance = widget.HighImportance

	stopButton := widget.NewButton("Остановить", v.handleStop)
	stopButton.Importance = widget.DangerImportance

	// Контейнер для кнопок - всегда показываем обе кнопки
	buttonContainer := container.NewGridWithColumns(2, startButton, stopButton)

	// Динамически включаем/отключаем кнопки в зависимости от состояния
	v.app.isRunning.AddListener(binding.NewDataListener(func() {
		running, _ := v.app.isRunning.Get()
		if running {
			startButton.Disable()
			stopButton.Enable()
		} else {
			startButton.Enable()
			stopButton.Disable()
		}
	}))

	// Инициализируем начальное состояние кнопок
	running, _ := v.app.isRunning.Get()
	if running {
		startButton.Disable()
		stopButton.Enable()
	} else {
		startButton.Enable()
		stopButton.Disable()
	}

	return buttonContainer
}

// buildAdditionalCard создает карточку дополнительных функций
func (v *MainView) buildAdditionalCard() *widget.Card {
	diagButton := widget.NewButton("Диагностика", v.runDiagnostics)
	clearCacheButton := widget.NewButton("Очистить кэш Discord", v.clearCache)
	reloadStrategiesButton := widget.NewButton("Перечитать список стратегий", v.reloadStrategies)

	content := container.NewGridWithColumns(3, diagButton, clearCacheButton, reloadStrategiesButton)
	return widget.NewCard("Дополнительно", "", content)
}

// buildInfoCard создает информационную карточку
func (v *MainView) buildInfoCard() *widget.Card {
	// Создаем функцию для генерации текста с актуальной версией
	getInfoText := func() string {
		version, _ := v.app.version.Get()
		return fmt.Sprintf(`**Zapret GUI** - Графический интерфейс для winws

**Версия:** %s

✅ Приложение готово к работе
`, version)
	}

	infoLabel := widget.NewRichTextFromMarkdown(getInfoText())

	// Добавляем слушатель для обновления текста при изменении версии
	v.app.version.AddListener(binding.NewDataListener(func() {
		infoLabel.ParseMarkdown(getInfoText())
		infoLabel.Refresh()
	}))

	return widget.NewCard("Информация", "", infoLabel)
}

// handleStart обрабатывает запуск стратегии
func (v *MainView) handleStart() {
	// Проверяем права администратора
	if !v.app.adminChecker.IsAdmin() {
		dialog.ShowInformation("Требуются права администратора",
			"Для работы winws требуются права администратора.\n"+
				"Пожалуйста, запустите приложение от имени администратора.",
			v.app.window)
		return
	}

	// Проверяем, не запущен ли уже процесс
	if v.app.processManager.IsRunning() {
		dialog.ShowInformation("Процесс уже запущен",
			"Процесс winws уже запущен. Остановите его перед запуском новой стратегии.",
			v.app.window)
		return
	}

	// Получаем выбранную стратегию
	strategyName, _ := v.app.selectedStrategy.Get()
	if strategyName == "" {
		dialog.ShowError(fmt.Errorf("стратегия не выбрана"), v.app.window)
		return
	}

	strategy, err := v.app.strategyService.GetByName(domain.StrategyName(strategyName))
	if err != nil {
		dialog.ShowError(fmt.Errorf("ошибка получения стратегии: %w", err), v.app.window)
		return
	}

	// Получаем путь к ресурсам
	assetsPath := v.app.configManager.GetAssetsPath()
	if assetsPath == "" {
		dialog.ShowError(fmt.Errorf("путь к ресурсам не установлен.\nПожалуйста, укажите путь в настройках"), v.app.window)
		return
	}

	// Получаем настройки
	gameFilter, _ := v.app.gameFilter.Get()

	// Запускаем стратегию
	err = v.app.processManager.StartStrategy(strategy, assetsPath, gameFilter)
	if err != nil {
		dialog.ShowError(fmt.Errorf("ошибка запуска стратегии:\n%w", err), v.app.window)
		// Обновляем статус даже при ошибке
		v.app.updateStatus()
		return
	}

	// Обновляем статус
	v.app.updateStatus()

	dialog.ShowInformation("Успех", fmt.Sprintf("Стратегия '%s' успешно запущена", strategyName), v.app.window)
}

// handleStop обрабатывает остановку процесса
func (v *MainView) handleStop() {
	// Проверяем, запущен ли процесс
	if !v.app.processManager.IsRunning() && !v.app.processManager.IsWinwsProcessRunning() {
		dialog.ShowInformation("Процесс не запущен",
			"Процесс winws не запущен.",
			v.app.window)
		return
	}

	// Останавливаем процесс в горутине
	err := v.app.processManager.StopProcess()

	// Все UI операции должны быть в главном потоке
	if err != nil {
		dialog.ShowError(fmt.Errorf("ошибка остановки процесса:\n%w", err), v.app.window)
		// Обновляем статус даже при ошибке
		v.app.updateStatus()
		return
	}

	// Обновляем статус
	v.app.updateStatus()

	dialog.ShowInformation("Успех", "Процесс успешно остановлен", v.app.window)
}

// runDiagnostics запускает диагностику
func (v *MainView) runDiagnostics() {
	// Создаем прогресс бар
	progressBar := widget.NewProgressBarInfinite()
	progressBar.Resize(fyne.NewSize(300, 30))

	// Создаем диалог с прогресс баром
	dialogContent := container.NewVBox(
		widget.NewLabel("Выполняется диагностика системы..."),
		progressBar,
	)

	dialog := dialog.NewCustomWithoutButtons("Диагностика", dialogContent, v.app.window)
	dialog.Show()

	// Запускаем диагностику в горутине
	go func() {
		results := v.app.diagnostics.RunAll()

		// Закрываем прогресс и показываем результаты
		fyne.Do(func() {
			dialog.Hide()
			v.showDiagnosticResults(results)
		})
	}()
}

// showDiagnosticResults показывает результаты диагностики
func (v *MainView) showDiagnosticResults(results []*domain.DiagnosticResult) {
	// Создаем таблицу с результатами
	table := widget.NewTable(
		func() (int, int) { return len(results), 3 },
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			if id.Row < len(results) {
				switch id.Col {
				case 0:
					label.SetText(results[id.Row].Name)
				case 1:
					if results[id.Row].Success {
						label.SetText("✓ Успех")
					} else {
						label.SetText("✗ Ошибка")
					}
				case 2:
					label.SetText(results[id.Row].Message)
				}
			}
		},
	)

	table.SetColumnWidth(0, 400)
	table.SetColumnWidth(1, 150)
	table.SetColumnWidth(2, 150)

	scroll := container.NewScroll(table)
	scroll.SetMinSize(fyne.NewSize(780, 450))

	dialog.ShowCustom("Результаты диагностики", "Закрыть", scroll, v.app.window)
}

// clearCache очищает кэш Discord
func (v *MainView) clearCache() {
	dialog.ShowConfirm("Подтверждение",
		"Вы уверены, что хотите очистить кэш Discord?\nDiscord будет закрыт.",
		func(confirmed bool) {
			if !confirmed {
				return
			}

			// Убиваем процессы Discord
			v.app.cacheService.KillDiscordProcesses()

			// Очищаем кэш
			if err := v.app.cacheService.ClearDiscordCache(); err != nil {
				dialog.ShowError(err, v.app.window)
				return
			}

			dialog.ShowInformation("Успех", "Кэш Discord успешно очищен", v.app.window)
		},
		v.app.window)
}

// showSettings показывает окно настроек
func (v *MainView) showSettings() {
	settingsView := NewSettingsView(v.app)
	settingsView.Show()
}

// showIncludedDomains показывает окно редактирования включенных доменов
func (v *MainView) showIncludedDomains() {
	filePath := v.app.configManager.GetExtraListPath()
	domainListView := NewDomainListView(v.app, filePath, "Включенные домены")
	domainListView.Show()
}

// showExcludedDomains показывает окно редактирования исключенных доменов
func (v *MainView) showExcludedDomains() {
	filePath := v.app.configManager.GetExcludeListPath()
	domainListView := NewDomainListView(v.app, filePath, "Исключенные домены")
	domainListView.Show()
}

// showCustomIpset показывает окно редактирования пользовательских подсетей
func (v *MainView) showCustomIpset() {
	filePath := v.app.configManager.GetCustomIpsetPath()
	ipsetListView := NewIpsetListView(v.app, filePath, "Пользовательские подсети (IPset)")
	ipsetListView.Show()
}

// showDomainCheck показывает окно проверки домена
func (v *MainView) showDomainCheck() {
	domainCheckView := NewDomainCheckView(v.app.fyneApp, v.app.window)
	domainCheckView.Show()
}

// showAppMonitor показывает окно мониторинга приложения
func (v *MainView) showAppMonitor() {
	appMonitorView := NewAppMonitorView(v.app)
	appMonitorView.Show()
}

// reloadStrategies перечитывает список стратегий и обновляет файлы в рабочей директории
func (v *MainView) reloadStrategies() {
	// Создаем прогресс бар
	progressBar := widget.NewProgressBarInfinite()
	progressBar.Resize(fyne.NewSize(300, 30))

	// Создаем диалог с прогресс баром
	dialogContent := container.NewVBox(
		widget.NewLabel("Обновление стратегий..."),
		progressBar,
	)

	progressDialog := dialog.NewCustomWithoutButtons("Обновление", dialogContent, v.app.window)
	progressDialog.Show()

	// Запускаем обновление в горутине
	go func() {
		err := v.app.ReloadStrategies()

		fyne.Do(func() {
			progressDialog.Hide()
			if err != nil {
				dialog.ShowError(err, v.app.window)
				return
			}
			dialog.ShowInformation("Успех", "Стратегии успешно обновлены", v.app.window)
		})
	}()
}
