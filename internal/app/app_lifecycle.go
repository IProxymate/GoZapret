package app

import (
	"os"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"

	"github.com/IProxymate/GoZapret/internal/domain"
)

// Run запускает приложение
func (a *App) Run() {
	a.setupSingleInstance()
	a.setupLifecycle()

	iconData := a.loadIcon()

	a.Window = a.FyneApp.NewWindow(domain.AppName)
	a.setupTray(iconData)
	a.requestAssetsPath()
	a.setupProcessErrorCallback()
	a.setupSelfUpdater()

	// Создаём MainView через фабрику
	if a.mainViewFactory != nil {
		a.mainView = a.mainViewFactory(a)
		a.Window.SetContent(a.mainView.Build())
	}

	if slices.Contains(os.Args[1:], domain.ArgAutostart) {
		a.runInAutostartMode()
	} else {
		a.runInNormalMode()
	}
}

// setupSingleInstance настраивает обработку одиночного экземпляра
func (a *App) setupSingleInstance() {
	a.SingleInstance.SetActivateCallback(a.ActivateWindow)
	if err := a.SingleInstance.StartIPCServer(); err != nil {
		a.Logger.Warn("Не удалось запустить IPC сервер", "error", err)
	}
}

// setupLifecycle настраивает обработчики жизненного цикла
func (a *App) setupLifecycle() {
	a.FyneApp.Lifecycle().SetOnStopped(a.onAppStopped)
}

// setupProcessErrorCallback устанавливает callback для обработки ошибок процесса
func (a *App) setupProcessErrorCallback() {
	a.Services.Process.SetErrorCallback(func(strategyName, errorMsg string) {
		fyne.Do(func() {
			a.ShowProcessError(strategyName, errorMsg)
		})
	})
}

// runInAutostartMode запускает приложение в режиме автозапуска
func (a *App) runInAutostartMode() {
	a.handleAutostart()
	a.UpdateStatus()
	a.AppStarted = true
	a.FyneApp.Run()
}

// runInNormalMode запускает приложение в обычном режиме
func (a *App) runInNormalMode() {
	a.UpdateStatus()
	a.AppStarted = true
	a.Window.Resize(fyne.NewSize(800, 600))
	a.Window.CenterOnScreen()
	a.Window.ShowAndRun()
}

// handleAutostart обрабатывает автозапуск приложения
func (a *App) handleAutostart() {
	lastStrategyName := a.Services.Config.GetLastStrategyName()
	if lastStrategyName == "" {
		a.Logger.Debug("Нет последней стратегии для автозапуска")
		a.Window.Hide()
		return
	}

	strategy, err := a.Services.Strategy.GetByName(lastStrategyName)
	if err != nil || strategy == nil {
		a.Logger.Warn("Последняя стратегия не найдена при автозапуске", "strategy", lastStrategyName, "error", err)
		a.Window.Hide()
		return
	}

	err = a.Services.Process.StartStrategy(strategy, a.Services.Config.GetAssetsPath(), a.Services.Config.GetGameFilter())
	if err != nil {
		a.Logger.Error("Ошибка запуска стратегии при автозапуске", "strategy", strategy.Name, "error", err)
	} else {
		a.Logger.Debug("Стратегия запущена при автозапуске", "strategy", strategy.Name)
	}

	a.Window.Hide()
}

// onAppStopped вызывается при завершении приложения
func (a *App) onAppStopped() {
	if !a.AppStarted {
		a.Logger.Debug("Игнорируем OnStopped - приложение ещё не запущено")
		return
	}
	a.Logger.Debug("Приложение завершается (Lifecycle.OnStopped)")
	a.stopWinwsProcess()
}

// Shutdown завершает приложение и останавливает все процессы
func (a *App) Shutdown() {
	a.Logger.Info("Завершение приложения")
	a.stopWinwsProcess()
	a.FyneApp.Quit()
}

// stopWinwsProcess останавливает процесс winws, если он запущен
func (a *App) stopWinwsProcess() {
	if !a.Services.Process.IsRunning() && !a.Services.Process.IsWinwsProcessRunning() {
		return
	}

	a.Logger.Debug("Остановка процесса winws")
	if err := a.Services.Process.StopProcess(); err != nil {
		a.Logger.Warn("Ошибка остановки процесса", "error", err)
	}
}

// ActivateWindow активирует и показывает главное окно приложения
func (a *App) ActivateWindow() {
	a.Logger.Debug("Активация окна приложения")

	fyne.Do(func() {
		if a.Window == nil {
			a.Logger.Warn("Окно не инициализировано")
			return
		}
		a.Window.Show()
		a.Window.RequestFocus()
		a.Logger.Info("Окно успешно активировано")
	})
}

// setupSelfUpdater настраивает сервис самообновления приложения
func (a *App) setupSelfUpdater() {
	if err := a.Services.SelfUpdate.SetupWithFyne(a.FyneApp, a.Window); err != nil {
		a.Logger.Warn("Не удалось настроить self-updater", "error", err)
	}
}

// requestAssetsPath запрашивает у пользователя путь к файлам zapret
func (a *App) requestAssetsPath() {
	assetsPath := a.Services.Config.GetAssetsPath()
	if assetsPath != "" {
		a.LoadStrategies(assetsPath)
		return
	}

	msg := "Добро пожаловать в GoZapret!\n\n" +
		"Для работы приложения необходимо указать путь к папке с файлами Zapret.\n" +
		"Обычно это папка, содержащая файлы winws.exe, service.bat и другие компоненты.\n\n" +
		"Нажмите OK, чтобы выбрать папку."

	infoDialog := dialog.NewInformation("Путь к файлам zapret не установлен", msg, a.Window)
	infoDialog.SetOnClosed(func() {
		a.ShowFolderDialog()
	})
	infoDialog.Show()
}

