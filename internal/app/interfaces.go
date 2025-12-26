package app

import "github.com/IProxymate/GoZapret/internal/domain"

// ConfigService определяет интерфейс для управления конфигурацией
type ConfigService interface {
	// Load загружает конфигурацию из файла
	Load() error
	// Save сохраняет конфигурацию в файл
	Save() error
	// GetConfig возвращает текущую конфигурацию
	GetConfig() *domain.Config

	// Пути
	GetAssetsPath() domain.AssetsPath
	SetAssetsPath(path domain.AssetsPath) error
	GetWorkingDir() string
	SetWorkingDir(dir string) error
	GetHostsDir() string
	GetExtraListPath() string
	GetExcludeListPath() string
	GetCustomIpsetPath() string

	// Стратегии
	GetLastStrategyName() domain.StrategyName
	SetLastStrategyName(name domain.StrategyName) error

	// Настройки
	GetAutoStart() bool
	SetAutoStart(enabled bool) error
	SetAutoStartWithService(enabled bool, autostartService interface{}) error
	GetGameFilter() bool
	SetGameFilter(enabled bool) error
	GetIpsetMode() string
	SetIpsetMode(mode string) error
	GetVersion() string
	SetVersion(version string) error

	// Рабочая директория
	PrepareWorkingDirectory() error
}

// StrategyService определяет интерфейс для управления стратегиями
type StrategyService interface {
	// LoadFromPath загружает стратегии из указанного пути
	LoadFromPath(assetsPath domain.AssetsPath) error
	// GetByName возвращает стратегию по имени
	GetByName(name domain.StrategyName) (*domain.Strategy, error)
	// GetAll возвращает все загруженные стратегии
	GetAll() []domain.Strategy
	// HasStrategies проверяет, есть ли загруженные стратегии
	HasStrategies() bool
	// ReadVersionFromServiceBat читает версию из файла service.bat
	ReadVersionFromServiceBat(assetsPath domain.AssetsPath) (string, error)
}

// AdminChecker определяет интерфейс для проверки прав администратора
type AdminChecker interface {
	// IsAdmin проверяет, запущено ли приложение с правами администратора
	IsAdmin() bool
}

// DiagnosticsRunner определяет интерфейс для диагностики системы
type DiagnosticsRunner interface {
	// RunAll выполняет все диагностические проверки
	RunAll() []*domain.DiagnosticResult
}

// IpsetManager определяет интерфейс для управления IPset
type IpsetManager interface {
	// UpdateIpsetFile обновляет содержимое файла ipset-all.txt в зависимости от режима
	UpdateIpsetFile(workingDir string, mode string) error
	// GetCurrentIpsetMode определяет текущий режим ipset по содержимому файла
	GetCurrentIpsetMode(workingDir string) (string, error)
	// BackupIpsetFile создает бэкап файла ipset-all.txt
	BackupIpsetFile(assetsPath domain.AssetsPath) error
}

// CacheCleaner определяет интерфейс для управления кэшем
type CacheCleaner interface {
	// ClearDiscordCache очищает кэш Discord
	ClearDiscordCache() error
	// KillDiscordProcesses убивает все процессы Discord
	KillDiscordProcesses() error
}

// AutostartManager определяет интерфейс для управления автозапуском
type AutostartManager interface {
	// IsAutoStartEnabled проверяет, включен ли автозапуск приложения
	IsAutoStartEnabled() (bool, error)
	// SetAutoStart устанавливает или убирает автозапуск приложения
	SetAutoStart(enabled bool) error
}

