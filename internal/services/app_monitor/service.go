package app_monitor

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
)

// Monitor интерфейс для мониторов
type Monitor interface {
	Start(processPath string) error
	Stop() *MonitorResult
	IsRunning() bool
	OnRequest(callback func(*NetworkRequest))
}

// Service сервис мониторинга приложений
type Service struct {
	logger        *slog.Logger
	monitor       Monitor
	workingDir    string
	ipsetChecker  *IpsetChecker
	domainChecker *DomainChecker
	useWinDivert  bool
}

// NewService создает новый сервис мониторинга
func NewService(workingDir string) *Service {
	ipsetChecker := NewIpsetChecker(workingDir)
	domainChecker := NewDomainChecker(workingDir)

	return &Service{
		logger:        slog.Default(),
		workingDir:    workingDir,
		ipsetChecker:  ipsetChecker,
		domainChecker: domainChecker,
		monitor:       NewConnectionMonitor(ipsetChecker, domainChecker),
		useWinDivert:  false,
	}
}

// StartMonitoring начинает мониторинг указанного приложения
func (s *Service) StartMonitoring(executablePath string) error {
	// Проверяем существование файла
	if _, err := os.Stat(executablePath); os.IsNotExist(err) {
		return fmt.Errorf("файл не найден: %s", executablePath)
	}

	// Проверяем расширение
	ext := filepath.Ext(executablePath)
	if ext != ".exe" {
		return fmt.Errorf("файл должен быть исполняемым (.exe): %s", executablePath)
	}

	s.logger.Info("Начало мониторинга приложения", "path", executablePath)
	return s.monitor.Start(executablePath)
}

// StopMonitoring останавливает мониторинг и возвращает результат
func (s *Service) StopMonitoring() *MonitorResult {
	s.logger.Info("Остановка мониторинга")
	result := s.monitor.Stop()

	if result != nil {
		// Сортируем статистику по количеству запросов
		sort.Slice(result.IPStatistics, func(i, j int) bool {
			return result.IPStatistics[i].Count > result.IPStatistics[j].Count
		})
		sort.Slice(result.DomainStats, func(i, j int) bool {
			return result.DomainStats[i].Count > result.DomainStats[j].Count
		})
	}

	return result
}

// IsMonitoring проверяет, запущен ли мониторинг
func (s *Service) IsMonitoring() bool {
	return s.monitor.IsRunning()
}

// OnRequest добавляет callback для обработки новых запросов
func (s *Service) OnRequest(callback func(*NetworkRequest)) {
	s.monitor.OnRequest(callback)
}

// RefreshCheckers обновляет чекеры ipset и доменов
func (s *Service) RefreshCheckers() {
	s.ipsetChecker = NewIpsetChecker(s.workingDir)
	s.domainChecker = NewDomainChecker(s.workingDir)
	s.monitor = NewConnectionMonitor(s.ipsetChecker, s.domainChecker)
}

// FormatResultAsText форматирует результат мониторинга в текст
func (s *Service) FormatResultAsText(result *MonitorResult) string {
	if result == nil {
		return "Нет данных"
	}

	var text string

	text += fmt.Sprintf("=== Результаты мониторинга ===\n")
	text += fmt.Sprintf("Приложение: %s\n", result.Session.ProcessName)
	text += fmt.Sprintf("Время мониторинга: %s - %s\n\n",
		result.Session.StartTime.Format("15:04:05"),
		result.Session.EndTime.Format("15:04:05"))

	text += fmt.Sprintf("=== Статистика по подсетям (/8) ===\n")
	for _, stats := range result.IPStatistics {
		status := "✅ В ipset"
		if !stats.InIpset {
			status = "❌ НЕТ в ipset"
		}
		text += fmt.Sprintf("%s: %d запросов %s\n", stats.Subnet, stats.Count, status)
		if len(stats.SampleIPs) > 0 {
			text += fmt.Sprintf("  Примеры: %v\n", stats.SampleIPs)
		}
	}

	if len(result.MissingIPSets) > 0 {
		text += fmt.Sprintf("\n=== Рекомендуется добавить в ipset ===\n")
		for _, subnet := range result.MissingIPSets {
			text += fmt.Sprintf("%s\n", subnet)
		}
	}

	text += fmt.Sprintf("\n=== Статистика по доменам ===\n")
	for _, stats := range result.DomainStats {
		status := "✅ В списке"
		if !stats.InDomains {
			status = "⚪ Нет в списке"
		}
		text += fmt.Sprintf("%s: %d запросов %s\n", stats.Domain, stats.Count, status)
	}

	return text
}
