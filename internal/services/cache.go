package services

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/windows"
)

// CacheService управляет кэшем Discord
type CacheService struct{}

// NewCacheService создает новый сервис управления кэшем
func NewCacheService() *CacheService {
	return &CacheService{}
}

// ClearDiscordCache очищает кэш Discord
func (c *CacheService) ClearDiscordCache() error {
	// Получаем путь к AppData
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return nil
	}

	// Путь к кэшу Discord
	discordCachePath := filepath.Join(appData, "discord", "Cache")

	// Проверяем существование директории
	if _, err := os.Stat(discordCachePath); os.IsNotExist(err) {
		return nil
	}

	// Удаляем содержимое директории кэша
	return os.RemoveAll(discordCachePath)
}

// KillDiscordProcesses убивает все процессы Discord
func (c *CacheService) KillDiscordProcesses() error {
	cmd := exec.Command("taskkill", "/IM", "Discord.exe", "/F")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.CREATE_NO_WINDOW,
	}

	// Игнорируем ошибку, если процесс не найден
	cmd.Run()
	return nil
}
