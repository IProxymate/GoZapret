package ipset

import (
	"github.com/IProxymate/GoZapret/internal/domain"
)

// Service управляет режимами ipset
type Service struct {
	fileManager   *FileManager
	backupManager *BackupManager
}

// NewService создает новый сервис управления ipset
func NewService() *Service {
	return &Service{
		fileManager:   NewFileManager(),
		backupManager: NewBackupManager(),
	}
}

// UpdateIpsetFile обновляет содержимое файла ipset-all.txt в зависимости от режима
func (s *Service) UpdateIpsetFile(assetsPath domain.AssetsPath, mode string) error {
	return s.fileManager.UpdateMode(assetsPath, mode)
}

// GetCurrentIpsetMode определяет текущий режим ipset по содержимому файла
func (s *Service) GetCurrentIpsetMode(assetsPath domain.AssetsPath) (string, error) {
	return s.fileManager.GetCurrentMode(assetsPath)
}

// BackupIpsetFile создает бэкап файла ipset-all.txt
func (s *Service) BackupIpsetFile(assetsPath domain.AssetsPath) error {
	return s.backupManager.Create(assetsPath)
}
