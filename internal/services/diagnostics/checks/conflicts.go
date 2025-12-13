package checks

import (
	"strings"
	"time"

	"github.com/IProxymate/GoZapret/internal/domain"
	"github.com/IProxymate/GoZapret/internal/utils"
)

// ConflictingBypassesCheck проверяет наличие конфликтующих байпасов
type ConflictingBypassesCheck struct{}

func NewConflictingBypassesCheck() *ConflictingBypassesCheck {
	return &ConflictingBypassesCheck{}
}

func (c *ConflictingBypassesCheck) Name() string {
	return "Проверка конфликтующих байпасов"
}

func (c *ConflictingBypassesCheck) Check() *domain.DiagnosticResult {
	start := time.Now()
	conflictingServices := []string{"GoodbyeDPI", "discordfix_zapret", "winws1", "winws2"}
	var foundConflicts []string

	for _, service := range conflictingServices {
		output, err := utils.CombinedOutputHidden("sc", "query", service)
		if err == nil {
			if !strings.Contains(string(output), "1060") {
				foundConflicts = append(foundConflicts, service)
			}
		}
	}

	result := &domain.DiagnosticResult{
		Name:      c.Name(),
		Timestamp: time.Now(),
		Duration:  time.Since(start),
		Details:   make(map[string]interface{}),
	}

	if len(foundConflicts) > 0 {
		result.Success = false
		result.Message = "ERROR"
		result.Details["result"] = "Найдены: " + strings.Join(foundConflicts, ", ")
		result.Details["conflicts"] = foundConflicts
	} else {
		result.Success = true
		result.Message = "OK"
		result.Details["result"] = "Конфликтующие байпасы не найдены"
	}

	return result
}

// ProxyCheck проверяет настройки системного прокси
type ProxyCheck struct{}

func NewProxyCheck() *ProxyCheck {
	return &ProxyCheck{}
}

func (c *ProxyCheck) Name() string {
	return "Проверка настроек системного прокси"
}

func (c *ProxyCheck) Check() *domain.DiagnosticResult {
	start := time.Now()
	output, err := utils.OutputHidden("reg", "query",
		"HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Internet Settings",
		"/v", "ProxyEnable")

	result := &domain.DiagnosticResult{
		Name:      c.Name(),
		Timestamp: time.Now(),
		Duration:  time.Since(start),
		Details:   make(map[string]interface{}),
	}

	proxyEnabled := false
	if err == nil && strings.Contains(string(output), "0x1") {
		proxyEnabled = true
	}

	if !proxyEnabled {
		result.Success = true
		result.Message = "OK"
		result.Details["result"] = "Прокси отключен"
		result.Details["output"] = string(output)
		return result
	}

	// Получаем адрес прокси-сервера
	output, _ = utils.OutputHidden("reg", "query",
		"HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Internet Settings",
		"/v", "ProxyServer")

	proxyServer := ""
	for _, line := range strings.Split(string(output), "\n") {
		if strings.Contains(line, "ProxyServer") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				proxyServer = parts[2]
			}
			break
		}
	}

	result.Success = false
	result.Message = "ERROR"
	result.Details["result"] = "Включён: " + proxyServer
	result.Details["proxy_server"] = proxyServer
	result.Details["output"] = string(output)

	return result
}

// TCPTimestampsCheck проверяет настройки TCP timestamps
type TCPTimestampsCheck struct{}

func NewTCPTimestampsCheck() *TCPTimestampsCheck {
	return &TCPTimestampsCheck{}
}

func (c *TCPTimestampsCheck) Name() string {
	return "Проверка настроек TCP timestamps"
}

func (c *TCPTimestampsCheck) Check() *domain.DiagnosticResult {
	start := time.Now()
	output, err := utils.OutputHidden("netsh", "interface", "tcp", "show", "global")

	result := &domain.DiagnosticResult{
		Name:      c.Name(),
		Timestamp: time.Now(),
		Duration:  time.Since(start),
		Details:   make(map[string]interface{}),
	}

	if err != nil {
		result.Success = false
		result.Message = "ERROR"
		result.Details["result"] = "Не удалось проверить TCP timestamps"
		result.Details["error"] = err.Error()
		return result
	}

	outputStr := string(output)
	if strings.Contains(outputStr, "timestamps") && strings.Contains(outputStr, "enabled") {
		result.Success = true
		result.Message = "OK"
		result.Details["result"] = "TCP timestamps включены"
		result.Details["output"] = outputStr
		return result
	}

	// Пытаемся включить timestamps
	utils.RunHidden("netsh", "interface", "tcp", "set", "global", "timestamps=enabled")

	result.Success = false
	result.Message = "ERROR"
	result.Details["result"] = "Отключены. Попытка включить..."
	result.Details["output"] = outputStr

	return result
}

// AdguardCheck проверяет наличие Adguard сервиса
type AdguardCheck struct{}

func NewAdguardCheck() *AdguardCheck {
	return &AdguardCheck{}
}

func (c *AdguardCheck) Name() string {
	return "Проверка наличия Adguard сервиса"
}

func (c *AdguardCheck) Check() *domain.DiagnosticResult {
	start := time.Now()
	output, err := utils.OutputHidden("tasklist", "/FI", "IMAGENAME eq AdguardSvc.exe")

	result := &domain.DiagnosticResult{
		Name:      c.Name(),
		Timestamp: time.Now(),
		Duration:  time.Since(start),
		Details:   make(map[string]interface{}),
	}

	if err != nil || !strings.Contains(string(output), "AdguardSvc.exe") {
		result.Success = true
		result.Message = "OK"
		result.Details["result"] = "Adguard сервис не найден"
		if err != nil {
			result.Details["error"] = err.Error()
		} else {
			result.Details["output"] = string(output)
		}
		return result
	}

	result.Success = false
	result.Message = "ERROR"
	result.Details["result"] = "Найден процесс AdguardSvc.exe"
	result.Details["output"] = string(output)

	return result
}

// KillerServicesCheck проверяет наличие конфликтующих Killer сервисов
type KillerServicesCheck struct{}

func NewKillerServicesCheck() *KillerServicesCheck {
	return &KillerServicesCheck{}
}

func (c *KillerServicesCheck) Name() string {
	return "Проверка наличия конфликтующих Killer сервисов"
}

func (c *KillerServicesCheck) Check() *domain.DiagnosticResult {
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

	if strings.Contains(strings.ToLower(string(output)), "killer") {
		result.Success = false
		result.Message = "ERROR"
		result.Details["result"] = "Найдены сервисы, конфликтующие с zapret"
		result.Details["output"] = string(output)
		return result
	}

	result.Success = true
	result.Message = "OK"
	result.Details["result"] = "Killer сервисы не найдены"

	return result
}

// IntelConnectivityCheck проверяет наличие конфликтующего Intel сервиса
type IntelConnectivityCheck struct{}

func NewIntelConnectivityCheck() *IntelConnectivityCheck {
	return &IntelConnectivityCheck{}
}

func (c *IntelConnectivityCheck) Name() string {
	return "Проверка наличия конфликтующего Intel сервиса"
}

func (c *IntelConnectivityCheck) Check() *domain.DiagnosticResult {
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
	if strings.Contains(outputLower, "intel") &&
		strings.Contains(outputLower, "connectivity") &&
		strings.Contains(outputLower, "network") {
		result.Success = false
		result.Message = "ERROR"
		result.Details["result"] = "Найден сервис, конфликтующий с zapret"
		result.Details["output"] = string(output)
		return result
	}

	result.Success = true
	result.Message = "OK"
	result.Details["result"] = "Intel сервисы не найдены"

	return result
}

// CheckPointCheck проверяет наличие сервисов Check Point
type CheckPointCheck struct{}

func NewCheckPointCheck() *CheckPointCheck {
	return &CheckPointCheck{}
}

func (c *CheckPointCheck) Name() string {
	return "Проверка наличия сервисов Check Point"
}

func (c *CheckPointCheck) Check() *domain.DiagnosticResult {
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
	checkpointFound := strings.Contains(outputLower, "tracsrvwrapper") ||
		strings.Contains(outputLower, "epwd")

	if checkpointFound {
		result.Success = false
		result.Message = "ERROR"
		result.Details["result"] = "Найдены сервисы, конфликтующие с zapret"
		result.Details["output"] = string(output)
		return result
	}

	result.Success = true
	result.Message = "OK"
	result.Details["result"] = "Сервисы Check Point не найдены"

	return result
}

// SmartByteCheck проверяет наличие сервисов SmartByte
type SmartByteCheck struct{}

func NewSmartByteCheck() *SmartByteCheck {
	return &SmartByteCheck{}
}

func (c *SmartByteCheck) Name() string {
	return "Проверка наличия сервисов SmartByte"
}

func (c *SmartByteCheck) Check() *domain.DiagnosticResult {
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

	if strings.Contains(strings.ToLower(string(output)), "smartbyte") {
		result.Success = false
		result.Message = "ERROR"
		result.Details["result"] = "Найдены сервисы, конфликтующие с zapret"
		result.Details["output"] = string(output)
		return result
	}

	result.Success = true
	result.Message = "OK"
	result.Details["result"] = "Сервисы SmartByte не найдены"

	return result
}

// VPNServicesCheck проверяет наличие VPN сервисов
type VPNServicesCheck struct{}

func NewVPNServicesCheck() *VPNServicesCheck {
	return &VPNServicesCheck{}
}

func (c *VPNServicesCheck) Name() string {
	return "Проверка наличия VPN сервисов"
}

func (c *VPNServicesCheck) Check() *domain.DiagnosticResult {
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

	if strings.Contains(strings.ToLower(string(output)), "vpn") {
		result.Success = false
		result.Message = "ERROR"
		result.Details["result"] = "Найдены сервисы, возможен конфликт"
		result.Details["output"] = string(output)
		return result
	}

	result.Success = true
	result.Message = "OK"
	result.Details["result"] = "VPN сервисы не найдены"

	return result
}
