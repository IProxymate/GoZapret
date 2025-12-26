package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/IProxymate/GoZapret/internal/app"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// DomainListView представляет окно редактирования списка доменов
type DomainListView struct {
	app      *app.App
	filePath string
	title    string
	window   fyne.Window
}

// NewDomainListView создает новое окно редактирования списка доменов
func NewDomainListView(a *app.App, filePath, title string) *DomainListView {
	return &DomainListView{
		app:      a,
		filePath: filePath,
		title:    title,
	}
}

// Show показывает окно редактирования
func (v *DomainListView) Show() {
	v.window = v.app.FyneApp.NewWindow(v.title)
	v.window.Resize(fyne.NewSize(600, 500))
	v.window.CenterOnScreen()

	content := v.buildContent()
	v.window.SetContent(content)
	v.window.Show()
}

// buildContent создает содержимое окна
func (v *DomainListView) buildContent() fyne.CanvasObject {
	// Загружаем текущее содержимое файла
	domains, err := v.loadDomains()
	if err != nil {
		v.app.Logger.Error("Ошибка загрузки списка доменов", "file", v.filePath, "error", err)
		domains = ""
	}

	// Создаем многострочное текстовое поле
	entry := widget.NewMultiLineEntry()
	entry.SetText(domains)
	entry.Wrapping = fyne.TextWrapWord
	entry.SetPlaceHolder("Введите домены построчно, например:\ndiscord.com\ntwitch.tv\nyoutube.com")

	// Информационная метка
	infoLabel := widget.NewLabel("Добавьте домены построчно. Каждый домен должен быть на отдельной строке.")
	infoLabel.Wrapping = fyne.TextWrapWord

	// Счетчик доменов
	countLabel := widget.NewLabel(v.getCountText(domains))

	// Обновляем счетчик при изменении текста
	entry.OnChanged = func(text string) {
		countLabel.SetText(v.getCountText(text))
	}

	// Кнопки
	saveButton := widget.NewButton("Сохранить", func() {
		v.saveDomains(entry.Text)
		v.restartStrategyIfRunning()
	})
	saveButton.Importance = widget.HighImportance

	cancelButton := widget.NewButton("Отмена", func() {
		v.window.Close()
	})

	clearButton := widget.NewButton("Очистить", func() {
		dialog.ShowConfirm("Подтверждение",
			"Вы уверены, что хотите очистить весь список?",
			func(confirmed bool) {
				if confirmed {
					entry.SetText("")
				}
			},
			v.window)
	})
	clearButton.Importance = widget.DangerImportance

	// Кнопка валидации
	validateButton := widget.NewButton("Проверить", func() {
		v.validateDomains(entry.Text)
	})

	buttonContainer := container.NewGridWithColumns(4, saveButton, validateButton, clearButton, cancelButton)

	// Основной контейнер
	content := container.NewBorder(
		container.NewVBox(
			widget.NewCard("", "", infoLabel),
			countLabel,
		),
		buttonContainer,
		nil,
		nil,
		container.NewScroll(entry),
	)

	return container.NewPadded(content)
}

// loadDomains загружает домены из файла
func (v *DomainListView) loadDomains() (string, error) {
	data, err := os.ReadFile(v.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// saveDomains сохраняет домены в файл
func (v *DomainListView) saveDomains(text string) {
	// Нормализуем текст: убираем пустые строки и лишние пробелы
	lines := strings.Split(text, "\n")
	var validLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			validLines = append(validLines, line)
		}
	}

	// Сохраняем в файл
	content := strings.Join(validLines, "\n")
	if len(validLines) > 0 {
		content += "\n" // Добавляем перевод строки в конце
	}

	err := os.WriteFile(v.filePath, []byte(content), 0644)
	if err != nil {
		dialog.ShowError(fmt.Errorf("ошибка сохранения файла: %v", err), v.window)
		return
	}

	dialog.ShowInformation("Успех",
		fmt.Sprintf("Список успешно сохранен.\nДоменов: %d", len(validLines)),
		v.window)
	v.window.Close()
}

// restartStrategyIfRunning перезапускает стратегию, если процесс запущен.
// Делегирует вызов App.RestartCurrentStrategy()
func (v *DomainListView) restartStrategyIfRunning() {
	v.app.RestartCurrentStrategy()
}

// validateDomains проверяет корректность доменов
func (v *DomainListView) validateDomains(text string) {
	lines := strings.Split(text, "\n")
	var validDomains []string
	var invalidLines []string

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Простая валидация домена
		if v.isValidDomain(line) {
			validDomains = append(validDomains, line)
		} else {
			invalidLines = append(invalidLines, fmt.Sprintf("Строка %d: %s", i+1, line))
		}
	}

	var message string
	if len(invalidLines) == 0 {
		message = fmt.Sprintf("✓ Все домены корректны\nВсего доменов: %d", len(validDomains))
	} else {
		message = fmt.Sprintf("Найдены некорректные домены:\n\n%s\n\nКорректных доменов: %d",
			strings.Join(invalidLines, "\n"), len(validDomains))
	}

	dialog.ShowInformation("Результат проверки", message, v.window)
}

// isValidDomain выполняет простую проверку корректности домена
func (v *DomainListView) isValidDomain(domain string) bool {
	// Базовая проверка: домен не должен быть пустым и должен содержать точку
	if domain == "" {
		return false
	}

	// Проверяем, что домен не содержит пробелов
	if strings.Contains(domain, " ") {
		return false
	}

	// Проверяем наличие точки (минимальное требование для домена)
	if !strings.Contains(domain, ".") {
		return false
	}

	// Проверяем, что домен не начинается и не заканчивается точкой
	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return false
	}

	// Проверяем, что домен содержит только допустимые символы
	for _, char := range domain {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '.' || char == '-') {
			return false
		}
	}

	return true
}

// getCountText возвращает текст со счетчиком доменов
func (v *DomainListView) getCountText(text string) string {
	lines := strings.Split(text, "\n")
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return fmt.Sprintf("Доменов в списке: %d", count)
}
