package services

import (
	"os"
	"path/filepath"

	"github.com/IProxymate/GoZapret/internal/utils"
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
	// Игнорируем ошибку, если процесс не найден
	utils.RunHidden("taskkill", "/IM", "Discord.exe", "/F")
	return nil
}
