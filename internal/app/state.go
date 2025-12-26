package app

import (
	"fyne.io/fyne/v2/data/binding"
	"github.com/IProxymate/GoZapret/internal/domain"
	"github.com/IProxymate/GoZapret/internal/services/process"
)

// State содержит состояние UI приложения через биндинги Fyne
// Биндинги позволяют автоматически обновлять UI при изменении данных
type State struct {
	// Статус процесса
	StatusText binding.String
	IsRunning  binding.Bool

	// Стратегии
	SelectedStrategy binding.String
	Strategies       binding.StringList

	// Настройки
	AutoStart  binding.Bool
	GameFilter binding.Bool
	IpsetMode  binding.String
	Version    binding.String
}

// NewState создает новое состояние UI
func NewState() *State {
	return &State{
		StatusText:       binding.NewString(),
		IsRunning:        binding.NewBool(),
		SelectedStrategy: binding.NewString(),
		Strategies:       binding.NewStringList(),
		AutoStart:        binding.NewBool(),
		GameFilter:       binding.NewBool(),
		IpsetMode:        binding.NewString(),
		Version:          binding.NewString(),
	}
}

// InitFromConfig инициализирует состояние из конфигурации
func (s *State) InitFromConfig(configService ConfigService, processManager *process.Manager) {
	cfg := configService.GetConfig()

	// Устанавливаем начальный статус с использованием констант
	if processManager.IsWinwsProcessRunning() {
		if cfg.LastStrategyName != "" {
			s.StatusText.Set(domain.StatusRunningWith.Format(cfg.LastStrategyName))
		} else {
			s.StatusText.Set(string(domain.StatusRunning))
		}
		s.IsRunning.Set(true)
	} else {
		s.StatusText.Set(string(domain.StatusStopped))
		s.IsRunning.Set(false)
	}

	// Настройки из конфига
	s.AutoStart.Set(cfg.AutoStart)
	s.GameFilter.Set(cfg.GameFilter)
	s.IpsetMode.Set(cfg.IpsetMode)
	s.Version.Set(cfg.Version)
}

