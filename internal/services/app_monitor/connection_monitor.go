package app_monitor

import (
	"fmt"
	"log/slog"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"time"

	psnet "github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// ConnectionMonitor мониторит сетевые подключения процесса через gopsutil
type ConnectionMonitor struct {
	mu            sync.RWMutex
	session       *MonitorSession
	stopChan      chan struct{}
	requests      []*NetworkRequest
	targetPIDs    map[int32]bool // Несколько PID (основной процесс + дочерние)
	targetProcess string
	ipsetChecker  *IpsetChecker
	domainChecker *DomainChecker
	dnsCache      *DNSCache
	callbacks     []func(*NetworkRequest)
	seenConns     map[string]time.Time // connKey -> время последнего наблюдения
	logger        *slog.Logger
}

// NewConnectionMonitor создает новый монитор подключений
func NewConnectionMonitor(ipsetChecker *IpsetChecker, domainChecker *DomainChecker) *ConnectionMonitor {
	return &ConnectionMonitor{
		ipsetChecker:  ipsetChecker,
		domainChecker: domainChecker,
		dnsCache:      NewDNSCache(),
		callbacks:     make([]func(*NetworkRequest), 0),
		seenConns:     make(map[string]time.Time),
		logger:        slog.Default(),
	}
}

// OnRequest добавляет callback для обработки новых запросов
func (m *ConnectionMonitor) OnRequest(callback func(*NetworkRequest)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callbacks = append(m.callbacks, callback)
}

// Start начинает мониторинг сетевых подключений для указанного процесса.
// processPath может быть полным путём к .exe или просто именем процесса (например, "Kiro.exe").
func (m *ConnectionMonitor) Start(processPath string) error {
	m.mu.Lock()
	if m.session != nil && m.session.IsRunning {
		m.mu.Unlock()
		return fmt.Errorf("мониторинг уже запущен")
	}

	processName := filepath.Base(processPath)

	// Пытаемся найти процесс, но не требуем его наличия
	pids, _ := m.findAllProcessPIDs(processPath)
	if pids == nil {
		pids = make(map[int32]bool)
	}

	m.targetPIDs = pids
	m.targetProcess = strings.TrimSuffix(strings.ToLower(processName), ".exe")
	m.stopChan = make(chan struct{})
	m.requests = make([]*NetworkRequest, 0)
	m.seenConns = make(map[string]time.Time)
	m.session = &MonitorSession{
		StartTime:   time.Now(),
		ProcessPath: processPath,
		ProcessName: processName,
		Requests:    m.requests,
		IsRunning:   true,
	}
	m.mu.Unlock()

	if len(pids) > 0 {
		m.logger.Info("Начало мониторинга", "process", processName, "pids", len(pids))
	} else {
		m.logger.Info("Начало мониторинга (ожидание запуска процесса)", "process", processName)
	}

	// Запускаем мониторинг в горутине
	go m.runMonitor()

	return nil
}

// findAllProcessPIDs находит все PID процессов, связанных с приложением.
// processPath может быть полным путём или просто именем процесса.
func (m *ConnectionMonitor) findAllProcessPIDs(processPath string) (map[int32]bool, error) {
	processName := strings.ToLower(filepath.Base(processPath))
	baseName := strings.TrimSuffix(processName, ".exe")
	isFullPath := filepath.IsAbs(processPath) || strings.Contains(processPath, string(filepath.Separator))

	// Также ищем связанные процессы (например, start_protected_game.exe для игр с античитом)
	relatedNames := []string{
		baseName,
		"start_protected_game",
		baseName + "_be",  // BattlEye
		baseName + "_eac", // Easy Anti-Cheat
	}

	pids := make(map[int32]bool)

	procs, err := process.Processes()
	if err != nil {
		return nil, err
	}

	for _, p := range procs {
		name, err := p.Name()
		if err != nil {
			continue
		}

		nameLower := strings.ToLower(name)
		nameWithoutExt := strings.TrimSuffix(nameLower, ".exe")

		// Проверяем совпадение с основным именем или связанными процессами
		for _, related := range relatedNames {
			if nameWithoutExt == related || strings.Contains(nameWithoutExt, baseName) {
				pids[p.Pid] = true
				m.logger.Debug("Найден процесс", "name", name, "pid", p.Pid)
				break
			}
		}

		// Проверяем полный путь только если передан полный путь
		if isFullPath {
			exe, err := p.Exe()
			if err == nil {
				exeDir := strings.ToLower(filepath.Dir(exe))
				targetDir := strings.ToLower(filepath.Dir(processPath))
				// Если процесс в той же папке или подпапке
				if strings.HasPrefix(exeDir, targetDir) || strings.HasPrefix(targetDir, exeDir) {
					pids[p.Pid] = true
					m.logger.Debug("Найден процесс по пути", "name", name, "pid", p.Pid, "exe", exe)
				}
			}
		}
	}

	if len(pids) == 0 {
		return nil, fmt.Errorf("процесс %s не найден", processName)
	}

	return pids, nil
}

// Stop останавливает мониторинг
func (m *ConnectionMonitor) Stop() *MonitorResult {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.session == nil || !m.session.IsRunning {
		return nil
	}

	close(m.stopChan)
	m.session.IsRunning = false
	m.session.EndTime = time.Now()
	m.session.Requests = m.requests

	m.logger.Info("Мониторинг остановлен", "requests", len(m.requests))

	// Генерируем результат
	result := m.generateResult()
	return result
}

// IsRunning проверяет, запущен ли мониторинг
func (m *ConnectionMonitor) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.session != nil && m.session.IsRunning
}

// GetCurrentRequests возвращает текущие запросы
func (m *ConnectionMonitor) GetCurrentRequests() []*NetworkRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*NetworkRequest, len(m.requests))
	copy(result, m.requests)
	return result
}

// runMonitor запускает цикл мониторинга
func (m *ConnectionMonitor) runMonitor() {
	// Более частый опрос для захвата короткоживущих подключений
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	// Периодически обновляем список PID (новые дочерние процессы)
	pidTicker := time.NewTicker(2 * time.Second)
	defer pidTicker.Stop()

	// Периодически очищаем устаревшие записи в seenConns
	cleanupTicker := time.NewTicker(30 * time.Second)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.checkConnections()
		case <-pidTicker.C:
			m.refreshPIDs()
		case <-cleanupTicker.C:
			m.cleanupSeenConns()
		}
	}
}

// cleanupSeenConns удаляет записи о подключениях старше 60 секунд,
// чтобы повторные подключения к тому же серверу фиксировались заново.
func (m *ConnectionMonitor) cleanupSeenConns() {
	m.mu.Lock()
	defer m.mu.Unlock()

	threshold := time.Now().Add(-60 * time.Second)
	for key, lastSeen := range m.seenConns {
		if lastSeen.Before(threshold) {
			delete(m.seenConns, key)
		}
	}
}

// refreshPIDs обновляет список отслеживаемых PID — добавляет новые и удаляет завершившиеся
func (m *ConnectionMonitor) refreshPIDs() {
	m.mu.RLock()
	processPath := m.session.ProcessPath
	m.mu.RUnlock()

	newPIDs, _ := m.findAllProcessPIDs(processPath)
	if newPIDs == nil {
		newPIDs = make(map[int32]bool)
	}

	m.mu.Lock()
	// Добавляем новые PID
	for pid := range newPIDs {
		if !m.targetPIDs[pid] {
			m.targetPIDs[pid] = true
			m.logger.Debug("Добавлен новый PID", "pid", pid)
		}
	}

	// Удаляем PID, которые больше не существуют (процесс завершился)
	for pid := range m.targetPIDs {
		if !newPIDs[pid] {
			// Проверяем, жив ли процесс
			if !isProcessAlive(pid) {
				delete(m.targetPIDs, pid)
				m.logger.Debug("Удалён завершившийся PID", "pid", pid)
			}
		}
	}
	m.mu.Unlock()
}

// isProcessAlive проверяет, существует ли процесс с данным PID
func isProcessAlive(pid int32) bool {
	p, err := process.NewProcess(pid)
	if err != nil {
		return false
	}
	// Проверяем, что процесс реально работает
	_, err = p.Status()
	return err == nil
}

// checkConnections проверяет текущие подключения процесса
func (m *ConnectionMonitor) checkConnections() {
	// Периодически обновляем DNS кэш
	m.dnsCache.LoadWindowsDNSCache()

	// Получаем список PID
	m.mu.RLock()
	pids := make(map[int32]bool)
	for pid := range m.targetPIDs {
		pids[pid] = true
	}
	m.mu.RUnlock()

	if len(pids) == 0 {
		return
	}

	// Получаем все подключения
	connections, err := psnet.Connections("all")
	if err != nil {
		m.logger.Warn("Ошибка получения подключений", "error", err)
		return
	}

	now := time.Now()

	for _, conn := range connections {
		if !pids[conn.Pid] {
			continue
		}

		// Фильтруем по состоянию — нас интересуют только активные подключения
		// ESTABLISHED — активное TCP-соединение
		// SYN_SENT — попытка установить соединение (исходящий запрос)
		// Для UDP (conn.Type == 2) состояние не проверяем — UDP stateless
		if conn.Type != 2 { // TCP
			status := strings.ToUpper(conn.Status)
			if status != "ESTABLISHED" && status != "SYN_SENT" && status != "SYN_RECV" {
				continue
			}
		}

		remoteIP := conn.Raddr.IP
		remotePort := conn.Raddr.Port

		// Пропускаем локальные и пустые адреса
		if remoteIP == "" || remoteIP == "0.0.0.0" || remoteIP == "::" ||
			remoteIP == "127.0.0.1" || remoteIP == "::1" ||
			strings.HasPrefix(remoteIP, "169.254.") { // link-local
			continue
		}

		// Создаем ключ для отслеживания уникальных подключений
		connKey := fmt.Sprintf("%s:%d:%d", remoteIP, remotePort, conn.Type)

		m.mu.Lock()
		if lastSeen, exists := m.seenConns[connKey]; exists {
			// Обновляем время последнего наблюдения
			m.seenConns[connKey] = now
			// Если подключение видели менее 60 секунд назад — пропускаем
			if now.Sub(lastSeen) < 60*time.Second {
				m.mu.Unlock()
				continue
			}
		}
		m.seenConns[connKey] = now
		m.mu.Unlock()

		// Определяем протокол
		protocol := "TCP"
		if conn.Type == 2 {
			protocol = "UDP"
		}

		// Парсим IP
		ip := net.ParseIP(remoteIP)
		if ip == nil {
			continue
		}

		// Определяем подсеть /8
		var subnet string
		if ip4 := ip.To4(); ip4 != nil {
			subnet = fmt.Sprintf("%d.0.0.0/8", ip4[0])
		}

		// Ищем hostname для IP (асинхронно не блокируем, используем только кэш)
		hostname := m.dnsCache.LookupCached(remoteIP)

		request := &NetworkRequest{
			Timestamp:   now,
			ProcessName: m.targetProcess,
			ProcessPath: m.session.ProcessPath,
			ProcessID:   uint32(conn.Pid),
			Hostname:    hostname,
			IPAddress:   ip,
			Port:        uint16(remotePort),
			Protocol:    protocol,
			Subnet:      subnet,
		}

		// Проверяем ipset
		if m.ipsetChecker != nil {
			request.InIpset = m.ipsetChecker.Contains(ip)
		}

		// Проверяем домен
		if hostname != "" && m.domainChecker != nil {
			request.InDomains = m.domainChecker.Contains(hostname)
		}

		m.addRequest(request)
	}
}

// addRequest добавляет запрос в список
func (m *ConnectionMonitor) addRequest(req *NetworkRequest) {
	m.mu.Lock()
	// Лимит на количество запросов — защита от утечки памяти при долгом мониторинге
	const maxRequests = 10000
	if len(m.requests) >= maxRequests {
		// Удаляем старейшие 10% записей
		cutoff := maxRequests / 10
		m.requests = m.requests[cutoff:]
	}
	m.requests = append(m.requests, req)
	callbacks := make([]func(*NetworkRequest), len(m.callbacks))
	copy(callbacks, m.callbacks)
	m.mu.Unlock()

	m.logger.Debug("Новое подключение",
		"ip", req.IPAddress.String(),
		"port", req.Port,
		"subnet", req.Subnet,
		"in_ipset", req.InIpset)

	// Вызываем callbacks вне блокировки
	for _, cb := range callbacks {
		cb(req)
	}
}

// generateResult генерирует результат мониторинга
func (m *ConnectionMonitor) generateResult() *MonitorResult {
	result := &MonitorResult{
		Session:       m.session,
		IPStatistics:  make([]*IPStats, 0),
		DomainStats:   make([]*DomainStats, 0),
		MissingIPSets: make([]string, 0),
	}

	// Статистика по подсетям
	subnetMap := make(map[string]*IPStats)
	// Статистика по доменам
	domainMap := make(map[string]*DomainStats)

	for _, req := range m.requests {
		// Статистика по подсетям
		if req.Subnet != "" {
			stats, ok := subnetMap[req.Subnet]
			if !ok {
				stats = &IPStats{
					Subnet:    req.Subnet,
					Count:     0,
					InIpset:   req.InIpset,
					SampleIPs: make([]string, 0),
				}
				subnetMap[req.Subnet] = stats
			}
			stats.Count++

			// Добавляем примеры IP (максимум 5)
			ipStr := req.IPAddress.String()
			found := false
			for _, s := range stats.SampleIPs {
				if s == ipStr {
					found = true
					break
				}
			}
			if !found && len(stats.SampleIPs) < 5 {
				stats.SampleIPs = append(stats.SampleIPs, ipStr)
			}
		}

		// Статистика по доменам
		if req.Hostname != "" {
			dStats, ok := domainMap[req.Hostname]
			if !ok {
				dStats = &DomainStats{
					Domain:    req.Hostname,
					Count:     0,
					InDomains: req.InDomains,
				}
				domainMap[req.Hostname] = dStats
			}
			dStats.Count++
		}
	}

	for _, stats := range subnetMap {
		result.IPStatistics = append(result.IPStatistics, stats)
		if !stats.InIpset {
			result.MissingIPSets = append(result.MissingIPSets, stats.Subnet)
		}
	}

	for _, dStats := range domainMap {
		result.DomainStats = append(result.DomainStats, dStats)
	}

	return result
}
