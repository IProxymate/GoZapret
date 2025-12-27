package domain_check

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/IProxymate/GoZapret/internal/config"
	"github.com/IProxymate/GoZapret/internal/utils"
)

// TestType тип теста
type TestType string

const (
	TestHTTP  TestType = "HTTP"
	TestTLS12 TestType = "TLS1.2"
	TestTLS13 TestType = "TLS1.3"
	TestPing  TestType = "Ping"
)

// TestResult результат одного теста
type TestResult struct {
	Type         TestType
	Success      bool
	HTTPCode     int
	ResponseTime float64
	Error        error
	Status       string // OK, ERROR, UNSUP, TIMEOUT
}

// TargetResult результат проверки одной цели
type TargetResult struct {
	Name       string
	URL        string
	PingTarget string
	Tests      []TestResult
	PingResult string
	IsURL      bool
}

// MultiCheckResult результат множественной проверки
type MultiCheckResult struct {
	Targets    []TargetResult
	TotalOK    int
	TotalError int
	TotalUnsup int
	PingOK     int
	PingFail   int
}

// Target цель для проверки
type Target struct {
	Name       string
	URL        string // URL для HTTP проверки (пустой для ping-only)
	PingTarget string // IP/домен для ping
}

// MultiChecker расширенный чекер с множественными тестами
type MultiChecker struct {
	timeout time.Duration
}

// NewMultiChecker создает новый MultiChecker
func NewMultiChecker() *MultiChecker {
	return &MultiChecker{
		timeout: 10 * time.Second,
	}
}

// GetDefaultTargets возвращает список целей по умолчанию из конфигурации
func GetDefaultTargets() []Target {
	extCfg := config.GetExternalConfig()
	targets := make([]Target, 0, len(extCfg.CheckDomains)+2)

	for _, d := range extCfg.CheckDomains {
		targets = append(targets, Target{
			Name:       d.Name,
			URL:        d.URL,
			PingTarget: d.Ping,
		})
	}

	// Добавляем DNS-серверы (только ping, без URL)
	targets = append(targets,
		Target{Name: "Cloudflare DNS", URL: "", PingTarget: "1.1.1.1"},
		Target{Name: "Google DNS", URL: "", PingTarget: "8.8.8.8"},
	)

	return targets
}

// CheckAll проверяет все цели параллельно
func (c *MultiChecker) CheckAll(targets []Target) *MultiCheckResult {
	result := &MultiCheckResult{
		Targets: make([]TargetResult, len(targets)),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, target := range targets {
		wg.Add(1)
		go func(idx int, t Target) {
			defer wg.Done()

			targetResult := c.checkTarget(t)

			mu.Lock()
			result.Targets[idx] = targetResult

			// Подсчитываем статистику
			for _, test := range targetResult.Tests {
				switch test.Status {
				case "OK":
					result.TotalOK++
				case "ERROR", "TIMEOUT":
					result.TotalError++
				case "UNSUP":
					result.TotalUnsup++
				}
			}

			if targetResult.PingResult != "" && targetResult.PingResult != "n/a" {
				if targetResult.PingResult == "Timeout" || strings.HasPrefix(targetResult.PingResult, "Error") {
					result.PingFail++
				} else {
					result.PingOK++
				}
			}
			mu.Unlock()
		}(i, target)
	}

	wg.Wait()
	return result
}

// checkTarget проверяет одну цель
func (c *MultiChecker) checkTarget(target Target) TargetResult {
	result := TargetResult{
		Name:       target.Name,
		URL:        target.URL,
		PingTarget: target.PingTarget,
		IsURL:      target.URL != "",
		Tests:      []TestResult{},
	}

	var wg sync.WaitGroup

	// HTTP тесты (если есть URL)
	if target.URL != "" {
		testTypes := []struct {
			Type TestType
			Args []string
		}{
			{TestHTTP, []string{"--http1.1"}},
			{TestTLS12, []string{"--tlsv1.2", "--tls-max", "1.2"}},
			{TestTLS13, []string{"--tlsv1.3", "--tls-max", "1.3"}},
		}

		var testMu sync.Mutex
		for _, tt := range testTypes {
			wg.Add(1)
			go func(testType TestType, args []string) {
				defer wg.Done()
				testResult := c.runCurlTest(target.URL, testType, args)
				testMu.Lock()
				result.Tests = append(result.Tests, testResult)
				testMu.Unlock()
			}(tt.Type, tt.Args)
		}
	}

	// Ping тест
	if target.PingTarget != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result.PingResult = c.runPingTest(target.PingTarget)
		}()
	} else {
		result.PingResult = "n/a"
	}

	wg.Wait()

	// Сортируем тесты в правильном порядке
	sortedTests := make([]TestResult, 0, len(result.Tests))
	for _, tt := range []TestType{TestHTTP, TestTLS12, TestTLS13} {
		for _, t := range result.Tests {
			if t.Type == tt {
				sortedTests = append(sortedTests, t)
				break
			}
		}
	}
	result.Tests = sortedTests

	return result
}

// runCurlTest выполняет curl тест
func (c *MultiChecker) runCurlTest(url string, testType TestType, extraArgs []string) TestResult {
	result := TestResult{
		Type:   testType,
		Status: "ERROR",
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	args := []string{
		"-I",
		"-s",
		"-m", "5",
		"-o", "nul",
		"-w", "%{http_code};%{time_total}",
	}
	args = append(args, extraArgs...)
	args = append(args, url)

	output, err := utils.OutputHiddenContext(ctx, "curl", args...)

	if err != nil {
		// Проверяем на unsupported
		outputStr := string(output)
		if strings.Contains(strings.ToLower(outputStr), "not support") ||
			strings.Contains(strings.ToLower(outputStr), "does not support") {
			result.Status = "UNSUP"
			result.Error = fmt.Errorf("протокол не поддерживается")
			return result
		}

		if ctx.Err() == context.DeadlineExceeded {
			result.Status = "TIMEOUT"
			result.Error = fmt.Errorf("таймаут")
			return result
		}

		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode := exitErr.ExitCode()
			if exitCode == 28 {
				result.Status = "TIMEOUT"
				result.Error = fmt.Errorf("таймаут curl")
				return result
			}
			// TLS версия не поддерживается сервером
			if exitCode == 35 || exitCode == 4 {
				result.Status = "UNSUP"
				result.Error = fmt.Errorf("TLS версия не поддерживается")
				return result
			}
		}

		result.Error = err
		return result
	}

	// Парсим результат
	parts := strings.Split(strings.TrimSpace(string(output)), ";")
	if len(parts) == 2 {
		if code, err := strconv.Atoi(parts[0]); err == nil {
			result.HTTPCode = code
		}
		if respTime, err := strconv.ParseFloat(parts[1], 64); err == nil {
			result.ResponseTime = respTime
		}
	}

	// Определяем успех
	if result.HTTPCode >= 200 && result.HTTPCode < 400 {
		result.Success = true
		result.Status = "OK"
	} else if result.HTTPCode == 0 {
		result.Status = "ERROR"
		result.Error = fmt.Errorf("нет ответа")
	} else {
		result.Status = "ERROR"
		result.Error = fmt.Errorf("HTTP %d", result.HTTPCode)
	}

	return result
}

// runPingTest выполняет ping тест
func (c *MultiChecker) runPingTest(target string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, err := utils.OutputHiddenContext(ctx, "ping", "-n", "3", "-w", "1000", target)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "Timeout"
		}
		return "Timeout"
	}

	// Парсим среднее время из вывода ping
	outputStr := string(output)

	// Ищем строку со средним временем (Average = XXms или Среднее = XXмс)
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		lineLower := strings.ToLower(line)
		if strings.Contains(lineLower, "average") || strings.Contains(lineLower, "среднее") {
			// Ищем число перед "ms" или "мс"
			parts := strings.Fields(line)
			for _, part := range parts {
				part = strings.TrimSuffix(part, "ms")
				part = strings.TrimSuffix(part, "мс")
				part = strings.TrimSuffix(part, ",")
				if ms, err := strconv.Atoi(part); err == nil && ms > 0 {
					return fmt.Sprintf("%d ms", ms)
				}
			}
		}
	}

	// Альтернативный парсинг - ищем время в строках ответа
	for _, line := range lines {
		if strings.Contains(line, "time=") || strings.Contains(line, "время=") {
			// Извлекаем время
			for _, part := range strings.Fields(line) {
				if strings.HasPrefix(part, "time=") || strings.HasPrefix(part, "время=") {
					timeStr := strings.TrimPrefix(part, "time=")
					timeStr = strings.TrimPrefix(timeStr, "время=")
					timeStr = strings.TrimSuffix(timeStr, "ms")
					timeStr = strings.TrimSuffix(timeStr, "мс")
					if ms, err := strconv.Atoi(timeStr); err == nil {
						return fmt.Sprintf("%d ms", ms)
					}
				}
			}
		}
	}

	// Проверяем на потерю пакетов
	if strings.Contains(outputStr, "100%") || strings.Contains(outputStr, "Received = 0") {
		return "Timeout"
	}

	return "OK"
}

// CheckSingleDomain проверяет один домен (для обратной совместимости)
func (c *MultiChecker) CheckSingleDomain(domain string) *TargetResult {
	// Нормализуем домен
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimSuffix(domain, "/")

	target := Target{
		Name:       domain,
		URL:        fmt.Sprintf("https://%s", domain),
		PingTarget: domain,
	}

	result := c.checkTarget(target)
	return &result
}
