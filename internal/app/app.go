package app

import (
	"embed"
	"fmt"
	"log/slog"

	"github.com/IProxymate/GoZapret/internal/domain"
	"github.com/IProxymate/GoZapret/internal/utils"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"
)

// MainViewInterface определяет интерфейс для главного представления
// Это позволяет избежать циклических зависимостей между app и ui пакетами
type MainViewInterface interface {
	Build() fyne.CanvasObject
	UpdateStrategyOptions(items []string, selectedStrategy string)
}

// MainViewFactory функция для создания MainView
// Устанавливается из ui пакета для избежания циклических зависимостей
type MainViewFactory func(app *App) MainViewInterface

// App представляет главное приложение
type App struct {
	// Fyne компоненты
	FyneApp fyne.App
	Window  fyne.Window
	Assets  embed.FS
	Logger  *slog.Logger

	// Сервисы (контейнер зависимостей)
	Services *Services

	// Состояние UI (биндинги)
	State *State

	// Внешние зависимости
	SingleInstance *utils.SingleInstance

	// Флаг полного запуска приложения
	AppStarted bool

	// UI компоненты
	mainView        MainViewInterface
	mainViewFactory MainViewFactory
}

// NewApp создает новое приложение
func NewApp(assets embed.FS, logger *slog.Logger, singleInstance *utils.SingleInstance, version string) *App {
	fyneApp := fyneapp.NewWithID(domain.AppID)

	// Создаем контейнер сервисов (версия передаётся через ldflags)
	services := NewServices(logger, version)

	// Создаем состояние UI
	state := NewState()
	state.InitFromConfig(services.Config, services.Process)

	// Синхронизируем автозапуск
	services.SyncAutostart(state.AutoStart, logger)

	app := &App{
		FyneApp:        fyneApp,
		Assets:         assets,
		Logger:         logger,
		Services:       services,
		State:          state,
		SingleInstance: singleInstance,
	}

	// Подписываемся на события
	app.subscribeToEvents()

	return app
}

// subscribeToEvents подписывает App на события от EventBus
func (a *App) subscribeToEvents() {
	eventBus := a.Services.EventBus

	// Подписка на изменение статуса
	eventBus.Subscribe(EventStatusChanged, func(event Event) {
		if data, ok := event.Data.(StatusEventData); ok {
			a.State.IsRunning.Set(data.IsRunning)
			a.State.StatusText.Set(data.StatusMessage)
			a.Logger.Debug("Статус обновлён через EventBus", "running", data.IsRunning, "message", data.StatusMessage)
		}
	})

	// Подписка на загрузку стратегий
	eventBus.Subscribe(EventStrategiesLoaded, func(event Event) {
		if data, ok := event.Data.(StrategiesLoadedData); ok {
			a.State.Strategies.Set(data.StrategyNames)
			a.State.SelectedStrategy.Set(data.SelectedStrategy)

			// Обновляем MainView если он существует
			if a.mainView != nil {
				a.mainView.UpdateStrategyOptions(data.StrategyNames, data.SelectedStrategy)
			}
			a.Logger.Debug("Стратегии обновлены через EventBus", "count", len(data.StrategyNames))
		}
	})

	// Подписка на ошибки стратегий
	eventBus.Subscribe(EventStrategyError, func(event Event) {
		if data, ok := event.Data.(StrategyEventData); ok && data.Error != nil {
			a.ShowProcessError(data.StrategyName, data.Error.Error())
		}
	})
}

// SetMainViewFactory устанавливает фабрику для создания MainView
func (a *App) SetMainViewFactory(factory MainViewFactory) {
	a.mainViewFactory = factory
}

// GetMainView возвращает главное представление
func (a *App) GetMainView() MainViewInterface {
	return a.mainView
}

// GetFyneApp возвращает Fyne приложение
func (a *App) GetFyneApp() fyne.App {
	return a.FyneApp
}

// GetWindow возвращает главное окно
func (a *App) GetWindow() fyne.Window {
	return a.Window
}

// UpdateStatus обновляет статус приложения
func (a *App) UpdateStatus() {
	fyne.Do(func() {
		running := a.Services.StrategyController.IsRunning()
		a.Logger.Debug("Обновление статуса приложения", "running", running)
		a.State.IsRunning.Set(running)

		statusMsg := a.Services.StrategyController.GetStatusMessage()
		a.State.StatusText.Set(statusMsg)
		a.Logger.Debug("Статус обновлен", "status", statusMsg)
	})
}

// ShowFolderDialog показывает диалог выбора папки
func (a *App) ShowFolderDialog() {
	fileDialog := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil || uri == nil {
			return
		}

		newPath := domain.AssetsPath(uri.Path())
		if err := a.Services.Config.SetAssetsPath(newPath); err != nil {
			dialog.ShowError(fmt.Errorf("ошибка сохранения пути: %v", err), a.Window)
			return
		}

		if err := a.Services.Config.PrepareWorkingDirectory(); err != nil {
			dialog.ShowError(fmt.Errorf("ошибка подготовки рабочей директории: %v", err), a.Window)
			return
		}

		a.LoadStrategies(newPath)
	}, a.Window)

	fileDialog.Resize(fyne.NewSize(800, 600))
	fileDialog.Show()
}

// ShowProcessError показывает диалог с ошибкой процесса
func (a *App) ShowProcessError(strategyName, errorMsg string) {
	a.Logger.Error("Ошибка процесса winws", "strategy", strategyName, "error", errorMsg)
	a.UpdateStatus()

	if a.Window != nil {
		a.Window.Show()
		a.Window.RequestFocus()
	}

	message := fmt.Sprintf("Стратегия '%s' завершилась с ошибкой:\n\n%s", strategyName, errorMsg)
	if len(message) > 1000 {
		message = message[:1000] + "\n\n... (сообщение обрезано)"
	}

	dialog.ShowError(fmt.Errorf("%s", message), a.Window)
}
