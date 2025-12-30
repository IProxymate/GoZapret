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
	mu       sync.RWMutex
	cache    map[string]string // IP -> hostname
	lastLoad time.Time
}

// NewDNSCache создает новый DNS кэш
func NewDNSCache() *DNSCache {
	return &DNSCache{
		cache: make(map[string]string),
	}
}

// LoadWindowsDNSCache загружает DNS кэш из Windows
func (d *DNSCache) LoadWindowsDNSCache() {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Не обновляем чаще чем раз в 5 секунд
	if time.Since(d.lastLoad) < 5*time.Second {
		return
	}
	d.lastLoad = time.Now()

	// Выполняем ipconfig /displaydns
	cmd := exec.Command("ipconfig", "/displaydns")
	output, err := cmd.Output()
	if err != nil {
		return
	}

	d.parseDisplayDNS(string(output))
}

// parseDisplayDNS парсит вывод ipconfig /displaydns
func (d *DNSCache) parseDisplayDNS(output string) {
	scanner := bufio.NewScanner(strings.NewReader(output))

	var currentHost string
	// Регулярка для A записей (IPv4)
	ipv4Regex := regexp.MustCompile(`(?:A \(Host\) Record|Запись A).*?:\s*(\d+\.\d+\.\d+\.\d+)`)
	// Регулярка для имени записи
	nameRegex := regexp.MustCompile(`(?:Record Name|Имя записи).*?:\s*(.+)`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Ищем имя записи
		if matches := nameRegex.FindStringSubmatch(line); len(matches) > 1 {
			currentHost = strings.TrimSpace(matches[1])
			continue
		}

		// Ищем A запись (IPv4)
		if matches := ipv4Regex.FindStringSubmatch(line); len(matches) > 1 && currentHost != "" {
			ip := strings.TrimSpace(matches[1])
			// Сохраняем только если ещё нет записи для этого IP
			if _, exists := d.cache[ip]; !exists {
				d.cache[ip] = currentHost
			}
		}
	}
}

// Lookup ищет домен по IP
func (d *DNSCache) Lookup(ip string) string {
	d.mu.RLock()
	hostname, ok := d.cache[ip]
	d.mu.RUnlock()

	if ok {
		return hostname
	}

	// Пробуем reverse DNS lookup
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
















