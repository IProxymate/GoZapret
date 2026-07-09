package app_monitor

import (
	"bufio"
	"net"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

// DNSCache кэширует соответствие IP -> домен
type DNSCache struct {
	mu             sync.RWMutex
	cache          map[string]string // IP -> hostname
	pendingLookups map[string]bool   // IP, для которых уже запущен reverse lookup
	lastLoad       time.Time

	// Предкомпилированные регулярки для парсинга ipconfig /displaydns
	ipv4Regex *regexp.Regexp
	ipv6Regex *regexp.Regexp
	nameRegex *regexp.Regexp
}

// NewDNSCache создает новый DNS кэш
func NewDNSCache() *DNSCache {
	return &DNSCache{
		cache:          make(map[string]string),
		pendingLookups: make(map[string]bool),
		ipv4Regex:      regexp.MustCompile(`(?:A \(Host\) Record|Запись A).*?:\s*(\d+\.\d+\.\d+\.\d+)`),
		ipv6Regex:      regexp.MustCompile(`(?:AAAA \(Host\) Record|Запись AAAA).*?:\s*([0-9a-fA-F:]+)`),
		nameRegex:      regexp.MustCompile(`(?:Record Name|Имя записи).*?:\s*(.+)`),
	}
}

// LoadWindowsDNSCache загружает DNS кэш из Windows
func (d *DNSCache) LoadWindowsDNSCache() {
	d.mu.Lock()

	// Не обновляем чаще чем раз в 10 секунд
	if time.Since(d.lastLoad) < 10*time.Second {
		d.mu.Unlock()
		return
	}
	d.lastLoad = time.Now()
	d.mu.Unlock()

	// Выполняем ipconfig /displaydns вне блокировки (может занять время)
	cmd := exec.Command("ipconfig", "/displaydns")
	output, err := cmd.Output()
	if err != nil {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	d.parseDisplayDNS(string(output))
}

// parseDisplayDNS парсит вывод ipconfig /displaydns
func (d *DNSCache) parseDisplayDNS(output string) {
	scanner := bufio.NewScanner(strings.NewReader(output))

	var currentHost string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Ищем имя записи
		if matches := d.nameRegex.FindStringSubmatch(line); len(matches) > 1 {
			currentHost = strings.TrimSpace(matches[1])
			continue
		}

		// Ищем A запись (IPv4)
		if matches := d.ipv4Regex.FindStringSubmatch(line); len(matches) > 1 && currentHost != "" {
			ip := strings.TrimSpace(matches[1])
			// Сохраняем только если ещё нет записи для этого IP
			if _, exists := d.cache[ip]; !exists {
				d.cache[ip] = currentHost
			}
			continue
		}

		// Ищем AAAA запись (IPv6)
		if matches := d.ipv6Regex.FindStringSubmatch(line); len(matches) > 1 && currentHost != "" {
			ip := strings.TrimSpace(matches[1])
			if _, exists := d.cache[ip]; !exists {
				d.cache[ip] = currentHost
			}
		}
	}
}

// LookupCached ищет домен по IP только в кэше (не блокирует).
// Если записи нет — запускает асинхронный reverse DNS lookup.
func (d *DNSCache) LookupCached(ip string) string {
	d.mu.RLock()
	hostname, ok := d.cache[ip]
	d.mu.RUnlock()

	if ok {
		return hostname
	}

	// Запускаем асинхронный reverse lookup если ещё не запущен
	d.mu.Lock()
	if !d.pendingLookups[ip] {
		d.pendingLookups[ip] = true
		go d.asyncReverseLookup(ip)
	}
	d.mu.Unlock()

	return ""
}

// asyncReverseLookup выполняет reverse DNS lookup в фоне
func (d *DNSCache) asyncReverseLookup(ip string) {
	names, err := net.LookupAddr(ip)

	d.mu.Lock()
	defer d.mu.Unlock()

	delete(d.pendingLookups, ip)

	if err == nil && len(names) > 0 {
		hostname := strings.TrimSuffix(names[0], ".")
		d.cache[ip] = hostname
	}
}

// Lookup ищет домен по IP (синхронный, с блокирующим reverse lookup при промахе)
func (d *DNSCache) Lookup(ip string) string {
	d.mu.RLock()
	hostname, ok := d.cache[ip]
	d.mu.RUnlock()

	if ok {
		return hostname
	}

	// Пробуем reverse DNS lookup с таймаутом
	names, err := net.LookupAddr(ip)
	if err == nil && len(names) > 0 {
		hostname = strings.TrimSuffix(names[0], ".")
		d.mu.Lock()
		d.cache[ip] = hostname
		d.mu.Unlock()
		return hostname
	}

	return ""
}

// Add добавляет запись в кэш
func (d *DNSCache) Add(ip, hostname string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.cache[ip] = hostname
}
