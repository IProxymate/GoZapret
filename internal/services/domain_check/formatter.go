package domain_check

import (
	"fmt"
	"strings"
)

// Formatter форматирует сообщения для UI
type Formatter struct{}

// NewFormatter создает новый форматтер
func NewFormatter() *Formatter {
	return &Formatter{}
}

// Format форматирует результаты анализа в сообщение
func (f *Formatter) Format(httpCode int, responseTime float64, analysis *AnalysisResult) string {
	var msg strings.Builder

	// Если есть ошибка, возвращаем её
	if analysis.Message != "" {
		return f.formatError(analysis.Message)
	}

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
	if analysis.IsBlocked {
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

// formatError форматирует сообщение об ошибке
func (f *Formatter) formatError(errMsg string) string {
	var msg strings.Builder

	msg.WriteString("❌ Домен недоступен\n\n")

	switch {
	case strings.Contains(errMsg, "таймаут"):
		msg.WriteString("⏱️ Превышено время ожидания\n")
		msg.WriteString("Время ответа превысило допустимый лимит.\n\n")
		msg.WriteString("⚠️ ВЕРОЯТНО ЗАБЛОКИРОВАН\n\n")
		msg.WriteString("💡 Рекомендация: добавьте домен в список включенных")
	case strings.Contains(errMsg, "подключиться"):
		msg.WriteString("🔌 Не удалось подключиться к хосту\n\n")
		msg.WriteString("⚠️ ВОЗМОЖНО ЗАБЛОКИРОВАН или хост недоступен")
	case strings.Contains(errMsg, "разрешить"):
		msg.WriteString("🌐 Не удалось разрешить имя хоста (DNS)\n\n")
		msg.WriteString("Проверьте правильность написания домена.")
	default:
		msg.WriteString(fmt.Sprintf("Ошибка: %s", errMsg))
	}

	return msg.String()
}
