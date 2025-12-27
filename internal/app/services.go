package app

import (
	"log/slog"
	"os"
	"path/filepath"

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
)

// Services содержит все сервисы приложения
// Это контейнер зависимостей для упрощения управления сервисами
type Services struct {
	// Шина событий
	EventBus *EventBus

	// Сервисы
	Config      ConfigService
	Strategy    StrategyService
	Process     *process.Manager
	Admin       AdminChecker
	Diagnostics DiagnosticsRunner
	Ipset       IpsetManager
	Cache       CacheCleaner
	Autostart   AutostartManager
	Update      *updates.Service
	SelfUpdate  *updates.SelfUpdater

	// Контроллеры
	StrategyController *StrategyController
}

// NewServices создает и инициализирует все сервисы приложения
func NewServices(logger *slog.Logger, version string) *Services {
	// Создаём шину событий
	eventBus := NewEventBus()

	// Инициализируем менеджер конфигурации
	configManager := initConfigManager(logger, version)

	// Создаем базовые сервисы
	adminChecker := services.NewAdminChecker()
	strategyService := strategy.NewService()

	// ProcessManager с зависимостями
	processManager := process.NewManager(
		process.NewProcessExecutor(),
		process.NewWindowsProcessMonitor(),
		process.NewBatParser(),
		process.NewArgsBuilder(configManager),
		adminChecker,
		configManager,
	)

	// DiagnosticsService с проверками
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

	// Создаём контроллер стратегий с EventBus
	strategyController := NewStrategyController(
		logger,
		configManager,
		strategyService,
		processManager,
		eventBus,
	)

	return &Services{
		EventBus:    eventBus,
		Config:      configManager,
		Strategy:    strategyService,
		Process:     processManager,
		Admin:       adminChecker,
		Diagnostics: diagnosticsService,
		Ipset:       ipset.NewService(),
		Cache:       services.NewCacheService(),
		Autostart:   autostart.NewService(),
		Update:      updates.NewService(config.GetExternalConfig().ZapretResourcesAPIURL()),
		SelfUpdate:  updates.NewSelfUpdater(version),

		StrategyController: strategyController,
	}
}

// initConfigManager инициализирует менеджер конфигурации
func initConfigManager(logger *slog.Logger, version string) *config.Manager {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	configPath := filepath.Join(configDir, domain.AppName, domain.ConfigFileName)

	configManager := config.NewManager(configPath, version)
	if err := configManager.Load(); err != nil {
		logger.Error("Ошибка загрузки конфигурации", "error", err)
	}

	return configManager
}

// SyncAutostart синхронизирует состояние автозапуска между конфигом и системой
func (s *Services) SyncAutostart(autoStartBinding interface{ Set(bool) error }, logger *slog.Logger) {
	cfg := s.Config.GetConfig()

	isEnabled, err := s.Autostart.IsAutoStartEnabled()
	if err != nil {
		logger.Warn("Ошибка проверки состояния автозапуска", "error", err)
		isEnabled = cfg.AutoStart
	}

	switch {
	case cfg.AutoStart && !isEnabled:
		if err := s.Autostart.SetAutoStart(true); err != nil {
			logger.Error("Ошибка установки автозапуска при старте", "error", err)
			autoStartBinding.Set(false)
		} else {
			autoStartBinding.Set(true)
		}
	case !cfg.AutoStart && isEnabled:
		if err := s.Autostart.SetAutoStart(false); err != nil {
			logger.Error("Ошибка отключения автозапуска при старте", "error", err)
		}
		autoStartBinding.Set(false)
	default:
		autoStartBinding.Set(cfg.AutoStart)
	}
}

