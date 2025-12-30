package ui

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/IProxymate/GoZapret/internal/app"
	"github.com/IProxymate/GoZapret/internal/services/hosts"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// HostsView представляет окно редактирования файла hosts
type HostsView struct {
	app          *app.App
	hostsService *hosts.Service
	window       fyne.Window
}

// NewHostsView создает новое окно редактирования hosts
func NewHostsView(a *app.App) *HostsView {
	return &HostsView{
		app:          a,
		hostsService: hosts.NewService(),
	}
}

// Show показывает окно редактирования
func (v *HostsView) Show() {
	v.window = v.app.FyneApp.NewWindow("Редактор файла hosts")
	v.window.Resize(fyne.NewSize(800, 600))
	v.window.CenterOnScreen()

	content := v.buildContent()
	v.window.SetContent(content)
	v.window.Show()
}

// buildContent создает содержимое окна
func (v *HostsView) buildContent() fyne.CanvasObject {
	// Загружаем текущее содержимое файла hosts
	content, err := v.hostsService.Read()
	if err != nil {
		v.app.Logger.Error("Ошибка загрузки файла hosts", "error", err)
		content = ""
	}

	// Создаем многострочное текстовое поле
	entry := widget.NewMultiLineEntry()
	entry.SetText(content)
	entry.Wrapping = fyne.TextWrapOff // Не переносим строки для hosts файла
	entry.SetPlaceHolder("# Файл hosts пуст\n# Формат: IP_адрес hostname\n# Пример:\n# 127.0.0.1 localhost\n# 192.168.1.1 myserver.local")

	// Информационная метка
	infoLabel := widget.NewLabel(fmt.Sprintf(
		"Редактирование системного файла hosts\nПуть: %s\n\n"+
			"Формат записи: IP_адрес<TAB>hostname [# комментарий]\n"+
			"Пример: 127.0.0.1    example.com    # блокировка сайта",
		v.hostsService.GetHostsPath(),
	))
	infoLabel.Wrapping = fyne.TextWrapWord

	// Счетчик записей
	countLabel := widget.NewLabel(v.getCountText(content))

	// Обновляем счетчик при изменении текста
	entry.OnChanged = func(text string) {
		countLabel.SetText(v.getCountText(text))
	}

	// Кнопки
	saveButton := widget.NewButton("Сохранить", func() {
		v.saveHosts(entry.Text)
	})
	saveButton.Importance = widget.HighImportance

	cancelButton := widget.NewButton("Отмена", func() {
		v.window.Close()
	})

	// Кнопка валидации
	validateButton := widget.NewButton("Проверить", func() {
		v.validateHosts(entry.Text)
	})

	// Ряд кнопок
	buttonContainer := container.NewGridWithColumns(3, saveButton, validateButton, cancelButton)

	// Предупреждение о правах администратора
	warningLabel := widget.NewLabel("⚠️ Для редактирования файла hosts требуются права администратора")
	warningLabel.Importance = widget.WarningImportance

	// Основной контейнер
	topContent := container.NewVBox(
		widget.NewCard("", "", infoLabel),
		warningLabel,
		countLabel,
	)

	mainContent := container.NewBorder(
		topContent,
		buttonContainer,
		nil,
		nil,
		container.NewScroll(entry),
	)

	return container.NewPadded(mainContent)
}

// saveHosts сохраняет содержимое в файл hosts
func (v *HostsView) saveHosts(text string) {
	// Подтверждение перед сохранением
	dialog.ShowConfirm("Подтверждение",
		"Вы уверены, что хотите сохранить изменения в файле hosts?\n\n"+
			"Будет создана резервная копия текущего файла.",
		func(confirmed bool) {
			if !confirmed {
				return
			}

			err := v.hostsService.Write(text)
			if err != nil {
				dialog.ShowError(fmt.Errorf("ошибка сохранения файла hosts: %v\n\n"+
					"Убедитесь, что приложение запущено от имени администратора.", err), v.window)
				return
			}

			// Предлагаем сбросить DNS кэш
			dialog.ShowConfirm("Файл сохранен",
				"Файл hosts успешно сохранен.\n\nСбросить DNS кэш для применения изменений?",
				func(flush bool) {
					if flush {
						v.flushDNS()
					}
					v.window.Close()
				},
				v.window)
		},
		v.window)
}

// flushDNS сбрасывает DNS кэш Windows
func (v *HostsView) flushDNS() {
	cmd := exec.Command("ipconfig", "/flushdns")
	output, err := cmd.CombinedOutput()
	if err != nil {
		dialog.ShowError(fmt.Errorf("ошибка сброса DNS кэша: %v\n%s", err, string(output)), v.window)
		return
	}

	dialog.ShowInformation("Успех", "DNS кэш успешно сброшен", v.window)
}

// validateHosts проверяет корректность записей в hosts
func (v *HostsView) validateHosts(text string) {
	lines := strings.Split(text, "\n")
	var validEntries int
	var invalidLines []string
	var comments int

	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Пропускаем пустые строки
		if line == "" {
			continue
		}

		// Считаем комментарии
		if strings.HasPrefix(line, "#") {
			comments++
			continue
		}

		// Парсим запись
		parts := strings.Fields(line)
		if len(parts) < 2 {
			invalidLines = append(invalidLines, fmt.Sprintf("Строка %d: недостаточно полей - %s", i+1, line))
			continue
		}

		ip := parts[0]
		hostname := parts[1]

		// Проверяем IP
		if !v.hostsService.ValidateIP(ip) {
			invalidLines = append(invalidLines, fmt.Sprintf("Строка %d: некорректный IP - %s", i+1, ip))
			continue
		}

		// Проверяем hostname
		if !v.hostsService.ValidateHostname(hostname) {
			invalidLines = append(invalidLines, fmt.Sprintf("Строка %d: некорректный hostname - %s", i+1, hostname))
			continue
		}

		validEntries++
	}

	var message string
	if len(invalidLines) == 0 {
		message = fmt.Sprintf("✓ Все записи корректны\n\nАктивных записей: %d\nКомментариев: %d",
			validEntries, comments)
	} else {
		message = fmt.Sprintf("Найдены некорректные записи:\n\n%s\n\nКорректных записей: %d\nКомментариев: %d",
			strings.Join(invalidLines, "\n"), validEntries, comments)
	}

	dialog.ShowInformation("Результат проверки", message, v.window)
}

// getCountText возвращает текст со счетчиком записей
func (v *HostsView) getCountText(text string) string {
	lines := strings.Split(text, "\n")
	activeCount := 0
	commentCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			commentCount++
		} else {
			// Проверяем, что это валидная запись (минимум 2 поля)
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				activeCount++
			}
		}
	}

	return fmt.Sprintf("Активных записей: %d | Комментариев/отключенных: %d", activeCount, commentCount)
}

