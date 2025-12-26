package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nwaples/rardecode"
)

// findCommonRoot находит общую корневую директорию для списка путей
func findCommonRoot(paths []string) string {
	if len(paths) == 0 {
		return ""
	}

	// Преобразуем все пути к стандартному формату с "/" в качестве разделителя
	var normalizedPaths []string
	for _, path := range paths {
		normalized := strings.ReplaceAll(path, "\\", "/")
		normalizedPaths = append(normalizedPaths, normalized)
	}

	// Разбиваем первый путь на части
	parts := strings.Split(normalizedPaths[0], "/")
	if len(parts) == 0 || parts[0] == "" {
		return ""
	}

	// Ищем общий префикс
	commonRoot := parts[0]

	// Проверяем, все ли файлы находятся в одной и той же корневой директории
	for _, path := range normalizedPaths {
		pathParts := strings.Split(path, "/")
		if len(pathParts) == 0 {
			continue
		}

		if pathParts[0] != commonRoot {
			// Если есть расхождение, возвращаем пустую строку (файлы в корне архива)
			return ""
		}
	}

	return commonRoot
}

// ExtractRar распаковывает RAR-архив в указанную директорию и возвращает путь к извлеченным файлам
func ExtractRar(archivePath, extractDir string) (string, error) {
	archive, err := rardecode.OpenReader(archivePath, "")
	if err != nil {
		return "", err
	}
	defer archive.Close()

	// Сначала определяем корневую директорию, пройдясь по всем файлам в архиве
	var allPaths []string
	for {
		header, err := archive.Next()
		if err != nil {
			break
		}
		allPaths = append(allPaths, header.Name)
	}

	// Определяем общую корневую директорию
	rootDir := findCommonRoot(allPaths)

	// Открываем архив заново для извлечения файлов
	archive, err = rardecode.OpenReader(archivePath, "")
	if err != nil {
		return "", err
	}
	defer archive.Close()

	// Извлекаем файлы и директории
	for {
		header, err := archive.Next()
		if err != nil {
			break
		}

		// Проверяем, что имя файла не содержит ".." для безопасности
		if strings.Contains(header.Name, "..") {
			continue
		}

		extractedFilePath := filepath.Join(extractDir, header.Name)

		if header.IsDir {
			// Создаём директорию
			err = os.MkdirAll(extractedFilePath, os.ModePerm)
			if err != nil {
				return "", err
			}
		} else {
			// Создаём директории при необходимости
			err = os.MkdirAll(filepath.Dir(extractedFilePath), os.ModePerm)
			if err != nil {
				return "", err
			}

			// Пытаемся создать файл с повторными попытками для заблокированных файлов
			var archiveFile *os.File
			maxRetries := 3
			for retry := 0; retry < maxRetries; retry++ {
				archiveFile, err = os.Create(extractedFilePath)
				if err == nil {
					break
				}
				// Если файл заблокирован, ждём и пробуем снова
				if retry < maxRetries-1 {
					time.Sleep(time.Duration(retry+1) * time.Second)
				}
			}
			if err != nil {
				return "", fmt.Errorf("не удалось создать файл %s: %w", extractedFilePath, err)
			}

			// Копируем содержимое
			_, err = io.Copy(archiveFile, archive)
			if err != nil {
				archiveFile.Close()
				return "", err
			}

			archiveFile.Close()
		}
	}

	// Возвращаем путь корневой директории архива
	return filepath.Join(extractDir, rootDir), nil
}
