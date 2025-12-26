package strategy

import (
	"github.com/IProxymate/GoZapret/internal/domain"
)

// Service управляет стратегиями
type Service struct {
	strategies    []domain.Strategy
	loader        *Loader
	versionReader *VersionReader
}

// NewService создает новый сервис стратегий
func NewService() *Service {
	descGenerator := NewDescriptionGenerator()
	return &Service{
		strategies:    make([]domain.Strategy, 0),
		loader:        NewLoader(descGenerator),
		versionReader: NewVersionReader(),
	}
}

// LoadFromPath загружает стратегии из указанного пути
func (s *Service) LoadFromPath(assetsPath domain.AssetsPath) error {
	strategies, err := s.loader.LoadFromPath(assetsPath)
	if err != nil {
		s.strategies = []domain.Strategy{}
		return err
	}

	s.strategies = strategies
	return nil
}

// GetByName возвращает стратегию по имени
func (s *Service) GetByName(name domain.StrategyName) (*domain.Strategy, error) {
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
func (s *Service) GetAll() []domain.Strategy {
	result := make([]domain.Strategy, len(s.strategies))
	copy(result, s.strategies)
	return result
}

// HasStrategies проверяет, есть ли загруженные стратегии
func (s *Service) HasStrategies() bool {
	return len(s.strategies) > 0
}

// ReadVersionFromServiceBat читает версию из файла service.bat
func (s *Service) ReadVersionFromServiceBat(assetsPath domain.AssetsPath) (string, error) {
	return s.versionReader.ReadFromServiceBat(assetsPath)
}
