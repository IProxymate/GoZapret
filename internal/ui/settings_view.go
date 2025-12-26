package ui

import (
	"fmt"

	"github.com/IProxymate/GoZapret/internal/app"
	"github.com/IProxymate/GoZapret/internal/domain"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// SettingsView представляет окно настроек
type SettingsView struct {
	app    *app.App
	window fyne.Window
}

// NewSettingsView создает новое окно настроек
func NewSettingsView(a *app.App) *SettingsView {
	return &SettingsView{
		app: a,
	}
}

// Show показывает окно настроек
func (v *SettingsView) Show() {
	v.window = v.app.FyneApp.NewWindow("Настройки")
	v.window.Resize(fyne.NewSize(600, 300))
	v.window.CenterOnScreen()

	content := v.buildContent()
	v.window.SetContent(content)
	v.window.Show()
}

// buildContent строит содержимое окна настроек
func (v *SettingsView) buildContent() fyne.CanvasObject {
	// Путь к ресурсам
	assetsPath := v.app.Services.Config.GetAssetsPath()
	assetsPathEntry := widget.NewEntry()
	assetsPathEntry.SetText(assetsPath.String())
	assetsPathEntry.SetPlaceHolder("Путь к файлам zapret...")

	browseButton := widget.NewButton("Обзор...", func() {
		fileDialog := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			assetsPathEntry.SetText(uri.Path())
		}, v.window)
		fileDialog.Resize(fyne.NewSize(800, 600))
		fileDialog.Show()
	})

	pathContainer := container.NewBorder(nil, nil, nil, browseButton, assetsPathEntry)

	// Автозапуск
	autoStartCheck := widget.NewCheckWithData("Запускать приложение автоматически", v.app.State.AutoStart)
	autoStartCheck.OnChanged = func(checked bool) {
		// Обновляем конфигурацию и применяем настройку автозапуска через сервис
		if err := v.app.Services.Config.SetAutoStartWithService(checked, v.app.Services.Autostart); err != nil {
			dialog.ShowError(fmt.Errorf("ошибка настройки автозапуска: %v", err), v.window)
		} else {
			// Обновляем состояние биндинга, чтобы гарантировать согласованность
			v.app.State.AutoStart.Set(checked)
		}
	}

	// Кнопки
	saveButton := widget.NewButton("Сохранить", func() {
		newPath := domain.AssetsPath(assetsPathEntry.Text)

		// Валидируем путь
		if newPath != "" {
			if err := newPath.Validate(); err != nil {
				dialog.ShowError(err, v.window)
				return
			}
		}

		// Сохраняем путь
		if err := v.app.Services.Config.SetAssetsPath(newPath); err != nil {
			dialog.ShowError(err, v.window)
			return
		}

		// Перезагружаем стратегии и обновляем рабочую директорию
		if newPath != "" {
			if err := v.app.ReloadStrategies(); err != nil {
				dialog.ShowError(err, v.window)
				return
			}
		}

		dialog.ShowInformation("Успех", "Настройки сохранены", v.window)
		v.window.Close()
	})
	saveButton.Importance = widget.HighImportance

	closeButton := widget.NewButton("Закрыть", func() {
		v.window.Close()
	})

	buttonsContainer := container.NewGridWithColumns(2, saveButton, closeButton)

	// Главный контейнер
	content := container.NewVBox(
		widget.NewLabel("Путь к файлам zapret:"),
		pathContainer,
		widget.NewSeparator(),
		widget.NewLabel("Дополнительные настройки:"),
		autoStartCheck,
		widget.NewSeparator(),
		buttonsContainer,
	)

	return container.NewPadded(content)
}
