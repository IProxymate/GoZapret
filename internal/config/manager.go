package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/IProxymate/GoZapret/internal/domain"
)

// Manager управляет конфигурацией приложения
type Manager struct {
	configPath string
	config     *domain.Config
}

// NewManager создает новый менеджер конфигурации
func NewManager(configPath string) *Manager {
	return &Manager{
		configPath: configPath,
		config:     &domain.Config{},
	}
}

// Load загружает конфигурацию из файла
func (m *Manager) Load() error {
	slog.Debug("Загрузка конфигурации", "path", m.configPath)

	data, err := os.ReadFile(m.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Info("Файл конфигурации не найден, используются значения по умолчанию")
			// Устанавливаем значения по умолчанию
			m.config = &domain.Config{
				LastStrategyName: "",
				AssetsPath:       "",
				AutoStart:        true,
				GameFilter:       false,
				IpsetMode:        "none",
				Version:          "1.0.0",
				UpdatedAt:        time.Now(),
				WorkingDir:       "",
			}
			return nil
		}
		slog.Error("Ошибка чтения файла конфигурации", "error", err)
		return err
	}

	if err := json.Unmarshal(data, m.config); err != nil {
		slog.Error("Ошибка парсинга конфигурации", "error", err)
		return err
	}

	slog.Debug("Конфигурация успешно загружена")
	return m.config.Validate()
}

// Save сохраняет конфигурацию в файл
func (m *Manager) Save() error {
	m.config.UpdatedAt = time.Now()

	if err := m.config.Validate(); err != nil {
		slog.Error("Ошибка валидации конфигурации", "error", err)
		return err
	}

	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		slog.Error("Ошибка сериализации конфигурации", "error", err)
		return err
	}

	// Создаем директорию, если она не существует
	configDir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		slog.Error("Ошибка создания директории конфигурации", "error", err, "dir", configDir)
		return err
	}

	if err := os.WriteFile(m.configPath, data, 0644); err != nil {
		slog.Error("Ошибка записи файла конфигурации", "error", err)
		return err
	}

	slog.Debug("Конфигурация успешно сохранена", "path", m.configPath)
	return nil
}

// GetConfig возвращает текущую конфигурацию
func (m *Manager) GetConfig() *domain.Config {
	return m.config
}

// UpdateConfig обновляет конфигурацию
func (m *Manager) UpdateConfig(updateFn func(*domain.Config)) error {
	updateFn(m.config)
	return m.Save()
}

// GetAssetsPath возвращает путь к ресурсам
func (m *Manager) GetAssetsPath() domain.AssetsPath {
	return m.config.AssetsPath
}

// SetAssetsPath устанавливает путь к ресурсам
func (m *Manager) SetAssetsPath(path domain.AssetsPath) error {
	m.config.AssetsPath = path
	return m.Save()
}

// GetLastStrategyName возвращает имя последней стратегии
func (m *Manager) GetLastStrategyName() domain.StrategyName {
	return m.config.LastStrategyName
}

// SetLastStrategyName устанавливает имя последней стратегии
func (m *Manager) SetLastStrategyName(name domain.StrategyName) error {
	m.config.LastStrategyName = name
	return m.Save()
}

// GetAutoStart возвращает состояние автозапуска
func (m *Manager) GetAutoStart() bool {
	return m.config.AutoStart
}

// SetAutoStart устанавливает состояние автозапуска
func (m *Manager) SetAutoStart(enabled bool) error {
	m.config.AutoStart = enabled
	return m.Save()
}

// SetAutoStartWithService устанавливает состояние автозапуска и применяет его через сервис
func (m *Manager) SetAutoStartWithService(enabled bool, autostartService interface{}) error {
	// Сначала сохраняем в конфиг
	if err := m.SetAutoStart(enabled); err != nil {
		return fmt.Errorf("ошибка сохранения настройки автозапуска: %v", err)
	}

	// Затем применяем через сервис автозапуска
	if service, ok := autostartService.(interface{ SetAutoStart(bool) error }); ok {
		if err := service.SetAutoStart(enabled); err != nil {
			return fmt.Errorf("ошибка настройки автозапуска через сервис: %v", err)
		}
	}

	return nil
}

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

// GetWorkingDir возвращает рабочую директорию
func (m *Manager) GetWorkingDir() string {
	return m.config.WorkingDir
}

// SetWorkingDir устанавливает рабочую директорию
func (m *Manager) SetWorkingDir(dir string) error {
	m.config.WorkingDir = dir
	return m.Save()
}

// GetVersion возвращает версию Zapret
func (m *Manager) GetVersion() string {
	return m.config.Version
}

// SetVersion устанавливает версию Zapret
func (m *Manager) SetVersion(version string) error {
	m.config.Version = version
	return m.Save()
}
