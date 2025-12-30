package app_monitor

import (
	"net"
	"time"
)

// NetworkRequest представляет сетевой запрос приложения
type NetworkRequest struct {
	Timestamp   time.Time `json:"timestamp"`
	ProcessName string    `json:"process_name"`
	ProcessPath string    `json:"process_path"`
	ProcessID   uint32    `json:"process_id"`
	Hostname    string    `json:"hostname"`    // DNS имя хоста
	IPAddress   net.IP    `json:"ip_address"`  // IP адрес
	Port        uint16    `json:"port"`        // Порт назначения
	Protocol    string    `json:"protocol"`    // TCP/UDP
	InIpset     bool      `json:"in_ipset"`    // Попадает ли IP в текущий ipset
	InDomains   bool      `json:"in_domains"`  // Попадает ли домен в список доменов
	Subnet      string    `json:"subnet"`      // Подсеть /8 для IP
}

// MonitorSession представляет сессию мониторинга
type MonitorSession struct {
	StartTime   time.Time         `json:"start_time"`
	EndTime     time.Time         `json:"end_time"`
	ProcessPath string            `json:"process_path"`
	ProcessName string            `json:"process_name"`
	Requests    []*NetworkRequest `json:"requests"`
	IsRunning   bool              `json:"is_running"`
}

// IPStats статистика по IP адресам
type IPStats struct {
	Subnet    string   `json:"subnet"`     // Подсеть /8
	Count     int      `json:"count"`      // Количество запросов
	InIpset   bool     `json:"in_ipset"`   // Покрыта ли подсеть текущим ipset
	SampleIPs []string `json:"sample_ips"` // Примеры IP адресов
}

// DomainStats статистика по доменам
type DomainStats struct {
	Domain    string `json:"domain"`
	Count     int    `json:"count"`
	InDomains bool   `json:"in_domains"` // Есть ли домен в списке
}

// MonitorResult результат мониторинга
type MonitorResult struct {
	Session       *MonitorSession `json:"session"`
	IPStatistics  []*IPStats      `json:"ip_statistics"`
	DomainStats   []*DomainStats  `json:"domain_statistics"`
	MissingIPSets []string        `json:"missing_ipsets"` // Подсети, которые нужно добавить
}
















