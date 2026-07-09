package app_monitor

import (
	"bufio"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/IProxymate/GoZapret/internal/domain"
)

// IpsetChecker проверяет IP адреса на вхождение в ipset
type IpsetChecker struct {
	subnets  []*net.IPNet       // Подсети (не /32)
	exactIPs map[string]bool    // Точные IP (/32) — быстрый lookup через map
}

// NewIpsetChecker создает новый чекер ipset
func NewIpsetChecker(workingDir string) *IpsetChecker {
	checker := &IpsetChecker{
		subnets:  make([]*net.IPNet, 0),
		exactIPs: make(map[string]bool),
	}
	checker.loadFromFile(workingDir)
	return checker
}

// loadFromFile загружает подсети из файла ipset-all.txt
func (c *IpsetChecker) loadFromFile(workingDir string) {
	ipsetPath := filepath.Join(workingDir, "lists", "ipset-all.txt")
	file, err := os.Open(ipsetPath)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Если нет маски — это точный IP
		if !strings.Contains(line, "/") {
			ip := net.ParseIP(line)
			if ip != nil {
				c.exactIPs[ip.String()] = true
			}
			continue
		}

		_, subnet, err := net.ParseCIDR(line)
		if err != nil {
			continue
		}

		// /32 для IPv4 и /128 для IPv6 — точные IP, кладём в map
		ones, bits := subnet.Mask.Size()
		if ones == bits {
			c.exactIPs[subnet.IP.String()] = true
		} else {
			c.subnets = append(c.subnets, subnet)
		}
	}
}

// Contains проверяет, содержится ли IP в ipset
func (c *IpsetChecker) Contains(ip net.IP) bool {
	// Быстрая проверка точных IP через map — O(1)
	if c.exactIPs[ip.String()] {
		return true
	}

	// Проверка подсетей — O(n), но n теперь значительно меньше
	for _, subnet := range c.subnets {
		if subnet.Contains(ip) {
			return true
		}
	}
	return false
}

// DomainChecker проверяет домены на вхождение в список
type DomainChecker struct {
	domains  map[string]bool
	patterns []*regexp.Regexp
}

// NewDomainChecker создает новый чекер доменов
func NewDomainChecker(workingDir string) *DomainChecker {
	checker := &DomainChecker{
		domains:  make(map[string]bool),
		patterns: make([]*regexp.Regexp, 0),
	}
	checker.loadFromFiles(workingDir)
	return checker
}

// loadFromFiles загружает домены из файлов
func (c *DomainChecker) loadFromFiles(workingDir string) {
	listsDir := filepath.Join(workingDir, "lists")

	// Загружаем все list-*.txt файлы из директории динамически
	matches, err := filepath.Glob(filepath.Join(listsDir, "list-*.txt"))
	if err == nil {
		for _, path := range matches {
			c.loadDomainsFromFile(path)
		}
	}

	// Загружаем пользовательские домены
	configDir, _ := os.UserConfigDir()
	if configDir != "" {
		extraPath := filepath.Join(configDir, domain.AppName, domain.HostsDirName, "extra-hosts.txt")
		c.loadDomainsFromFile(extraPath)
	}
}

// loadDomainsFromFile загружает домены из одного файла
func (c *DomainChecker) loadDomainsFromFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Поддержка wildcard доменов
		if strings.HasPrefix(line, "*.") {
			pattern := regexp.QuoteMeta(line[2:])
			re, err := regexp.Compile(`(?i)(^|\.)` + pattern + `$`)
			if err == nil {
				c.patterns = append(c.patterns, re)
			}
		} else {
			c.domains[strings.ToLower(line)] = true
		}
	}
}

// Contains проверяет, содержится ли домен в списке
func (c *DomainChecker) Contains(domain string) bool {
	domain = strings.ToLower(domain)

	// Прямое совпадение
	if c.domains[domain] {
		return true
	}

	// Проверка паттернов
	for _, pattern := range c.patterns {
		if pattern.MatchString(domain) {
			return true
		}
	}

	// Проверка родительских доменов
	parts := strings.Split(domain, ".")
	for i := 1; i < len(parts); i++ {
		parent := strings.Join(parts[i:], ".")
		if c.domains[parent] {
			return true
		}
	}

	return false
}

