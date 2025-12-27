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

// ExternalConfig содержит внешние URL и репозитории
type ExternalConfig struct {
	Repositories struct {
		GoZapret        string `json:"gozapret"`
		ZapretResources string `json:"zapret_resources"`
		OriginalZapret  string `json:"original_zapret"`
	} `json:"repositories"`
	URLs struct {
		IpsetList string `json:"ipset_list"`
	} `json:"urls"`
	CheckDomains []CheckDomain `json:"check_domains"`
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

// DownloadURL возвращает URL для скачивания релиза
func (c *ExternalConfig) DownloadURL(version string) string {
	return fmt.Sprintf("https://github.com/%s/releases/download/v%s/GoZapret.exe", c.Repositories.GoZapret, version)
}
