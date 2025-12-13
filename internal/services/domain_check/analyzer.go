package domain_check

// AnalysisResult содержит результаты анализа
type AnalysisResult struct {
	IsBlocked bool
	Message   string
}

// Analyzer анализирует результаты проверки
type Analyzer struct{}

// NewAnalyzer создает новый анализатор
func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

// Analyze анализирует результаты проверки
func (a *Analyzer) Analyze(result *RawResult) *AnalysisResult {
	if result.Error != nil {
		return &AnalysisResult{
			IsBlocked: true,
			Message:   result.Error.Error(),
		}
	}

	return &AnalysisResult{
		IsBlocked: a.IsLikelyBlocked(result.HTTPCode, result.ResponseTime),
		Message:   "",
	}
}

// IsLikelyBlocked определяет, вероятно ли домен заблокирован
func (a *Analyzer) IsLikelyBlocked(httpCode int, responseTime float64) bool {
	// Признаки блокировки:
	// 1. Очень долгий ответ (>2 сек) - DPI может замедлять
	// 2. Нет HTTP ответа (код 0)
	if responseTime > 2.0 {
		return true
	}

	if httpCode == 0 {
		return true
	}

	return false
}
