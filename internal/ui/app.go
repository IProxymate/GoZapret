package ui

import (
	"embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"

	"github.com/IProxymate/GoZapret/internal/config"
	"github.com/IProxymate/GoZapret/internal/domain"
	"github.com/IProxymate/GoZapret/internal/services"

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

	// Embedded ресурсы
	assets embed.FS

	// Логгер
	logger *slog.Logger

	// Сервисы
	configManager    *config.Manager
	strategyService  *services.StrategyService
	processManager   *services.ProcessManager
	adminChecker     *services.AdminChecker
	diagnostics      *services.DiagnosticsService
	ipsetService     *services.IpsetService
	cacheService     *services.CacheService
	autostartService *services.AutostartService
	updateService    *services.UpdateService

	// Биндинги для UI
	statusText       binding.String
	isRunning        binding.Bool
	selectedStrategy binding.String
	strategies       binding.StringList
	autoStart        binding.Bool
	gameFilter       binding.Bool
	ipsetMode        binding.String
	version          binding.String

	// Флаг автозапуска
	autostart bool

	// UI компоненты
	mainView *MainView
}

// NewApp создает новое приложение
func NewApp(assets embed.FS, logger *slog.Logger) *App {
	fyneApp := app.NewWithID("com.zapret.gui")

	// Получаем путь к конфигурации
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	configPath := filepath.Join(configDir, "GoZapret", "config.json")

	// Инициализируем сервисы
	configManager := config.NewManager(configPath)
	adminChecker := services.NewAdminChecker()
	strategyService := services.NewStrategyService()
	processManager := services.NewProcessManager(adminChecker, configManager)
	diagnostics := services.NewDiagnosticsService(adminChecker)
	ipsetService := services.NewIpsetService()
	cacheService := services.NewCacheService()
	autostartService := services.NewAutostartService()
	updateService := services.NewUpdateService()

	// Загружаем конфигурацию
	if err := configManager.Load(); err != nil {
		logger.Error("Ошибка загрузки конфигурации", "error", err)
	}

	// Создаем биндинги
	statusText := binding.NewString()
	isRunning := binding.NewBool()
	selectedStrategy := binding.NewString()
	strategies := binding.NewStringList()
	autoStart := binding.NewBool()
	gameFilter := binding.NewBool()
	ipsetMode := binding.NewString()
	version := binding.NewString()

	// Устанавливаем начальные значения из конфигурации
	cfg := configManager.GetConfig()

	// Проверяем, запущен ли процесс winws.exe в системе
	processRunning := processManager.IsWinwsProcessRunning()
	if processRunning {
		// Если процесс запущен, устанавливаем соответствующий статус
		lastStrategy := cfg.LastStrategyName
		if lastStrategy != "" {
			statusText.Set(fmt.Sprintf("Статус: Запущено (%s)", lastStrategy))
		} else {
			statusText.Set("Статус: Запущено")
		}
		isRunning.Set(true)
	} else {
		// Процесс не запущен
		statusText.Set("Статус: Остановлено")
		isRunning.Set(false)
	}

	autoStart.Set(cfg.AutoStart)
	gameFilter.Set(cfg.GameFilter)
	ipsetMode.Set(cfg.IpsetMode)
	version.Set(cfg.Version)

	// Синхронизируем состояние автозапуска между конфигом и системой
	isEnabled, err := autostartService.IsAutoStartEnabled()
	if err != nil {
		logger.Warn("Ошибка проверки состояния автозапуска", "error", err)
		// Если не удалось проверить, используем значение из конфига
		isEnabled = cfg.AutoStart
	}

	// Приоритет отдаем значению из конфига
	// Если автозапуск включен в конфиге, но не установлен в системе, устанавливаем его
	if cfg.AutoStart && !isEnabled {
		if err := autostartService.SetAutoStart(true); err != nil {
			logger.Error("Ошибка установки автозапуска при старте", "error", err)
			// Если не удалось установить, обновляем биндинг в false
			autoStart.Set(false)
		} else {
			// Успешно установили, обновляем биндинг в true
			autoStart.Set(true)
		}
	} else if !cfg.AutoStart && isEnabled {
		// Если автозапуск выключен в конфиге, но установлен в системе, убираем его
		if err := autostartService.SetAutoStart(false); err != nil {
			logger.Error("Ошибка отключения автозапуска при старте", "error", err)
		}
		// Обновляем биндинг согласно конфигу
		autoStart.Set(false)
	} else {
		// Состояния совпадают, просто устанавливаем биндинг из конфига
		autoStart.Set(cfg.AutoStart)
	}

	return &App{
		fyneApp:          fyneApp,
		assets:           assets,
		logger:           logger,
		configManager:    configManager,
		strategyService:  strategyService,
		processManager:   processManager,
		adminChecker:     adminChecker,
		diagnostics:      diagnostics,
		ipsetService:     ipsetService,
		cacheService:     cacheService,
		autostartService: autostartService,
		updateService:    updateService,
		statusText:       statusText,
		isRunning:        isRunning,
		selectedStrategy: selectedStrategy,
		strategies:       strategies,
		autoStart:        autoStart,
		gameFilter:       gameFilter,
		ipsetMode:        ipsetMode,
		version:          version,
		autostart:        false, // по умолчанию false, будет установлено через SetAutostartFlag
	}
}

// Run запускает приложение
func (a *App) Run() {
	// Проверяем аргументы командной строки
	autostart := slices.Contains(os.Args[1:], "/autostart")

	// Загружаем иконку из embedded ресурсов
	iconData, err := a.loadIconData()
	if err == nil {
		iconResource := fyne.NewStaticResource("icon256.png", iconData)
		a.fyneApp.SetIcon(iconResource)
	}

	a.window = a.fyneApp.NewWindow("GoZapret")

	// Настройка сворачивания в трей
	a.setupTray(iconData)

	// Загружаем стратегии, если путь к ресурсам установлен
	a.requestAssetsPath()

	// Создаем главный вид
	a.mainView = NewMainView(a)
	a.window.SetContent(a.mainView.Build())

	// Обновляем статус
	a.updateStatus()

	// Если запуск с флагом /autostart, запускаем стратегию без показа окна
	if autostart {
		a.handleAutostart()
		// Запускаем приложение в фоновом режиме (в трее)
		a.fyneApp.Run()
	} else {
		// Обычный режим - показываем окно
		a.window.Resize(fyne.NewSize(800, 600))
		a.window.CenterOnScreen()
		a.window.ShowAndRun()
	}
}

// requestAssetsPath запрашивает у пользователя путь к файлам zapret
func (a *App) requestAssetsPath() {
	assetsPath := a.configManager.GetAssetsPath()
	if assetsPath != "" {
		a.loadStrategies(assetsPath)
		return
	} else {
		// Создаем информационный диалог с кнопкой OK
		msg := "Добро пожаловать в GoZapret!\n\n" +
			"Для работы приложения необходимо указать путь к папке с файлами Zapret.\n" +
			"Обычно это папка, содержащая файлы winws.exe, service.bat и другие компоненты.\n\n" +
			"Нажмите OK, чтобы выбрать папку."
		infoDialog := dialog.NewInformation("Путь к файлам zapret не установлен", msg, a.window)

		// Переопределяем действие при закрытии диалога, чтобы открыть FileDialog после нажатия OK
		infoDialog.SetOnClosed(func() {
			// После закрытия информационного диалога (нажатия OK) открываем FileDialog для выбора папки
			fileDialog := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
				if err != nil || uri == nil {
					return
				}

				// Устанавливаем выбранный путь в конфиг
				newPath := domain.AssetsPath(uri.Path())
				if err := a.configManager.SetAssetsPath(newPath); err != nil {
					dialog.ShowError(fmt.Errorf("ошибка сохранения пути к файлам zapret: %v", err), a.window)
					return
				}

				// Загружаем стратегии из новой директории
				a.loadStrategies(newPath)
			}, a.window)
			fileDialog.Resize(fyne.NewSize(800, 600))
			fileDialog.Show()
		})

		infoDialog.Show()
	}

}

// handleAutostart обрабатывает автозапуск приложения
func (a *App) handleAutostart() {
	// Запускаем стратегию из конфига
	lastStrategyName := a.configManager.GetLastStrategyName()
	if lastStrategyName != "" {
		strategy, err := a.strategyService.GetByName(lastStrategyName)
		if err == nil && strategy != nil {
			// Запускаем стратегию
			err = a.processManager.StartStrategy(strategy, a.configManager.GetAssetsPath(), a.configManager.GetGameFilter())
			if err != nil {
				a.logger.Error("Ошибка запуска стратегии при автозапуске", "strategy", strategy.Name, "error", err)
			} else {
				a.logger.Debug("Стратегия запущена при автозапуске", "strategy", strategy.Name)
			}
		} else {
			a.logger.Warn("Последняя стратегия не найдена при автозапуске", "strategy", lastStrategyName, "error", err)
		}
	} else {
		a.logger.Debug("Нет последней стратегии для автозапуска")
	}

	// Скрываем окно без его предварительного показа
	a.window.Hide()
}

// updateStatus обновляет статус приложения
func (a *App) updateStatus() {
	fyne.Do(func() {
		running := a.processManager.IsRunning()
		a.logger.Debug("Обновление статуса приложения", "running", running)

		a.isRunning.Set(running)

		if running {
			processInfo := a.processManager.GetCurrentProcess()
			if processInfo != nil && processInfo.Strategy != "" {
				statusMsg := fmt.Sprintf("Статус: Запущено (%s)", processInfo.Strategy)
				a.statusText.Set(statusMsg)
				a.logger.Debug("Статус обновлен", "status", statusMsg)
			} else {
				// Если процесс запущен, но информации о стратегии нет, используем последнюю стратегию из конфига
				lastStrategy := a.configManager.GetLastStrategyName()
				if lastStrategy != "" {
					statusMsg := fmt.Sprintf("Статус: Запущено (%s)", lastStrategy)
					a.statusText.Set(statusMsg)
					a.logger.Debug("Статус обновлен", "status", statusMsg)
				} else {
					a.statusText.Set("Статус: Запущено")
					a.logger.Debug("Статус обновлен: Запущено (без информации о процессе)")
				}
			}
		} else {
			lastStrategy := a.configManager.GetLastStrategyName()
			if lastStrategy != "" {
				statusMsg := fmt.Sprintf("Статус: Остановлено (последняя: %s)", lastStrategy)
				a.statusText.Set(statusMsg)
				a.logger.Debug("Статус обновлен", "status", statusMsg)
			} else {
				a.statusText.Set("Статус: Остановлено")
				a.logger.Debug("Статус обновлен: Остановлено")
			}
		}
	})
}

// loadStrategies загружает стратегии из указанного пути
func (a *App) loadStrategies(assetsPath domain.AssetsPath) {
	if err := a.strategyService.LoadFromPath(assetsPath); err != nil {
		e := fmt.Errorf("ошибка загрузки стратегий: %v", err)
		strategyNames := make([]string, 0)
		a.strategies.Set(strategyNames)
		dialog.ShowError(e, a.window)
		return
	}

	// Читаем версию из service.bat при каждой загрузке стратегий
	if newVersion, err := a.strategyService.ReadVersionFromServiceBat(assetsPath); err == nil {
		currentVersion := a.configManager.GetVersion()
		// Обновляем версию, если она изменилась
		if currentVersion != newVersion {
			if err := a.configManager.SetVersion(newVersion); err != nil {
				a.logger.Warn("Не удалось сохранить версию в конфиг", "version", newVersion, "error", err)
			} else {
				a.logger.Debug("Версия Zapret обновлена в конфиге", "old_version", currentVersion, "new_version", newVersion)
				// Обновляем биндинг версии для UI
				a.version.Set(newVersion)
			}
		}
	} else {
		a.logger.Warn("Не удалось прочитать версию из service.bat", "error", err)
	}

	// Обновляем список стратегий в биндинге
	strategyList := a.strategyService.GetAll()
	strategyNames := make([]string, len(strategyList))
	for i, s := range strategyList {
		strategyNames[i] = s.Name.String()
	}

	a.strategies.Set(strategyNames)

	// Устанавливаем последнюю выбранную стратегию
	lastStrategy := a.configManager.GetLastStrategyName()
	if lastStrategy != "" {
		a.selectedStrategy.Set(lastStrategy.String())
	} else if len(strategyNames) > 0 {
		a.selectedStrategy.Set(strategyNames[0])
	}
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
	// Проверяем, является ли приложение desktop-приложением
	if deskApp, ok := a.fyneApp.(desktop.App); ok {
		// Создаем меню для трея
		menu := fyne.NewMenu("GoZapret",
			fyne.NewMenuItem("Показать", func() {
				a.window.Show()
				a.window.Resize(fyne.NewSize(800, 600))
				a.window.CenterOnScreen()
				a.window.RequestFocus()
			}),
		)

		// Устанавливаем меню для системного трея
		deskApp.SetSystemTrayMenu(menu)

		// Обработчик закрытия окна - сворачиваем в трей вместо закрытия
		a.window.SetCloseIntercept(func() {
			a.window.Hide()
		})

		// Устанавливаем иконку в трее с задержкой, чтобы дать время трю быть готовым
		go func() {
			// Используем Fyne таймер для отложенной установки иконки
			fyne.Do(func() {
				// Небольшая задержка перед установкой иконки
				trayIcon := fyne.NewStaticResource("tray-icon.png", icon)
				deskApp.SetSystemTrayIcon(trayIcon)
				a.logger.Debug("Иконка в трее установлена")
			})
		}()
	}
}

// loadIconData загружает иконку из embedded ресурсов или файла
func (a *App) loadIconData() ([]byte, error) {
	// Попробуем загрузить иконку из embedded ресурсов
	iconData, err := a.assets.ReadFile("assets/icon256.png")
	if err == nil {
		return iconData, nil
	}

	// Резервный вариант - загрузка из файла
	iconData, err = os.ReadFile("assets/icon256.png")
	if err == nil {
		return iconData, nil
	}

	// Если оба варианта неудачны, возвращаем ошибку
	return nil, err
}
