package checks

import (
	"strings"
	"time"

	"github.com/IProxymate/GoZapret/internal/domain"
	"github.com/IProxymate/GoZapret/internal/utils"
)

// ServiceCheckConfig конфигурация для проверки сервисов
type ServiceCheckConfig struct {
	Name           string   // Название проверки
	SearchPatterns []string // Паттерны для поиска (lower case)
	SuccessMessage string   // Сообщение при успехе
	ErrorMessage   string   // Сообщение при ошибке
	CaseSensitive  bool     // Учитывать регистр
}

// ServiceCheck базовая проверка наличия сервисов через sc query
type ServiceCheck struct {
	config ServiceCheckConfig
}

// NewServiceCheck создаёт новую проверку сервисов
func NewServiceCheck(config ServiceCheckConfig) *ServiceCheck {
	return &ServiceCheck{config: config}
}

func (c *ServiceCheck) Name() string {
	return c.config.Name
}

func (c *ServiceCheck) Check() *domain.DiagnosticResult {
	start := time.Now()
	output, err := utils.OutputHidden("sc", "query")

	result := &domain.DiagnosticResult{
		Name:      c.Name(),
		Timestamp: time.Now(),
		Duration:  time.Since(start),
		Details:   make(map[string]interface{}),
	}

	if err != nil {
		result.Success = false
		result.Message = "ERROR"
		result.Details["result"] = "Не удалось выполнить команду sc query"
		result.Details["error"] = err.Error()
		return result
	}

	outputStr := string(output)
	searchOutput := outputStr
	if !c.config.CaseSensitive {
		searchOutput = strings.ToLower(outputStr)
	}

	// Проверяем все паттерны
	found := false
	for _, pattern := range c.config.SearchPatterns {
		searchPattern := pattern
		if !c.config.CaseSensitive {
			searchPattern = strings.ToLower(pattern)
		}
		if strings.Contains(searchOutput, searchPattern) {
			found = true
			break
		}
	}

	if found {
		result.Success = false
		result.Message = "ERROR"
		result.Details["result"] = c.config.ErrorMessage
		result.Details["output"] = outputStr
	} else {
		result.Success = true
		result.Message = "OK"
		result.Details["result"] = c.config.SuccessMessage
	}

	return result
}

// ProcessCheck проверка наличия процесса через tasklist
type ProcessCheck struct {
	name           string
	processName    string
	successMessage string
	errorMessage   string
}

// NewProcessCheck создаёт новую проверку процесса
func NewProcessCheck(name, processName, successMessage, errorMessage string) *ProcessCheck {
	return &ProcessCheck{
		name:           name,
		processName:    processName,
		successMessage: successMessage,
		errorMessage:   errorMessage,
	}
}

func (c *ProcessCheck) Name() string {
	return c.name
}

func (c *ProcessCheck) Check() *domain.DiagnosticResult {
	start := time.Now()
	output, err := utils.OutputHidden("tasklist", "/FI", "IMAGENAME eq "+c.processName)

	result := &domain.DiagnosticResult{
		Name:      c.Name(),
		Timestamp: time.Now(),
		Duration:  time.Since(start),
		Details:   make(map[string]interface{}),
	}

	if err != nil || !strings.Contains(string(output), c.processName) {
		result.Success = true
		result.Message = "OK"
		result.Details["result"] = c.successMessage
		if err != nil {
			result.Details["error"] = err.Error()
		} else {
			result.Details["output"] = string(output)
		}
		return result
	}

	result.Success = false
	result.Message = "ERROR"
	result.Details["result"] = c.errorMessage
	result.Details["output"] = string(output)

	return result
}

// MultiPatternServiceCheck проверка сервисов с несколькими паттернами (все должны совпасть)
type MultiPatternServiceCheck struct {
	name           string
	patterns       []string // Все паттерны должны быть найдены
	successMessage string
	errorMessage   string
}

// NewMultiPatternServiceCheck создаёт проверку с несколькими обязательными паттернами
func NewMultiPatternServiceCheck(name string, patterns []string, successMessage, errorMessage string) *MultiPatternServiceCheck {
	return &MultiPatternServiceCheck{
		name:           name,
		patterns:       patterns,
		successMessage: successMessage,
		errorMessage:   errorMessage,
	}
}

func (c *MultiPatternServiceCheck) Name() string {
	return c.name
}

func (c *MultiPatternServiceCheck) Check() *domain.DiagnosticResult {
	start := time.Now()
	output, err := utils.OutputHidden("sc", "query")

	result := &domain.DiagnosticResult{
		Name:      c.Name(),
		Timestamp: time.Now(),
		Duration:  time.Since(start),
		Details:   make(map[string]interface{}),
	}

	if err != nil {
		result.Success = false
		result.Message = "ERROR"
		result.Details["result"] = "Не удалось выполнить команду sc query"
		result.Details["error"] = err.Error()
		return result
	}

	outputLower := strings.ToLower(string(output))

	// Все паттерны должны быть найдены
	allFound := true
	for _, pattern := range c.patterns {
		if !strings.Contains(outputLower, strings.ToLower(pattern)) {
			allFound = false
			break
		}
	}

	if allFound {
		result.Success = false
		result.Message = "ERROR"
		result.Details["result"] = c.errorMessage
		result.Details["output"] = string(output)
	} else {
		result.Success = true
		result.Message = "OK"
		result.Details["result"] = c.successMessage
	}

	return result
}

