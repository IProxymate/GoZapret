package process

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// ConfigManager интерфейс для работы с конфигурацией
type ConfigManager interface {
	GetExtraListPath() string
	GetExcludeListPath() string
	GetWorkingDir() string
}

// ArgsBuilder отвечает за построение аргументов командной строки
type ArgsBuilder struct {
	configManager ConfigManager
}

// NewArgsBuilder создает новый построитель аргументов
func NewArgsBuilder(configManager ConfigManager) *ArgsBuilder {
	return &ArgsBuilder{
		configManager: configManager,
	}
}

// Build строит финальные аргументы для запуска winws
func (b *ArgsBuilder) Build(parsedArgs []string, workDir string, gameFilterEnabled bool) []string {
	// Парсим аргументы с учетом Game Filter
	args := b.parseWinwsArgs(strings.Join(parsedArgs, " "), gameFilterEnabled)

	// Заменяем пути
	b.replacePathsInArgs(args, workDir)

	// Добавляем пользовательские списки
	args = b.addCustomHostlistArgs(args)

	return args
}

// parseWinwsArgs парсит аргументы winws
func (b *ArgsBuilder) parseWinwsArgs(argsStr string, gameFilterEnabled bool) []string {
	slog.Debug("Парсинг аргументов winws", "gameFilter", gameFilterEnabled)

	// Сначала заменяем %GameFilter% на нужное значение
	gameFilterValue := "12"
	if gameFilterEnabled {
		gameFilterValue = "1024-65535"
	}
	argsStr = strings.ReplaceAll(argsStr, "%GameFilter%", gameFilterValue)

	var args []string
	var currentArg strings.Builder
	inQuotes := false

	for i := 0; i < len(argsStr); i++ {
		char := argsStr[i]

		if char == '"' {
			inQuotes = !inQuotes
			continue
		}

		if char == ' ' && !inQuotes {
			if currentArg.Len() > 0 {
				args = append(args, currentArg.String())
				currentArg.Reset()
			}
			continue
		}

		currentArg.WriteByte(char)
	}

	if currentArg.Len() > 0 {
		args = append(args, currentArg.String())
	}

	// Применяем Game Filter (удаляем --filter-udp если Game Filter отключен)
	if !gameFilterEnabled {
		filteredArgs := make([]string, 0, len(args))
		skipNext := false

		for i, arg := range args {
			if skipNext {
				skipNext = false
				continue
			}

			if arg == "--filter-udp" {
				if i+1 < len(args) {
					skipNext = true
				}
				continue
			}

			filteredArgs = append(filteredArgs, arg)
		}

		args = filteredArgs
	}

	return args
}

// replacePathsInArgs заменяет относительные пути и переменные на абсолютные в аргументах
func (b *ArgsBuilder) replacePathsInArgs(args []string, workDir string) {
	tempDir := filepath.Dir(workDir)
	binDir := workDir
	listsDir := filepath.Join(tempDir, "lists")

	slog.Debug("Замена путей в аргументах", "binDir", binDir, "listsDir", listsDir)

	for i, arg := range args {
		// Заменяем переменные %BIN% и %LISTS% на пути
		arg = strings.ReplaceAll(arg, "%BIN%", binDir+string(filepath.Separator))
		arg = strings.ReplaceAll(arg, "%LISTS%", listsDir+string(filepath.Separator))

		// %GameFilter% должен быть уже заменен в parseWinwsArgs
		if strings.Contains(arg, "%GameFilter%") {
			slog.Error("Обнаружена незамененная переменная %GameFilter%", "arg", arg)
		}

		args[i] = arg
	}
}

// addCustomHostlistArgs добавляет аргументы для пользовательских списков доменов
func (b *ArgsBuilder) addCustomHostlistArgs(args []string) []string {
	if b.configManager == nil {
		return args
	}

	extraListPath := b.configManager.GetExtraListPath()
	excludeListPath := b.configManager.GetExcludeListPath()

	// Проверяем, существуют ли файлы и не пусты ли они
	extraExists := b.isFileNotEmpty(extraListPath)
	excludeExists := b.isFileNotEmpty(excludeListPath)

	if !extraExists && !excludeExists {
		slog.Debug("Пользовательские списки доменов пусты или не существуют")
		return args
	}

	// Создаем новый массив аргументов
	newArgs := make([]string, 0, len(args)+20)

	// Флаг: добавили ли мы уже наши аргументы в текущий профиль
	addedInCurrentProfile := false

	for i, arg := range args {
		newArgs = append(newArgs, arg)

		// Если встретили --new, сбрасываем флаг для нового профиля
		if arg == "--new" {
			addedInCurrentProfile = false
			continue
		}

		// Проверяем, является ли это первым --hostlist= в профиле
		if !addedInCurrentProfile &&
			strings.HasPrefix(arg, "--hostlist=") &&
			!strings.HasPrefix(arg, "--hostlist-exclude=") &&
			!strings.HasPrefix(arg, "--hostlist-domains=") {

			// Добавляем наш --hostlist для включенных доменов
			if extraExists {
				newArgs = append(newArgs, "--hostlist="+extraListPath)
				slog.Debug("Добавлен --hostlist в профиль", "path", extraListPath, "position", i)
			}

			// Добавляем --hostlist-exclude для исключенных доменов
			if excludeExists {
				newArgs = append(newArgs, "--hostlist-exclude="+excludeListPath)
				slog.Debug("Добавлен --hostlist-exclude в профиль", "path", excludeListPath, "position", i)
			}

			addedInCurrentProfile = true
		}
	}

	slog.Info("Пользовательские списки доменов интегрированы в команду",
		"extraList", extraExists, "excludeList", excludeExists)

	return newArgs
}

// isFileNotEmpty проверяет, существует ли файл и не пуст ли он
func (b *ArgsBuilder) isFileNotEmpty(filePath string) bool {
	info, err := os.Stat(filePath)
	if err != nil {
		return false
	}
	return info.Size() > 0
}
