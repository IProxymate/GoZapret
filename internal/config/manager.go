package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
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
				LastAssetsPath:   "",
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

	// Создаем директорию hosts и файлы списков доменов
	if err := m.ensureHostsDirectory(); err != nil {
		slog.Warn("Ошибка создания директории hosts", "error", err)
		// Не прерываем сохранение конфига из-за этой ошибки
	}

	if err := os.WriteFile(m.configPath, data, 0644); err != nil {
		slog.Error("Ошибка записи файла конфигурации", "error", err)
		return err
	}

	slog.Debug("Конфигурация успешно сохранена", "path", m.configPath)
	return nil
}

// ensureHostsDirectory создает директорию hosts и файлы списков доменов, если они не существуют
func (m *Manager) ensureHostsDirectory() error {
	configDir := filepath.Dir(m.configPath)
	hostsDir := filepath.Join(configDir, "hosts")

	// Создаем директорию hosts
	if err := os.MkdirAll(hostsDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории hosts: %v", err)
	}

	// Создаем файл list-extra.txt, если он не существует
	extraListPath := filepath.Join(hostsDir, "list-extra.txt")
	if _, err := os.Stat(extraListPath); os.IsNotExist(err) {
		if err := os.WriteFile(extraListPath, []byte(""), 0644); err != nil {
			return fmt.Errorf("ошибка создания файла list-extra.txt: %v", err)
		}
		slog.Debug("Создан файл list-extra.txt", "path", extraListPath)
	}

	// Создаем файл list-extra-exclude.txt, если он не существует
	excludeListPath := filepath.Join(hostsDir, "list-extra-exclude.txt")
	if _, err := os.Stat(excludeListPath); os.IsNotExist(err) {
		if err := os.WriteFile(excludeListPath, []byte(""), 0644); err != nil {
			return fmt.Errorf("ошибка создания файла list-extra-exclude.txt: %v", err)
		}
		slog.Debug("Создан файл list-extra-exclude.txt", "path", excludeListPath)
	}

	// Создаем файл ipset-custom.txt, если он не существует
	customIpsetPath := filepath.Join(hostsDir, "ipset-custom.txt")
	if _, err := os.Stat(customIpsetPath); os.IsNotExist(err) {
		if err := os.WriteFile(customIpsetPath, []byte(""), 0644); err != nil {
			return fmt.Errorf("ошибка создания файла ipset-custom.txt: %v", err)
		}
		slog.Debug("Создан файл ipset-custom.txt", "path", customIpsetPath)
	}

	return nil
}

// GetHostsDir возвращает путь к директории hosts
func (m *Manager) GetHostsDir() string {
	configDir := filepath.Dir(m.configPath)
	return filepath.Join(configDir, "hosts")
}

// GetExtraListPath возвращает путь к файлу list-extra.txt
func (m *Manager) GetExtraListPath() string {
	return filepath.Join(m.GetHostsDir(), "list-extra.txt")
}

// GetExcludeListPath возвращает путь к файлу list-extra-exclude.txt
func (m *Manager) GetExcludeListPath() string {
	return filepath.Join(m.GetHostsDir(), "list-extra-exclude.txt")
}

// GetCustomIpsetPath возвращает путь к файлу пользовательских подсетей ipset-custom.txt
func (m *Manager) GetCustomIpsetPath() string {
	return filepath.Join(m.GetHostsDir(), "ipset-custom.txt")
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
	// Если путь изменился, сбрасываем LastAssetsPath чтобы файлы скопировались заново
	if m.config.AssetsPath != path {
		slog.Debug("Путь к ресурсам изменен, сбрасываем LastAssetsPath", "old", m.config.AssetsPath, "new", path)
		m.config.LastAssetsPath = ""
	}
	m.config.AssetsPath = path
	return m.Save()
}

// PrepareWorkingDirectory подготавливает рабочую директорию с файлами из ресурсов
// Этот метод должен вызываться после изменения AssetsPath
func (m *Manager) PrepareWorkingDirectory() error {
	slog.Info("Подготовка рабочей директории", "assetsPath", m.config.AssetsPath)

	if m.config.AssetsPath == "" {
		return fmt.Errorf("путь к ресурсам не задан")
	}

	if err := m.config.AssetsPath.Validate(); err != nil {
		return fmt.Errorf("невалидный путь к ресурсам: %w", err)
	}

	// Определяем рабочую директорию
	workingDir := m.ensureWorkingDir()

	// Создаем поддиректории
	binDir := filepath.Join(workingDir, "bin")
	listsDir := filepath.Join(workingDir, "lists")

	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории bin: %w", err)
	}

	if err := os.MkdirAll(listsDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории lists: %w", err)
	}

	// Копируем файлы
	slog.Debug("Копирование файлов из ресурсов", "from", m.config.AssetsPath, "to", workingDir)
	if err := m.copyRequiredFiles(m.config.AssetsPath, binDir, listsDir); err != nil {
		return fmt.Errorf("ошибка копирования файлов: %w", err)
	}

	// Сохраняем путь
	m.config.LastAssetsPath = m.config.AssetsPath

	// Читаем и сохраняем версию
	if version, err := m.readVersionFromServiceBat(m.config.AssetsPath); err == nil {
		m.config.Version = version
	}

	// Сохраняем конфиг
	if err := m.Save(); err != nil {
		slog.Warn("Не удалось сохранить конфигурацию после подготовки директории", "error", err)
	}

	slog.Info("Рабочая директория успешно подготовлена", "dir", workingDir)
	return nil
}

// ensureWorkingDir определяет и создает рабочую директорию
func (m *Manager) ensureWorkingDir() string {
	workingDir := m.config.WorkingDir

	if workingDir == "" {
		configDir, err := os.UserConfigDir()
		if err != nil {
			configDir = "."
		}
		workingDir = filepath.Join(configDir, "GoZapret", "working")
		m.config.WorkingDir = workingDir
	}

	os.MkdirAll(workingDir, 0755)
	return workingDir
}

// copyRequiredFiles копирует все файлы из директорий bin и lists в рабочую директорию
func (m *Manager) copyRequiredFiles(assetsPath domain.AssetsPath, binDir, listsDir string) error {
	// Копируем все файлы из bin
	srcBinDir := filepath.Join(assetsPath.String(), "bin")
	if err := m.copyAllFilesFromDir(srcBinDir, binDir); err != nil {
		slog.Warn("Ошибка копирования файлов из bin", "error", err)
	}

	// Копируем все файлы из lists
	srcListsDir := filepath.Join(assetsPath.String(), "lists")
	if err := m.copyAllFilesFromDir(srcListsDir, listsDir); err != nil {
		slog.Warn("Ошибка копирования файлов из lists", "error", err)
	}

	return nil
}

// copyAllFilesFromDir копирует все файлы из исходной директории в целевую (с заменой)
func (m *Manager) copyAllFilesFromDir(srcDir, dstDir string) error {
	// Проверяем существование исходной директории
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		slog.Debug("Исходная директория не существует", "dir", srcDir)
		return nil
	}

	// Читаем содержимое исходной директории
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("ошибка чтения директории %s: %w", srcDir, err)
	}

	copiedCount := 0
	for _, entry := range entries {
		// Пропускаем директории, копируем только файлы
		if entry.IsDir() {
			continue
		}

		srcPath := filepath.Join(srcDir, entry.Name())
		dstPath := filepath.Join(dstDir, entry.Name())

		// Читаем исходный файл
		data, err := os.ReadFile(srcPath)
		if err != nil {
			slog.Warn("Ошибка чтения файла", "file", srcPath, "error", err)
			continue
		}

		// Записываем в целевую директорию (с заменой существующего)
		if err := os.WriteFile(dstPath, data, 0755); err != nil {
			slog.Warn("Ошибка записи файла", "file", dstPath, "error", err)
			continue
		}

		copiedCount++
	}

	slog.Debug("Файлы скопированы", "from", srcDir, "to", dstDir, "count", copiedCount)
	return nil
}

// readVersionFromServiceBat читает версию из файла service.bat
func (m *Manager) readVersionFromServiceBat(assetsPath domain.AssetsPath) (string, error) {
	serviceBatPath := filepath.Join(assetsPath.String(), "service.bat")

	file, err := os.Open(serviceBatPath)
	if err != nil {
		slog.Warn("Не удалось открыть service.bat для чтения версии", "path", serviceBatPath, "error", err)
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	versionRegex := regexp.MustCompile(`set\s+"LOCAL_VERSION=([^"]+)"`)

	for scanner.Scan() {
		line := scanner.Text()
		matches := versionRegex.FindStringSubmatch(line)
		if len(matches) > 1 {
			version := matches[1]
			slog.Info("Версия Zapret найдена в service.bat", "version", version)
			return version, nil
		}
	}

	if err := scanner.Err(); err != nil {
		slog.Error("Ошибка чтения service.bat", "error", err)
		return "", err
	}

	slog.Warn("Версия не найдена в service.bat")
	return "", fmt.Errorf("версия не найдена в service.bat")
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

// GetLastAssetsPath возвращает последний путь к ресурсам, для которого копировались файлы
func (m *Manager) GetLastAssetsPath() domain.AssetsPath {
	return m.config.LastAssetsPath
}

// SetLastAssetsPath устанавливает последний путь к ресурсам
func (m *Manager) SetLastAssetsPath(path domain.AssetsPath) error {
	m.config.LastAssetsPath = path
	return m.Save()
}
