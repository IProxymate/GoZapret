package checks

import (
	"strings"
	"time"

	"github.com/IProxymate/GoZapret/internal/domain"
	"github.com/IProxymate/GoZapret/internal/utils"
)

// AdminChecker интерфейс для проверки прав администратора
type AdminChecker interface {
	IsAdmin() bool
}

// AdminCheck проверяет права администратора
type AdminCheck struct {
	adminChecker AdminChecker
}

func NewAdminCheck(adminChecker AdminChecker) *AdminCheck {
	return &AdminCheck{adminChecker: adminChecker}
}

func (c *AdminCheck) Name() string {
	return "Проверка прав администратора"
}

func (c *AdminCheck) Check() *domain.DiagnosticResult {
	start := time.Now()
	isAdmin := c.adminChecker.IsAdmin()

	result := &domain.DiagnosticResult{
		Name:      c.Name(),
		Success:   isAdmin,
		Timestamp: time.Now(),
		Duration:  time.Since(start),
		Details:   make(map[string]interface{}),
	}

	if isAdmin {
		result.Message = "OK"
		result.Details["result"] = "Приложение запущено с правами администратора"
	} else {
		result.Message = "ERROR"
		result.Details["result"] = "Приложение запущено БЕЗ прав администратора"
	}

	return result
}

// WinDivertCheck проверяет наличие драйвера WinDivert
type WinDivertCheck struct{}

func NewWinDivertCheck() *WinDivertCheck {
	return &WinDivertCheck{}
}

func (c *WinDivertCheck) Name() string {
	return "Проверка драйвера WinDivert"
}

func (c *WinDivertCheck) Check() *domain.DiagnosticResult {
	start := time.Now()
	output, err := utils.OutputHidden("sc", "query", "WinDivert")

	result := &domain.DiagnosticResult{
		Name:      c.Name(),
		Timestamp: time.Now(),
		Duration:  time.Since(start),
		Details:   make(map[string]interface{}),
	}

	if err == nil && len(output) > 0 {
		result.Success = true
		result.Message = "OK"
		result.Details["result"] = "Драйвер WinDivert установлен"
		result.Details["output"] = string(output)
	} else {
		result.Success = false
		result.Message = "ERROR"
		result.Details["result"] = "Драйвер WinDivert не найден или не установлен"
		if err != nil {
			result.Details["error"] = err.Error()
		}
	}

	return result
}

// NetworkCheck проверяет сетевое подключение
type NetworkCheck struct{}

func NewNetworkCheck() *NetworkCheck {
	return &NetworkCheck{}
}

func (c *NetworkCheck) Name() string {
	return "Проверка сетевого подключения"
}

func (c *NetworkCheck) Check() *domain.DiagnosticResult {
	start := time.Now()
	err := utils.RunHidden("ping", "-n", "1", "8.8.8.8")

	result := &domain.DiagnosticResult{
		Name:      c.Name(),
		Timestamp: time.Now(),
		Duration:  time.Since(start),
		Details:   make(map[string]interface{}),
	}

	if err == nil {
		result.Success = true
		result.Message = "OK"
		result.Details["result"] = "Сетевое подключение работает"
	} else {
		result.Success = false
		result.Message = "ERROR"
		result.Details["result"] = "Проблемы с сетевым подключением"
		result.Details["error"] = err.Error()
	}

	return result
}

// WinwsCheck проверяет запущенный процесс winws.exe
type WinwsCheck struct{}

func NewWinwsCheck() *WinwsCheck {
	return &WinwsCheck{}
}

func (c *WinwsCheck) Name() string {
	return "Проверка запущенного winws.exe"
}

func (c *WinwsCheck) Check() *domain.DiagnosticResult {
	start := time.Now()
	winwsOutput, err := utils.OutputHidden("tasklist", "/FI", "IMAGENAME eq winws.exe")
	winwsRunning := strings.Contains(string(winwsOutput), "winws.exe")

	result := &domain.DiagnosticResult{
		Name:      c.Name(),
		Timestamp: time.Now(),
		Duration:  time.Since(start),
		Details:   make(map[string]interface{}),
	}

	if err == nil && winwsRunning {
		result.Success = true
		result.Message = "OK"
		result.Details["result"] = "winws.exe запущен"
		result.Details["output"] = string(winwsOutput)
	} else {
		result.Success = false
		result.Message = "ERROR"
		result.Details["result"] = "winws.exe не запущен"
		if err != nil {
			result.Details["error"] = err.Error()
		}
	}

	return result
}

// BFECheck проверяет сервис Base Filtering Engine
type BFECheck struct{}

func NewBFECheck() *BFECheck {
	return &BFECheck{}
}

func (c *BFECheck) Name() string {
	return "Проверка сервиса Base Filtering Engine"
}

func (c *BFECheck) Check() *domain.DiagnosticResult {
	start := time.Now()
	output, err := utils.OutputHidden("sc", "query", "BFE")

	result := &domain.DiagnosticResult{
		Name:      c.Name(),
		Timestamp: time.Now(),
		Duration:  time.Since(start),
		Details:   make(map[string]interface{}),
	}

	if err != nil {
		result.Success = false
		result.Message = "ERROR"
		result.Details["result"] = "Не удалось выполнить команду sc query BFE"
		result.Details["error"] = err.Error()
		return result
	}

	if strings.Contains(string(output), "RUNNING") {
		result.Success = true
		result.Message = "OK"
		result.Details["result"] = "Сервис запущен"
		result.Details["output"] = string(output)
	} else {
		result.Success = false
		result.Message = "ERROR"
		result.Details["result"] = "Сервис не запущен. Требуется для работы zapret."
		result.Details["output"] = string(output)
	}

	return result
}
