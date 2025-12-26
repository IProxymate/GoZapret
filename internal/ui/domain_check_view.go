package ui

import (
	"fmt"
	"strings"

	"github.com/IProxymate/GoZapret/internal/app"
	"github.com/IProxymate/GoZapret/internal/services/domain_check"

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
	checkWindow.Resize(fyne.NewSize(800, 600))
	checkWindow.CenterOnScreen()

	// Создаём табы для разных режимов проверки
	singleCheckTab := v.createSingleCheckTab(checkWindow)
	multiCheckTab := v.createMultiCheckTab(checkWindow)

	tabs := container.NewAppTabs(
		container.NewTabItem("Проверка домена", singleCheckTab),
		container.NewTabItem("Массовая проверка", multiCheckTab),
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
	scroll.SetMinSize(fyne.NewSize(750, 300))

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
	includeDefaultsCheck := widget.NewCheck("Включить стандартные цели (Discord, YouTube, Google...)", nil)
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
	scroll.SetMinSize(fyne.NewSize(750, 200))

	return container.NewBorder(topContent, nil, nil, nil, scroll)
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
