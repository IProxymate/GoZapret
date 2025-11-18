package services

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/IProxymate/GoZapret/internal/domain"
)

// StrategyService управляет стратегиями
type StrategyService struct {
	strategies []domain.Strategy
}

// NewStrategyService создает новый сервис стратегий
func NewStrategyService() *StrategyService {
	return &StrategyService{
		strategies: make([]domain.Strategy, 0),
	}
}

// LoadFromPath загружает стратегии из указанного пути
func (s *StrategyService) LoadFromPath(assetsPath domain.AssetsPath) error {
	slog.Info("Загрузка стратегий", "path", assetsPath)

	if err := assetsPath.Validate(); err != nil {
		slog.Error("Ошибка валидации пути к ресурсам", "error", err)
		s.strategies = []domain.Strategy{}
		return err
	}

	if !assetsPath.HasWinws() {
		slog.Error("winws.exe не найден в указанном пути", "path", assetsPath)
		s.strategies = []domain.Strategy{}
		return domain.ErrWinwsNotFound
	}

	files, err := os.ReadDir(assetsPath.String())
	if err != nil {
		slog.Error("Ошибка чтения директории", "path", assetsPath, "error", err)
		s.strategies = []domain.Strategy{}
		return err
	}

	strategies, err := s.parseStrategiesFromFiles(files)
	if err != nil {
		slog.Error("Ошибка парсинга стратегий", "error", err)
		s.strategies = []domain.Strategy{}
		return err
	}

	if len(strategies) == 0 {
		slog.Warn("Стратегии не найдены", "path", assetsPath)
		s.strategies = []domain.Strategy{}
		return domain.ErrNoStrategiesFound
	}

	s.strategies = strategies
	slog.Debug("Стратегии успешно загружены", "count", len(strategies))
	return nil
}

// GetByName возвращает стратегию по имени
func (s *StrategyService) GetByName(name domain.StrategyName) (*domain.Strategy, error) {
	if err := name.Validate(); err != nil {
		return nil, err
	}

	for i := range s.strategies {
		if s.strategies[i].Name == name {
			return &s.strategies[i], nil
		}
	}

	return nil, domain.ErrStrategyNotFound
}

// GetAll возвращает все загруженные стратегии
func (s *StrategyService) GetAll() []domain.Strategy {
	result := make([]domain.Strategy, len(s.strategies))
	copy(result, s.strategies)
	return result
}

// HasStrategies проверяет, есть ли загруженные стратегии
func (s *StrategyService) HasStrategies() bool {
	return len(s.strategies) > 0
}

// parseStrategiesFromFiles парсит стратегии из файлов директории
func (s *StrategyService) parseStrategiesFromFiles(files []os.DirEntry) ([]domain.Strategy, error) {
	var strategies []domain.Strategy
	batFileRegex := regexp.MustCompile(`^.*\.bat$`)

	for _, file := range files {
		if file.IsDir() || !batFileRegex.MatchString(file.Name()) {
			continue
		}

		if s.isServiceFile(file.Name()) {
			continue
		}

		strategy, err := s.createStrategyFromFile(file.Name())
		if err != nil {
			continue
		}

		strategies = append(strategies, *strategy)
	}

	return strategies, nil
}

// isServiceFile проверяет, является ли файл служебным
func (s *StrategyService) isServiceFile(fileName string) bool {
	servicePrefixes := []string{"service", "install", "uninstall", "setup"}
	lowerName := strings.ToLower(fileName)

	for _, prefix := range servicePrefixes {
		if strings.Contains(lowerName, prefix) {
			return true
		}
	}

	return false
}

// createStrategyFromFile создает стратегию из имени файла
func (s *StrategyService) createStrategyFromFile(fileName string) (*domain.Strategy, error) {
	batFile := domain.BatFileName(fileName)
	if err := batFile.Validate(); err != nil {
		return nil, err
	}

	name := domain.StrategyName(batFile.WithoutExtension())
	if err := name.Validate(); err != nil {
		return nil, err
	}

	description := s.generateDescription(name)

	strategy := &domain.Strategy{
		Name:        name,
		Description: description,
		BatFile:     batFile,
		CreatedAt:   time.Now(),
	}

	if err := strategy.Validate(); err != nil {
		return nil, err
	}

	return strategy, nil
}

// generateDescription генерирует описание стратегии на основе имени
func (s *StrategyService) generateDescription(name domain.StrategyName) string {
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

// ProcessManagerInterface определяет интерфейс для управления процессами
type ProcessManagerInterface interface {
	IsRunning() bool
	StopProcess() error
	StartStrategy(strategy *domain.Strategy, assetsPath domain.AssetsPath, gameFilterEnabled bool) error
}

// RestartStrategy перезапускает стратегию с проверкой "Если запущено"
func (s *StrategyService) RestartStrategy(processManager ProcessManagerInterface, strategy *domain.Strategy, assetsPath domain.AssetsPath, gameFilterEnabled bool) error {
	slog.Info("Перезапуск стратегии", "strategy", strategy.Name)

	// Проверяем, запущен ли процесс
	if processManager.IsRunning() {
		slog.Debug("Остановка текущего процесса перед перезапуском")
		// Если запущен, останавливаем его
		if err := processManager.StopProcess(); err != nil {
			slog.Error("Ошибка остановки текущего процесса", "error", err)
			return fmt.Errorf("ошибка остановки текущего процесса: %w", err)
		}
		// Ждем короткое время, чтобы убедиться, что процесс полностью остановлен
		time.Sleep(500 * time.Millisecond)
	}

	// Запускаем новую стратегию
	if err := processManager.StartStrategy(strategy, assetsPath, gameFilterEnabled); err != nil {
		slog.Error("Ошибка запуска стратегии при перезапуске", "strategy", strategy.Name, "error", err)
		return fmt.Errorf("ошибка запуска стратегии: %w", err)
	}

	slog.Debug("Стратегия успешно перезапущена", "strategy", strategy.Name)
	return nil
}

// ReadVersionFromServiceBat читает версию из файла service.bat
func (s *StrategyService) ReadVersionFromServiceBat(assetsPath domain.AssetsPath) (string, error) {
	serviceBatPath := filepath.Join(assetsPath.String(), "service.bat")

	file, err := os.Open(serviceBatPath)
	if err != nil {
		slog.Warn("Не удалось открыть service.bat для чтения версии", "path", serviceBatPath, "error", err)
		return "", err
	}
	defer file.Close()

	// Регулярное выражение для поиска строки с версией
	versionRegex := regexp.MustCompile(`set\s+"LOCAL_VERSION=([^"]+)"`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		matches := versionRegex.FindStringSubmatch(line)
		if len(matches) > 1 {
			version := matches[1]
			slog.Info("Версия Zapret найдена в service.bat", "version", version)
			return version, nil
		}
	}

	if err := scanner.Err(); err != nil {
		slog.Error("Ошибка чтения service.bat", "error", err)
		return "", err
	}

	slog.Warn("Версия не найдена в service.bat")
	return "", fmt.Errorf("версия не найдена в service.bat")
}
