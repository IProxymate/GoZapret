package services

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"
	"time"
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

// DomainChecker сервис для проверки доступности доменов
type DomainChecker struct{}

// NewDomainChecker создает новый экземпляр DomainChecker
func NewDomainChecker() *DomainChecker {
	return &DomainChecker{}
}

// CheckDomain проверяет доступность домена через curl
func (dc *DomainChecker) CheckDomain(domain string) *DomainCheckResult {
	result := &DomainCheckResult{
		Domain: domain,
	}

	// Нормализуем домен (убираем протокол если есть)
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimSuffix(domain, "/")

	// Формируем URL
	url := fmt.Sprintf("https://%s", domain)

	slog.Info("Проверка домена", "domain", domain, "url", url)

	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Формируем команду curl
	// -o nul - не сохранять вывод
	// -s - тихий режим
	// -w - формат вывода: HTTP код;время ответа
	// --max-time 10 - максимальное время ожидания
	// -L - следовать редиректам
	// -k - игнорировать ошибки SSL (для тестирования)
	cmd := exec.CommandContext(ctx, "curl",
		"-o", "nul",
		"-s",
		"-w", "%{http_code};%{time_total}",
		"--max-time", "10",
		"-L",
		"-k",
		url,
	)

	// Выполняем команду
	output, err := cmd.Output()
	if err != nil {
		// Проверяем, не истек ли таймаут контекста
		if ctx.Err() == context.DeadlineExceeded {
			result.Error = fmt.Errorf("превышено время ожидания (>15 сек)")
			result.IsBlocked = true
			result.Message = "❌ Домен недоступен (таймаут контекста).\n\n⚠️ ВЕРОЯТНО ЗАБЛОКИРОВАН"
			slog.Warn("Таймаут контекста при проверке домена", "domain", domain)
			return result
		}

		// Проверяем exit code curl
		// Exit code 28 = таймаут операции (--max-time)
		// Exit code 7 = не удалось подключиться к хосту
		// Exit code 6 = не удалось разрешить хост
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode := exitErr.ExitCode()
			switch exitCode {
			case 28:
				result.Error = fmt.Errorf("таймаут операции curl (>10 сек)")
				result.IsBlocked = true
				result.Message = "❌ Домен недоступен (таймаут curl).\n" +
					"Время ответа превысило 10 секунд.\n\n" +
					"⚠️ ВЕРОЯТНО ЗАБЛОКИРОВАН\n\n" +
					"💡 Рекомендация: добавьте домен в список включенных"
				slog.Warn("Таймаут curl при проверке домена", "domain", domain, "exitCode", exitCode)
				return result
			case 7:
				result.Error = fmt.Errorf("не удалось подключиться к хосту")
				result.IsBlocked = true
				result.Message = "❌ Не удалось подключиться к хосту.\n\n" +
					"⚠️ ВОЗМОЖНО ЗАБЛОКИРОВАН или хост недоступен"
				slog.Warn("Не удалось подключиться к хосту", "domain", domain, "exitCode", exitCode)
				return result
			case 6:
				result.Error = fmt.Errorf("не удалось разрешить имя хоста")
				result.Message = "❌ Не удалось разрешить имя хоста (DNS).\n\n" +
					"Проверьте правильность написания домена."
				slog.Warn("Не удалось разрешить имя хоста", "domain", domain, "exitCode", exitCode)
				return result
			default:
				result.Error = fmt.Errorf("curl завершился с кодом %d", exitCode)
				result.Message = fmt.Sprintf("❌ Ошибка curl (код %d).\n\n%v", exitCode, err)
				slog.Error("Ошибка curl при проверке домена", "domain", domain, "exitCode", exitCode, "error", err)
				return result
			}
		}

		result.Error = err
		result.Message = fmt.Sprintf("❌ Ошибка при проверке:\n%v", err)
		slog.Error("Ошибка при проверке домена", "domain", domain, "error", err)
		return result
	}

	// Парсим результат
	parts := strings.Split(strings.TrimSpace(string(output)), ";")
	if len(parts) != 2 {
		result.Error = fmt.Errorf("неверный формат ответа curl: %s", string(output))
		result.Message = "❌ Ошибка парсинга ответа"
		slog.Error("Неверный формат ответа curl", "output", string(output))
		return result
	}

	// Парсим HTTP код
	httpCode, err := strconv.Atoi(parts[0])
	if err != nil {
		result.Error = fmt.Errorf("не удалось распарсить HTTP код: %v", err)
		result.Message = "❌ Ошибка парсинга HTTP кода"
		slog.Error("Ошибка парсинга HTTP кода", "code", parts[0], "error", err)
		return result
	}
	result.HTTPCode = httpCode

	// Парсим время ответа
	responseTime, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		result.Error = fmt.Errorf("не удалось распарсить время ответа: %v", err)
		result.Message = "❌ Ошибка парсинга времени ответа"
		slog.Error("Ошибка парсинга времени ответа", "time", parts[1], "error", err)
		return result
	}
	result.ResponseTime = responseTime

	// Анализируем результат
	result.Message = dc.analyzeResult(httpCode, responseTime)
	result.IsBlocked = dc.isLikelyBlocked(httpCode, responseTime)

	slog.Info("Результат проверки домена",
		"domain", domain,
		"httpCode", httpCode,
		"responseTime", responseTime,
		"isBlocked", result.IsBlocked,
	)

	return result
}

// analyzeResult анализирует результаты проверки и формирует сообщение
func (dc *DomainChecker) analyzeResult(httpCode int, responseTime float64) string {
	var msg strings.Builder

	// Анализ HTTP кода
	switch {
	case httpCode >= 200 && httpCode < 300:
		msg.WriteString("✅ Домен доступен")
	case httpCode >= 300 && httpCode < 400:
		msg.WriteString("↪️ Редирект")
	case httpCode >= 400 && httpCode < 500:
		msg.WriteString("⚠️ Ошибка клиента")
	case httpCode >= 500:
		msg.WriteString("❌ Ошибка сервера")
	case httpCode == 0:
		msg.WriteString("❌ Нет ответа")
	default:
		msg.WriteString("❓ Неизвестный статус")
	}

	msg.WriteString(fmt.Sprintf(" (HTTP %d)\n", httpCode))

	// Анализ времени ответа
	msg.WriteString(fmt.Sprintf("⏱️ Время ответа: %.3f сек\n", responseTime))

	// Оценка скорости
	switch {
	case responseTime < 0.5:
		msg.WriteString("🚀 Очень быстро")
	case responseTime < 1.0:
		msg.WriteString("⚡ Быстро")
	case responseTime < 2.0:
		msg.WriteString("🐌 Медленно")
	case responseTime < 5.0:
		msg.WriteString("🐢 Очень медленно")
	default:
		msg.WriteString("⏳ Критически медленно")
	}

	msg.WriteString("\n\n")

	// Вывод о блокировке
	if dc.isLikelyBlocked(httpCode, responseTime) {
		msg.WriteString("⚠️ ВЕРОЯТНО ЗАБЛОКИРОВАН\n")
		msg.WriteString("Признаки:\n")
		if responseTime > 2.0 {
			msg.WriteString("• Слишком долгий ответ (>2 сек)\n")
		}
		if httpCode == 0 {
			msg.WriteString("• Нет HTTP ответа\n")
		}
		msg.WriteString("\n💡 Рекомендация: добавьте домен в список включенных")
	} else {
		msg.WriteString("✅ СКОРЕЕ ВСЕГО НЕ ЗАБЛОКИРОВАН\n")
		if responseTime < 1.0 && httpCode >= 200 && httpCode < 400 {
			msg.WriteString("Домен работает нормально")
		}
	}

	return msg.String()
}

// isLikelyBlocked определяет, вероятно ли домен заблокирован
func (dc *DomainChecker) isLikelyBlocked(httpCode int, responseTime float64) bool {
	// Признаки блокировки:
	// 1. Очень долгий ответ (>2 сек) - DPI может замедлять
	// 2. Нет HTTP ответа (код 0)
	// 3. Таймаут (обрабатывается отдельно)

	if responseTime > 2.0 {
		return true
	}

	if httpCode == 0 {
		return true
	}

	return false
}
