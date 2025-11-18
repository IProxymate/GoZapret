package services

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/IProxymate/GoZapret/internal/domain"
)

// IpsetService отвечает за управление файлом ipset-all.txt
type IpsetService struct{}

// NewIpsetService создает новый сервис управления ipset
func NewIpsetService() *IpsetService {
	return &IpsetService{}
}

// UpdateIpsetFile обновляет содержимое файла ipset-all.txt в зависимости от режима
func (s *IpsetService) UpdateIpsetFile(assetsPath domain.AssetsPath, mode string) error {
	// Проверяем корректность режима
	if mode != "any" && mode != "none" && mode != "loaded" {
		return fmt.Errorf("некорректный режим ipset: %s", mode)
	}

	// Формируем путь к файлу ipset-all.txt
	ipsetFilePath := filepath.Join(assetsPath.String(), "lists", "ipset-all.txt")
	ipsetBackupPath := ipsetFilePath + ".backup"

	// Создаем директорию, если она не существует
	listsDir := filepath.Dir(ipsetFilePath)
	if err := os.MkdirAll(listsDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории lists: %w", err)
	}

	// В зависимости от режима, записываем соответствующее содержимое
	switch mode {
	case "any":
		// Режим "any" - создаем пустой файл
		if err := os.WriteFile(ipsetFilePath, []byte(""), 0644); err != nil {
			return fmt.Errorf("ошибка создания пустого файла ipset-all.txt: %w", err)
		}
		log.Printf("Режим ipset 'any': создан пустой файл %s", ipsetFilePath)

	case "none":
		// Режим "none" - записываем только "203.0.113/32"
		content := "203.0.113.113/32\n"
		if err := os.WriteFile(ipsetFilePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("ошибка записи файла ipset-all.txt для режима none: %w", err)
		}
		log.Printf("Режим ipset 'none': файл %s содержит только 203.0.113/32", ipsetFilePath)

	case "loaded":
		// Режим "loaded" - восстанавливаем из бэкапа или используем стандартный список
		// Проверяем, существует ли бэкап
		if _, err := os.Stat(ipsetBackupPath); err == nil {
			// Если бэкап существует, восстанавливаем из него
			if err := s.restoreFromBackup(ipsetBackupPath, ipsetFilePath); err != nil {
				return fmt.Errorf("ошибка восстановления из бэкапа: %w", err)
			}
			log.Printf("Режим ipset 'loaded': восстановлен из бэкапа %s", ipsetBackupPath)
		} else {
			// Если бэкап не существует, используем какой-то стандартный список
			// В реальном приложении это может быть загрузка из внешнего источника
			// или восстановление из предустановленного списка
			defaultContent := "# Default ipset list\n"
			if err := os.WriteFile(ipsetFilePath, []byte(defaultContent), 0644); err != nil {
				return fmt.Errorf("ошибка записи стандартного списка ipset: %w", err)
			}
			log.Printf("Режим ipset 'loaded': создан файл с содержимым по умолчанию")
		}
	}

	return nil
}

// restoreFromBackup восстанавливает файл из бэкапа
func (s *IpsetService) restoreFromBackup(backupPath, targetPath string) error {
	// Читаем содержимое бэкапа
	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("ошибка чтения бэкапа: %w", err)
	}

	// Записываем содержимое в целевой файл
	if err := os.WriteFile(targetPath, backupContent, 0644); err != nil {
		return fmt.Errorf("ошибка записи восстановленного файла: %w", err)
	}

	return nil
}

// BackupIpsetFile создает бэкап файла ipset-all.txt
func (s *IpsetService) BackupIpsetFile(assetsPath domain.AssetsPath) error {
	ipsetFilePath := filepath.Join(assetsPath.String(), "lists", "ipset-all.txt")
	ipsetBackupPath := ipsetFilePath + ".backup"

	// Проверяем, существует ли файл
	if _, err := os.Stat(ipsetFilePath); os.IsNotExist(err) {
		// Если файл не существует, создаем пустой файл и затем бэкапим его
		if err := os.WriteFile(ipsetFilePath, []byte(""), 0644); err != nil {
			return fmt.Errorf("ошибка создания файла ipset-all.txt: %w", err)
		}
	}

	// Читаем содержимое исходного файла
	content, err := os.ReadFile(ipsetFilePath)
	if err != nil {
		return fmt.Errorf("ошибка чтения файла для бэкапа: %w", err)
	}

	// Создаем бэкап
	if err := os.WriteFile(ipsetBackupPath, content, 0644); err != nil {
		return fmt.Errorf("ошибка создания бэкапа: %w", err)
	}

	log.Printf("Создан бэкап файла ipset-all.txt: %s", ipsetBackupPath)
	return nil
}

// GetCurrentIpsetMode определяет текущий режим ipset по содержимому файла
func (s *IpsetService) GetCurrentIpsetMode(assetsPath domain.AssetsPath) (string, error) {
	ipsetFilePath := filepath.Join(assetsPath.String(), "lists", "ipset-all.txt")

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

	// Если файл содержит только "203.0.113.13/32", это режим "none"
	if contentStr == "203.0.113.13/32\n" || contentStr == "203.0.113.113/32" {
		return "none", nil
	}

	// В остальных случаях считаем, что это режим "loaded"
	return "loaded", nil
}
