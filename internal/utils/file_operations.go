package utils

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExtractZip распаковывает ZIP-архив в указанную директорию и возвращает путь к извлеченным файлам
func ExtractZip(archivePath, extractDir string) (string, error) {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	var rootDir string
	for i, file := range reader.File {
		// Определяем корневую директорию архива (первый элемент)
		if i == 0 {
			parts := strings.Split(strings.ReplaceAll(file.Name, "\\", "/"), "/")
			if len(parts) > 0 && parts[0] != "" {
				rootDir = parts[0]
			}
		}

		extractedFilePath := filepath.Join(extractDir, file.Name)

		if file.FileInfo().IsDir() {
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

			// Открываем файл из архива
			archiveFile, err := file.Open()
			if err != nil {
				return "", err
			}
			defer archiveFile.Close()

			// Создаём файл в файловой системе
			dstFile, err := os.Create(extractedFilePath)
			if err != nil {
				return "", err
			}
			defer dstFile.Close()

			// Копируем содержимое
			_, err = io.Copy(dstFile, archiveFile)
			if err != nil {
				return "", err
			}
		}
	}

	// Возвращаем путь корневой директории архива
	return filepath.Join(extractDir, rootDir), nil
}

