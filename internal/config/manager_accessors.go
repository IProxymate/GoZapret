package config

import (
	"fmt"

	"github.com/IProxymate/GoZapret/internal/domain"
)

// --- Пути к файлам (делегирование в WorkspaceManager) ---

// GetHostsDir возвращает путь к директории hosts
func (m *Manager) GetHostsDir() string {
	return m.workspace.GetHostsDir()
}

// GetExtraListPath возвращает путь к файлу list-extra.txt
func (m *Manager) GetExtraListPath() string {
	return m.workspace.GetExtraListPath()
}

// GetExcludeListPath возвращает путь к файлу list-extra-exclude.txt
func (m *Manager) GetExcludeListPath() string {
	return m.workspace.GetExcludeListPath()
}

// GetCustomIpsetPath возвращает путь к файлу ipset-include-user.txt
func (m *Manager) GetCustomIpsetPath() string {
	return m.workspace.GetCustomIpsetPath()
}

// --- AssetsPath ---

// GetAssetsPath возвращает путь к ресурсам
func (m *Manager) GetAssetsPath() domain.AssetsPath {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.AssetsPath
}

// SetAssetsPath устанавливает путь к ресурсам
func (m *Manager) SetAssetsPath(path domain.AssetsPath) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.config.AssetsPath != path {
		m.config.LastAssetsPath = ""
	}
	m.config.AssetsPath = path
	return m.saveLocked()
}

// --- Рабочая директория ---

// GetWorkingDir возвращает рабочую директорию
func (m *Manager) GetWorkingDir() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.WorkingDir
}

// SetWorkingDir устанавливает рабочую директорию
func (m *Manager) SetWorkingDir(dir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.WorkingDir = dir
	return m.saveLocked()
}

// PrepareWorkingDirectory подготавливает рабочую директорию
func (m *Manager) PrepareWorkingDirectory() error {
	m.mu.Lock()
	if m.config.AssetsPath == "" {
		m.mu.Unlock()
		return fmt.Errorf("путь к ресурсам не задан")
	}

	workingDir := m.workspace.EnsureWorkingDir(m.config.WorkingDir)
	m.config.WorkingDir = workingDir
	assetsPath := m.config.AssetsPath
	m.mu.Unlock()

	// Файловые операции без блокировки
	if err := m.workspace.PrepareWorkingDirectory(assetsPath, workingDir); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.LastAssetsPath = m.config.AssetsPath
	return m.saveLocked()
}

// --- Стратегии ---

// GetLastStrategyName возвращает имя последней стратегии
func (m *Manager) GetLastStrategyName() domain.StrategyName {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.LastStrategyName
}

// SetLastStrategyName устанавливает имя последней стратегии
func (m *Manager) SetLastStrategyName(name domain.StrategyName) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.LastStrategyName = name
	return m.saveLocked()
}

// --- Автозапуск ---

// GetAutoStart возвращает состояние автозапуска
func (m *Manager) GetAutoStart() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.AutoStart
}

// SetAutoStart устанавливает состояние автозапуска
func (m *Manager) SetAutoStart(enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.AutoStart = enabled
	return m.saveLocked()
}

// AutostartSetter интерфейс для установки автозапуска
type AutostartSetter interface {
	SetAutoStart(enabled bool) error
}

// SetAutoStartWithService устанавливает состояние автозапуска через сервис
func (m *Manager) SetAutoStartWithService(enabled bool, autostartService AutostartSetter) error {
	m.mu.Lock()
	m.config.AutoStart = enabled
	if err := m.saveLocked(); err != nil {
		m.mu.Unlock()
		return fmt.Errorf("ошибка сохранения настройки автозапуска: %w", err)
	}
	m.mu.Unlock()

	if err := autostartService.SetAutoStart(enabled); err != nil {
		return fmt.Errorf("ошибка настройки автозапуска через сервис: %w", err)
	}

	return nil
}

// --- Настройки ---

// GetGameFilterMode возвращает режим Game Filter
func (m *Manager) GetGameFilterMode() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.GetGameFilterMode().String()
}

// SetGameFilterMode устанавливает режим Game Filter
func (m *Manager) SetGameFilterMode(mode string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.SetGameFilterMode(domain.GameFilterMode(mode))
	return m.saveLocked()
}

// GetGameFilter возвращает состояние Game Filter (deprecated, для обратной совместимости)
func (m *Manager) GetGameFilter() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.GetGameFilterMode().IsEnabled()
}

// SetGameFilter устанавливает состояние Game Filter (deprecated, для обратной совместимости)
func (m *Manager) SetGameFilter(enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	mode := domain.GameFilterModeFromBool(enabled)
	m.config.SetGameFilterMode(mode)
	return m.saveLocked()
}

// GetIpsetMode возвращает режим ipset
func (m *Manager) GetIpsetMode() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.IpsetMode
}

// SetIpsetMode устанавливает режим ipset
func (m *Manager) SetIpsetMode(mode string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.IpsetMode = mode
	return m.saveLocked()
}

// --- Версия ---

// GetVersion возвращает версию Zapret
func (m *Manager) GetVersion() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Version
}

// SetVersion устанавливает версию Zapret
func (m *Manager) SetVersion(version string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.Version = version
	return m.saveLocked()
}

// --- LastAssetsPath ---

// GetLastAssetsPath возвращает последний путь к ресурсам
func (m *Manager) GetLastAssetsPath() domain.AssetsPath {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.LastAssetsPath
}

// SetLastAssetsPath устанавливает последний путь к ресурсам
func (m *Manager) SetLastAssetsPath(path domain.AssetsPath) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.LastAssetsPath = path
	return m.saveLocked()
}
