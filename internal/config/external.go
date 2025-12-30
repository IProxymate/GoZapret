package config

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"
)

//go:embed external.json
var externalConfigJSON []byte

// CheckDomain представляет домен для проверки
type CheckDomain struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Ping string `json:"ping"`
}

// DNSServer представляет DNS сервер для ping-проверки
type DNSServer struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
}

// DPISuiteTarget представляет цель для DPI проверки
type DPISuiteTarget struct {
	ID       string `json:"id"`
	Provider string `json:"provider"`
	URL      string `json:"url"`
	Times    int    `json:"times,omitempty"` // количество повторений (по умолчанию 1)
}

// ExternalConfig содержит внешние URL и репозитории
type ExternalConfig struct {
	Repositories struct {
		GoZapret        string `json:"gozapret"`
		ZapretResources string `json:"zapret_resources"`
		OriginalZapret  string `json:"original_zapret"`
	} `json:"repositories"`
	URLs struct {
		IpsetList   string `json:"ipset_list"`
		TargetsFile string `json:"targets_file"`
	} `json:"urls"`
	CheckDomains []CheckDomain    `json:"check_domains"`
	DNSServers   []DNSServer      `json:"dns_servers"`
	DPISuite     []DPISuiteTarget `json:"dpi_suite"`
}

var (
	externalConfig     *ExternalConfig
	externalConfigOnce sync.Once
)

// GetExternalConfig возвращает внешнюю конфигурацию (singleton)
func GetExternalConfig() *ExternalConfig {
	externalConfigOnce.Do(func() {
		externalConfig = &ExternalConfig{}
		if err := json.Unmarshal(externalConfigJSON, externalConfig); err != nil {
			panic(fmt.Sprintf("ошибка парсинга external.json: %v", err))
		}
	})
	return externalConfig
}

// GoZapretRepo возвращает репозиторий GoZapret
func (c *ExternalConfig) GoZapretRepo() string {
	return c.Repositories.GoZapret
}

// ZapretResourcesRepo возвращает репозиторий ресурсов zapret
func (c *ExternalConfig) ZapretResourcesRepo() string {
	return c.Repositories.ZapretResources
}

// GoZapretReleasesURL возвращает URL для загрузки релизов GoZapret
func (c *ExternalConfig) GoZapretReleasesURL() string {
	return fmt.Sprintf("https://github.com/%s/releases/download/v{{.Version}}/GoZapret{{.Ext}}", c.Repositories.GoZapret)
}

// GoZapretAPIURL возвращает API URL для проверки обновлений GoZapret
func (c *ExternalConfig) GoZapretAPIURL() string {
	return fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", c.Repositories.GoZapret)
}

// ZapretResourcesAPIURL возвращает API URL для проверки обновлений ресурсов
func (c *ExternalConfig) ZapretResourcesAPIURL() string {
	return fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", c.Repositories.ZapretResources)
}

// GoZapretGitHubURL возвращает URL репозитория GoZapret
func (c *ExternalConfig) GoZapretGitHubURL() string {
	return fmt.Sprintf("https://github.com/%s", c.Repositories.GoZapret)
}

// GoZapretIssuesURL возвращает URL для issues
func (c *ExternalConfig) GoZapretIssuesURL() string {
	return fmt.Sprintf("https://github.com/%s/issues", c.Repositories.GoZapret)
}

// ZapretResourcesGitHubURL возвращает URL репозитория ресурсов
func (c *ExternalConfig) ZapretResourcesGitHubURL() string {
	return fmt.Sprintf("https://github.com/%s", c.Repositories.ZapretResources)
}

// OriginalZapretGitHubURL возвращает URL оригинального zapret
func (c *ExternalConfig) OriginalZapretGitHubURL() string {
	return fmt.Sprintf("https://github.com/%s", c.Repositories.OriginalZapret)
}

// IpsetListURL возвращает URL для загрузки списка IPset
func (c *ExternalConfig) IpsetListURL() string {
	return c.URLs.IpsetList
}

// TargetsFileURL возвращает URL для загрузки файла targets.txt
func (c *ExternalConfig) TargetsFileURL() string {
	return c.URLs.TargetsFile
}

// DownloadURL возвращает URL для скачивания релиза
func (c *ExternalConfig) DownloadURL(version string) string {
	return fmt.Sprintf("https://github.com/%s/releases/download/v%s/GoZapret.exe", c.Repositories.GoZapret, version)
}

// GetDPISuiteTargets возвращает развернутый список DPI целей с учетом повторений
func (c *ExternalConfig) GetDPISuiteTargets() []DPISuiteTarget {
	var targets []DPISuiteTarget
	for _, t := range c.DPISuite {
		times := t.Times
		if times < 1 {
			times = 1
		}
		for i := 0; i < times; i++ {
			target := t
			if times > 1 {
				target.ID = fmt.Sprintf("%s@%d", t.ID, i)
			}
			targets = append(targets, target)
		}
	}
	return targets
}
