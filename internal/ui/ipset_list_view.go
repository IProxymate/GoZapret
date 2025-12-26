package ui

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/IProxymate/GoZapret/internal/app"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// IpsetListView представляет окно редактирования списка подсетей
type IpsetListView struct {
	app      *app.App
	filePath string
	title    string
	window   fyne.Window
}

// NewIpsetListView создает новое окно редактирования списка подсетей
func NewIpsetListView(a *app.App, filePath, title string) *IpsetListView {
	return &IpsetListView{
		app:      a,
		filePath: filePath,
		title:    title,
	}
}

// Show показывает окно редактирования
func (v *IpsetListView) Show() {
	v.window = v.app.FyneApp.NewWindow(v.title)
	v.window.Resize(fyne.NewSize(600, 500))
	v.window.CenterOnScreen()

	content := v.buildContent()
	v.window.SetContent(content)
	v.window.Show()
}

// buildContent создает содержимое окна
func (v *IpsetListView) buildContent() fyne.CanvasObject {
	// Загружаем текущее содержимое файла
	subnets, err := v.loadSubnets()
	if err != nil {
		v.app.Logger.Error("Ошибка загрузки списка подсетей", "file", v.filePath, "error", err)
		subnets = ""
	}

	// Создаем многострочное текстовое поле
	entry := widget.NewMultiLineEntry()
	entry.SetText(subnets)
	entry.Wrapping = fyne.TextWrapWord
	entry.SetPlaceHolder("Введите IP-адреса или подсети построчно, например:\n192.168.1.0/24\n10.0.0.0/8\n34.149.116.40/32\n3.0.0.0/8")

	// Информационная метка
	infoLabel := widget.NewLabel("Добавьте IP-адреса или подсети в формате CIDR построчно.\nПримеры: 192.168.1.0/24, 10.0.0.1/32, 3.0.0.0/8\nДля одиночного IP используйте /32 (например: 34.149.116.40/32)")
	infoLabel.Wrapping = fyne.TextWrapWord

	// Счетчик записей
	countLabel := widget.NewLabel(v.getCountText(subnets))

	// Обновляем счетчик при изменении текста
	entry.OnChanged = func(text string) {
		countLabel.SetText(v.getCountText(text))
	}

	// Кнопки
	saveButton := widget.NewButton("Сохранить", func() {
		v.saveSubnets(entry.Text)
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
		v.validateSubnets(entry.Text)
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

// loadSubnets загружает подсети из файла
func (v *IpsetListView) loadSubnets() (string, error) {
	data, err := os.ReadFile(v.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// saveSubnets сохраняет подсети в файл
func (v *IpsetListView) saveSubnets(text string) {
	// Нормализуем текст: убираем пустые строки и лишние пробелы
	lines := strings.Split(text, "\n")
	var validLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			// Если указан просто IP без маски, добавляем /32
			if !strings.Contains(line, "/") {
				if net.ParseIP(line) != nil {
					line = line + "/32"
				}
			}
			validLines = append(validLines, line)
		} else if strings.HasPrefix(line, "#") {
			// Сохраняем комментарии
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

	// Обновляем ipset файл
	v.updateIpsetFile()

	dialog.ShowInformation("Успех",
		fmt.Sprintf("Список успешно сохранен.\nЗаписей: %d", len(validLines)),
		v.window)
	v.window.Close()
}

// updateIpsetFile обновляет файл ipset-all.txt с учетом пользовательских подсетей
func (v *IpsetListView) updateIpsetFile() {
	workingDir := v.app.Services.Config.GetWorkingDir()
	if workingDir == "" {
		return
	}

	mode, _ := v.app.State.IpsetMode.Get()
	if mode == "" {
		mode = "loaded"
	}

	// Обновляем ipset файл через сервис
	v.app.Services.Ipset.UpdateIpsetFile(workingDir, mode)
}

// restartStrategyIfRunning перезапускает стратегию, если процесс запущен.
// Делегирует вызов App.RestartCurrentStrategy()
func (v *IpsetListView) restartStrategyIfRunning() {
	v.app.RestartCurrentStrategy()
}

// validateSubnets проверяет корректность подсетей
func (v *IpsetListView) validateSubnets(text string) {
	lines := strings.Split(text, "\n")
	var validSubnets []string
	var invalidLines []string

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Проверяем корректность подсети
		if v.isValidSubnet(line) {
			validSubnets = append(validSubnets, line)
		} else {
			invalidLines = append(invalidLines, fmt.Sprintf("Строка %d: %s", i+1, line))
		}
	}

	var message string
	if len(invalidLines) == 0 {
		message = fmt.Sprintf("✓ Все записи корректны\nВсего записей: %d", len(validSubnets))
	} else {
		message = fmt.Sprintf("Найдены некорректные записи:\n\n%s\n\nКорректных записей: %d",
			strings.Join(invalidLines, "\n"), len(validSubnets))
	}

	dialog.ShowInformation("Результат проверки", message, v.window)
}

// isValidSubnet проверяет корректность IP-адреса или подсети
func (v *IpsetListView) isValidSubnet(subnet string) bool {
	subnet = strings.TrimSpace(subnet)
	if subnet == "" {
		return false
	}

	// Если нет маски, проверяем как обычный IP
	if !strings.Contains(subnet, "/") {
		return net.ParseIP(subnet) != nil
	}

	// Проверяем как CIDR
	_, _, err := net.ParseCIDR(subnet)
	return err == nil
}

// getCountText возвращает текст со счетчиком записей
func (v *IpsetListView) getCountText(text string) string {
	lines := strings.Split(text, "\n")
	count := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			count++
		}
	}
	return fmt.Sprintf("Записей в списке: %d", count)
}

