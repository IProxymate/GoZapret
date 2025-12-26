package diagnostics

import (
	"github.com/IProxymate/GoZapret/internal/domain"
)

// Checker интерфейс для диагностических проверок
type Checker interface {
	Name() string
	Check() *domain.DiagnosticResult
}
