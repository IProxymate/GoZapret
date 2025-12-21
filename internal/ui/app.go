package ui

import (
	"embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/IProxymate/GoZapret/internal/config"
	"github.com/IProxymate/GoZapret/internal/domain"
	"github.com/IProxymate/GoZapret/internal/services"
	"github.com/IProxymate/GoZapret/internal/services/autostart"
	"github.com/IProxymate/GoZapret/internal/services/diagnostics"
	"github.com/IProxymate/GoZapret/internal/services/diagnostics/checks"
	"github.com/IProxymate/GoZapret/internal/services/ipset"
	"github.com/IProxymate/GoZapret/internal/services/process"
	"github.com/IProxymate/GoZapret/internal/services/strategy"
	"github.com/IProxymate/GoZapret/internal/services/updates"
	"github.com/IProxymate/GoZapret/internal/utils"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
)

// App представляет главное приложение
type App struct {
	fyneApp fyne.App
	window  fyne.Window
	assets  embed.FS
	logger  *slog.Logger

	// Сервисы
	configManager    *config.Manager
	strategyService  *strategy.Service
	processManager   *process.Manager
	adminChecker     *services.AdminChecker
	diagnostics      *diagnostics.Service
	ipsetService     *ipset.Service
	cacheService     *services.CacheService
	autostartService *autostart.Service
	updateService    *updates.Service
	singleInstance   *utils.SingleInstance

	// Биндинги для UI
	statusText       binding.String
	isRunning        binding.Bool
	selectedStrategy binding.String
	strategies       binding.StringList
	autoStart        binding.Bool
	gameFilter       binding.Bool
	ipsetMode        binding.String
	version          binding.String

	// Флаг полного запуска приложения
	appStarted bool

	// UI компоненты
	mainView *MainView
}

// NewApp создает новое приложение
func NewApp(assets embed.FS, logger *slog.Logger, singleInstance *utils.SingleInstance) *App {
	fyneApp := app.NewWithID("com.zapret.gui")

	configManager := initConfigManager(logger)
	services := initServices(configManager, logger)

	bindings := initBindings(configManager, services.processManager)

	syncAutostart(configManager, services.autostartService, bindings.autoStart, logger)

	return &App{
		fyneApp:          fyneApp,
		assets:           assets,
		logger:           logger,
		configManager:    configManager,
		strategyService:  services.strategyService,
		processManager:   services.processManager,
		adminChecker:     services.adminChecker,
		diagnostics:      services.diagnosticsService,
		ipsetService:     services.ipsetService,
		cacheService:     services.cacheService,
		autostartService: services.autostartService,
		updateService:    services.updateService,
		singleInstance:   singleInstance,
		statusText:       bindings.statusText,
		isRunning:        bindings.isRunning,
		selectedStrategy: bindings.selectedStrategy,
		strategies:       bindings.strategies,
		autoStart:        bindings.autoStart,
		gameFilter:       bindings.gameFilter,
		ipsetMode:        bindings.ipsetMode,
		version:          bindings.version,
	}
}

// appServices содержит все сервисы приложения
type appServices struct {
	strategyService    *strategy.Service
	processManager     *process.Manager
	adminChecker       *services.AdminChecker
	diagnosticsService *diagnostics.Service
	ipsetService       *ipset.Service
	cacheService       *services.CacheService
	autostartService   *autostart.Service
	updateService      *updates.Service
}

// appBindings содержит все биндинги UI
type appBindings struct {
	statusText       binding.String
	isRunning        binding.Bool
	selectedStrategy binding.String
	strategies       binding.StringList
	autoStart        binding.Bool
	gameFilter       binding.Bool
	ipsetMode        binding.String
	version          binding.String
}

// initConfigManager инициализирует менеджер конфигурации
func initConfigManager(logger *slog.Logger) *config.Manager {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	configPath := filepath.Join(configDir, "GoZapret", "config.json")

	configManager := config.NewManager(configPath)
	if err := configManager.Load(); err != nil {
		logger.Error("Ошибка загрузки конфигурации", "error", err)
	}

	return configManager
}

// initServices инициализирует все сервисы приложения
func initServices(configManager *config.Manager, logger *slog.Logger) *appServices {
	adminChecker := services.NewAdminChecker()
	strategyService := strategy.NewService()

	// ProcessManager
	processManager := process.NewManager(
		process.NewProcessExecutor(),
		process.NewWindowsProcessMonitor(),
		process.NewBatParser(),
		process.NewArgsBuilder(configManager),
		adminChecker,
		configManager,
	)

	// DiagnosticsService
	diagnosticsService := diagnostics.NewService([]diagnostics.Checker{
		checks.NewAdminCheck(adminChecker),
		checks.NewWinDivertCheck(),
		checks.NewNetworkCheck(),
		checks.NewWinwsCheck(),
		checks.NewBFECheck(),
		checks.NewConflictingBypassesCheck(),
		checks.NewProxyCheck(),
		checks.NewTCPTimestampsCheck(),
		checks.NewAdguardCheck(),
		checks.NewKillerServicesCheck(),
		checks.NewIntelConnectivityCheck(),
		checks.NewCheckPointCheck(),
		checks.NewSmartByteCheck(),
		checks.NewVPNServicesCheck(),
	})

	return &appServices{
		strategyService:    strategyService,
		processManager:     processManager,
		adminChecker:       adminChecker,
		diagnosticsService: diagnosticsService,
		ipsetService:       ipset.NewService(),
		cacheService:       services.NewCacheService(),
		autostartService:   autostart.NewService(),
		updateService:      updates.NewService("https://api.github.com/repos/Flowseal/zapret-discord-youtube/releases/latest"),
	}
}

// initBindings инициализирует биндинги UI
func initBindings(configManager *config.Manager, processManager *process.Manager) *appBindings {
	statusText := binding.NewString()
	isRunning := binding.NewBool()

	cfg := configManager.GetConfig()

	// Устанавливаем начальный статус
	if processManager.IsWinwsProcessRunning() {
		if cfg.LastStrategyName != "" {
			statusText.Set(fmt.Sprintf("Статус: Запущено (%s)", cfg.LastStrategyName))
		} else {
			statusText.Set("Статус: Запущено")
		}
		isRunning.Set(true)
	} else {
		statusText.Set("Статус: Остановлено")
		isRunning.Set(false)
	}

	autoStart := binding.NewBool()
	autoStart.Set(cfg.AutoStart)

	gameFilter := binding.NewBool()
	gameFilter.Set(cfg.GameFilter)

	ipsetMode := binding.NewString()
	ipsetMode.Set(cfg.IpsetMode)

	version := binding.NewString()
	version.Set(cfg.Version)

	return &appBindings{
		statusText:       statusText,
		isRunning:        isRunning,
		selectedStrategy: binding.NewString(),
		strategies:       binding.NewStringList(),
		autoStart:        autoStart,
		gameFilter:       gameFilter,
		ipsetMode:        ipsetMode,
		version:          version,
	}
}

// syncAutostart синхронизирует состояние автозапуска между конфигом и системой
func syncAutostart(configManager *config.Manager, autostartService *autostart.Service, autoStartBinding binding.Bool, logger *slog.Logger) {
	cfg := configManager.GetConfig()

	isEnabled, err := autostartService.IsAutoStartEnabled()
	if err != nil {
		logger.Warn("Ошибка проверки состояния автозапуска", "error", err)
		isEnabled = cfg.AutoStart
	}

	switch {
	case cfg.AutoStart && !isEnabled:
		if err := autostartService.SetAutoStart(true); err != nil {
			logger.Error("Ошибка установки автозапуска при старте", "error", err)
			autoStartBinding.Set(false)
		} else {
			autoStartBinding.Set(true)
		}
	case !cfg.AutoStart && isEnabled:
		if err := autostartService.SetAutoStart(false); err != nil {
			logger.Error("Ошибка отключения автозапуска при старте", "error", err)
		}
		autoStartBinding.Set(false)
	default:
		autoStartBinding.Set(cfg.AutoStart)
	}
}

// Run запускает приложение
func (a *App) Run() {
	a.setupSingleInstance()
	a.setupLifecycle()

	iconData := a.loadIcon()

	a.window = a.fyneApp.NewWindow("GoZapret")
	a.setupTray(iconData)
	a.requestAssetsPath()
	a.setupProcessErrorCallback()

	a.mainView = NewMainView(a)
	a.window.SetContent(a.mainView.Build())

	if slices.Contains(os.Args[1:], "/autostart") {
		a.runInAutostartMode()
	} else {
		a.runInNormalMode()
	}
}

// setupSingleInstance настраивает обработку одиночного экземпляра
func (a *App) setupSingleInstance() {
	a.singleInstance.SetActivateCallback(a.ActivateWindow)
	if err := a.singleInstance.StartIPCServer(); err != nil {
		a.logger.Warn("Не удалось запустить IPC сервер", "error", err)
	}
}

// setupLifecycle настраивает обработчики жизненного цикла
func (a *App) setupLifecycle() {
	a.fyneApp.Lifecycle().SetOnStopped(a.onAppStopped)
}

// loadIcon загружает иконку приложения
func (a *App) loadIcon() []byte {
	iconData, err := a.loadIconData()
	if err == nil {
		a.fyneApp.SetIcon(fyne.NewStaticResource("icon256.png", iconData))
	}
	return iconData
}

// setupProcessErrorCallback устанавливает callback для обработки ошибок процесса
func (a *App) setupProcessErrorCallback() {
	a.processManager.SetErrorCallback(func(strategyName, errorMsg string) {
		fyne.Do(func() {
			a.showProcessError(strategyName, errorMsg)
		})
	})
}

// runInAutostartMode запускает приложение в режиме автозапуска
func (a *App) runInAutostartMode() {
	a.handleAutostart()
	a.updateStatus()
	a.appStarted = true
	a.fyneApp.Run()
}

// runInNormalMode запускает приложение в обычном режиме
func (a *App) runInNormalMode() {
	a.updateStatus()
	a.appStarted = true
	a.window.Resize(fyne.NewSize(800, 600))
	a.window.CenterOnScreen()
	a.window.ShowAndRun()
}

// requestAssetsPath запрашивает у пользователя путь к файлам zapret
func (a *App) requestAssetsPath() {
	assetsPath := a.configManager.GetAssetsPath()
	if assetsPath != "" {
		a.loadStrategies(assetsPath)
		return
	}

	msg := "Добро пожаловать в GoZapret!\n\n" +
		"Для работы приложения необходимо указать путь к папке с файлами Zapret.\n" +
		"Обычно это папка, содержащая файлы winws.exe, service.bat и другие компоненты.\n\n" +
		"Нажмите OK, чтобы выбрать папку."

	infoDialog := dialog.NewInformation("Путь к файлам zapret не установлен", msg, a.window)
	infoDialog.SetOnClosed(func() {
		a.showFolderDialog()
	})
	infoDialog.Show()
}

// showFolderDialog показывает диалог выбора папки
func (a *App) showFolderDialog() {
	fileDialog := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil || uri == nil {
			return
		}

		newPath := domain.AssetsPath(uri.Path())
		if err := a.configManager.SetAssetsPath(newPath); err != nil {
			dialog.ShowError(fmt.Errorf("ошибка сохранения пути: %v", err), a.window)
			return
		}

		if err := a.configManager.PrepareWorkingDirectory(); err != nil {
			dialog.ShowError(fmt.Errorf("ошибка подготовки рабочей директории: %v", err), a.window)
			return
		}

		a.loadStrategies(newPath)
	}, a.window)

	fileDialog.Resize(fyne.NewSize(800, 600))
	fileDialog.Show()
}

// handleAutostart обрабатывает автозапуск приложения
func (a *App) handleAutostart() {
	lastStrategyName := a.configManager.GetLastStrategyName()
	if lastStrategyName == "" {
		a.logger.Debug("Нет последней стратегии для автозапуска")
		a.window.Hide()
		return
	}

	strategy, err := a.strategyService.GetByName(lastStrategyName)
	if err != nil || strategy == nil {
		a.logger.Warn("Последняя стратегия не найдена при автозапуске", "strategy", lastStrategyName, "error", err)
		a.window.Hide()
		return
	}

	err = a.processManager.StartStrategy(strategy, a.configManager.GetAssetsPath(), a.configManager.GetGameFilter())
	if err != nil {
		a.logger.Error("Ошибка запуска стратегии при автозапуске", "strategy", strategy.Name, "error", err)
	} else {
		a.logger.Debug("Стратегия запущена при автозапуске", "strategy", strategy.Name)
	}

	a.window.Hide()
}

// updateStatus обновляет статус приложения
func (a *App) updateStatus() {
	fyne.Do(func() {
		running := a.processManager.IsRunning()
		a.logger.Debug("Обновление статуса приложения", "running", running)
		a.isRunning.Set(running)

		statusMsg := a.buildStatusMessage(running)
		a.statusText.Set(statusMsg)
		a.logger.Debug("Статус обновлен", "status", statusMsg)
	})
}

// buildStatusMessage формирует сообщение о статусе
func (a *App) buildStatusMessage(running bool) string {
	if running {
		if processInfo := a.processManager.GetCurrentProcess(); processInfo != nil && processInfo.Strategy != "" {
			return fmt.Sprintf("Статус: Запущено (%s)", processInfo.Strategy)
		}
		if lastStrategy := a.configManager.GetLastStrategyName(); lastStrategy != "" {
			return fmt.Sprintf("Статус: Запущено (%s)", lastStrategy)
		}
		return "Статус: Запущено"
	}

	if lastStrategy := a.configManager.GetLastStrategyName(); lastStrategy != "" {
		return fmt.Sprintf("Статус: Остановлено (последняя: %s)", lastStrategy)
	}
	return "Статус: Остановлено"
}

// ReloadStrategies перечитывает стратегии и обновляет файлы в рабочей директории
func (a *App) ReloadStrategies() error {
	assetsPath := a.configManager.GetAssetsPath()
	if assetsPath == "" {
		return fmt.Errorf("путь к ресурсам не установлен")
	}

	currentSelected, _ := a.selectedStrategy.Get()

	if err := a.configManager.PrepareWorkingDirectory(); err != nil {
		return fmt.Errorf("ошибка подготовки рабочей директории: %w", err)
	}

	a.reloadStrategiesWithSelection(assetsPath, currentSelected)
	return nil
}

// loadStrategies загружает стратегии из указанного пути (первая загрузка)
func (a *App) loadStrategies(assetsPath domain.AssetsPath) {
	strategyNames := a.loadStrategiesInternal(assetsPath)
	if strategyNames == nil {
		return
	}

	a.strategies.Set(strategyNames)

	if lastStrategy := a.configManager.GetLastStrategyName(); lastStrategy != "" {
		a.selectedStrategy.Set(lastStrategy.String())
	} else if len(strategyNames) > 0 {
		a.selectedStrategy.Set(strategyNames[0])
	}
}

// reloadStrategiesWithSelection перезагружает стратегии с сохранением текущего выбора
func (a *App) reloadStrategiesWithSelection(assetsPath domain.AssetsPath, currentSelected string) {
	strategyNames := a.loadStrategiesInternal(assetsPath)
	if strategyNames == nil {
		return
	}

	a.logger.Info("Перезагрузка стратегий", "count", len(strategyNames), "currentSelected", currentSelected)

	selectedStrategy := a.findSelectedStrategy(strategyNames, currentSelected)

	fyne.Do(func() {
		a.strategies.Set(strategyNames)
		a.selectedStrategy.Set(selectedStrategy)

		if a.mainView != nil {
			a.mainView.UpdateStrategyOptions(strategyNames, selectedStrategy)
		}
	})
}

// findSelectedStrategy находит выбранную стратегию в списке или возвращает первую
func (a *App) findSelectedStrategy(strategyNames []string, currentSelected string) string {
	if currentSelected != "" {
		for _, name := range strategyNames {
			if name == currentSelected {
				return currentSelected
			}
		}
	}

	if len(strategyNames) > 0 {
		return strategyNames[0]
	}
	return ""
}

// loadStrategiesInternal загружает стратегии и возвращает список имён
func (a *App) loadStrategiesInternal(assetsPath domain.AssetsPath) []string {
	if err := a.strategyService.LoadFromPath(assetsPath); err != nil {
		a.strategies.Set([]string{})
		dialog.ShowError(fmt.Errorf("ошибка загрузки стратегий: %v", err), a.window)
		return nil
	}

	a.updateVersionFromServiceBat(assetsPath)

	strategyList := a.strategyService.GetAll()
	strategyNames := make([]string, len(strategyList))
	for i, s := range strategyList {
		strategyNames[i] = s.Name.String()
	}

	a.logger.Info("Загружены стратегии", "count", len(strategyNames), "names", strategyNames)
	return strategyNames
}

// updateVersionFromServiceBat обновляет версию из service.bat
func (a *App) updateVersionFromServiceBat(assetsPath domain.AssetsPath) {
	newVersion, err := a.strategyService.ReadVersionFromServiceBat(assetsPath)
	if err != nil {
		a.logger.Warn("Не удалось прочитать версию из service.bat", "error", err)
		return
	}

	currentVersion := a.configManager.GetVersion()
	if currentVersion == newVersion {
		return
	}

	if err := a.configManager.SetVersion(newVersion); err != nil {
		a.logger.Warn("Не удалось сохранить версию в конфиг", "version", newVersion, "error", err)
		return
	}

	a.logger.Debug("Версия Zapret обновлена в конфиге", "old_version", currentVersion, "new_version", newVersion)
	a.version.Set(newVersion)
}

// GetFyneApp возвращает Fyne приложение
func (a *App) GetFyneApp() fyne.App {
	return a.fyneApp
}

// GetWindow возвращает главное окно
func (a *App) GetWindow() fyne.Window {
	return a.window
}

// setupTray настраивает сворачивание приложения в системный трей
func (a *App) setupTray(icon []byte) {
	deskApp, ok := a.fyneApp.(desktop.App)
	if !ok {
		return
	}

	menu := fyne.NewMenu("GoZapret",
		fyne.NewMenuItem("Показать", func() {
			a.window.Show()
			a.window.Resize(fyne.NewSize(800, 600))
			a.window.CenterOnScreen()
			a.window.RequestFocus()
		}),
	)

	deskApp.SetSystemTrayMenu(menu)
	a.window.SetCloseIntercept(func() {
		a.window.Hide()
	})

	go a.setTrayIconWithRetry(deskApp, icon)
}

// setTrayIconWithRetry устанавливает иконку в трее с повторными попытками
func (a *App) setTrayIconWithRetry(deskApp desktop.App, icon []byte) {
	time.Sleep(2 * time.Second)

	trayIcon := fyne.NewStaticResource("tray-icon.png", icon)
	const maxAttempts = 5
	baseDelay := time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		a.logger.Debug("Попытка установки иконки в трее", "attempt", attempt, "max", maxAttempts)

		if a.trySetTrayIcon(deskApp, trayIcon) {
			a.logger.Info("Иконка в трее успешно установлена", "attempt", attempt)
			return
		}

		if attempt < maxAttempts {
			delay := time.Duration(attempt) * baseDelay
			a.logger.Debug("Ожидание перед следующей попыткой", "delay", delay)
			time.Sleep(delay)
		}
	}

	a.logger.Error("Не удалось установить иконку в трее после всех попыток", "attempts", maxAttempts)
}

// trySetTrayIcon пытается установить иконку в трее
func (a *App) trySetTrayIcon(deskApp desktop.App, trayIcon *fyne.StaticResource) bool {
	success := make(chan bool, 1)

	fyne.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				a.logger.Warn("Паника при установке иконки в трее", "error", r)
				success <- false
			}
		}()
		deskApp.SetSystemTrayIcon(trayIcon)
		success <- true
	})

	select {
	case ok := <-success:
		return ok
	case <-time.After(3 * time.Second):
		a.logger.Warn("Таймаут при установке иконки в трее")
		return false
	}
}

// loadIconData загружает иконку из embedded ресурсов или файла
func (a *App) loadIconData() ([]byte, error) {
	if iconData, err := a.assets.ReadFile("assets/icon256.png"); err == nil {
		return iconData, nil
	}

	if iconData, err := os.ReadFile("assets/icon256.png"); err == nil {
		return iconData, nil
	}

	return nil, fmt.Errorf("иконка не найдена")
}

// ActivateWindow активирует и показывает главное окно приложения
func (a *App) ActivateWindow() {
	a.logger.Debug("Активация окна приложения")

	fyne.Do(func() {
		if a.window == nil {
			a.logger.Warn("Окно не инициализировано")
			return
		}
		a.window.Show()
		a.window.RequestFocus()
		a.logger.Info("Окно успешно активировано")
	})
}

// showProcessError показывает диалог с ошибкой процесса
func (a *App) showProcessError(strategyName, errorMsg string) {
	a.logger.Error("Ошибка процесса winws", "strategy", strategyName, "error", errorMsg)
	a.updateStatus()

	if a.window != nil {
		a.window.Show()
		a.window.RequestFocus()
	}

	message := fmt.Sprintf("Стратегия '%s' завершилась с ошибкой:\n\n%s", strategyName, errorMsg)
	if len(message) > 1000 {
		message = message[:1000] + "\n\n... (сообщение обрезано)"
	}

	dialog.ShowError(fmt.Errorf("%s", message), a.window)
}

// Shutdown завершает приложение и останавливает все процессы
func (a *App) Shutdown() {
	a.logger.Info("Завершение приложения")
	a.stopWinwsProcess()
	a.fyneApp.Quit()
}

// onAppStopped вызывается при завершении приложения
func (a *App) onAppStopped() {
	if !a.appStarted {
		a.logger.Debug("Игнорируем OnStopped - приложение ещё не запущено")
		return
	}
	a.logger.Debug("Приложение завершается (Lifecycle.OnStopped)")
	a.stopWinwsProcess()
}

// stopWinwsProcess останавливает процесс winws, если он запущен
func (a *App) stopWinwsProcess() {
	if !a.processManager.IsRunning() && !a.processManager.IsWinwsProcessRunning() {
		return
	}

	a.logger.Debug("Остановка процесса winws")
	if err := a.processManager.StopProcess(); err != nil {
		a.logger.Warn("Ошибка остановки процесса", "error", err)
	}
}
