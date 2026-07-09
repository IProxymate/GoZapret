package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/IProxymate/GoZapret/internal/domain"
)

// WorkspaceManager управляет рабочей директорией и файлами
type WorkspaceManager struct {
	configDir string
}

// NewWorkspaceManager создаёт новый менеджер рабочего пространства
func NewWorkspaceManager(configDir string) *WorkspaceManager {
	return &WorkspaceManager{
		configDir: configDir,
	}
}

// GetHostsDir возвращает путь к директории hosts
func (w *WorkspaceManager) GetHostsDir() string {
	return filepath.Join(w.configDir, domain.HostsDirName)
}

// GetExtraListPath возвращает путь к файлу list-general-user.txt
func (w *WorkspaceManager) GetExtraListPath() string {
	return filepath.Join(w.GetHostsDir(), domain.ListExtraFile)
}

// GetExcludeListPath возвращает путь к файлу list-exclude-user.txt
func (w *WorkspaceManager) GetExcludeListPath() string {
	return filepath.Join(w.GetHostsDir(), domain.ListExcludeFile)
}

// GetCustomIpsetPath возвращает путь к файлу ipset-include-user.txt
func (w *WorkspaceManager) GetCustomIpsetPath() string {
	return filepath.Join(w.GetHostsDir(), domain.IpsetIncludeUserFile)
}

// EnsureHostsDirectory создает директорию hosts и файлы списков доменов
func (w *WorkspaceManager) EnsureHostsDirectory() error {
	hostsDir := w.GetHostsDir()

	// Создаем директорию hosts
	if err := os.MkdirAll(hostsDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории hosts: %w", err)
	}

	// Миграция: переименовываем старые файлы в новые
	migrations := [][2]string{
		{domain.LegacyListExtraFile, domain.ListExtraFile},
		{domain.LegacyListExcludeFile, domain.ListExcludeFile},
		// ipset-exclude-user.txt переименован в ipset-include-user.txt (по факту это
		// include-список). Более новое имя мигрируем первым, чтобы при наличии обоих
		// файлов сохранилось актуальное содержимое.
		{domain.LegacyIpsetExcludeUserFile, domain.IpsetIncludeUserFile},
		{domain.LegacyIpsetCustomFile, domain.IpsetIncludeUserFile},
	}

	for _, m := range migrations {
		oldPath := filepath.Join(hostsDir, m[0])
		newPath := filepath.Join(hostsDir, m[1])
		if _, err := os.Stat(oldPath); err == nil {
			// Старый файл существует
			if _, err := os.Stat(newPath); os.IsNotExist(err) {
				// Новый файл НЕ существует — переименовываем
				if err := os.Rename(oldPath, newPath); err != nil {
					slog.Warn("Ошибка миграции файла", "old", oldPath, "new", newPath, "error", err)
				} else {
					slog.Info("Файл мигрирован", "old", m[0], "new", m[1])
				}
			}
		}
	}

	// Создаем пустые файлы, если они не существуют
	filesToCreate := []string{
		domain.ListExtraFile,
		domain.ListExcludeFile,
		domain.IpsetIncludeUserFile,
	}

	for _, filename := range filesToCreate {
		filePath := filepath.Join(hostsDir, filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			if err := os.WriteFile(filePath, []byte(""), 0644); err != nil {
				return fmt.Errorf("ошибка создания файла %s: %w", filename, err)
			}
			slog.Debug("Создан файл", "path", filePath)
		}
	}

	return nil
}

// EnsureWorkingDir создаёт и возвращает путь к рабочей директории
func (w *WorkspaceManager) EnsureWorkingDir(currentWorkingDir string) string {
	if currentWorkingDir != "" {
		os.MkdirAll(currentWorkingDir, 0755)
		return currentWorkingDir
	}

	workingDir := filepath.Join(w.configDir, domain.WorkingDirName)
	os.MkdirAll(workingDir, 0755)
	return workingDir
}

// PrepareWorkingDirectory подготавливает рабочую директорию с файлами из ресурсов
func (w *WorkspaceManager) PrepareWorkingDirectory(assetsPath domain.AssetsPath, workingDir string) error {
	slog.Info("Подготовка рабочей директории", "assetsPath", assetsPath, "workingDir", workingDir)

	if assetsPath == "" {
		return fmt.Errorf("путь к ресурсам не задан")
	}

	if err := assetsPath.Validate(); err != nil {
		return fmt.Errorf("невалидный путь к ресурсам: %w", err)
	}

	// Создаем поддиректории
	binDir := filepath.Join(workingDir, domain.BinDirName)
	listsDir := filepath.Join(workingDir, domain.ListsDirName)

	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории bin: %w", err)
	}

	if err := os.MkdirAll(listsDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории lists: %w", err)
	}

	// Копируем файлы
	slog.Debug("Копирование файлов из ресурсов", "from", assetsPath, "to", workingDir)
	if err := w.copyRequiredFiles(assetsPath, binDir, listsDir); err != nil {
		return fmt.Errorf("ошибка копирования файлов: %w", err)
	}

	slog.Info("Рабочая директория успешно подготовлена", "dir", workingDir)
	return nil
}

// copyRequiredFiles копирует файлы из директорий bin и lists
func (w *WorkspaceManager) copyRequiredFiles(assetsPath domain.AssetsPath, binDir, listsDir string) error {
	// Копируем все файлы из bin
	srcBinDir := filepath.Join(assetsPath.String(), domain.BinDirName)
	if err := w.copyAllFilesFromDir(srcBinDir, binDir); err != nil {
		slog.Warn("Ошибка копирования файлов из bin", "error", err)
	}

	// Копируем все файлы из lists
	srcListsDir := filepath.Join(assetsPath.String(), domain.ListsDirName)
	if err := w.copyAllFilesFromDir(srcListsDir, listsDir); err != nil {
		slog.Warn("Ошибка копирования файлов из lists", "error", err)
	}

	return nil
}

// copyAllFilesFromDir копирует все файлы из исходной директории в целевую
func (w *WorkspaceManager) copyAllFilesFromDir(srcDir, dstDir string) error {
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		slog.Debug("Исходная директория не существует", "dir", srcDir)
		return nil
	}

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("ошибка чтения директории %s: %w", srcDir, err)
	}

	copiedCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		srcPath := filepath.Join(srcDir, entry.Name())
		dstPath := filepath.Join(dstDir, entry.Name())

		data, err := os.ReadFile(srcPath)
		if err != nil {
			slog.Warn("Ошибка чтения файла", "file", srcPath, "error", err)
			continue
		}

		if err := os.WriteFile(dstPath, data, 0755); err != nil {
			slog.Warn("Ошибка записи файла", "file", dstPath, "error", err)
			continue
		}

		copiedCount++
	}

	slog.Debug("Файлы скопированы", "from", srcDir, "to", dstDir, "count", copiedCount)
	return nil
}
