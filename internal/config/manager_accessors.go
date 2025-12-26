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

// GetCustomIpsetPath возвращает путь к файлу ipset-custom.txt
func (m *Manager) GetCustomIpsetPath() string {
	return m.workspace.GetCustomIpsetPath()
}

// --- AssetsPath ---

// GetAssetsPath возвращает путь к ресурсам
func (m *Manager) GetAssetsPath() domain.AssetsPath {
	return m.config.AssetsPath
}

// SetAssetsPath устанавливает путь к ресурсам
func (m *Manager) SetAssetsPath(path domain.AssetsPath) error {
	if m.config.AssetsPath != path {
		m.config.LastAssetsPath = ""
	}
	m.config.AssetsPath = path
	return m.Save()
}

// --- Рабочая директория ---

// GetWorkingDir возвращает рабочую директорию
func (m *Manager) GetWorkingDir() string {
	return m.config.WorkingDir
}

// SetWorkingDir устанавливает рабочую директорию
func (m *Manager) SetWorkingDir(dir string) error {
	m.config.WorkingDir = dir
	return m.Save()
}

// PrepareWorkingDirectory подготавливает рабочую директорию
func (m *Manager) PrepareWorkingDirectory() error {
	if m.config.AssetsPath == "" {
		return fmt.Errorf("путь к ресурсам не задан")
	}

	workingDir := m.workspace.EnsureWorkingDir(m.config.WorkingDir)
	m.config.WorkingDir = workingDir

	if err := m.workspace.PrepareWorkingDirectory(m.config.AssetsPath, workingDir); err != nil {
		return err
	}

	m.config.LastAssetsPath = m.config.AssetsPath
	return m.Save()
}

// --- Стратегии ---

// GetLastStrategyName возвращает имя последней стратегии
func (m *Manager) GetLastStrategyName() domain.StrategyName {
	return m.config.LastStrategyName
}

// SetLastStrategyName устанавливает имя последней стратегии
func (m *Manager) SetLastStrategyName(name domain.StrategyName) error {
	m.config.LastStrategyName = name
	return m.Save()
}

// --- Автозапуск ---

// GetAutoStart возвращает состояние автозапуска
func (m *Manager) GetAutoStart() bool {
	return m.config.AutoStart
}

// SetAutoStart устанавливает состояние автозапуска
func (m *Manager) SetAutoStart(enabled bool) error {
	m.config.AutoStart = enabled
	return m.Save()
}

// SetAutoStartWithService устанавливает состояние автозапуска через сервис
func (m *Manager) SetAutoStartWithService(enabled bool, autostartService interface{}) error {
	if err := m.SetAutoStart(enabled); err != nil {
		return fmt.Errorf("ошибка сохранения настройки автозапуска: %w", err)
	}

	if service, ok := autostartService.(interface{ SetAutoStart(bool) error }); ok {
		if err := service.SetAutoStart(enabled); err != nil {
			return fmt.Errorf("ошибка настройки автозапуска через сервис: %w", err)
		}
	}

	return nil
}

// --- Настройки ---

// GetGameFilter возвращает состояние Game Filter
func (m *Manager) GetGameFilter() bool {
	return m.config.GameFilter
}

// SetGameFilter устанавливает состояние Game Filter
func (m *Manager) SetGameFilter(enabled bool) error {
	m.config.GameFilter = enabled
	return m.Save()
}

// GetIpsetMode возвращает режим ipset
func (m *Manager) GetIpsetMode() string {
	return m.config.IpsetMode
}

// SetIpsetMode устанавливает режим ipset
func (m *Manager) SetIpsetMode(mode string) error {
	m.config.IpsetMode = mode
	return m.Save()
}

// --- Версия ---

// GetVersion возвращает версию Zapret
func (m *Manager) GetVersion() string {
	return m.config.Version
}

// SetVersion устанавливает версию Zapret
func (m *Manager) SetVersion(version string) error {
	m.config.Version = version
	return m.Save()
}

// --- LastAssetsPath ---

// GetLastAssetsPath возвращает последний путь к ресурсам
func (m *Manager) GetLastAssetsPath() domain.AssetsPath {
	return m.config.LastAssetsPath
}

// SetLastAssetsPath устанавливает последний путь к ресурсам
func (m *Manager) SetLastAssetsPath(path domain.AssetsPath) error {
	m.config.LastAssetsPath = path
	return m.Save()
}

