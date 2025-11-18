package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CopyDir копирует директорию и всё её содержимое в другую директорию
func CopyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !srcInfo.IsDir() {
		return fmt.Errorf("source is not a directory: %s", src)
	}

	// Создаём целевую директорию
	err = os.MkdirAll(dst, srcInfo.Mode())
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			err = CopyDir(srcPath, dstPath)
			if err != nil {
				return err
			}
		} else {
			err = CopyFile(srcPath, dstPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// CopyFile копирует файл из одного места в другое
func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(srcFile, dstFile)
	if err != nil {
		return err
	}

	// Копируем права доступа
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, srcInfo.Mode())
}

// RemoveAllContents удаляет всё содержимое директории, оставляя саму директорию
func RemoveAllContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()

	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}

	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}

	return nil
}

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

// GetSubdirectories возвращает список поддиректорий в указанной директории
func GetSubdirectories(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var subdirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			subdirs = append(subdirs, entry.Name())
		}
	}

	return subdirs, nil
}
