package domain

import (
	"os/exec"
	"time"
)

// Strategy представляет стратегию запуска
type Strategy struct {
	Name        StrategyName `json:"name"`
	Description string       `json:"description"`
	BatFile     BatFileName  `json:"bat_file"`
	Args        []string     `json:"args,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
}

// Validate проверяет корректность стратегии
func (s *Strategy) Validate() error {
	if err := s.Name.Validate(); err != nil {
		return err
	}

	if err := s.BatFile.Validate(); err != nil {
		return err
	}

	if s.Description == "" {
		return ErrInvalidStrategy
	}

	return nil
}

// Config представляет конфигурацию приложения
type Config struct {
	LastStrategyName StrategyName `json:"last_strategy_name"`
	AssetsPath       AssetsPath   `json:"assets_path"`
	LastAssetsPath   AssetsPath   `json:"last_assets_path,omitempty"` // Последний путь, для которого копировались файлы
	AutoStart        bool         `json:"auto_start"`
	GameFilter       bool         `json:"game_filter"`
	IpsetMode        string       `json:"ipset_mode"` // "any", "none", "loaded"
	Version          string       `json:"version"`
	UpdatedAt        time.Time    `json:"updated_at"`
	WorkingDir       string       `json:"working_dir,omitempty"`
}

// Validate проверяет корректность конфигурации
func (c *Config) Validate() error {
	if c.AssetsPath != "" {
		if err := c.AssetsPath.Validate(); err != nil {
			return err
		}
	}

	if c.LastStrategyName != "" {
		if err := c.LastStrategyName.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// ProcessInfo представляет информацию о запущенном процессе
type ProcessInfo struct {
	PID       ProcessID     `json:"pid"`
	Strategy  StrategyName  `json:"strategy"`
	StartedAt time.Time     `json:"started_at"`
	Status    ProcessStatus `json:"status"`
	Command   *exec.Cmd     `json:"-"`
	LastError string        `json:"last_error,omitempty"` // Последняя ошибка процесса
}

// ProcessStatus представляет статус процесса
type ProcessStatus string

const (
	ProcessStatusStarting ProcessStatus = "starting"
	ProcessStatusRunning  ProcessStatus = "running"
	ProcessStatusStopping ProcessStatus = "stopping"
	ProcessStatusStopped  ProcessStatus = "stopped"
	ProcessStatusFailed   ProcessStatus = "failed"
)

// IsActive проверяет, активен ли процесс
func (s ProcessStatus) IsActive() bool {
	return s == ProcessStatusStarting || s == ProcessStatusRunning
}

// DiagnosticResult представляет результат диагностики
type DiagnosticResult struct {
	Name      string                 `json:"name"`
	Success   bool                   `json:"success"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Duration  time.Duration          `json:"duration"`
	Timestamp time.Time              `json:"timestamp"`
}
