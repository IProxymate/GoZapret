package services

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/IProxymate/GoZapret/internal/config"
	"github.com/IProxymate/GoZapret/internal/domain"
	"github.com/IProxymate/GoZapret/internal/utils"

	"golang.org/x/sys/windows"
)

// ProcessManager управляет процессами winws
type ProcessManager struct {
	currentProcess *domain.ProcessInfo
	workingDir     string // Постоянная рабочая директория
	adminChecker   *AdminChecker
	configManager  *config.Manager
}

// NewProcessManager создает новый менеджер процессов
func NewProcessManager(adminChecker *AdminChecker, configManager *config.Manager) *ProcessManager {
	return &ProcessManager{
		adminChecker:  adminChecker,
		configManager: configManager,
	}
}

// StartStrategy запускает стратегию
func (pm *ProcessManager) StartStrategy(strategy *domain.Strategy, assetsPath domain.AssetsPath, gameFilterEnabled bool) error {
	slog.Info("Начало запуска стратегии", "strategy", strategy.Name, "gameFilter", gameFilterEnabled)

	if err := strategy.Validate(); err != nil {
		slog.Error("Ошибка валидации стратегии", "strategy", strategy.Name, "error", err)
		return err
	}

	if err := assetsPath.Validate(); err != nil {
		slog.Error("Ошибка валидации пути к ресурсам", "path", assetsPath, "error", err)
		return err
	}

	if !pm.adminChecker.IsAdmin() {
		slog.Warn("Нет прав администратора для запуска стратегии")
		return domain.ErrAdminRightsRequired
	}

	if pm.IsRunning() {
		slog.Warn("Попытка запуска стратегии при уже запущенном процессе")
		return domain.ErrProcessAlreadyRunning
	}

	// Получаем рабочую директорию
	workDir := pm.getWorkingBinDir()
	if workDir == "" {
		slog.Error("Рабочая директория не подготовлена")
		return fmt.Errorf("рабочая директория не подготовлена, необходимо сначала задать путь к ресурсам")
	}

	// Парсим bat файл
	slog.Debug("Парсинг bat файла", "file", strategy.BatFile)
	args, err := pm.parseBatFile(strategy.BatFile, assetsPath, gameFilterEnabled)
	if err != nil {
		slog.Error("Ошибка парсинга bat файла", "file", strategy.BatFile, "error", err)
		return err
	}

	// Запускаем процесс
	cmd, err := pm.startProcess(args, workDir)
	if err != nil {
		slog.Error("Ошибка запуска процесса", "error", err)
		return err
	}

	// Сохраняем информацию о процессе
	pm.currentProcess = &domain.ProcessInfo{
		PID:       domain.ProcessID(cmd.Process.Pid),
		Strategy:  strategy.Name,
		StartedAt: time.Now(),
		Status:    domain.ProcessStatusStarting,
		Command:   cmd,
	}

	// Запускаем мониторинг процесса
	go pm.monitorProcess(cmd)

	slog.Debug("Стратегия успешно запущена", "strategy", strategy.Name, "pid", cmd.Process.Pid)
	return nil
}

// RestartStrategy перезапускает стратегию с проверкой "Если запущено"
func (pm *ProcessManager) RestartStrategy(strategy *domain.Strategy, assetsPath domain.AssetsPath, gameFilterEnabled bool) error {
	// Проверяем, запущен ли процесс
	if pm.IsRunning() {
		// Если запущен, останавливаем его
		if err := pm.StopProcess(); err != nil {
			return fmt.Errorf("ошибка остановки текущего процесса: %w", err)
		}
		// Ждем короткое время, чтобы убедиться, что процесс полностью остановлен
		time.Sleep(500 * time.Millisecond)
	}

	// Запускаем новую стратегию
	if err := pm.StartStrategy(strategy, assetsPath, gameFilterEnabled); err != nil {
		return fmt.Errorf("ошибка запуска стратегии: %w", err)
	}

	return nil
}

// StopProcess останавливает текущий процесс
func (pm *ProcessManager) StopProcess() error {
	if !pm.IsRunning() {
		if pm.IsWinwsProcessRunning() {
			if err := pm.killAllWinwsProcesses(); err != nil {
				return err
			}
			pm.cleanupTempDir()
			return nil
		}
		return domain.ErrProcessNotRunning
	}

	var cmd *exec.Cmd
	if pm.currentProcess != nil {
		cmd = pm.currentProcess.Command
		pm.currentProcess.Status = domain.ProcessStatusStopping
	}

	if cmd != nil {
		if err := pm.gracefulShutdown(cmd); err != nil {
			if killErr := pm.forceKill(cmd); killErr != nil {
				if finalErr := pm.killAllWinwsProcesses(); finalErr != nil {
					return finalErr
				}
			}
		}
	} else {
		if err := pm.killAllWinwsProcesses(); err != nil {
			return err
		}
	}

	if pm.currentProcess != nil {
		pm.currentProcess.Status = domain.ProcessStatusStopped
		pm.currentProcess = nil
	}

	pm.cleanupTempDir()
	return nil
}

// IsRunning проверяет, запущен ли процесс
func (pm *ProcessManager) IsRunning() bool {
	if pm.currentProcess == nil {
		// Если внутреннее состояние отсутствует, проверяем наличие процесса winws.exe в системе
		isRunning := pm.IsWinwsProcessRunning()
		slog.Debug("Результат проверки процесса winws.exe в системе", "running", isRunning)
		return isRunning
	}

	slog.Debug("Проверка статуса процесса", "status", pm.currentProcess.Status)

	if !pm.currentProcess.Status.IsActive() {
		slog.Debug("Статус процесса не активен")
		return false
	}

	// В Windows проверяем процесс через системный вызов OpenProcess
	if pm.currentProcess.Command != nil && pm.currentProcess.Command.Process != nil {
		pid := pm.currentProcess.Command.Process.Pid
		slog.Debug("Проверка процесса через OpenProcess", "pid", pid)

		// Используем OpenProcess для проверки существования процесса
		handle, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION, false, uint32(pid))
		if err != nil {
			slog.Debug("Не удалось открыть процесс", "pid", pid, "error", err)
			pm.currentProcess.Status = domain.ProcessStatusStopped
			return false
		}
		defer windows.CloseHandle(handle)

		// Получаем статус выхода процесса
		var exitCode uint32
		err = windows.GetExitCodeProcess(handle, &exitCode)
		if err != nil {
			slog.Debug("Не удалось получить статус выхода процесса", "pid", pid, "error", err)
			pm.currentProcess.Status = domain.ProcessStatusStopped
			return false
		}

		// Если exitCode != STILL_ACTIVE, процесс завершен
		if exitCode != 259 { // STILL_ACTIVE = 259
			slog.Debug("Процесс завершен", "exitCode", exitCode)
			pm.currentProcess.Status = domain.ProcessStatusStopped
			return false
		}

		slog.Debug("Процесс активен", "pid", pid)

		// Убедимся, что статус running, если процесс действительно запущен
		if pm.currentProcess.Status == domain.ProcessStatusStarting {
			pm.currentProcess.Status = domain.ProcessStatusRunning
		}
	}

	isActive := pm.currentProcess.Status.IsActive()
	slog.Debug("Результат проверки процесса", "active", isActive)
	return isActive
}

// GetCurrentProcess возвращает информацию о текущем процессе
func (pm *ProcessManager) GetCurrentProcess() *domain.ProcessInfo {
	if pm.currentProcess != nil && pm.currentProcess.Command != nil && pm.currentProcess.Command.Process != nil {
		// Проверяем существование процесса через Windows API
		pid := pm.currentProcess.Command.Process.Pid
		handle, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION, false, uint32(pid))
		if err != nil {
			pm.currentProcess.Status = domain.ProcessStatusStopped
		} else {
			defer windows.CloseHandle(handle)

			// Получаем статус выхода процесса
			var exitCode uint32
			err = windows.GetExitCodeProcess(handle, &exitCode)
			if err != nil || exitCode != 259 { // STILL_ACTIVE = 259
				pm.currentProcess.Status = domain.ProcessStatusStopped
			} else {
				// Убедимся, что статус running, если процесс действительно запущен
				if pm.currentProcess.Status == domain.ProcessStatusStarting {
					pm.currentProcess.Status = domain.ProcessStatusRunning
				}
			}
		}
	} else if pm.currentProcess == nil {
		// Если внутреннее состояние отсутствует, но процесс запущен в системе, создаем минимальную информацию о процессе
		if pm.IsWinwsProcessRunning() {
			slog.Debug("Процесс winws.exe запущен в системе, но отсутствует внутреннее состояние")
			// В этом случае мы не можем создать полную информацию о процессе без PID из Command
			// Но можем вернуть информацию о том, что процесс запущен где-то в системе
			return &domain.ProcessInfo{
				PID:       0, // Неизвестный PID
				Strategy:  "",
				StartedAt: time.Time{}, // Нулевое время
				Status:    domain.ProcessStatusRunning,
				Command:   nil,
			}
		}
	}
	return pm.currentProcess
}

// IsWinwsProcessRunning проверяет, запущен ли процесс winws.exe в системе
func (pm *ProcessManager) IsWinwsProcessRunning() bool {
	output, err := utils.OutputHidden("tasklist", "/FI", "IMAGENAME eq winws.exe")
	if err != nil {
		return false
	}

	return strings.Contains(string(output), "winws.exe")
}

// parseBatFile парсит bat файл и извлекает аргументы
func (pm *ProcessManager) parseBatFile(batFile domain.BatFileName, assetsPath domain.AssetsPath, gameFilterEnabled bool) ([]string, error) {
	batPath := filepath.Join(assetsPath.String(), batFile.String())

	data, err := os.ReadFile(batPath)
	if err != nil {
		return nil, domain.ErrBatFileNotFound
	}

	args, err := pm.extractWinwsArgs(string(data), gameFilterEnabled)
	if err != nil {
		return nil, domain.ErrBatFileParseFailed
	}

	return args, nil
}

// extractWinwsArgs извлекает аргументы winws из содержимого bat файла
func (pm *ProcessManager) extractWinwsArgs(content string, gameFilterEnabled bool) ([]string, error) {
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
			argPart := pm.extractArgsFromLine(line)
			if argPart != "" {
				commandLines = append(commandLines, argPart)
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

	return pm.parseWinwsArgs(fullCommand, gameFilterEnabled), nil
}

// extractArgsFromLine извлекает аргументы из строки с winws.exe
func (pm *ProcessManager) extractArgsFromLine(line string) string {
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

// parseWinwsArgs парсит аргументы winws
func (pm *ProcessManager) parseWinwsArgs(argsStr string, gameFilterEnabled bool) []string {
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

// getWorkingBinDir возвращает путь к директории bin в рабочей директории
func (pm *ProcessManager) getWorkingBinDir() string {
	if pm.configManager == nil {
		return ""
	}

	workingDir := pm.configManager.GetWorkingDir()
	if workingDir == "" {
		return ""
	}

	return filepath.Join(workingDir, "bin")
}

// startProcess запускает процесс winws
func (pm *ProcessManager) startProcess(args []string, workDir string) (*exec.Cmd, error) {
	winwsPath := filepath.Join(workDir, "winws.exe")

	pm.replacePathsInArgs(args, workDir)

	// Добавляем аргументы для пользовательских списков доменов
	args = pm.addCustomHostlistArgs(args)

	// Логируем полную команду запуска
	slog.Debug("Запуск winws.exe", "executable", winwsPath, "args", args)

	ctx := context.Background()
	cmd := utils.NewHiddenCommandContext(ctx, winwsPath, args...)

	if err := cmd.Start(); err != nil {
		return nil, domain.ErrProcessStartFailed
	}

	return cmd, nil
}

// addCustomHostlistArgs добавляет аргументы для пользовательских списков доменов
// --hostlist добавляется после первого существующего --hostlist в каждом профиле
// --hostlist-exclude добавляется ОДИН РАЗ в каждый профиль (работает глобально для всех --hostlist в профиле)
func (pm *ProcessManager) addCustomHostlistArgs(args []string) []string {
	if pm.configManager == nil {
		return args
	}

	extraListPath := pm.configManager.GetExtraListPath()
	excludeListPath := pm.configManager.GetExcludeListPath()

	// Проверяем, существуют ли файлы и не пусты ли они
	extraExists := pm.isFileNotEmpty(extraListPath)
	excludeExists := pm.isFileNotEmpty(excludeListPath)

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

		// Проверяем, является ли это первым --hostlist= в профиле (не --hostlist-exclude и не --hostlist-domains)
		if !addedInCurrentProfile &&
			strings.HasPrefix(arg, "--hostlist=") &&
			!strings.HasPrefix(arg, "--hostlist-exclude=") &&
			!strings.HasPrefix(arg, "--hostlist-domains=") {

			// Добавляем наш --hostlist для включенных доменов, если файл существует
			if extraExists {
				newArgs = append(newArgs, "--hostlist="+extraListPath)
				slog.Debug("Добавлен --hostlist в профиль", "path", extraListPath, "position", i)
			}

			// Добавляем --hostlist-exclude для исключенных доменов, если файл существует
			// Он работает глобально для всех --hostlist в профиле
			if excludeExists {
				newArgs = append(newArgs, "--hostlist-exclude="+excludeListPath)
				slog.Debug("Добавлен --hostlist-exclude в профиль", "path", excludeListPath, "position", i)
			}

			// Отмечаем, что в этом профиле уже добавили наши аргументы
			addedInCurrentProfile = true
		}
	}

	slog.Info("Пользовательские списки доменов интегрированы в команду",
		"extraList", extraExists, "excludeList", excludeExists)

	return newArgs
}

// isFileNotEmpty проверяет, существует ли файл и не пуст ли он
func (pm *ProcessManager) isFileNotEmpty(filePath string) bool {
	info, err := os.Stat(filePath)
	if err != nil {
		return false
	}
	return info.Size() > 0
}

// replacePathsInArgs заменяет относительные пути и переменные на абсолютные в аргументах
func (pm *ProcessManager) replacePathsInArgs(args []string, workDir string) {
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

// monitorProcess мониторит процесс в отдельной горутине
func (pm *ProcessManager) monitorProcess(cmd *exec.Cmd) {
	err := cmd.Wait()

	if pm.currentProcess != nil {
		if err != nil {
			pm.currentProcess.Status = domain.ProcessStatusFailed
		} else {
			pm.currentProcess.Status = domain.ProcessStatusStopped
		}
	}

	pm.cleanupTempDir()
}

// gracefulShutdown пытается корректно завершить процесс
func (pm *ProcessManager) gracefulShutdown(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}

	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-done:
		return nil
	case <-time.After(3 * time.Second): // Уменьшили таймаут до 3 секунд
		return fmt.Errorf("таймаут ожидания завершения процесса")
	}
}

// forceKill принудительно убивает процесс
func (pm *ProcessManager) forceKill(cmd *exec.Cmd) error {
	if cmd.Process != nil {
		return cmd.Process.Kill()
	}
	return nil
}

// killAllWinwsProcesses убивает все процессы winws.exe в системе
func (pm *ProcessManager) killAllWinwsProcesses() error {
	err := utils.RunHidden("taskkill", "/IM", "winws.exe", "/F")
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() == 128 || exitError.ExitCode() == 129 {
				return nil
			}
		}
		// Возвращаем nil даже при ошибке, так как процесс мог уже завершиться
		return nil
	}

	return nil
}

// cleanupTempDir удаляет временную директорию
func (pm *ProcessManager) cleanupTempDir() {
	if pm.workingDir != "" {
		// Не удаляем постоянную рабочую директорию
		// os.RemoveAll(pm.workingDir)
		// pm.workingDir = ""
	}
}
