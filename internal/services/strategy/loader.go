package strategy

import (
	"log/slog"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/IProxymate/GoZapret/internal/domain"
)

// Loader загружает стратегии из файловой системы
type Loader struct {
	descGenerator *DescriptionGenerator
}

// NewLoader создает новый загрузчик стратегий
func NewLoader(descGenerator *DescriptionGenerator) *Loader {
	return &Loader{
		descGenerator: descGenerator,
	}
}

// LoadFromPath загружает стратегии из указанного пути
func (l *Loader) LoadFromPath(assetsPath domain.AssetsPath) ([]domain.Strategy, error) {
	slog.Info("Загрузка стратегий", "path", assetsPath)

	if err := assetsPath.Validate(); err != nil {
		slog.Error("Ошибка валидации пути к ресурсам", "error", err)
		return []domain.Strategy{}, err
	}

	if !assetsPath.HasWinws() {
		slog.Error("winws.exe не найден в указанном пути", "path", assetsPath)
		return []domain.Strategy{}, domain.ErrWinwsNotFound
	}

	files, err := os.ReadDir(assetsPath.String())
	if err != nil {
		slog.Error("Ошибка чтения директории", "path", assetsPath, "error", err)
		return []domain.Strategy{}, err
	}

	strategies, err := l.parseStrategiesFromFiles(files)
	if err != nil {
		slog.Error("Ошибка парсинга стратегий", "error", err)
		return []domain.Strategy{}, err
	}

	if len(strategies) == 0 {
		slog.Warn("Стратегии не найдены", "path", assetsPath)
		return []domain.Strategy{}, domain.ErrNoStrategiesFound
	}

	slog.Debug("Стратегии успешно загружены", "count", len(strategies))
	return strategies, nil
}

// parseStrategiesFromFiles парсит стратегии из файлов директории
func (l *Loader) parseStrategiesFromFiles(files []os.DirEntry) ([]domain.Strategy, error) {
	var strategies []domain.Strategy
	batFileRegex := regexp.MustCompile(`^.*\.bat$`)

	for _, file := range files {
		if file.IsDir() || !batFileRegex.MatchString(file.Name()) {
			continue
		}

		if l.isServiceFile(file.Name()) {
			continue
		}

		strategy, err := l.createStrategyFromFile(file.Name())
		if err != nil {
			continue
		}

		strategies = append(strategies, *strategy)
	}

	return strategies, nil
}

// isServiceFile проверяет, является ли файл служебным
func (l *Loader) isServiceFile(fileName string) bool {
	servicePrefixes := []string{"service", "install", "uninstall", "setup"}
	lowerName := strings.ToLower(fileName)

	for _, prefix := range servicePrefixes {
		if strings.HasPrefix(lowerName, prefix) {
			return true
		}
	}

	return false
}

// createStrategyFromFile создает стратегию из имени файла
func (l *Loader) createStrategyFromFile(fileName string) (*domain.Strategy, error) {
	batFile := domain.BatFileName(fileName)
	if err := batFile.Validate(); err != nil {
		return nil, err
	}

	name := domain.StrategyName(batFile.WithoutExtension())
	if err := name.Validate(); err != nil {
		return nil, err
	}

	description := l.descGenerator.Generate(name)

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
