package ipset

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/IProxymate/GoZapret/internal/domain"
)

// BackupManager управляет бэкапами файлов ipset
type BackupManager struct{}

// NewBackupManager создает новый менеджер бэкапов
func NewBackupManager() *BackupManager {
	return &BackupManager{}
}

// Create создает бэкап файла ipset-all.txt
func (b *BackupManager) Create(assetsPath domain.AssetsPath) error {
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

// Restore восстанавливает файл из бэкапа
func (b *BackupManager) Restore(assetsPath domain.AssetsPath) error {
	ipsetFilePath := filepath.Join(assetsPath.String(), "lists", "ipset-all.txt")
	ipsetBackupPath := ipsetFilePath + ".backup"

	// Читаем содержимое бэкапа
	backupContent, err := os.ReadFile(ipsetBackupPath)
	if err != nil {
		return fmt.Errorf("ошибка чтения бэкапа: %w", err)
	}

	// Записываем содержимое в целевой файл
	if err := os.WriteFile(ipsetFilePath, backupContent, 0644); err != nil {
		return fmt.Errorf("ошибка записи восстановленного файла: %w", err)
	}

	log.Printf("Файл восстановлен из бэкапа: %s", ipsetBackupPath)
	return nil
}
