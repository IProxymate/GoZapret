package ipset

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
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
	log.Printf("Режим ipset 'any': создан пустой файл %s", ipsetFilePath)
	return nil
}

// writeNoneMode записывает специальный IP для режима "none"
func (f *FileManager) writeNoneMode(ipsetFilePath string) error {
	content := "203.0.113.113/32\n"
	if err := os.WriteFile(ipsetFilePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("ошибка записи файла ipset-all.txt для режима none: %w", err)
	}
	log.Printf("Режим ipset 'none': файл %s содержит только 203.0.113/32", ipsetFilePath)
	return nil
}

// writeLoadedMode восстанавливает файл из бэкапа или создает стандартный список
func (f *FileManager) writeLoadedMode(workingDir string, ipsetFilePath string) error {
	ipsetBackupPath := ipsetFilePath + ".backup"

	// Проверяем, существует ли бэкап
	if _, err := os.Stat(ipsetBackupPath); err == nil {
		// Если бэкап существует, восстанавливаем из него
		backupContent, err := os.ReadFile(ipsetBackupPath)
		if err != nil {
			return fmt.Errorf("ошибка чтения бэкапа: %w", err)
		}

		if err := os.WriteFile(ipsetFilePath, backupContent, 0644); err != nil {
			return fmt.Errorf("ошибка записи восстановленного файла: %w", err)
		}
		log.Printf("Режим ipset 'loaded': восстановлен из бэкапа %s", ipsetBackupPath)
	} else {
		// Если бэкап не существует, используем стандартный список
		defaultContent := "# Default ipset list\n"
		if err := os.WriteFile(ipsetFilePath, []byte(defaultContent), 0644); err != nil {
			return fmt.Errorf("ошибка записи стандартного списка ipset: %w", err)
		}
		log.Printf("Режим ipset 'loaded': создан файл с содержимым по умолчанию")
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
