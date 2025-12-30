package ui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/IProxymate/GoZapret/internal/app"
	"github.com/IProxymate/GoZapret/internal/domain"
	"github.com/IProxymate/GoZapret/internal/services/domain_check"
	"github.com/IProxymate/GoZapret/internal/services/process"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// DomainCheckView представление для проверки доменов
type DomainCheckView struct {
	app          *app.App
	multiChecker *domain_check.MultiChecker
}

// NewDomainCheckView создает новое представление для проверки доменов
func NewDomainCheckView(a *app.App) *DomainCheckView {
	return &DomainCheckView{
		app:          a,
		multiChecker: domain_check.NewMultiChecker(),
	}
}

// Show отображает окно проверки домена
func (v *DomainCheckView) Show() {
	// Создаем окно
	checkWindow := v.app.FyneApp.NewWindow("Проверить домен")
	checkWindow.Resize(fyne.NewSize(900, 700))
	checkWindow.CenterOnScreen()

	// Создаём табы для разных режимов проверки
	singleCheckTab := v.createSingleCheckTab(checkWindow)
	multiCheckTab := v.createMultiCheckTab(checkWindow)
	dpiCheckTab := v.createDPICheckTab(checkWindow)
	strategyTestTab := v.createStrategyTestTab(checkWindow)

	tabs := container.NewAppTabs(
		container.NewTabItem("Проверка домена", singleCheckTab),
		container.NewTabItem("Массовая проверка", multiCheckTab),
		container.NewTabItem("DPI проверка", dpiCheckTab),
		container.NewTabItem("Тест стратегий", strategyTestTab),
	)

	checkWindow.SetContent(tabs)
	checkWindow.Show()
}

// createSingleCheckTab создаёт вкладку для проверки одного домена
func (v *DomainCheckView) createSingleCheckTab(window fyne.Window) fyne.CanvasObject {
	// Поле ввода домена
	domainEntry := widget.NewEntry()
	domainEntry.SetPlaceHolder("Введите домен (например: discord.com)")

	// Контейнер для результатов
	resultsContainer := container.NewVBox()

	// Индикатор загрузки
	progressBar := widget.NewProgressBarInfinite()
	progressBar.Hide()

	// Кнопка проверки
	var checkButton *widget.Button
	checkButton = widget.NewButton("Проверить", func() {
		domain := domainEntry.Text
		if domain == "" {
			dialog.ShowError(fmt.Errorf("введите домен"), window)
			return
		}

		// Показываем индикатор загрузки
		progressBar.Show()
		checkButton.Disable()
		resultsContainer.RemoveAll()
		resultsContainer.Add(widget.NewLabel("⏳ Проверка домена...\nЭто может занять до 15 секунд..."))

		// Запускаем проверку в горутине
		go func() {
			result := v.multiChecker.CheckSingleDomain(domain)

			// Обновляем UI в главном потоке
			fyne.Do(func() {
				checkButton.Enable()
				progressBar.Hide()
				resultsContainer.RemoveAll()
				v.showSingleResult(resultsContainer, result)
			})
		}()
	})
	checkButton.Importance = widget.HighImportance

	// Инструкция
	instructionLabel := widget.NewLabel(
		"💡 Проверка выполняет тесты HTTP, TLS 1.2, TLS 1.3 и Ping.\n\n" +
			"Статусы:\n" +
			"• OK - тест пройден успешно\n" +
			"• ERROR - ошибка подключения (возможна блокировка)\n" +
			"• UNSUP - протокол не поддерживается сервером\n" +
			"• TIMEOUT - превышено время ожидания",
	)
	instructionLabel.Wrapping = fyne.TextWrapWord

	// Компоновка
	topContent := container.NewVBox(
		widget.NewLabelWithStyle("Проверка доступности домена", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		instructionLabel,
		widget.NewSeparator(),
		widget.NewForm(
			widget.NewFormItem("Домен:", domainEntry),
		),
		checkButton,
		progressBar,
	)

	scroll := container.NewScroll(resultsContainer)
	scroll.SetMinSize(fyne.NewSize(850, 350))

	return container.NewBorder(topContent, nil, nil, nil, scroll)
}

// createMultiCheckTab создаёт вкладку для массовой проверки
func (v *DomainCheckView) createMultiCheckTab(window fyne.Window) fyne.CanvasObject {
	// Контейнер для результатов
	resultsContainer := container.NewVBox()

	// Индикатор загрузки
	progressBar := widget.NewProgressBarInfinite()
	progressBar.Hide()

	// Статистика
	statsLabel := widget.NewLabel("")
	statsLabel.Hide()

	// Текстовое поле для пользовательских доменов
	customDomainsEntry := widget.NewMultiLineEntry()
	customDomainsEntry.SetPlaceHolder("Введите дополнительные домены (по одному на строку):\nexample.com\napi.example.com")
	customDomainsEntry.SetMinRowsVisible(4)

	// Чекбокс для включения стандартных целей
	includeDefaultsCheck := widget.NewCheck("Включить стандартные цели (Discord, YouTube, Google, DNS...)", nil)
	includeDefaultsCheck.SetChecked(true)

	// Кнопка проверки
	var checkButton *widget.Button
	checkButton = widget.NewButton("Запустить проверку", func() {
		// Собираем цели
		var targets []domain_check.Target

		// Добавляем стандартные цели если выбрано
		if includeDefaultsCheck.Checked {
			targets = append(targets, domain_check.GetDefaultTargets()...)
		}

		// Добавляем пользовательские домены
		customDomains := strings.TrimSpace(customDomainsEntry.Text)
		if customDomains != "" {
			lines := strings.Split(customDomains, "\n")
			for _, line := range lines {
				domain := strings.TrimSpace(line)
				if domain == "" {
					continue
				}
				// Нормализуем домен
				domain = strings.TrimPrefix(domain, "https://")
				domain = strings.TrimPrefix(domain, "http://")
				domain = strings.TrimSuffix(domain, "/")

				targets = append(targets, domain_check.Target{
					Name:       domain,
					URL:        fmt.Sprintf("https://%s", domain),
					PingTarget: domain,
				})
			}
		}

		if len(targets) == 0 {
			dialog.ShowError(fmt.Errorf("добавьте хотя бы одну цель для проверки"), window)
			return
		}

		progressBar.Show()
		checkButton.Disable()
		statsLabel.Hide()
		resultsContainer.RemoveAll()
		resultsContainer.Add(widget.NewLabel(fmt.Sprintf("⏳ Выполняется проверка %d целей...\nЭто может занять до %d секунд...", len(targets), len(targets)*3)))

		go func() {
			result := v.multiChecker.CheckAll(targets)

			fyne.Do(func() {
				checkButton.Enable()
				progressBar.Hide()
				resultsContainer.RemoveAll()
				v.showMultiResults(resultsContainer, statsLabel, result)
				statsLabel.Show()
			})
		}()
	})
	checkButton.Importance = widget.HighImportance

	// Инструкция
	instructionLabel := widget.NewLabel(
		"💡 Массовая проверка тестирует доступность сервисов.\n" +
			"Для каждой цели выполняются тесты: HTTP, TLS 1.2, TLS 1.3, Ping",
	)
	instructionLabel.Wrapping = fyne.TextWrapWord

	// Список стандартных целей (сворачиваемый)
	defaultTargetsList := widget.NewLabel(v.formatTargetsList())
	defaultTargetsList.Wrapping = fyne.TextWrapWord

	// Аккордеон для стандартных целей
	defaultTargetsAccordion := widget.NewAccordion(
		widget.NewAccordionItem("Стандартные цели", defaultTargetsList),
	)

	// Карточка для пользовательских доменов
	customDomainsCard := widget.NewCard("Дополнительные домены", "", container.NewVBox(
		customDomainsEntry,
	))

	topContent := container.NewVBox(
		widget.NewLabelWithStyle("Массовая проверка доступности", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		instructionLabel,
		widget.NewSeparator(),
		includeDefaultsCheck,
		defaultTargetsAccordion,
		customDomainsCard,
		widget.NewSeparator(),
		checkButton,
		progressBar,
		statsLabel,
	)

	scroll := container.NewScroll(resultsContainer)
	scroll.SetMinSize(fyne.NewSize(850, 200))

	return container.NewBorder(topContent, nil, nil, nil, scroll)
}

// createDPICheckTab создаёт вкладку для DPI проверки
func (v *DomainCheckView) createDPICheckTab(window fyne.Window) fyne.CanvasObject {
	// Контейнер для результатов
	resultsContainer := container.NewVBox()

	// Индикатор загрузки
	progressBar := widget.NewProgressBarInfinite()
	progressBar.Hide()

	// Статистика
	statsLabel := widget.NewLabel("")
	statsLabel.Hide()

	// Кнопка проверки
	var checkButton *widget.Button
	checkButton = widget.NewButton("Запустить DPI проверку", func() {
		targets := domain_check.GetDPITargets()

		if len(targets) == 0 {
			dialog.ShowError(fmt.Errorf("список DPI целей пуст"), window)
			return
		}

		progressBar.Show()
		checkButton.Disable()
		statsLabel.Hide()
		resultsContainer.RemoveAll()
		resultsContainer.Add(widget.NewLabel(fmt.Sprintf("⏳ Выполняется DPI проверка %d целей...\nЭто может занять несколько минут...", len(targets))))

		go func() {
			result := v.multiChecker.CheckDPI(targets)

			fyne.Do(func() {
				checkButton.Enable()
				progressBar.Hide()
				resultsContainer.RemoveAll()
				v.showDPIResults(resultsContainer, statsLabel, result)
				statsLabel.Show()
			})
		}()
	})
	checkButton.Importance = widget.HighImportance

	// Инструкция
	instructionLabel := widget.NewLabel(
		"💡 DPI проверка (TCP 16-20 freeze detection)\n\n" +
			"Эта проверка обнаруживает блокировку на уровне DPI (Deep Packet Inspection).\n" +
			"Паттерн 16-20KB freeze означает, что цензор обрезает соединение после передачи 16-20KB данных.\n\n" +
			"Статусы:\n" +
			"• OK - соединение работает нормально\n" +
			"• FAIL - ошибка соединения\n" +
			"• UNSUPPORTED - протокол не поддерживается\n" +
			"• LIKELY_BLOCKED - обнаружен паттерн блокировки DPI",
	)
	instructionLabel.Wrapping = fyne.TextWrapWord

	// Список DPI целей
	dpiTargetsList := widget.NewLabel(v.formatDPITargetsList())
	dpiTargetsList.Wrapping = fyne.TextWrapWord

	// Аккордеон для DPI целей
	dpiTargetsAccordion := widget.NewAccordion(
		widget.NewAccordionItem("DPI цели (провайдеры)", dpiTargetsList),
	)

	topContent := container.NewVBox(
		widget.NewLabelWithStyle("DPI проверка (TCP 16-20 freeze)", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		instructionLabel,
		widget.NewSeparator(),
		dpiTargetsAccordion,
		widget.NewSeparator(),
		checkButton,
		progressBar,
		statsLabel,
	)

	scroll := container.NewScroll(resultsContainer)
	scroll.SetMinSize(fyne.NewSize(850, 200))

	return container.NewBorder(topContent, nil, nil, nil, scroll)
}

// createStrategyTestTab создаёт вкладку для тестирования стратегий
func (v *DomainCheckView) createStrategyTestTab(window fyne.Window) fyne.CanvasObject {
	// Контейнер для результатов
	resultsContainer := container.NewVBox()

	// Индикатор загрузки
	progressBar := widget.NewProgressBar()
	progressBar.Hide()

	// Статус текущего тестирования
	statusLabel := widget.NewLabel("")
	statusLabel.Hide()

	// Статистика
	statsLabel := widget.NewLabel("")
	statsLabel.Hide()

	// Выбор режима тестирования
	modeSelect := widget.NewSelect([]string{"Стандартные тесты (HTTP/Ping)", "DPI тесты (TCP 16-20)"}, nil)
	modeSelect.SetSelected("Стандартные тесты (HTTP/Ping)")

	// Контекст для отмены
	var cancelFunc context.CancelFunc
	var tester *domain_check.StrategyTester

	// Кнопка отмены
	cancelButton := widget.NewButton("Отмена", func() {
		if tester != nil {
			tester.Cancel()
		}
		if cancelFunc != nil {
			cancelFunc()
		}
	})
	cancelButton.Importance = widget.DangerImportance
	cancelButton.Hide()

	// Кнопка проверки
	var checkButton *widget.Button
	checkButton = widget.NewButton("Запустить тест всех стратегий", func() {
		// Проверяем права администратора
		if !v.app.Services.Admin.IsAdmin() {
			dialog.ShowError(fmt.Errorf("для тестирования стратегий требуются права администратора"), window)
			return
		}

		// Проверяем, не запущен ли уже winws
		if v.app.Services.Process.IsRunning() || v.app.Services.Process.IsWinwsProcessRunning() {
			dialog.ShowConfirm("Внимание",
				"Для тестирования необходимо остановить текущий процесс zapret.\nОстановить?",
				func(confirmed bool) {
					if confirmed {
						_ = v.app.Services.StrategyController.StopStrategy()
						// Запускаем тестирование после остановки
						v.startStrategyTest(window, modeSelect, progressBar, statusLabel, statsLabel, resultsContainer, checkButton, cancelButton, &tester, &cancelFunc)
					}
				},
				window)
			return
		}

		v.startStrategyTest(window, modeSelect, progressBar, statusLabel, statsLabel, resultsContainer, checkButton, cancelButton, &tester, &cancelFunc)
	})
	checkButton.Importance = widget.HighImportance

	// Инструкция
	instructionLabel := widget.NewLabel(
		"💡 Тестирование стратегий\n\n" +
			"Эта функция последовательно запускает каждую стратегию и проверяет её эффективность.\n" +
			"Для каждой стратегии:\n" +
			"1. Запускается winws с данной конфигурацией\n" +
			"2. Выполняются тесты доступности\n" +
			"3. Процесс останавливается\n\n" +
			"По завершении будет определена лучшая стратегия на основе результатов тестов.\n\n" +
			"⚠️ Требуются права администратора\n" +
			"⚠️ Текущий процесс zapret будет остановлен на время тестирования",
	)
	instructionLabel.Wrapping = fyne.TextWrapWord

	// Список стратегий
	strategies, _ := v.app.State.Strategies.Get()
	strategiesText := "Нет загруженных стратегий"
	if len(strategies) > 0 {
		strategiesText = fmt.Sprintf("Загружено стратегий: %d\n%s", len(strategies), strings.Join(strategies, ", "))
	}
	strategiesLabel := widget.NewLabel(strategiesText)
	strategiesLabel.Wrapping = fyne.TextWrapWord

	strategiesAccordion := widget.NewAccordion(
		widget.NewAccordionItem("Доступные стратегии", strategiesLabel),
	)

	buttonsContainer := container.NewHBox(checkButton, cancelButton)

	topContent := container.NewVBox(
		widget.NewLabelWithStyle("Тестирование стратегий", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		instructionLabel,
		widget.NewSeparator(),
		widget.NewForm(
			widget.NewFormItem("Режим тестирования:", modeSelect),
		),
		strategiesAccordion,
		widget.NewSeparator(),
		buttonsContainer,
		progressBar,
		statusLabel,
		statsLabel,
	)

	scroll := container.NewScroll(resultsContainer)
	scroll.SetMinSize(fyne.NewSize(850, 150))

	return container.NewBorder(topContent, nil, nil, nil, scroll)
}

// startStrategyTest запускает тестирование стратегий
func (v *DomainCheckView) startStrategyTest(
	window fyne.Window,
	modeSelect *widget.Select,
	progressBar *widget.ProgressBar,
	statusLabel *widget.Label,
	statsLabel *widget.Label,
	resultsContainer *fyne.Container,
	checkButton *widget.Button,
	cancelButton *widget.Button,
	testerPtr **domain_check.StrategyTester,
	cancelFuncPtr *context.CancelFunc,
) {
	// Получаем список стратегий
	strategyNames, _ := v.app.State.Strategies.Get()
	if len(strategyNames) == 0 {
		dialog.ShowError(fmt.Errorf("нет загруженных стратегий"), window)
		return
	}

	// Получаем объекты стратегий
	var strategies []*domain.Strategy
	for _, name := range strategyNames {
		strategy, err := v.app.Services.Strategy.GetByName(domain.StrategyName(name))
		if err == nil {
			strategies = append(strategies, strategy)
		}
	}

	if len(strategies) == 0 {
		dialog.ShowError(fmt.Errorf("не удалось загрузить стратегии"), window)
		return
	}

	// Определяем режим
	var mode domain_check.StrategyTestMode
	if strings.Contains(modeSelect.Selected, "DPI") {
		mode = domain_check.TestModeDPI
	} else {
		mode = domain_check.TestModeStandard
	}

	// Подготавливаем тестер
	workingDir := v.app.Services.Config.GetWorkingDir()
	if workingDir == "" {
		dialog.ShowError(fmt.Errorf("рабочая директория не настроена"), window)
		return
	}

	assetsPath := v.app.Services.Config.GetAssetsPath()
	gameFilter, _ := v.app.State.GameFilter.Get()

	batParser := process.NewBatParser()
	argsBuilder := process.NewArgsBuilder(v.app.Services.Config)
	workingBinDir := filepath.Join(workingDir, "bin")

	tester := domain_check.NewStrategyTester(batParser, argsBuilder, workingBinDir, assetsPath, gameFilter)
	*testerPtr = tester

	// Создаём контекст с отменой
	ctx, cancel := context.WithCancel(context.Background())
	*cancelFuncPtr = cancel

	// Настраиваем UI
	progressBar.SetValue(0)
	progressBar.Show()
	statusLabel.Show()
	statsLabel.Hide()
	checkButton.Disable()
	cancelButton.Show()
	resultsContainer.RemoveAll()

	// Настраиваем callback прогресса
	tester.SetProgressCallback(func(progress domain_check.StrategyTestProgress) {
		fyne.Do(func() {
			// Вычисляем прогресс с защитой от отрицательных значений
			var progressValue float64
			if progress.TotalStrategies > 0 && progress.CurrentIndex > 0 {
				progressValue = float64(progress.CurrentIndex-1) / float64(progress.TotalStrategies)
				// Добавляем прогресс внутри текущей стратегии
				switch progress.Phase {
				case "starting":
					progressValue += 0.1 / float64(progress.TotalStrategies)
				case "testing":
					progressValue += 0.5 / float64(progress.TotalStrategies)
				case "stopping":
					progressValue += 0.9 / float64(progress.TotalStrategies)
				}
			}
			// Ограничиваем значение от 0 до 1
			if progressValue < 0 {
				progressValue = 0
			}
			if progressValue > 1 {
				progressValue = 1
			}
			progressBar.SetValue(progressValue)

			phaseText := ""
			switch progress.Phase {
			case "starting":
				phaseText = "запуск"
			case "testing":
				phaseText = "тестирование"
			case "stopping":
				phaseText = "остановка"
			}

			statusLabel.SetText(fmt.Sprintf("⏳ [%d/%d] %s: %s...",
				progress.CurrentIndex, progress.TotalStrategies,
				progress.CurrentStrategy, phaseText))
		})
	})

	// Запускаем тестирование в горутине
	go func() {
		result := tester.TestAllStrategies(ctx, strategies, mode, nil)

		fyne.Do(func() {
			progressBar.Hide()
			statusLabel.Hide()
			checkButton.Enable()
			cancelButton.Hide()

			v.showStrategyTestResults(resultsContainer, statsLabel, result)
			statsLabel.Show()
		})
	}()
}

// showStrategyTestResults отображает результаты тестирования стратегий
func (v *DomainCheckView) showStrategyTestResults(container *fyne.Container, statsLabel *widget.Label, result *domain_check.AllStrategiesTestResult) {
	// Статистика
	statsText := fmt.Sprintf("📊 Протестировано стратегий: %d | Время: %.1f сек",
		len(result.Results), result.TotalTime.Seconds())

	if result.BestStrategy != "" {
		statsText += fmt.Sprintf("\n🏆 Лучшая стратегия: %s (балл: %d)", result.BestStrategy, result.BestScore)
	}

	statsLabel.SetText(statsText)

	// Результаты по каждой стратегии (сортируем по баллу)
	for _, res := range result.Results {
		card := v.createStrategyResultCard(&res, res.StrategyName == result.BestStrategy)
		container.Add(card)
	}
}

// createStrategyResultCard создаёт карточку результата для одной стратегии
func (v *DomainCheckView) createStrategyResultCard(result *domain_check.StrategyTestResult, isBest bool) *widget.Card {
	content := container.NewVBox()

	// Ошибка
	if result.Error != nil {
		errorLabel := widget.NewLabel(fmt.Sprintf("❌ Ошибка: %s", result.Error.Error()))
		errorLabel.Importance = widget.DangerImportance
		content.Add(errorLabel)
		return widget.NewCard(result.StrategyName, "", content)
	}

	// Статистика
	var statsText string
	if result.StandardResult != nil {
		statsText = fmt.Sprintf("HTTP OK: %d | ERROR: %d | UNSUP: %d | Ping OK: %d",
			result.StandardResult.TotalOK, result.StandardResult.TotalError,
			result.StandardResult.TotalUnsup, result.StandardResult.PingOK)
	} else if result.DPIResult != nil {
		statsText = fmt.Sprintf("OK: %d | FAIL: %d | BLOCKED: %d",
			result.DPIResult.TotalOK, result.DPIResult.TotalFail, result.DPIResult.TotalBlocked)
	}

	statsLabel := widget.NewLabel(statsText)
	content.Add(statsLabel)

	// Балл и время
	scoreText := fmt.Sprintf("Балл: %d | Время: %.1f сек", result.Score, result.Duration.Seconds())
	scoreLabel := widget.NewLabel(scoreText)
	if isBest {
		scoreLabel.Importance = widget.SuccessImportance
	}
	content.Add(scoreLabel)

	// Заголовок
	title := result.StrategyName
	if isBest {
		title = "🏆 " + title
	}

	return widget.NewCard(title, "", content)
}

// formatTargetsList форматирует список целей
func (v *DomainCheckView) formatTargetsList() string {
	targets := domain_check.GetDefaultTargets()
	var parts []string
	for _, t := range targets {
		if t.URL != "" {
			parts = append(parts, t.Name)
		} else {
			parts = append(parts, fmt.Sprintf("%s (ping)", t.Name))
		}
	}
	return strings.Join(parts, ", ")
}

// formatDPITargetsList форматирует список DPI целей
func (v *DomainCheckView) formatDPITargetsList() string {
	targets := domain_check.GetDPITargets()
	providerMap := make(map[string]int)
	for _, t := range targets {
		providerMap[t.Provider]++
	}

	var parts []string
	for provider, count := range providerMap {
		parts = append(parts, fmt.Sprintf("%s (%d)", provider, count))
	}
	return fmt.Sprintf("Всего целей: %d\nПровайдеры: %s", len(targets), strings.Join(parts, ", "))
}

// showSingleResult отображает результат проверки одного домена
func (v *DomainCheckView) showSingleResult(container *fyne.Container, result *domain_check.TargetResult) {
	// Заголовок
	titleLabel := widget.NewLabelWithStyle(
		fmt.Sprintf("Результаты проверки: %s", result.Name),
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)
	container.Add(titleLabel)
	container.Add(widget.NewSeparator())

	// HTTP тесты
	if len(result.Tests) > 0 {
		httpCard := v.createTestResultsCard("HTTP/TLS тесты", result.Tests)
		container.Add(httpCard)
	}

	// Ping результат
	pingCard := v.createPingResultCard(result.PingTarget, result.PingResult)
	container.Add(pingCard)

	// Итоговый статус
	summaryCard := v.createSummaryCard(result)
	container.Add(summaryCard)
}

// showMultiResults отображает результаты массовой проверки
func (v *DomainCheckView) showMultiResults(container *fyne.Container, statsLabel *widget.Label, result *domain_check.MultiCheckResult) {
	// Обновляем статистику
	statsText := fmt.Sprintf("📊 Статистика: HTTP OK: %d | ERROR: %d | UNSUP: %d | Ping OK: %d | Ping Fail: %d",
		result.TotalOK, result.TotalError, result.TotalUnsup, result.PingOK, result.PingFail)
	statsLabel.SetText(statsText)

	// Результаты по каждой цели
	for _, target := range result.Targets {
		card := v.createTargetResultCard(&target)
		container.Add(card)
	}
}

// showDPIResults отображает результаты DPI проверки
func (v *DomainCheckView) showDPIResults(container *fyne.Container, statsLabel *widget.Label, result *domain_check.DPICheckResult) {
	// Обновляем статистику
	statsText := fmt.Sprintf("📊 DPI Статистика: OK: %d | FAIL: %d | UNSUP: %d | BLOCKED: %d",
		result.TotalOK, result.TotalFail, result.TotalUnsup, result.TotalBlocked)

	if result.WarnDetected {
		statsText += "\n⚠️ ВНИМАНИЕ: Обнаружен паттерн DPI блокировки (16-20KB freeze)!"
	} else {
		statsText += "\n✅ Паттерн DPI блокировки не обнаружен"
	}

	statsLabel.SetText(statsText)

	// Результаты по каждой цели
	for _, target := range result.Targets {
		card := v.createDPITargetResultCard(&target)
		container.Add(card)
	}

	// Итоговая рекомендация
	if result.WarnDetected {
		warnCard := widget.NewCard("⚠️ Рекомендации", "",
			widget.NewLabel("Обнаружен паттерн DPI блокировки на некоторых целях.\n\n"+
				"Рекомендуется:\n"+
				"• Попробовать другую стратегию\n"+
				"• Изменить SNI/IP настройки\n"+
				"• Проверить работу с другим провайдером"))
		container.Add(warnCard)
	}
}

// createTestResultsCard создаёт карточку с результатами HTTP/TLS тестов
func (v *DomainCheckView) createTestResultsCard(title string, tests []domain_check.TestResult) *widget.Card {
	content := container.NewVBox()

	for _, test := range tests {
		var statusIcon string
		var importance widget.Importance

		switch test.Status {
		case "OK":
			statusIcon = "✓"
			importance = widget.SuccessImportance
		case "UNSUP":
			statusIcon = "⚠"
			importance = widget.WarningImportance
		case "TIMEOUT":
			statusIcon = "⏱"
			importance = widget.WarningImportance
		default:
			statusIcon = "✗"
			importance = widget.DangerImportance
		}

		var details string
		if test.HTTPCode > 0 {
			details = fmt.Sprintf("HTTP %d, %.2fs", test.HTTPCode, test.ResponseTime)
		} else if test.Error != nil {
			details = test.Error.Error()
		}

		label := widget.NewLabel(fmt.Sprintf("%s %s: %s %s", statusIcon, test.Type, test.Status, details))
		label.Importance = importance
		content.Add(label)
	}

	return widget.NewCard(title, "", content)
}

// createPingResultCard создаёт карточку с результатом ping
func (v *DomainCheckView) createPingResultCard(target, result string) *widget.Card {
	var statusIcon string
	var importance widget.Importance

	if result == "n/a" {
		statusIcon = "—"
		importance = widget.MediumImportance
	} else if result == "Timeout" || strings.HasPrefix(result, "Error") {
		statusIcon = "✗"
		importance = widget.DangerImportance
	} else {
		statusIcon = "✓"
		importance = widget.SuccessImportance
	}

	label := widget.NewLabel(fmt.Sprintf("%s Ping %s: %s", statusIcon, target, result))
	label.Importance = importance

	return widget.NewCard("Ping тест", "", label)
}

// createSummaryCard создаёт итоговую карточку
func (v *DomainCheckView) createSummaryCard(result *domain_check.TargetResult) *widget.Card {
	// Подсчитываем статистику
	okCount := 0
	errorCount := 0
	for _, test := range result.Tests {
		if test.Status == "OK" {
			okCount++
		} else if test.Status == "ERROR" || test.Status == "TIMEOUT" {
			errorCount++
		}
	}

	var summaryText string
	var importance widget.Importance

	if errorCount == 0 && okCount > 0 {
		summaryText = "✅ Домен доступен, все тесты пройдены успешно"
		importance = widget.SuccessImportance
	} else if errorCount > 0 && okCount > 0 {
		summaryText = fmt.Sprintf("⚠️ Частичная доступность: %d OK, %d ошибок", okCount, errorCount)
		importance = widget.WarningImportance
	} else if errorCount > 0 {
		summaryText = "❌ Домен недоступен или заблокирован"
		importance = widget.DangerImportance
	} else {
		summaryText = "ℹ️ Недостаточно данных для оценки"
		importance = widget.MediumImportance
	}

	// Добавляем рекомендации
	if errorCount > 0 {
		summaryText += "\n\n💡 Рекомендации:\n"
		summaryText += "• Убедитесь, что zapret запущен\n"
		summaryText += "• Попробуйте другую стратегию\n"
		summaryText += "• Проверьте настройки DNS"
	}

	label := widget.NewLabel(summaryText)
	label.Wrapping = fyne.TextWrapWord
	label.Importance = importance

	return widget.NewCard("Итог", "", label)
}

// createTargetResultCard создаёт карточку результата для одной цели
func (v *DomainCheckView) createTargetResultCard(target *domain_check.TargetResult) *widget.Card {
	content := container.NewHBox()

	// HTTP тесты
	if target.IsURL && len(target.Tests) > 0 {
		for _, test := range target.Tests {
			var statusText string
			var importance widget.Importance

			switch test.Status {
			case "OK":
				statusText = fmt.Sprintf("%s:OK", test.Type)
				importance = widget.SuccessImportance
			case "UNSUP":
				statusText = fmt.Sprintf("%s:UNSUP", test.Type)
				importance = widget.WarningImportance
			case "TIMEOUT":
				statusText = fmt.Sprintf("%s:TIMEOUT", test.Type)
				importance = widget.WarningImportance
			default:
				statusText = fmt.Sprintf("%s:ERROR", test.Type)
				importance = widget.DangerImportance
			}

			label := widget.NewLabel(statusText)
			label.Importance = importance
			content.Add(label)
		}

		// Разделитель
		content.Add(widget.NewLabel("|"))
	}

	// Ping
	pingLabel := widget.NewLabel(fmt.Sprintf("Ping: %s", target.PingResult))
	if target.PingResult == "Timeout" || strings.HasPrefix(target.PingResult, "Error") {
		pingLabel.Importance = widget.DangerImportance
	} else if target.PingResult != "n/a" {
		pingLabel.Importance = widget.SuccessImportance
	}
	content.Add(pingLabel)

	return widget.NewCard(target.Name, "", content)
}

// createDPITargetResultCard создаёт карточку результата DPI для одной цели
func (v *DomainCheckView) createDPITargetResultCard(target *domain_check.DPITargetResult) *widget.Card {
	content := container.NewVBox()

	// Заголовок с провайдером
	headerLabel := widget.NewLabel(fmt.Sprintf("Провайдер: %s", target.Provider))
	headerLabel.TextStyle = fyne.TextStyle{Italic: true}
	content.Add(headerLabel)

	// Результаты тестов
	testsContainer := container.NewHBox()
	for _, test := range target.Tests {
		var statusText string
		var importance widget.Importance

		switch test.Status {
		case "OK":
			statusText = fmt.Sprintf("%s:OK", test.TestLabel)
			importance = widget.SuccessImportance
		case "UNSUPPORTED":
			statusText = fmt.Sprintf("%s:UNSUP", test.TestLabel)
			importance = widget.WarningImportance
		case "LIKELY_BLOCKED":
			statusText = fmt.Sprintf("%s:BLOCKED", test.TestLabel)
			importance = widget.DangerImportance
		default:
			statusText = fmt.Sprintf("%s:FAIL", test.TestLabel)
			importance = widget.DangerImportance
		}

		label := widget.NewLabel(statusText)
		label.Importance = importance
		testsContainer.Add(label)
	}
	content.Add(testsContainer)

	// Предупреждение о блокировке
	if target.Warned {
		warnLabel := widget.NewLabel("⚠️ Обнаружен паттерн 16-20KB freeze!")
		warnLabel.Importance = widget.DangerImportance
		content.Add(warnLabel)
	}

	return widget.NewCard(target.TargetID, "", content)
}
