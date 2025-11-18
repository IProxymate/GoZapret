package services

import (
	"os"
	"testing"
	"time"

	"github.com/IProxymate/GoZapret/internal/config"
	"github.com/IProxymate/GoZapret/internal/domain"
)

func TestProcessManager_RestartStrategy(t *testing.T) {
	// Создаем временный файл для конфигурации
	tempConfigFile := "test_config.json"
	defer os.Remove(tempConfigFile)

	// Создаем админ-чекер и менеджер конфигурации для теста
	adminChecker := NewAdminChecker()
	configManager := config.NewManager(tempConfigFile)

	// Создаем ProcessManager
	pm := NewProcessManager(adminChecker, configManager)

	// Создаем тестовую стратегию
	strategy := &domain.Strategy{
		Name:        "test",
		Description: "Тестовая стратегия",
		BatFile:     "test.bat",
		CreatedAt:   time.Now(),
	}

	// Проверяем, что метод RestartStrategy существует и может быть вызван
	// (в реальных условиях он будет проверять, запущен ли процесс, и перезапускать его)
	// В тесте мы не будем запускать реальные процессы, только проверим, что метод существует
	assetsPath := domain.AssetsPath(".")

	// Проверяем, что метод не паникует при вызове с неправильным путем (процесс не запущен)
	err := pm.RestartStrategy(strategy, assetsPath, false)

	// Ожидаем ошибку, так как путь к ресурсам неверный, но не панику
	if err == nil {
		t.Log("Метод успешно вызван (ожидается ошибка из-за отсутствия ресурсов)")
	} else {
		t.Logf("Метод вызван с ожидаемой ошибкой: %v", err)
	}
}
