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
	subnets []*net.IPNet
}

// NewIpsetChecker создает новый чекер ipset
func NewIpsetChecker(workingDir string) *IpsetChecker {
	checker := &IpsetChecker{
		subnets: make([]*net.IPNet, 0),
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

		// Добавляем /32 если нет маски
		if !strings.Contains(line, "/") {
			line += "/32"
		}

		_, subnet, err := net.ParseCIDR(line)
		if err == nil {
			c.subnets = append(c.subnets, subnet)
		}
	}
}

// Contains проверяет, содержится ли IP в ipset
func (c *IpsetChecker) Contains(ip net.IP) bool {
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
	// Загружаем из list-general.txt и других файлов списков
	listFiles := []string{
		filepath.Join(workingDir, "lists", "list-general.txt"),
		filepath.Join(workingDir, "lists", "list-discord.txt"),
		filepath.Join(workingDir, "lists", "list-youtube.txt"),
	}

	for _, path := range listFiles {
		c.loadDomainsFromFile(path)
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

