package services

import (
	"testing"
	"time"

	"github.com/IProxymate/GoZapret/internal/domain"
)

// MockProcessManager - тестовая реализация ProcessManager
type MockProcessManager struct {
	running bool
}

func (m *MockProcessManager) IsRunning() bool {
	return m.running
}

func (m *MockProcessManager) StopProcess() error {
	if m.running {
		m.running = false
	}
	return nil
}

func (m *MockProcessManager) StartStrategy(strategy *domain.Strategy, assetsPath domain.AssetsPath, gameFilterEnabled bool) error {
	m.running = true
	return nil
}

func TestRestartStrategy(t *testing.T) {
	// Создаем стратегию для тестирования
	strategy := &domain.Strategy{
		Name:        "test",
		Description: "Тестовая стратегия",
		BatFile:     "test.bat",
		CreatedAt:   time.Now(),
	}

	// Тестируем сценарий, когда процесс не запущен
	t.Run("Restart when not running", func(t *testing.T) {
		mockPM := &MockProcessManager{running: false}
		service := NewStrategyService()
		assetsPath := domain.AssetsPath(".")

		err := service.RestartStrategy(mockPM, strategy, assetsPath, false)
		if err != nil {
			t.Errorf("Ожидается успешное выполнение, получена ошибка: %v", err)
		}

		if !mockPM.running {
			t.Error("Ожидается, что процесс будет запущен после вызова RestartStrategy")
		}
	})

	// Тестируем сценарий, когда процесс уже запущен
	t.Run("Restart when running", func(t *testing.T) {
		mockPM := &MockProcessManager{running: true}
		service := NewStrategyService()
		assetsPath := domain.AssetsPath(".")

		err := service.RestartStrategy(mockPM, strategy, assetsPath, false)
		if err != nil {
			t.Errorf("Ожидается успешное выполнение, получена ошибка: %v", err)
		}

		if !mockPM.running {
			t.Error("Ожидается, что процесс будет запущен после перезапуска")
		}
	})
}
