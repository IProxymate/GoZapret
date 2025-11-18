package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/IProxymate/GoZapret/internal/services"
)

// DomainCheckView представление для проверки доменов
type DomainCheckView struct {
	app           fyne.App
	window        fyne.Window
	domainChecker *services.DomainChecker
}

// NewDomainCheckView создает новое представление для проверки доменов
func NewDomainCheckView(app fyne.App, window fyne.Window) *DomainCheckView {
	return &DomainCheckView{
		app:           app,
		window:        window,
		domainChecker: services.NewDomainChecker(),
	}
}

// Show отображает окно проверки домена
func (v *DomainCheckView) Show() {
	// Создаем окно
	checkWindow := v.app.NewWindow("Проверить домен")
	checkWindow.Resize(fyne.NewSize(600, 400))
	checkWindow.CenterOnScreen()

	// Поле ввода домена
	domainEntry := widget.NewEntry()
	domainEntry.SetPlaceHolder("Введите домен (например: discord.com)")

	// Метка с результатом
	resultLabel := widget.NewLabel("")
	resultLabel.Wrapping = fyne.TextWrapWord

	// Индикатор загрузки
	progressBar := widget.NewProgressBarInfinite()
	progressBar.Hide()

	// Кнопка проверки (объявляем заранее для использования в замыкании)
	var checkButton *widget.Button
	checkButton = widget.NewButton("Проверить", func() {
		domain := domainEntry.Text
		if domain == "" {
			dialog.ShowError(fmt.Errorf("введите домен"), checkWindow)
			return
		}

		// Показываем индикатор загрузки
		progressBar.Show()
		checkButton.Disable()
		resultLabel.SetText("⏳ Проверка домена...\nЭто может занять до 10 секунд...")

		// Запускаем проверку в горутине
		go func() {
			result := v.domainChecker.CheckDomain(domain)

			// Обновляем UI в главном потоке через fyne.Do
			fyne.Do(func() {
				checkButton.Enable()
				progressBar.Hide()

				if result.Error != nil {
					resultLabel.SetText(result.Message)
				} else {
					resultLabel.SetText(result.Message)
				}
			})
		}()
	})

	// Кнопка закрытия
	closeButton := widget.NewButton("Закрыть", func() {
		checkWindow.Close()
	})

	// Инструкция
	instructionLabel := widget.NewLabel(
		"💡 Эта функция проверяет доступность домена через curl.\n\n" +
			"Если домен заблокирован:\n" +
			"• Время ответа будет > 2 секунд\n" +
			"• HTTP код может быть 0 (нет ответа)\n\n" +
			"Если домен работает нормально:\n" +
			"• Время ответа < 1 секунды\n" +
			"• HTTP код 200-399",
	)
	instructionLabel.Wrapping = fyne.TextWrapWord

	// Компоновка
	content := container.NewBorder(
		// Top
		container.NewVBox(
			widget.NewLabelWithStyle("Проверка доступности домена", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewSeparator(),
			instructionLabel,
			widget.NewSeparator(),
			widget.NewForm(
				widget.NewFormItem("Домен:", domainEntry),
			),
			checkButton,
			progressBar,
		),
		// Bottom
		container.NewVBox(
			widget.NewSeparator(),
			closeButton,
		),
		// Left, Right
		nil, nil,
		// Center
		container.NewScroll(resultLabel),
	)

	checkWindow.SetContent(content)
	checkWindow.Show()
}
