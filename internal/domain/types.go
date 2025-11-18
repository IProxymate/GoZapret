package domain

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AssetsPath представляет путь к файлам Zapret
type AssetsPath string

// Validate проверяет корректность пути к ресурсам
func (p AssetsPath) Validate() error {
	if p == "" {
		return fmt.Errorf("путь к ресурсам не может быть пустым")
	}

	path := string(p)
	if !filepath.IsAbs(path) {
		return fmt.Errorf("путь должен быть абсолютным: %s", path)
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("путь недоступен: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("путь должен указывать на директорию: %s", path)
	}

	return nil
}

// String возвращает строковое представление пути
func (p AssetsPath) String() string {
	return string(p)
}

// HasWinws проверяет наличие winws.exe в директории
func (p AssetsPath) HasWinws() bool {
	winwsPath := filepath.Join(string(p), "winws.exe")
	if _, err := os.Stat(winwsPath); err == nil {
		return true
	}

	// Проверяем в подпапке bin
	binWinwsPath := filepath.Join(string(p), "bin", "winws.exe")
	_, err := os.Stat(binWinwsPath)
	return err == nil
}

// StrategyName представляет имя стратегии
type StrategyName string

// Validate проверяет корректность имени стратегии
func (s StrategyName) Validate() error {
	if s == "" {
		return fmt.Errorf("имя стратегии не может быть пустым")
	}

	name := string(s)
	if strings.ContainsAny(name, "/\\:*?\"<>|") {
		return fmt.Errorf("имя стратегии содержит недопустимые символы: %s", name)
	}

	return nil
}

// String возвращает строковое представление имени
func (s StrategyName) String() string {
	return string(s)
}

// ProcessID представляет идентификатор процесса
type ProcessID int

// IsValid проверяет корректность ID процесса
func (p ProcessID) IsValid() bool {
	return p > 0
}

// BatFileName представляет имя bat файла
type BatFileName string

// Validate проверяет корректность имени bat файла
func (b BatFileName) Validate() error {
	if b == "" {
		return fmt.Errorf("имя bat файла не может быть пустым")
	}

	name := string(b)
	if !strings.HasSuffix(strings.ToLower(name), ".bat") {
		return fmt.Errorf("файл должен иметь расширение .bat: %s", name)
	}

	return nil
}

// String возвращает строковое представление имени файла
func (b BatFileName) String() string {
	return string(b)
}

// WithoutExtension возвращает имя файла без расширения
func (b BatFileName) WithoutExtension() string {
	return strings.TrimSuffix(string(b), ".bat")
}
