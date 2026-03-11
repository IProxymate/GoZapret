package strategy

import (
	"strings"

	"github.com/IProxymate/GoZapret/internal/domain"
)

// DescriptionGenerator генерирует описания стратегий
type DescriptionGenerator struct{}

// NewDescriptionGenerator создает новый генератор описаний
func NewDescriptionGenerator() *DescriptionGenerator {
	return &DescriptionGenerator{}
}

// Generate генерирует описание стратегии на основе имени
func (g *DescriptionGenerator) Generate(name domain.StrategyName) string {
	nameLower := strings.ToLower(name.String())

	switch {
	case nameLower == "general":
		return "Основная стратегия для Discord и YouTube"
	case strings.Contains(nameLower, "discord"):
		return "Стратегия для Discord " + name.String()
	case strings.Contains(nameLower, "youtube"):
		return "Стратегия для YouTube " + name.String()
	case strings.Contains(nameLower, "alt"):
		return "Альтернативная стратегия " + name.String()
	case strings.Contains(nameLower, "simple fake"):
		return "Стратегия с простым фейком " + name.String()
	case strings.Contains(nameLower, "fake") && strings.Contains(nameLower, "tls"):
		return "Стратегия с FAKE TLS " + name.String()
	case strings.Contains(nameLower, "quic"):
		return "QUIC стратегия " + name.String()
	case strings.Contains(nameLower, "dpi"):
		return "DPI обход стратегия " + name.String()
	default:
		return "Стратегия " + name.String()
	}
}
