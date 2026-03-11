package process

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/IProxymate/GoZapret/internal/domain"
)

// BatParser отвечает за парсинг BAT-файлов
type BatParser struct{}

// NewBatParser создает новый парсер BAT-файлов
func NewBatParser() *BatParser {
	return &BatParser{}
}

// Parse парсит bat файл и извлекает аргументы
func (p *BatParser) Parse(batFile domain.BatFileName, assetsPath domain.AssetsPath) ([]string, error) {
	batPath := filepath.Join(assetsPath.String(), batFile.String())

	data, err := os.ReadFile(batPath)
	if err != nil {
		return nil, domain.ErrBatFileNotFound
	}

	args, err := p.extractWinwsArgs(string(data))
	if err != nil {
		return nil, domain.ErrBatFileParseFailed
	}

	return args, nil
}

// extractWinwsArgs извлекает аргументы winws из содержимого bat файла
func (p *BatParser) extractWinwsArgs(content string) ([]string, error) {
	lines := strings.Split(content, "\n")
	foundWinws := false
	var commandLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "::") || strings.HasPrefix(line, "@") || line == "" {
			continue
		}

		if strings.Contains(line, "winws.exe") && strings.HasPrefix(strings.ToLower(line), "start ") {
			foundWinws = true
			argPart := p.extractArgsFromLine(line)
			if argPart != "" {
				// Убираем символ переноса строки bat-файла (^), как и для продолжающих строк
				argPart = strings.TrimSuffix(argPart, "^")
				argPart = strings.TrimSpace(argPart)
				if argPart != "" {
					commandLines = append(commandLines, argPart)
				}
			}
			continue
		}

		if foundWinws {
			cleanLine := strings.TrimSuffix(line, "^")
			cleanLine = strings.TrimSpace(cleanLine)
			if cleanLine != "" {
				commandLines = append(commandLines, cleanLine)
			}

			if !strings.HasSuffix(line, "^") {
				break
			}
		}
	}

	if !foundWinws {
		return nil, fmt.Errorf("команда winws не найдена в bat файле")
	}

	fullCommand := strings.Join(commandLines, " ")
	fullCommand = strings.TrimSpace(fullCommand)

	return strings.Fields(fullCommand), nil
}

// extractArgsFromLine извлекает аргументы из строки с winws.exe
func (p *BatParser) extractArgsFromLine(line string) string {
	winwsIndex := strings.Index(line, "winws.exe\"")
	if winwsIndex != -1 {
		argStart := winwsIndex + len("winws.exe\"")
		if argStart < len(line) {
			return strings.TrimLeft(line[argStart:], " \t")
		}
	}

	winwsIdx := strings.Index(line, "winws.exe")
	if winwsIdx != -1 {
		argStart := winwsIdx + len("winws.exe")
		if argStart < len(line) {
			return strings.TrimLeft(line[argStart:], " \t")
		}
	}

	return ""
}
