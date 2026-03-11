package ipset

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/IProxymate/GoZapret/internal/domain"
)

// FileManager управляет файлами ipset
type FileManager struct{}

// NewFileManager создает новый менеджер файлов
func NewFileManager() *FileManager {
	return &FileManager{}
}

// UpdateMode обновляет содержимое файла ipset-all.txt в зависимости от режима
func (f *FileManager) UpdateMode(workingDir string, mode string) error {
	// Проверяем корректность режима
	if mode != "any" && mode != "none" && mode != "loaded" {
		return fmt.Errorf("некорректный режим ipset: %s", mode)
	}

	// Формируем путь к файлу ipset-all.txt в рабочей директории
	ipsetFilePath := filepath.Join(workingDir, "lists", "ipset-all.txt")

	// Создаем директорию, если она не существует
	listsDir := filepath.Dir(ipsetFilePath)
	if err := os.MkdirAll(listsDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории lists: %w", err)
	}

	// В зависимости от режима, записываем соответствующее содержимое
	switch mode {
	case "any":
		return f.writeAnyMode(ipsetFilePath)
	case "none":
		return f.writeNoneMode(ipsetFilePath)
	case "loaded":
		return f.writeLoadedMode(workingDir, ipsetFilePath)
	}

	return nil
}

// writeAnyMode создает пустой файл для режима "any"
func (f *FileManager) writeAnyMode(ipsetFilePath string) error {
	if err := os.WriteFile(ipsetFilePath, []byte(""), 0644); err != nil {
		return fmt.Errorf("ошибка создания пустого файла ipset-all.txt: %w", err)
	}
	slog.Debug("Режим ipset 'any': создан пустой файл", "path", ipsetFilePath)
	return nil
}

// writeNoneMode записывает специальный IP для режима "none"
func (f *FileManager) writeNoneMode(ipsetFilePath string) error {
	content := "203.0.113.113/32\n"
	if err := os.WriteFile(ipsetFilePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("ошибка записи файла ipset-all.txt для режима none: %w", err)
	}
	slog.Debug("Режим ipset 'none': файл содержит только 203.0.113.113/32", "path", ipsetFilePath)
	return nil
}

// writeLoadedMode восстанавливает файл из бэкапа или создает стандартный список
func (f *FileManager) writeLoadedMode(workingDir string, ipsetFilePath string) error {
	ipsetBackupPath := ipsetFilePath + ".backup"

	var baseContent string

	// Проверяем, существует ли бэкап
	if _, err := os.Stat(ipsetBackupPath); err == nil {
		// Если бэкап существует, читаем из него
		backupData, err := os.ReadFile(ipsetBackupPath)
		if err != nil {
			return fmt.Errorf("ошибка чтения бэкапа: %w", err)
		}
		baseContent = string(backupData)
		slog.Debug("Режим ipset 'loaded': загружен бэкап", "path", ipsetBackupPath)
	} else {
		// Если бэкап не существует, используем стандартный список
		baseContent = "# Default ipset list\n"
		slog.Debug("Режим ipset 'loaded': используется содержимое по умолчанию")
	}

	// Загружаем пользовательские подсети
	customSubnets := f.loadCustomSubnets(workingDir)

	// Объединяем базовый контент с пользовательскими подсетями
	finalContent := baseContent
	if len(customSubnets) > 0 {
		if !strings.HasSuffix(finalContent, "\n") {
			finalContent += "\n"
		}
		finalContent += strings.Join(customSubnets, "\n") + "\n"
		slog.Debug("Режим ipset 'loaded': добавлены пользовательские подсети", "count", len(customSubnets))
	}

	if err := os.WriteFile(ipsetFilePath, []byte(finalContent), 0644); err != nil {
		return fmt.Errorf("ошибка записи файла ipset: %w", err)
	}

	return nil
}

// GetCurrentMode определяет текущий режим ipset по содержимому файла
func (f *FileManager) GetCurrentMode(workingDir string) (string, error) {
	ipsetFilePath := filepath.Join(workingDir, "lists", "ipset-all.txt")

	// Проверяем, существует ли файл
	fileInfo, err := os.Stat(ipsetFilePath)
	if os.IsNotExist(err) {
		// Если файл не существует, считаем, что режим "any" (пустой)
		return "any", nil
	} else if err != nil {
		return "", fmt.Errorf("ошибка проверки файла ipset: %w", err)
	}

	// Если файл существует, но пустой, это режим "any"
	if fileInfo.Size() == 0 {
		return "any", nil
	}

	// Читаем содержимое файла
	content, err := os.ReadFile(ipsetFilePath)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения файла ipset: %w", err)
	}

	contentStr := string(content)

	// Если файл содержит только "203.0.113.113/32", это режим "none"
	if contentStr == "203.0.113.113/32\n" || contentStr == "203.0.113.113/32" {
		return "none", nil
	}

	// В остальных случаях считаем, что это режим "loaded"
	return "loaded", nil
}

// loadCustomSubnets загружает пользовательские подсети из файла ipset-custom.txt
func (f *FileManager) loadCustomSubnets(workingDir string) []string {
	// Путь к файлу пользовательских подсетей в директории конфигурации
	configDir, err := os.UserConfigDir()
	if err != nil {
		slog.Warn("Ошибка получения директории конфигурации", "error", err)
		return nil
	}

	customIpsetPath := filepath.Join(configDir, domain.AppName, domain.HostsDirName, domain.IpsetExcludeUserFile)

	// Проверяем существование файла
	if _, err := os.Stat(customIpsetPath); os.IsNotExist(err) {
		return nil
	}

	// Читаем файл
	file, err := os.Open(customIpsetPath)
	if err != nil {
		slog.Warn("Ошибка открытия файла пользовательских подсетей", "error", err)
		return nil
	}
	defer file.Close()

	var subnets []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Пропускаем пустые строки и комментарии
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		subnets = append(subnets, line)
	}

	if err := scanner.Err(); err != nil {
		slog.Warn("Ошибка чтения файла пользовательских подсетей", "error", err)
		return nil
	}

	return subnets
}
