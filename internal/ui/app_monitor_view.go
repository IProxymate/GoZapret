package ui

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/IProxymate/GoZapret/internal/services/app_monitor"
)

// AppMonitorView представление для мониторинга сетевой активности приложений
type AppMonitorView struct {
	app            *App
	monitorService *app_monitor.Service
	monitorWindow  fyne.Window

	// UI компоненты
	pathEntry    *widget.Entry
	startButton  *widget.Button
	stopButton   *widget.Button
	statusLabel  *widget.Label
	requestsList *widget.List
	resultText   *widget.Entry
	progressBar  *widget.ProgressBarInfinite
	countLabel   *widget.Label

	// Данные
	requests []*app_monitor.NetworkRequest
}

// NewAppMonitorView создает новое представление мониторинга
func NewAppMonitorView(app *App) *AppMonitorView {
	workingDir := app.configManager.GetWorkingDir()
	return &AppMonitorView{
		app:            app,
		monitorService: app_monitor.NewService(workingDir),
		requests:       make([]*app_monitor.NetworkRequest, 0),
	}
}

// Show отображает окно мониторинга
func (v *AppMonitorView) Show() {
	v.monitorWindow = v.app.fyneApp.NewWindow("Мониторинг сетевой активности")
	v.monitorWindow.Resize(fyne.NewSize(950, 700))
	v.monitorWindow.CenterOnScreen()

	// Создаем UI компоненты
	v.createComponents()

	// Компоновка
	content := v.buildLayout()
	v.monitorWindow.SetContent(content)

	// Обработка закрытия окна
	v.monitorWindow.SetOnClosed(func() {
		if v.monitorService.IsMonitoring() {
			v.monitorService.StopMonitoring()
		}
	})

	v.monitorWindow.Show()
}

// createComponents создает UI компоненты
func (v *AppMonitorView) createComponents() {
	// Поле ввода пути
	v.pathEntry = widget.NewEntry()
	v.pathEntry.SetPlaceHolder("Путь к исполняемому файлу (.exe)")

	// Кнопки управления
	v.startButton = widget.NewButtonWithIcon("Начать мониторинг", theme.MediaPlayIcon(), v.startMonitoring)
	v.startButton.Importance = widget.HighImportance

	v.stopButton = widget.NewButtonWithIcon("Остановить", theme.MediaStopIcon(), v.stopMonitoring)
	v.stopButton.Importance = widget.DangerImportance
	v.stopButton.Disable()

	// Статус
	v.statusLabel = widget.NewLabel("Готов к мониторингу")
	v.countLabel = widget.NewLabel("Подключений: 0")

	// Прогресс бар
	v.progressBar = widget.NewProgressBarInfinite()
	v.progressBar.Hide()

	// Список запросов (вместо таблицы - более гибкий)
	v.requestsList = widget.NewList(
		func() int {
			return len(v.requests)
		},
		func() fyne.CanvasObject {
			return v.createRequestItem()
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			v.updateRequestItem(id, obj)
		},
	)

	// Текстовое поле для результатов
	v.resultText = widget.NewMultiLineEntry()
	v.resultText.Wrapping = fyne.TextWrapWord
	v.resultText.SetPlaceHolder("Результаты анализа появятся здесь после остановки мониторинга...")
}

// createRequestItem создает элемент списка для запроса
func (v *AppMonitorView) createRequestItem() fyne.CanvasObject {
	timeLabel := widget.NewLabel("00:00:00")
	timeLabel.TextStyle = fyne.TextStyle{Monospace: true}
	timeLabel.Alignment = fyne.TextAlignCenter

	ipLabel := widget.NewLabel("000.000.000.000")
	ipLabel.TextStyle = fyne.TextStyle{Monospace: true}
	ipLabel.Alignment = fyne.TextAlignCenter

	portLabel := widget.NewLabel("00000")
	portLabel.TextStyle = fyne.TextStyle{Monospace: true}
	portLabel.Alignment = fyne.TextAlignCenter

	domainLabel := widget.NewLabel("")
	domainLabel.Alignment = fyne.TextAlignLeading
	domainLabel.Truncation = fyne.TextTruncateEllipsis

	subnetLabel := widget.NewLabel("000.0.0.0/8")
	subnetLabel.TextStyle = fyne.TextStyle{Monospace: true}
	subnetLabel.Alignment = fyne.TextAlignCenter

	statusLabel := widget.NewLabel("❌")
	statusLabel.Alignment = fyne.TextAlignCenter

	// Пропорции: время 0.8, IP 1.5, порт 0.6, домен 2.5, подсеть 1, статус 0.5
	row := container.New(&proportionalLayout{
		proportions: []float32{0.8, 1.5, 0.6, 2.5, 1, 0.5},
	},
		container.NewCenter(timeLabel),
		container.NewCenter(ipLabel),
		container.NewCenter(portLabel),
		domainLabel,
		container.NewCenter(subnetLabel),
		container.NewCenter(statusLabel),
	)

	return row
}

// updateRequestItem обновляет элемент списка
func (v *AppMonitorView) updateRequestItem(id widget.ListItemID, obj fyne.CanvasObject) {
	if id >= len(v.requests) {
		return
	}

	req := v.requests[id]
	row := obj.(*fyne.Container)

	// Индексы: 0=time, 1=ip, 2=port, 3=domain, 4=subnet, 5=status
	// Время
	timeCenter := row.Objects[0].(*fyne.Container)
	timeLabel := timeCenter.Objects[0].(*widget.Label)
	timeLabel.SetText(req.Timestamp.Format("15:04:05"))

	// IP
	ipCenter := row.Objects[1].(*fyne.Container)
	ipLabel := ipCenter.Objects[0].(*widget.Label)
	ipLabel.SetText(req.IPAddress.String())

	// Порт
	portCenter := row.Objects[2].(*fyne.Container)
	portLabel := portCenter.Objects[0].(*widget.Label)
	portLabel.SetText(fmt.Sprintf("%d", req.Port))

	// Домен
	domainLabel := row.Objects[3].(*widget.Label)
	if req.Hostname != "" {
		domainLabel.SetText(req.Hostname)
	} else {
		domainLabel.SetText("-")
	}

	// Подсеть
	subnetCenter := row.Objects[4].(*fyne.Container)
	subnetLabel := subnetCenter.Objects[0].(*widget.Label)
	subnetLabel.SetText(req.Subnet)

	// Статус
	statusCenter := row.Objects[5].(*fyne.Container)
	statusLabel := statusCenter.Objects[0].(*widget.Label)
	if req.InIpset {
		statusLabel.SetText("✓")
	} else {
		statusLabel.SetText("✗")
	}
}

// buildLayout создает компоновку окна
func (v *AppMonitorView) buildLayout() fyne.CanvasObject {
	// Кнопка выбора файла
	browseButton := widget.NewButtonWithIcon("Обзор", theme.FolderOpenIcon(), v.showFileDialog)
	pathContainer := container.NewBorder(nil, nil, nil, browseButton, v.pathEntry)

	// Инструкция (компактная)
	instructionText := widget.NewRichTextFromMarkdown(`
**Как использовать:**
1. Укажите путь к .exe файлу игры → 2. Нажмите "Начать" → 3. Запустите игру → 4. Играйте → 5. Нажмите "Остановить"
`)

	// Панель управления
	controlPanel := container.NewVBox(
		container.NewBorder(nil, nil, widget.NewLabel("Приложение:"), nil, pathContainer),
		container.NewHBox(v.startButton, v.stopButton, widget.NewSeparator(), v.progressBar),
	)

	// Заголовок таблицы
	headerTime := widget.NewLabelWithStyle("Время", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	headerIP := widget.NewLabelWithStyle("IP адрес", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	headerPort := widget.NewLabelWithStyle("Порт", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	headerDomain := widget.NewLabelWithStyle("Домен", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	headerSubnet := widget.NewLabelWithStyle("Подсеть", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	headerStatus := widget.NewLabelWithStyle("IPset", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	tableHeader := container.New(&proportionalLayout{
		proportions: []float32{0.8, 1.5, 0.6, 2.5, 1, 0.5},
	},
		container.NewCenter(headerTime),
		container.NewCenter(headerIP),
		container.NewCenter(headerPort),
		container.NewCenter(headerDomain),
		container.NewCenter(headerSubnet),
		container.NewCenter(headerStatus),
	)

	// Карточка со списком
	listCard := widget.NewCard("", "", container.NewBorder(
		tableHeader,
		nil, nil, nil,
		v.requestsList,
	))

	// Статус панель
	statusPanel := container.NewHBox(
		v.statusLabel,
		widget.NewSeparator(),
		v.countLabel,
	)

	// Вкладка "Реальное время"
	realtimeTab := container.NewBorder(
		statusPanel,
		nil, nil, nil,
		listCard,
	)

	// Вкладка "Результаты"
	resultsTab := container.NewScroll(v.resultText)

	// Вкладки
	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon("Подключения", theme.ListIcon(), realtimeTab),
		container.NewTabItemWithIcon("Анализ", theme.DocumentIcon(), resultsTab),
	)

	// Кнопки внизу
	copyButton := widget.NewButtonWithIcon("Копировать недостающие", theme.ContentCopyIcon(), v.copyMissingSubnets)
	addButton := widget.NewButtonWithIcon("Добавить в IPset", theme.ContentAddIcon(), v.addMissingToIpset)
	closeButton := widget.NewButtonWithIcon("Закрыть", theme.CancelIcon(), func() {
		v.monitorWindow.Close()
	})

	bottomPanel := container.NewBorder(
		widget.NewSeparator(),
		nil, nil, nil,
		container.NewHBox(copyButton, addButton, widget.NewLabel(""), closeButton),
	)

	// Верхняя панель
	topPanel := container.NewVBox(
		instructionText,
		widget.NewSeparator(),
		controlPanel,
		widget.NewSeparator(),
	)

	return container.NewBorder(topPanel, bottomPanel, nil, nil, tabs)
}

// showFileDialog показывает диалог выбора файла
func (v *AppMonitorView) showFileDialog() {
	fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return
		}
		defer reader.Close()

		path := reader.URI().Path()
		// Убираем ведущий слеш на Windows
		if len(path) > 2 && path[0] == '/' && path[2] == ':' {
			path = path[1:]
		}
		v.pathEntry.SetText(path)
	}, v.monitorWindow)

	fd.SetFilter(storage.NewExtensionFileFilter([]string{".exe"}))
	fd.Resize(fyne.NewSize(800, 600))
	fd.Show()
}

// startMonitoring начинает мониторинг
func (v *AppMonitorView) startMonitoring() {
	path := v.pathEntry.Text
	if path == "" {
		dialog.ShowError(fmt.Errorf("укажите путь к исполняемому файлу"), v.monitorWindow)
		return
	}

	// Очищаем предыдущие данные
	v.requests = make([]*app_monitor.NetworkRequest, 0)
	v.requestsList.Refresh()
	v.resultText.SetText("")
	v.countLabel.SetText("Подключений: 0")

	// Обновляем чекеры и пересоздаём монитор
	v.monitorService.RefreshCheckers()

	// Регистрируем callback для новых запросов
	v.monitorService.OnRequest(func(req *app_monitor.NetworkRequest) {
		fyne.Do(func() {
			v.addRequest(req)
		})
	})

	// Запускаем мониторинг
	err := v.monitorService.StartMonitoring(path)
	if err != nil {
		dialog.ShowError(fmt.Errorf("ошибка запуска мониторинга: %v", err), v.monitorWindow)
		return
	}

	// Обновляем UI
	v.startButton.Disable()
	v.stopButton.Enable()
	v.progressBar.Show()
	v.statusLabel.SetText("⏳ Ожидание запуска приложения...")

	// Запускаем обновление статуса
	go v.updateStatus()
}

// stopMonitoring останавливает мониторинг
func (v *AppMonitorView) stopMonitoring() {
	result := v.monitorService.StopMonitoring()

	// Обновляем UI
	v.startButton.Enable()
	v.stopButton.Disable()
	v.progressBar.Hide()
	v.statusLabel.SetText("✅ Мониторинг завершен")
	v.countLabel.SetText(fmt.Sprintf("Подключений: %d", len(v.requests)))

	// Показываем результаты
	if result != nil {
		text := v.monitorService.FormatResultAsText(result)
		v.resultText.SetText(text)
	}
}

// addRequest добавляет запрос в список
func (v *AppMonitorView) addRequest(req *app_monitor.NetworkRequest) {
	v.requests = append(v.requests, req)
	v.requestsList.Refresh()

	// Прокручиваем к последнему элементу
	if len(v.requests) > 0 {
		v.requestsList.ScrollToBottom()
	}

	// Обновляем счетчик
	v.countLabel.SetText(fmt.Sprintf("Подключений: %d", len(v.requests)))
}

// updateStatus обновляет статус во время мониторинга
func (v *AppMonitorView) updateStatus() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	startTime := time.Now()
	processFound := false

	for {
		select {
		case <-ticker.C:
			if !v.monitorService.IsMonitoring() {
				return
			}
			elapsed := time.Since(startTime).Round(time.Second)
			requestCount := len(v.requests)

			fyne.Do(func() {
				if requestCount > 0 {
					processFound = true
					v.statusLabel.SetText(fmt.Sprintf("🔍 Мониторинг %s", elapsed.String()))
				} else if !processFound {
					v.statusLabel.SetText(fmt.Sprintf("⏳ Ожидание %s — запустите игру!", elapsed.String()))
				} else {
					v.statusLabel.SetText(fmt.Sprintf("🔍 Мониторинг %s", elapsed.String()))
				}
			})
		}
	}
}

// copyMissingSubnets копирует недостающие подсети в буфер обмена
func (v *AppMonitorView) copyMissingSubnets() {
	missing := make(map[string]bool)
	for _, req := range v.requests {
		if !req.InIpset && req.Subnet != "" {
			missing[req.Subnet] = true
		}
	}

	if len(missing) == 0 {
		dialog.ShowInformation("Информация", "Все подсети уже есть в IPset!", v.monitorWindow)
		return
	}

	var subnets []string
	for subnet := range missing {
		subnets = append(subnets, subnet)
	}

	text := strings.Join(subnets, "\n")
	v.monitorWindow.Clipboard().SetContent(text)

	dialog.ShowInformation("Скопировано",
		fmt.Sprintf("Скопировано %d подсетей:\n\n%s", len(subnets), text),
		v.monitorWindow)
}

// addMissingToIpset добавляет недостающие подсети в ipset
func (v *AppMonitorView) addMissingToIpset() {
	missing := make(map[string]bool)
	for _, req := range v.requests {
		if !req.InIpset && req.Subnet != "" {
			missing[req.Subnet] = true
		}
	}

	if len(missing) == 0 {
		dialog.ShowInformation("Информация", "Все подсети уже есть в IPset!", v.monitorWindow)
		return
	}

	var subnets []string
	for subnet := range missing {
		subnets = append(subnets, subnet)
	}

	dialog.ShowConfirm("Добавить подсети",
		fmt.Sprintf("Добавить %d подсетей в пользовательский IPset?\n\n%s", len(subnets), strings.Join(subnets, "\n")),
		func(confirmed bool) {
			if !confirmed {
				return
			}
			v.doAddToIpset(subnets)
		},
		v.monitorWindow)
}

// doAddToIpset выполняет добавление подсетей в ipset
func (v *AppMonitorView) doAddToIpset(subnets []string) {
	customIpsetPath := v.app.configManager.GetCustomIpsetPath()

	// Открываем окно редактирования ipset
	ipsetListView := NewIpsetListView(v.app, customIpsetPath, "Пользовательские подсети (IPset)")
	ipsetListView.Show()

	dialog.ShowInformation("Добавьте подсети",
		fmt.Sprintf("Добавьте следующие подсети в открывшемся окне:\n\n%s", strings.Join(subnets, "\n")),
		v.monitorWindow)
}

// proportionalLayout распределяет элементы пропорционально
type proportionalLayout struct {
	proportions []float32
}

func (p *proportionalLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	height := float32(30)
	for _, obj := range objects {
		if obj.MinSize().Height > height {
			height = obj.MinSize().Height
		}
	}
	return fyne.NewSize(400, height)
}

func (p *proportionalLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}

	// Вычисляем сумму пропорций
	var totalProportion float32
	for i := 0; i < len(objects) && i < len(p.proportions); i++ {
		totalProportion += p.proportions[i]
	}

	// Распределяем ширину
	x := float32(0)
	for i, obj := range objects {
		proportion := float32(1)
		if i < len(p.proportions) {
			proportion = p.proportions[i]
		}

		width := (proportion / totalProportion) * size.Width
		obj.Resize(fyne.NewSize(width, size.Height))
		obj.Move(fyne.NewPos(x, 0))
		x += width
	}
}
