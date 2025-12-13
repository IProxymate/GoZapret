package strategy

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"

	"github.com/IProxymate/GoZapret/internal/domain"
)

// VersionReader читает версию из файла service.bat
type VersionReader struct{}

// NewVersionReader создает новый читатель версий
func NewVersionReader() *VersionReader {
	return &VersionReader{}
}

// ReadFromServiceBat читает версию из файла service.bat
func (v *VersionReader) ReadFromServiceBat(assetsPath domain.AssetsPath) (string, error) {
	serviceBatPath := filepath.Join(assetsPath.String(), "service.bat")

	file, err := os.Open(serviceBatPath)
	if err != nil {
		slog.Warn("Не удалось открыть service.bat для чтения версии", "path", serviceBatPath, "error", err)
		return "", err
	}
	defer file.Close()

	// Регулярное выражение для поиска строки с версией
	versionRegex := regexp.MustCompile(`set\s+"LOCAL_VERSION=([^"]+)"`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		matches := versionRegex.FindStringSubmatch(line)
		if len(matches) > 1 {
			version := matches[1]
			slog.Info("Версия Zapret найдена в service.bat", "version", version)
			return version, nil
		}
	}

	if err := scanner.Err(); err != nil {
		slog.Error("Ошибка чтения service.bat", "error", err)
		return "", err
	}

	slog.Warn("Версия не найдена в service.bat")
	return "", fmt.Errorf("версия не найдена в service.bat")
}
