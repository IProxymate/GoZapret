package domain_check

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/IProxymate/GoZapret/internal/utils"
)

// RawResult содержит сырые результаты HTTP запроса
type RawResult struct {
	HTTPCode     int
	ResponseTime float64
	Error        error
}

// HTTPClient интерфейс для HTTP клиента
type HTTPClient interface {
	Check(domain string) (*RawResult, error)
}

// CurlClient реализация HTTP клиента через curl
type CurlClient struct{}

// NewCurlClient создает новый curl клиент
func NewCurlClient() *CurlClient {
	return &CurlClient{}
}

// Check проверяет домен через curl
func (c *CurlClient) Check(domain string) (*RawResult, error) {
	// Нормализуем домен
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimSuffix(domain, "/")

	url := fmt.Sprintf("https://%s", domain)

	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Выполняем curl запрос
	output, err := utils.OutputHiddenContext(ctx, "curl",
		"-o", "nul",
		"-s",
		"-w", "%{http_code};%{time_total}",
		"--max-time", "10",
		"-L",
		"-k",
		url,
	)

	result := &RawResult{}

	if err != nil {
		// Проверяем таймаут контекста
		if ctx.Err() == context.DeadlineExceeded {
			result.Error = fmt.Errorf("превышено время ожидания (>15 сек)")
			return result, result.Error
		}

		// Проверяем exit code curl
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode := exitErr.ExitCode()
			switch exitCode {
			case 28:
				result.Error = fmt.Errorf("таймаут операции curl (>10 сек)")
			case 7:
				result.Error = fmt.Errorf("не удалось подключиться к хосту")
			case 6:
				result.Error = fmt.Errorf("не удалось разрешить имя хоста")
			default:
				result.Error = fmt.Errorf("curl завершился с кодом %d", exitCode)
			}
			return result, result.Error
		}

		result.Error = err
		return result, err
	}

	// Парсим результат
	parts := strings.Split(strings.TrimSpace(string(output)), ";")
	if len(parts) != 2 {
		result.Error = fmt.Errorf("неверный формат ответа curl: %s", string(output))
		return result, result.Error
	}

	// Парсим HTTP код
	httpCode, err := strconv.Atoi(parts[0])
	if err != nil {
		result.Error = fmt.Errorf("не удалось распарсить HTTP код: %v", err)
		return result, result.Error
	}
	result.HTTPCode = httpCode

	// Парсим время ответа
	responseTime, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		result.Error = fmt.Errorf("не удалось распарсить время ответа: %v", err)
		return result, result.Error
	}
	result.ResponseTime = responseTime

	return result, nil
}
