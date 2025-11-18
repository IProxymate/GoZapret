package services

import (
	"log/slog"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/IProxymate/GoZapret/internal/domain"

	"golang.org/x/sys/windows"
)

// DiagnosticsService выполняет диагностику системы
type DiagnosticsService struct {
	adminChecker *AdminChecker
}

// NewDiagnosticsService создает новый сервис диагностики
func NewDiagnosticsService(adminChecker *AdminChecker) *DiagnosticsService {
	return &DiagnosticsService{
		adminChecker: adminChecker,
	}
}

func (d *DiagnosticsService) createNoWindowCommand(name string, arg ...string) *exec.Cmd {
	cmd := exec.Command(name, arg...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
	return cmd
}

// RunAll выполняет все диагностические проверки
func (d *DiagnosticsService) RunAll() []*domain.DiagnosticResult {
	slog.Debug("Запуск полной диагностики системы")
	start := time.Now()

	var results []*domain.DiagnosticResult

	results = append(results, d.checkAdminRights())
	results = append(results, d.checkWinDivertDriver())
	results = append(results, d.checkWinws())
	results = append(results, d.checkNetworkConnectivity())
	results = append(results, d.checkConflictingBypasses())
	results = append(results, d.checkBaseFilteringEngine())
	results = append(results, d.checkProxySettings())
	results = append(results, d.checkTCPTimestamps())
	results = append(results, d.checkAdguard())
	results = append(results, d.checkKillerServices())
	results = append(results, d.checkIntelConnectivity())
	results = append(results, d.checkCheckPoint())
	results = append(results, d.checkSmartByte())
	results = append(results, d.checkVPNServices())

	// Подсчитываем успешные и неуспешные проверки
	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
	}

	slog.Debug("Диагностика завершена",
		"duration", time.Since(start),
		"total", len(results),
		"success", successCount,
		"failed", len(results)-successCount)

	return results
}

// checkAdminRights проверяет права администратора
func (d *DiagnosticsService) checkAdminRights() *domain.DiagnosticResult {
	start := time.Now()
	isAdmin := d.adminChecker.IsAdmin()

	result := &domain.DiagnosticResult{
		Name:      "Проверка прав администратора",
		Success:   isAdmin,
		Timestamp: time.Now(),
		Duration:  time.Since(start),
		Details:   make(map[string]interface{}),
	}

	if isAdmin {
		result.Message = "OK"
		result.Details["result"] = "Приложение запущено с правами администратора"
		slog.Debug("Диагностика: права администратора присутствуют")
	} else {
		result.Message = "ERROR"
		result.Details["result"] = "Приложение запущено БЕЗ прав администратора"
		slog.Warn("Диагностика: приложение запущено без прав администратора")
	}

	return result
}

// checkWinDivertDriver проверяет наличие драйвера WinDivert
func (d *DiagnosticsService) checkWinDivertDriver() *domain.DiagnosticResult {
	start := time.Now()

	cmd := exec.Command("sc", "query", "WinDivert")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.CREATE_NO_WINDOW,
	}

	output, err := cmd.Output()

	result := &domain.DiagnosticResult{
		Name:      "Проверка драйвера WinDivert",
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
		result.Details["error"] = err.Error()
	}

	return result
}

// checkNetworkConnectivity проверяет сетевое подключение
func (d *DiagnosticsService) checkNetworkConnectivity() *domain.DiagnosticResult {
	start := time.Now()

	cmd := exec.Command("ping", "-n", "1", "8.8.8.8")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.CREATE_NO_WINDOW,
	}

	err := cmd.Run()

	result := &domain.DiagnosticResult{
		Name:      "Проверка сетевого подключения",
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

func (d *DiagnosticsService) checkWinws() *domain.DiagnosticResult {
	start := time.Now()
	cmd := d.createNoWindowCommand("tasklist", "/FI", "IMAGENAME eq winws.exe")
	winwsOutput, err := cmd.Output()
	winwsRunning := strings.Contains(string(winwsOutput), "winws.exe")

	result := &domain.DiagnosticResult{
		Name:      "Проверка запущенного winws.exe",
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

// checkConflictingBypasses проверяет наличие конфликтующих байпасов
func (d *DiagnosticsService) checkConflictingBypasses() *domain.DiagnosticResult {
	start := time.Now()
	conflictingServices := []string{"GoodbyeDPI", "discordfix_zapret", "winws1", "winws2"}
	var foundConflicts []string

	for _, service := range conflictingServices {
		cmd := d.createNoWindowCommand("sc", "query", service)
		output, err := cmd.CombinedOutput()
		if err == nil {
			// Проверяем, что вывод содержит информацию о сервисе, а не только ошибку
			if !strings.Contains(string(output), "1060") { // ERROR_SERVICE_DOES_NOT_EXIST
				foundConflicts = append(foundConflicts, service)
			}
		}
	}

	result := &domain.DiagnosticResult{
		Name:      "Проверка конфликтующих байпасов",
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

// checkBaseFilteringEngine проверяет сервис Base Filtering Engine
func (d *DiagnosticsService) checkBaseFilteringEngine() *domain.DiagnosticResult {
	start := time.Now()
	cmd := d.createNoWindowCommand("sc", "query", "BFE")
	output, err := cmd.Output()

	result := &domain.DiagnosticResult{
		Name:      "Проверка сервиса Base Filtering Engine",
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

// checkProxySettings проверяет настройки системного прокси
func (d *DiagnosticsService) checkProxySettings() *domain.DiagnosticResult {
	start := time.Now()
	cmd := d.createNoWindowCommand("reg", "query",
		"HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Internet Settings",
		"/v", "ProxyEnable")
	output, err := cmd.Output()

	result := &domain.DiagnosticResult{
		Name:      "Проверка настроек системного прокси",
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
	cmd = d.createNoWindowCommand("reg", "query",
		"HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Internet Settings",
		"/v", "ProxyServer")
	output, _ = cmd.Output()

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

// checkTCPTimestamps проверяет настройки TCP timestamps
func (d *DiagnosticsService) checkTCPTimestamps() *domain.DiagnosticResult {
	start := time.Now()
	cmd := d.createNoWindowCommand("netsh", "interface", "tcp", "show", "global")
	output, err := cmd.Output()

	result := &domain.DiagnosticResult{
		Name:      "Проверка настроек TCP timestamps",
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
	cmd = d.createNoWindowCommand("netsh", "interface", "tcp", "set", "global", "timestamps=enabled")
	cmd.Run() // Игнорируем ошибку

	result.Success = false
	result.Message = "ERROR"
	result.Details["result"] = "Отключены. Попытка включить..."
	result.Details["output"] = outputStr

	return result
}

// checkAdguard проверяет наличие Adguard сервиса
func (d *DiagnosticsService) checkAdguard() *domain.DiagnosticResult {
	start := time.Now()
	cmd := d.createNoWindowCommand("tasklist", "/FI", "IMAGENAME eq AdguardSvc.exe")
	output, err := cmd.Output()

	result := &domain.DiagnosticResult{
		Name:      "Проверка наличия Adguard сервиса",
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

// checkKillerServices проверяет наличие конфликтующих Killer сервисов
func (d *DiagnosticsService) checkKillerServices() *domain.DiagnosticResult {
	start := time.Now()
	cmd := d.createNoWindowCommand("sc", "query")
	output, err := cmd.Output()

	result := &domain.DiagnosticResult{
		Name:      "Проверка наличия конфликтующих Killer сервисов",
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

// checkIntelConnectivity проверяет наличие конфликтующего Intel сервиса
func (d *DiagnosticsService) checkIntelConnectivity() *domain.DiagnosticResult {
	start := time.Now()
	cmd := d.createNoWindowCommand("sc", "query")
	output, err := cmd.Output()

	result := &domain.DiagnosticResult{
		Name:      "Проверка наличия конфликтующего Intel сервиса",
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

// checkCheckPoint проверяет наличие сервисов Check Point
func (d *DiagnosticsService) checkCheckPoint() *domain.DiagnosticResult {
	start := time.Now()
	cmd := d.createNoWindowCommand("sc", "query")
	output, err := cmd.Output()

	result := &domain.DiagnosticResult{
		Name:      "Проверка наличия сервисов Check Point",
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

// checkSmartByte проверяет наличие сервисов SmartByte
func (d *DiagnosticsService) checkSmartByte() *domain.DiagnosticResult {
	start := time.Now()
	cmd := d.createNoWindowCommand("sc", "query")
	output, err := cmd.Output()

	result := &domain.DiagnosticResult{
		Name:      "Проверка наличия сервисов SmartByte",
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

// checkVPNServices проверяет наличие VPN сервисов
func (d *DiagnosticsService) checkVPNServices() *domain.DiagnosticResult {
	start := time.Now()
	cmd := d.createNoWindowCommand("sc", "query")
	output, err := cmd.Output()

	result := &domain.DiagnosticResult{
		Name:      "Проверка наличия VPN сервисов",
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
