package domain_check

import (
	"log/slog"
)

// DomainCheckResult содержит результаты проверки домена
type DomainCheckResult struct {
	Domain       string
	HTTPCode     int
	ResponseTime float64
	Error        error
	IsBlocked    bool
	Message      string
}

// Checker основной сервис проверки доменов
type Checker struct {
	httpClient HTTPClient
	analyzer   *Analyzer
	formatter  *Formatter
}

// NewChecker создает новый чекер доменов
func NewChecker(httpClient HTTPClient, analyzer *Analyzer, formatter *Formatter) *Checker {
	return &Checker{
		httpClient: httpClient,
		analyzer:   analyzer,
		formatter:  formatter,
	}
}

// CheckDomain проверяет доступность домена
func (c *Checker) CheckDomain(domain string) *DomainCheckResult {
	slog.Info("Проверка домена", "domain", domain)

	result := &DomainCheckResult{
		Domain: domain,
	}

	// Выполняем HTTP запрос
	rawResult, err := c.httpClient.Check(domain)
	if err != nil {
		result.Error = err
		result.IsBlocked = true

		// Анализируем ошибку
		analysis := c.analyzer.Analyze(rawResult)
		result.Message = c.formatter.formatError(analysis.Message)

		slog.Warn("Ошибка при проверке домена", "domain", domain, "error", err)
		return result
	}

	result.HTTPCode = rawResult.HTTPCode
	result.ResponseTime = rawResult.ResponseTime

	// Анализируем результат
	analysis := c.analyzer.Analyze(rawResult)
	result.IsBlocked = analysis.IsBlocked

	// Форматируем сообщение
	result.Message = c.formatter.Format(rawResult.HTTPCode, rawResult.ResponseTime, analysis)

	slog.Info("Результат проверки домена",
		"domain", domain,
		"httpCode", rawResult.HTTPCode,
		"responseTime", rawResult.ResponseTime,
		"isBlocked", result.IsBlocked,
	)

	return result
}
