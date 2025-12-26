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
		result.Details["result"] = "Найдены конфликтующие сервисы: " + strings.Join(foundConflicts, ", ")
		result.Details["conflicts"] = foundConflicts
		result.Details["hint"] = "Удалите эти сервисы через меню 'Удалить службы' или вручную через services.msc"
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
	result.Message = "WARN"
	result.Details["result"] = "Системный прокси включён: " + proxyServer
	result.Details["proxy_server"] = proxyServer
	result.Details["hint"] = "Убедитесь, что прокси работает корректно, или отключите его если не используете"

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

// NewAdguardCheck создаёт проверку Adguard сервиса (через процесс)
func NewAdguardCheck() *ProcessCheckWithHint {
	return NewProcessCheckWithHint(
		"Проверка наличия Adguard сервиса",
		"AdguardSvc.exe",
		"Adguard сервис не найден",
		"Найден процесс AdguardSvc.exe. Adguard может вызывать проблемы с Discord",
		"https://github.com/Flowseal/zapret-discord-youtube/issues/417",
	)
}

// NewKillerServicesCheck создаёт проверку Killer сервисов
func NewKillerServicesCheck() *ServiceCheckWithHint {
	return NewServiceCheckWithHint(ServiceCheckConfig{
		Name:           "Проверка наличия конфликтующих Killer сервисов",
		SearchPatterns: []string{"killer"},
		SuccessMessage: "Killer сервисы не найдены",
		ErrorMessage:   "Найдены Killer сервисы, конфликтующие с zapret",
	}, "https://github.com/Flowseal/zapret-discord-youtube/issues/2512#issuecomment-2821119513")
}

// NewIntelConnectivityCheck создаёт проверку Intel Connectivity сервиса
func NewIntelConnectivityCheck() *MultiPatternServiceCheckWithHint {
	return NewMultiPatternServiceCheckWithHint(
		"Проверка наличия конфликтующего Intel сервиса",
		[]string{"intel", "connectivity", "network"},
		"Intel сервисы не найдены",
		"Найден Intel Connectivity Network Service, конфликтующий с zapret",
		"https://github.com/ValdikSS/GoodbyeDPI/issues/541#issuecomment-2661670982",
	)
}

// NewCheckPointCheck создаёт проверку Check Point сервисов
func NewCheckPointCheck() *ServiceCheckWithHint {
	return NewServiceCheckWithHint(ServiceCheckConfig{
		Name:           "Проверка наличия сервисов Check Point",
		SearchPatterns: []string{"tracsrvwrapper", "epwd"},
		SuccessMessage: "Сервисы Check Point не найдены",
		ErrorMessage:   "Найдены сервисы Check Point, конфликтующие с zapret",
	}, "Попробуйте удалить Check Point")
}

// NewSmartByteCheck создаёт проверку SmartByte сервисов
func NewSmartByteCheck() *ServiceCheckWithHint {
	return NewServiceCheckWithHint(ServiceCheckConfig{
		Name:           "Проверка наличия сервисов SmartByte",
		SearchPatterns: []string{"smartbyte"},
		SuccessMessage: "Сервисы SmartByte не найдены",
		ErrorMessage:   "Найдены сервисы SmartByte, конфликтующие с zapret",
	}, "Попробуйте удалить или отключить SmartByte через services.msc")
}

// VPNServicesCheck проверка VPN сервисов с выводом списка найденных
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

	// Ищем VPN сервисы и собираем их имена
	var foundVPNServices []string
	lines := strings.Split(string(output), "\n")
	
	for _, line := range lines {
		lineLower := strings.ToLower(line)
		if strings.Contains(lineLower, "vpn") {
			// Извлекаем имя сервиса из строки SERVICE_NAME: xxx
			if strings.Contains(lineLower, "service_name") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) >= 2 {
					serviceName := strings.TrimSpace(parts[1])
					if serviceName != "" {
						foundVPNServices = append(foundVPNServices, serviceName)
					}
				}
			}
		}
	}

	if len(foundVPNServices) > 0 {
		result.Success = false
		result.Message = "WARN"
		result.Details["result"] = "Найдены VPN сервисы: " + strings.Join(foundVPNServices, ", ")
		result.Details["vpn_services"] = foundVPNServices
		result.Details["hint"] = "Некоторые VPN могут конфликтовать с zapret. Убедитесь, что все VPN отключены."
	} else {
		result.Success = true
		result.Message = "OK"
		result.Details["result"] = "VPN сервисы не найдены"
	}

	return result
}
