package diagnostics

import (
	"sync"

	"github.com/IProxymate/GoZapret/internal/domain"
)

// ParallelExecutor выполняет проверки параллельно
type ParallelExecutor struct {
	checkers []Checker
}

// NewParallelExecutor создает новый параллельный исполнитель
func NewParallelExecutor(checkers []Checker) *ParallelExecutor {
	return &ParallelExecutor{
		checkers: checkers,
	}
}

// RunAll выполняет все проверки параллельно
func (e *ParallelExecutor) RunAll() []*domain.DiagnosticResult {
	var wg sync.WaitGroup
	results := make([]*domain.DiagnosticResult, len(e.checkers))

	for i, checker := range e.checkers {
		wg.Add(1)
		go func(idx int, c Checker) {
			defer wg.Done()
			results[idx] = c.Check()
		}(i, checker)
	}

	wg.Wait()
	return results
}
