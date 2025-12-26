package domain_check

// NewCheckerWithDefaults создает Checker со всеми зависимостями по умолчанию
func NewCheckerWithDefaults() *Checker {
	httpClient := NewCurlClient()
	analyzer := NewAnalyzer()
	formatter := NewFormatter()

	return NewChecker(httpClient, analyzer, formatter)
}
