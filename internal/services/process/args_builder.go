package process

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/IProxymate/GoZapret/internal/domain"
)

// ConfigManager интерфейс для работы с конфигурацией
type ConfigManager interface {
	GetExtraListPath() string
	GetExcludeListPath() string
	GetCustomIpsetPath() string
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
func (b *ArgsBuilder) Build(parsedArgs []string, workDir string, gameFilterMode domain.GameFilterMode) []string {
	rawArgs := strings.Join(parsedArgs, " ")

	// Определяем формат bat-файла (новый или старый)
	isNewFormat := b.detectNewFormat(rawArgs)
	slog.Debug("Определён формат bat-файла", "newFormat", isNewFormat, "gameFilterMode", gameFilterMode)

	// Заменяем переменные GameFilter
	argsStr := b.replaceGameFilterVars(rawArgs, gameFilterMode, isNewFormat)

	// Парсим аргументы с учётом кавычек
	args := b.parseQuotedArgs(argsStr)

	// Разбиваем --key=value на отдельные --key и value,
	// чтобы воспроизвести поведение CMD.exe, который обрабатывает '=' как разделитель
	args = b.splitEqualsArgs(args)

	// Заменяем пути %BIN% и %LISTS%
	b.replacePathsInArgs(args, workDir)

	// Инжектируем пользовательские списки GoZapret в аргументы (для обоих форматов).
	// Используем пути напрямую из configManager, без копирования файлов.
	args = b.injectUserLists(args)

	return args
}

// detectNewFormat определяет, использует ли bat-файл новый формат (v1.9.7+).
// Новый формат содержит %GameFilterTCP% / %GameFilterUDP% и/или ссылки на user-списки.
func (b *ArgsBuilder) detectNewFormat(argsStr string) bool {
	return strings.Contains(argsStr, "%GameFilterTCP%") ||
		strings.Contains(argsStr, "%GameFilterUDP%") ||
		strings.Contains(argsStr, domain.ListExtraFile) ||
		strings.Contains(argsStr, domain.ListExcludeFile)
}

// replaceGameFilterVars заменяет переменные GameFilter на значения
func (b *ArgsBuilder) replaceGameFilterVars(argsStr string, mode domain.GameFilterMode, isNewFormat bool) string {
	if isNewFormat {
		// Новый формат: раздельные переменные TCP и UDP
		argsStr = strings.ReplaceAll(argsStr, "%GameFilterTCP%", mode.TCPValue())
		argsStr = strings.ReplaceAll(argsStr, "%GameFilterUDP%", mode.UDPValue())
		slog.Debug("Замена GameFilter (новый формат)", "tcp", mode.TCPValue(), "udp", mode.UDPValue())
	}

	// Старый формат (или fallback): единая переменная %GameFilter%
	argsStr = strings.ReplaceAll(argsStr, "%GameFilter%", mode.LegacyValue())

	return argsStr
}

// parseQuotedArgs парсит строку аргументов с учётом кавычек
func (b *ArgsBuilder) parseQuotedArgs(argsStr string) []string {
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

	return args
}

// splitEqualsArgs разбивает аргументы формата --key=value на отдельные --key и value.
// CMD.exe обрабатывает '=' как разделитель, поэтому bat-файл --dpi-desync=fake
// превращается в два аргумента: --dpi-desync и fake.
// Без этого winws.exe может некорректно обрабатывать некоторые параметры.
func (b *ArgsBuilder) splitEqualsArgs(args []string) []string {
	result := make([]string, 0, len(args)*2)

	for _, arg := range args {
		// Обрабатываем только аргументы, начинающиеся с --
		if strings.HasPrefix(arg, "--") && strings.Contains(arg, "=") {
			eqIdx := strings.Index(arg, "=")
			key := arg[:eqIdx]
			value := arg[eqIdx+1:]

			result = append(result, key)
			if value != "" {
				result = append(result, value)
			}
		} else {
			result = append(result, arg)
		}
	}

	return result
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

		args[i] = arg
	}
}

// injectUserLists инжектирует пользовательские списки GoZapret в аргументы.
// Использует пути к файлам напрямую из configManager (без копирования).
// Работает для обоих форматов bat-файлов (старого и нового).
// Аргументы уже разбиты splitEqualsArgs, поэтому --hostlist и значение — отдельные элементы.
//
// Инжекция происходит ТОЛЬКО в "общие" профили — те, которые уже содержат ссылку
// на list-general.txt или list-general-user.txt. Специализированные профили
// (list-google.txt, hostlist-domains и т.д.) не затрагиваются.
func (b *ArgsBuilder) injectUserLists(args []string) []string {
	if b.configManager == nil {
		return args
	}

	extraListPath := b.configManager.GetExtraListPath()
	excludeListPath := b.configManager.GetExcludeListPath()

	// Проверяем, существуют ли файлы и не пусты ли они
	extraExists := b.isFileNotEmpty(extraListPath)
	excludeExists := b.isFileNotEmpty(excludeListPath)

	if !extraExists && !excludeExists {
		slog.Debug("Пользовательские списки GoZapret пусты — инжекция не требуется")
		return args
	}

	// Разбиваем аргументы на профили (секции между --new)
	profiles := b.splitProfiles(args)

	// Собираем результат
	newArgs := make([]string, 0, len(args)+30)

	for i, profile := range profiles {
		if i > 0 {
			newArgs = append(newArgs, "--new")
		}

		// Инжектируем только в "общие" профили (содержащие list-general)
		if b.isGeneralProfile(profile) {
			profile = b.injectIntoProfile(profile, extraListPath, excludeListPath, extraExists, excludeExists)
		}

		newArgs = append(newArgs, profile...)
	}

	if extraExists || excludeExists {
		slog.Info("Пользовательские списки GoZapret инжектированы",
			"extraList", extraExists, "excludeList", excludeExists)
	}

	return newArgs
}

// splitProfiles разбивает аргументы на профили (секции, разделённые --new)
func (b *ArgsBuilder) splitProfiles(args []string) [][]string {
	var profiles [][]string
	var current []string

	for _, arg := range args {
		if arg == "--new" {
			profiles = append(profiles, current)
			current = nil
		} else {
			current = append(current, arg)
		}
	}
	if len(current) > 0 {
		profiles = append(profiles, current)
	}

	return profiles
}

// isGeneralProfile проверяет, является ли профиль "общим" — содержит ли он
// ссылку на list-general.txt или list-general-user.txt.
// Специализированные профили (list-google, hostlist-domains и т.д.) не затрагиваются.
func (b *ArgsBuilder) isGeneralProfile(profile []string) bool {
	for _, arg := range profile {
		// Проверяем, содержит ли значение аргумента "list-general"
		if strings.Contains(filepath.Base(arg), "list-general") {
			return true
		}
	}
	return false
}

// injectIntoProfile инжектирует пользовательские списки в один профиль
func (b *ArgsBuilder) injectIntoProfile(profile []string, extraListPath, excludeListPath string, extraExists, excludeExists bool) []string {
	newProfile := make([]string, 0, len(profile)+4)

	addedExtra := false
	addedExclude := false
	pendingInject := ""

	for _, arg := range profile {
		newProfile = append(newProfile, arg)

		// Если ожидаем значение после флага — инжектируем после него
		if pendingInject != "" {
			switch pendingInject {
			case "hostlist":
				if !addedExtra {
					if extraExists {
						newProfile = append(newProfile, "--hostlist", extraListPath)
						slog.Debug("Инжектирован --hostlist GoZapret в профиль", "path", extraListPath)
					}
					if excludeExists && !addedExclude {
						newProfile = append(newProfile, "--hostlist-exclude", excludeListPath)
						addedExclude = true
					}
					addedExtra = true
				}
			case "hostlist-exclude":
				if !addedExclude && excludeExists {
					newProfile = append(newProfile, "--hostlist-exclude", excludeListPath)
					slog.Debug("Инжектирован --hostlist-exclude GoZapret в профиль", "path", excludeListPath)
					addedExclude = true
				}
			}
			pendingInject = ""
			continue
		}

		// Запоминаем флаг — следующий аргумент будет его значением
		if !addedExtra && arg == "--hostlist" {
			pendingInject = "hostlist"
		} else if !addedExclude && arg == "--hostlist-exclude" {
			pendingInject = "hostlist-exclude"
		}
	}

	return newProfile
}

// isFileNotEmpty проверяет, существует ли файл и содержит ли он реальные данные
func (b *ArgsBuilder) isFileNotEmpty(filePath string) bool {
	info, err := os.Stat(filePath)
	if err != nil {
		return false
	}
	return info.Size() > 0
}
