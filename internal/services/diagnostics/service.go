package diagnostics

import (
	"log/slog"
	"time"

	"github.com/IProxymate/GoZapret/internal/domain"
)

// Service координирует диагностические проверки
type Service struct {
	executor *ParallelExecutor
}

// NewService создает новый сервис диагностики
func NewService(checkers []Checker) *Service {
	return &Service{
		executor: NewParallelExecutor(checkers),
	}
}

// RunAll выполняет все диагностические проверки
func (s *Service) RunAll() []*domain.DiagnosticResult {
	slog.Debug("Запуск полной диагностики системы")
	start := time.Now()

	results := s.executor.RunAll()

	// Подсчитываем успешные и неуспешные проверки
	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
	}

	slog.Debug("Диагностика завершена",
		"duration", time.Since(start),
		"total", len(results),
		"success", successCount,
		"failed", len(results)-successCount)

	return results
}
