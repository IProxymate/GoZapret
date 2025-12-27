package config

import (
	"encoding/json"
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
	workspace  *WorkspaceManager
	version    string // версия приложения, передаётся при сборке
}

// NewManager создает новый менеджер конфигурации
func NewManager(configPath string, version string) *Manager {
	configDir := filepath.Dir(configPath)
	return &Manager{
		configPath: configPath,
		config:     &domain.Config{},
		workspace:  NewWorkspaceManager(configDir),
		version:    version,
	}
}

// Load загружает конфигурацию из файла
func (m *Manager) Load() error {
	slog.Debug("Загрузка конфигурации", "path", m.configPath)

	data, err := os.ReadFile(m.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Info("Файл конфигурации не найден, используются значения по умолчанию")
			m.config = m.defaultConfig()
			return nil
		}
		slog.Error("Ошибка чтения файла конфигурации", "error", err)
		return err
	}

	if err := json.Unmarshal(data, m.config); err != nil {
		slog.Error("Ошибка парсинга конфигурации", "error", err)
		return err
	}

	// Всегда обновляем версию из билда (после обновления приложения)
	m.config.Version = m.version

	slog.Debug("Конфигурация успешно загружена")
	return m.config.Validate()
}

// defaultConfig возвращает конфигурацию по умолчанию
func (m *Manager) defaultConfig() *domain.Config {
	return &domain.Config{
		LastStrategyName: "",
		AssetsPath:       "",
		LastAssetsPath:   "",
		AutoStart:        domain.DefaultAutoStart,
		GameFilter:       domain.DefaultGameFilter,
		IpsetMode:        domain.DefaultIpsetMode.String(),
		Version:          m.version,
		UpdatedAt:        time.Now(),
		WorkingDir:       "",
	}
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

	// Создаем директорию hosts и файлы списков доменов
	if err := m.workspace.EnsureHostsDirectory(); err != nil {
		slog.Warn("Ошибка создания директории hosts", "error", err)
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
