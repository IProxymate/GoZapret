package autostart

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows/registry"
)

const autostartKey = `Software\Microsoft\Windows\CurrentVersion\Run`
const autostartValueName = "GoZapret"

// RegistryManager управляет записями автозапуска в реестре Windows (legacy)
type RegistryManager struct{}

// NewRegistryManager создает новый менеджер реестра
func NewRegistryManager() *RegistryManager {
	return &RegistryManager{}
}

// IsEnabled проверяет наличие записи в реестре
func (r *RegistryManager) IsEnabled() (bool, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, autostartKey, registry.READ)
	if err != nil {
		return false, err
	}
	defer key.Close()

	value, _, err := key.GetStringValue(autostartValueName)
	if err != nil {
		if err == registry.ErrNotExist {
			return false, nil
		}
		return false, err
	}

	execPath, err := os.Executable()
	if err != nil {
		return false, fmt.Errorf("не удалось получить путь к исполняемому файлу: %v", err)
	}
	expectedValue := fmt.Sprintf(`"%s" /autostart`, execPath)

	return value == expectedValue, nil
}

// Remove удаляет запись из реестра
func (r *RegistryManager) Remove() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, autostartKey, registry.WRITE)
	if err != nil {
		return err
	}
	defer key.Close()

	return key.DeleteValue(autostartValueName)
}
