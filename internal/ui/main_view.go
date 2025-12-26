package ui

import (
	"fmt"

	"github.com/IProxymate/GoZapret/internal/app"
	"github.com/IProxymate/GoZapret/internal/domain"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// MainView представляет главный вид приложения
type MainView struct {
	app      *app.App
	helpView *HelpView

	// Флаг завершения инициализации UI
	uiInitialized bool

	// Виджет выбора стратегии для обновления извне
	strategySelect *widget.Select

	// Кнопка выбора стратегии (показывает текущую выбранную)
	strategyButton *widget.Button
}

// NewMainView создает новый главный вид
func NewMainView(a *app.App) *MainView {
	return &MainView{
		app:           a,
		helpView:      NewHelpView(a),
		uiInitialized: false,
	}
}

// restartCurrentStrategyIfRunning перезапускает текущую стратегию, если процесс запущен.
// Делегирует вызов App.RestartCurrentStrategy()
func (v *MainView) restartCurrentStrategyIfRunning() {
	v.app.RestartCurrentStrategy()
}

// Build строит UI главного вида
func (v *MainView) Build() fyne.CanvasObject {
	// Создаем меню
	mainMenu := v.buildMenu()
	v.app.Window.SetMainMenu(mainMenu)

	// Статус
	statusLabel := widget.NewLabelWithData(v.app.State.StatusText)
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
		fyne.NewMenuItem("Обновить приложение", v.helpView.CheckForAppUpdates),
		fyne.NewMenuItem("Обновить ресурсы zapret", v.helpView.CheckForUpdates),
		fyne.NewMenuItem("Обновить список IPset", v.helpView.UpdateIpsetList),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("О программе", v.helpView.ShowAbout),
	)

	return fyne.NewMainMenu(fileMenu, listsMenu, toolsMenu, helpMenu)
}

// buildControlCard создает карточку управления
func (v *MainView) buildControlCard() *widget.Card {
	// Создаем компоненты
	strategyButton := v.createStrategyButton()
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
		strategyButton,
		filterContainer, // centered filter container with padding
		widget.NewLabel(""),
		buttonContainer,
	)

	return widget.NewCard("Управление", "", content)
}

// createStrategyButton создает кнопку для выбора стратегии
func (v *MainView) createStrategyButton() *widget.Button {
	// Получаем текущую выбранную стратегию
	selected, _ := v.app.State.SelectedStrategy.Get()
	buttonText := "Выбрать стратегию..."
	if selected != "" {
		buttonText = fmt.Sprintf("Стратегия: %s", selected)
	}

	v.strategyButton = widget.NewButton(buttonText, func() {
		v.showStrategySelectionDialog()
	})

	// Привязываем обновление текста кнопки к изменению выбранной стратегии
	v.app.State.SelectedStrategy.AddListener(binding.NewDataListener(func() {
		selected, _ := v.app.State.SelectedStrategy.Get()
		if v.strategyButton != nil {
			if selected != "" {
				v.strategyButton.SetText(fmt.Sprintf("Стратегия: %s", selected))
			} else {
				v.strategyButton.SetText("Выбрать стратегию...")
			}
		}
	}))

	return v.strategyButton
}

// showStrategySelectionDialog показывает диалог выбора стратегии
func (v *MainView) showStrategySelectionDialog() {
	// Получаем список стратегий
	items, _ := v.app.State.Strategies.Get()
	if items == nil || len(items) == 0 {
		dialog.ShowInformation("Нет стратегий", "Список стратегий пуст. Пожалуйста, укажите путь к ресурсам.", v.app.Window)
		return
	}

	// Получаем текущую выбранную стратегию
	currentSelected, _ := v.app.State.SelectedStrategy.Get()
	tempSelected := currentSelected

	// Создаем список для выбора
	strategyList := widget.NewList(
		func() int {
			return len(items)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("template strategy name")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			if id < len(items) {
				label.SetText(items[id])
			}
		},
	)

	// Устанавливаем начальный выбор
	for i, item := range items {
		if item == currentSelected {
			strategyList.Select(i)
			break
		}
	}

	// Обработчик выбора элемента
	strategyList.OnSelected = func(id widget.ListItemID) {
		if id < len(items) {
			tempSelected = items[id]
		}
	}

	// Создаем контейнер со списком
	listContainer := container.NewScroll(strategyList)
	listContainer.SetMinSize(fyne.NewSize(400, 300))

	// Создаем диалог с кнопками
	var customDialog dialog.Dialog

	applyButton := widget.NewButton("Применить", func() {
		if tempSelected != "" && tempSelected != currentSelected {
			// Применяем выбор
			v.app.State.SelectedStrategy.Set(tempSelected)
			v.app.Services.Config.SetLastStrategyName(domain.StrategyName(tempSelected))

			// Перезапускаем стратегию, если процесс запущен
			v.restartCurrentStrategyIfRunning()
		}
		customDialog.Hide()
	})
	applyButton.Importance = widget.HighImportance

	cancelButton := widget.NewButton("Отменить", func() {
		customDialog.Hide()
	})

	buttonContainer := container.NewGridWithColumns(2, applyButton, cancelButton)

	content := container.NewBorder(
		widget.NewLabel("Выберите стратегию из списка:"),
		buttonContainer,
		nil,
		nil,
		listContainer,
	)

	customDialog = dialog.NewCustomWithoutButtons("Выбор стратегии", content, v.app.Window)
	customDialog.Resize(fyne.NewSize(450, 400))
	customDialog.Show()
}

// createStrategySelect создает выпадающий список для выбора стратегии (для обратной совместимости)
func (v *MainView) createStrategySelect() *widget.Select {
	// Получаем начальный список стратегий
	items, _ := v.app.State.Strategies.Get()
	if items == nil {
		items = []string{}
	}

	v.strategySelect = widget.NewSelect(items, func(s string) {
		// Игнорируем изменения до завершения инициализации UI
		if !v.uiInitialized {
			return
		}

		v.app.State.SelectedStrategy.Set(s)
		v.app.Services.Config.SetLastStrategyName(domain.StrategyName(s))

		// Перезапускаем стратегию, если процесс запущен
		v.restartCurrentStrategyIfRunning()
	})

	// Привязываем выбранную стратегию
	v.app.State.SelectedStrategy.AddListener(binding.NewDataListener(func() {
		selected, _ := v.app.State.SelectedStrategy.Get()
		if v.strategySelect != nil {
			v.strategySelect.SetSelected(selected)
		}
	}))

	// Устанавливаем начальные значения
	selected, _ := v.app.State.SelectedStrategy.Get()
	if selected != "" {
		v.strategySelect.SetSelected(selected)
	}
	return v.strategySelect
}

// UpdateStrategyOptions обновляет список стратегий в виджете (вызывается из App)
func (v *MainView) UpdateStrategyOptions(items []string, selectedStrategy string) {
	v.app.Logger.Debug("UpdateStrategyOptions: обновление виджета", "count", len(items), "selected", selectedStrategy)

	// Проверяем, существует ли выбранная стратегия в новом списке
	found := false
	for _, item := range items {
		if item == selectedStrategy {
			found = true
			break
		}
	}

	finalSelected := selectedStrategy
	if !found && len(items) > 0 {
		finalSelected = items[0]
	}

	// Обновляем кнопку выбора стратегии
	if v.strategyButton != nil {
		if finalSelected != "" {
			v.strategyButton.SetText(fmt.Sprintf("Стратегия: %s", finalSelected))
		} else {
			v.strategyButton.SetText("Выбрать стратегию...")
		}
	}

	// Обновляем Select для обратной совместимости
	if v.strategySelect != nil {
		v.strategySelect.Options = items
		if finalSelected != "" {
			v.strategySelect.SetSelected(finalSelected)
	}
	v.strategySelect.Refresh()
	}
}

// createGameFilterCheck создает чекбокс для Game Filter
func (v *MainView) createGameFilterCheck() *widget.Check {
	// Получаем начальное значение из конфига
	initialValue, _ := v.app.State.GameFilter.Get()

	// Создаем обычный чекбокс без привязки данных
	gameFilterCheck := widget.NewCheck("Game Filter", func(checked bool) {
		// Игнорируем изменения до завершения инициализации UI
		if !v.uiInitialized {
			return
		}

		// Сохраняем новое значение в конфиг
		v.app.Services.Config.SetGameFilter(checked)
		v.app.State.GameFilter.Set(checked)

		// Перезапускаем стратегию, если процесс запущен
		v.restartCurrentStrategyIfRunning()
	})

	// Устанавливаем начальное значение
	gameFilterCheck.Checked = initialValue

	return gameFilterCheck
}

// createIpsetSelect создает выпадающий список для выбора режима IPset
func (v *MainView) createIpsetSelect() *widget.Select {
	// Создаем Select с начальными опциями
	options := domain.AllIpsetModeStrings()
	ipsetSelect := widget.NewSelect(options, func(mode string) {
		// Игнорируем изменения до завершения инициализации UI
		if !v.uiInitialized {
			return
		}

		// Сохраняем новое значение в конфиг
		v.app.State.IpsetMode.Set(mode)
		v.app.Services.Config.SetIpsetMode(mode)

		// Обновляем файл ipset при изменении режима
		workingDir := v.app.Services.Config.GetWorkingDir()
		if workingDir != "" {
			v.app.Services.Ipset.UpdateIpsetFile(workingDir, mode)
		}

		// Перезапускаем стратегию, если процесс запущен
		v.restartCurrentStrategyIfRunning()
	})

	// Привязываем режим ipset
	v.app.State.IpsetMode.AddListener(binding.NewDataListener(func() {
		mode, _ := v.app.State.IpsetMode.Get()
		ipsetSelect.SetSelected(mode)
	}))

	// Устанавливаем начальное значение
	mode, _ := v.app.State.IpsetMode.Get()
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
	v.app.State.IsRunning.AddListener(binding.NewDataListener(func() {
		running, _ := v.app.State.IsRunning.Get()
		if running {
			startButton.Disable()
			stopButton.Enable()
		} else {
			startButton.Enable()
			stopButton.Disable()
		}
	}))

	// Инициализируем начальное состояние кнопок
	running, _ := v.app.State.IsRunning.Get()
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
		version, _ := v.app.State.Version.Get()
		return fmt.Sprintf(`**Zapret GUI** - Графический интерфейс для winws

**Версия:** %s

✅ Приложение готово к работе
`, version)
	}

	infoLabel := widget.NewRichTextFromMarkdown(getInfoText())

	// Добавляем слушатель для обновления текста при изменении версии
	v.app.State.Version.AddListener(binding.NewDataListener(func() {
		infoLabel.ParseMarkdown(getInfoText())
		infoLabel.Refresh()
	}))

	return widget.NewCard("Информация", "", infoLabel)
}

// handleStart обрабатывает запуск стратегии
func (v *MainView) handleStart() {
	// Проверяем права администратора
	if !v.app.Services.Admin.IsAdmin() {
		dialog.ShowInformation("Требуются права администратора",
			"Для работы winws требуются права администратора.\n"+
				"Пожалуйста, запустите приложение от имени администратора.",
			v.app.Window)
		return
	}

	// Получаем выбранную стратегию и настройки
	strategyName, _ := v.app.State.SelectedStrategy.Get()
	gameFilter, _ := v.app.State.GameFilter.Get()

	// Запускаем через контроллер (статус обновится автоматически через EventBus)
	err := v.app.Services.StrategyController.StartStrategy(strategyName, gameFilter)
	if err != nil {
		dialog.ShowError(err, v.app.Window)
		return
	}

	dialog.ShowInformation("Успех", fmt.Sprintf("Стратегия '%s' успешно запущена", strategyName), v.app.Window)
}

// handleStop обрабатывает остановку процесса
func (v *MainView) handleStop() {
	// Останавливаем через контроллер (статус обновится автоматически через EventBus)
	err := v.app.Services.StrategyController.StopStrategy()
	if err != nil {
		dialog.ShowError(err, v.app.Window)
		return
	}

	dialog.ShowInformation("Успех", "Процесс успешно остановлен", v.app.Window)
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

	dialog := dialog.NewCustomWithoutButtons("Диагностика", dialogContent, v.app.Window)
	dialog.Show()

	// Запускаем диагностику в горутине
	go func() {
		results := v.app.Services.Diagnostics.RunAll()

		// Закрываем прогресс и показываем результаты
		fyne.Do(func() {
			dialog.Hide()
			v.showDiagnosticResults(results)
		})
	}()
}

// showDiagnosticResults показывает результаты диагностики
func (v *MainView) showDiagnosticResults(results []*domain.DiagnosticResult) {
	// Подсчитываем статистику
	successCount := 0
	warnCount := 0
	errorCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		} else if r.Message == "WARN" {
			warnCount++
		} else {
			errorCount++
		}
	}

	// Создаём заголовок со статистикой
	statsText := fmt.Sprintf("Всего проверок: %d | ✓ Успешно: %d | ⚠ Предупреждения: %d | ✗ Ошибки: %d",
		len(results), successCount, warnCount, errorCount)
	statsLabel := widget.NewLabel(statsText)
	statsLabel.Importance = widget.MediumImportance

	// Создаем список результатов с детальной информацией
	resultsList := container.NewVBox()

	for _, result := range results {
		// Определяем статус и цвет
		var statusIcon string
		var importance widget.Importance
		if result.Success {
			statusIcon = "✓"
			importance = widget.SuccessImportance
		} else if result.Message == "WARN" {
			statusIcon = "⚠"
			importance = widget.WarningImportance
		} else {
			statusIcon = "✗"
			importance = widget.DangerImportance
		}

		// Название проверки со статусом
		titleLabel := widget.NewLabel(fmt.Sprintf("%s %s", statusIcon, result.Name))
		titleLabel.Importance = importance
		titleLabel.TextStyle = fyne.TextStyle{Bold: true}

		// Детали результата
		detailText := ""
		if resultDetail, ok := result.Details["result"].(string); ok {
			detailText = resultDetail
		}

		// Добавляем подсказку если есть
		if hint, ok := result.Details["hint"].(string); ok && hint != "" {
			detailText += "\n💡 " + hint
		}

		detailLabel := widget.NewLabel(detailText)
		detailLabel.Wrapping = fyne.TextWrapWord

		// Создаем карточку для каждого результата
		card := widget.NewCard("", "", container.NewVBox(titleLabel, detailLabel))
		resultsList.Add(card)
	}

	// Скролл для списка результатов
	scroll := container.NewScroll(resultsList)
	scroll.SetMinSize(fyne.NewSize(600, 400))

	// Общий контейнер
	content := container.NewBorder(
		statsLabel, // top
		nil,        // bottom
		nil,        // left
		nil,        // right
		scroll,     // center
	)

	dialog.ShowCustom("Результаты диагностики", "Закрыть", content, v.app.Window)
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
			v.app.Services.Cache.KillDiscordProcesses()

			// Очищаем кэш
			if err := v.app.Services.Cache.ClearDiscordCache(); err != nil {
				dialog.ShowError(err, v.app.Window)
				return
			}

			dialog.ShowInformation("Успех", "Кэш Discord успешно очищен", v.app.Window)
		},
		v.app.Window)
}

// showSettings показывает окно настроек
func (v *MainView) showSettings() {
	settingsView := NewSettingsView(v.app)
	settingsView.Show()
}

// showIncludedDomains показывает окно редактирования включенных доменов
func (v *MainView) showIncludedDomains() {
	filePath := v.app.Services.Config.GetExtraListPath()
	domainListView := NewDomainListView(v.app, filePath, "Включенные домены")
	domainListView.Show()
}

// showExcludedDomains показывает окно редактирования исключенных доменов
func (v *MainView) showExcludedDomains() {
	filePath := v.app.Services.Config.GetExcludeListPath()
	domainListView := NewDomainListView(v.app, filePath, "Исключенные домены")
	domainListView.Show()
}

// showCustomIpset показывает окно редактирования пользовательских подсетей
func (v *MainView) showCustomIpset() {
	filePath := v.app.Services.Config.GetCustomIpsetPath()
	ipsetListView := NewIpsetListView(v.app, filePath, "Пользовательские подсети (IPset)")
	ipsetListView.Show()
}

// showDomainCheck показывает окно проверки домена
func (v *MainView) showDomainCheck() {
	domainCheckView := NewDomainCheckView(v.app)
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

	progressDialog := dialog.NewCustomWithoutButtons("Обновление", dialogContent, v.app.Window)
	progressDialog.Show()

	// Запускаем обновление в горутине
	go func() {
		err := v.app.ReloadStrategies()

		fyne.Do(func() {
			progressDialog.Hide()
			if err != nil {
				dialog.ShowError(err, v.app.Window)
				return
			}
			dialog.ShowInformation("Успех", "Стратегии успешно обновлены", v.app.Window)
		})
	}()
}
