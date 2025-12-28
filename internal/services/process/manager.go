package process

import (
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/IProxymate/GoZapret/internal/domain"
)

// AdminChecker интерфейс для проверки прав администратора
type AdminChecker interface {
	IsAdmin() bool
}

// ErrorCallback функция обратного вызова при ошибке процесса
type ErrorCallback func(strategyName string, errorMsg string)

// Manager координирует работу с процессами
type Manager struct {
	currentProcess *domain.ProcessInfo
	executor       *ProcessExecutor
	monitor        *WindowsProcessMonitor
	batParser      *BatParser
	argsBuilder    *ArgsBuilder
	adminChecker   AdminChecker
	configManager  ConfigManager
	errorCallback  ErrorCallback
}

// NewManager создает новый менеджер процессов
func NewManager(
	executor *ProcessExecutor,
	monitor *WindowsProcessMonitor,
	batParser *BatParser,
	argsBuilder *ArgsBuilder,
	adminChecker AdminChecker,
	configManager ConfigManager,
) *Manager {
	return &Manager{
		executor:      executor,
		monitor:       monitor,
		batParser:     batParser,
		argsBuilder:   argsBuilder,
		adminChecker:  adminChecker,
		configManager: configManager,
	}
}

// StartStrategy запускает стратегию
func (m *Manager) StartStrategy(strategy *domain.Strategy, assetsPath domain.AssetsPath, gameFilterEnabled bool) error {
	slog.Info("Начало запуска стратегии", "strategy", strategy.Name, "gameFilter", gameFilterEnabled)

	if err := strategy.Validate(); err != nil {
		slog.Error("Ошибка валидации стратегии", "strategy", strategy.Name, "error", err)
		return err
	}

	if err := assetsPath.Validate(); err != nil {
		slog.Error("Ошибка валидации пути к ресурсам", "path", assetsPath, "error", err)
		return err
	}

	if !m.adminChecker.IsAdmin() {
		slog.Warn("Нет прав администратора для запуска стратегии")
		return domain.ErrAdminRightsRequired
	}

	if m.IsRunning() {
		slog.Warn("Попытка запуска стратегии при уже запущенном процессе")
		return domain.ErrProcessAlreadyRunning
	}

	// Получаем рабочую директорию
	workDir := m.getWorkingBinDir()
	if workDir == "" {
		slog.Error("Рабочая директория не подготовлена")
		return fmt.Errorf("рабочая директория не подготовлена, необходимо сначала задать путь к ресурсам")
	}

	// Парсим bat файл
	slog.Debug("Парсинг bat файла", "file", strategy.BatFile)
	parsedArgs, err := m.batParser.Parse(strategy.BatFile, assetsPath)
	if err != nil {
		slog.Error("Ошибка парсинга bat файла", "file", strategy.BatFile, "error", err)
		return err
	}

	// Строим финальные аргументы
	args := m.argsBuilder.Build(parsedArgs, workDir, gameFilterEnabled)

	// Запускаем процесс
	winwsPath := filepath.Join(workDir, "winws.exe")
	result, err := m.executor.Start(winwsPath, args)
	if err != nil {
		slog.Error("Ошибка запуска процесса", "error", err)
		return err
	}

	// Сохраняем информацию о процессе
	m.currentProcess = &domain.ProcessInfo{
		PID:       domain.ProcessID(result.Cmd.Process.Pid),
		Strategy:  strategy.Name,
		StartedAt: time.Now(),
		Status:    domain.ProcessStatusStarting,
		Command:   result.Cmd,
	}

	// Запускаем мониторинг процесса
	go m.monitorProcess(result)

	slog.Debug("Стратегия успешно запущена", "strategy", strategy.Name, "pid", result.Cmd.Process.Pid)
	return nil
}

// StopProcess останавливает текущий процесс
func (m *Manager) StopProcess() error {
	if !m.IsRunning() {
		if m.monitor.IsWinwsRunning() {
			if err := m.executor.KillAllWinws(); err != nil {
				return err
			}
			return nil
		}
		return domain.ErrProcessNotRunning
	}

	var cmd *exec.Cmd
	if m.currentProcess != nil {
		cmd = m.currentProcess.Command
		m.currentProcess.Status = domain.ProcessStatusStopping
	}

	if cmd != nil {
		if err := m.executor.Stop(cmd); err != nil {
			if finalErr := m.executor.KillAllWinws(); finalErr != nil {
				return finalErr
			}
		}
	} else {
		if err := m.executor.KillAllWinws(); err != nil {
			return err
		}
	}

	if m.currentProcess != nil {
		m.currentProcess.Status = domain.ProcessStatusStopped
		m.currentProcess = nil
	}

	return nil
}

// IsRunning проверяет, запущен ли процесс
func (m *Manager) IsRunning() bool {
	if m.currentProcess == nil {
		isRunning := m.monitor.IsWinwsRunning()
		slog.Debug("Результат проверки процесса winws.exe в системе", "running", isRunning)
		return isRunning
	}

	slog.Debug("Проверка статуса процесса", "status", m.currentProcess.Status)

	if !m.currentProcess.Status.IsActive() {
		slog.Debug("Статус процесса не активен")
		return false
	}

	if m.currentProcess.Command != nil && m.currentProcess.Command.Process != nil {
		pid := m.currentProcess.Command.Process.Pid
		slog.Debug("Проверка процесса через OpenProcess", "pid", pid)

		if !m.monitor.IsProcessRunning(pid) {
			m.currentProcess.Status = domain.ProcessStatusStopped
			return false
		}

		// Убедимся, что статус running, если процесс действительно запущен
		if m.currentProcess.Status == domain.ProcessStatusStarting {
			m.currentProcess.Status = domain.ProcessStatusRunning
		}
	}

	isActive := m.currentProcess.Status.IsActive()
	slog.Debug("Результат проверки процесса", "active", isActive)
	return isActive
}

// GetCurrentProcess возвращает информацию о текущем процессе
func (m *Manager) GetCurrentProcess() *domain.ProcessInfo {
	if m.currentProcess != nil && m.currentProcess.Command != nil && m.currentProcess.Command.Process != nil {
		pid := m.currentProcess.Command.Process.Pid

		if !m.monitor.IsProcessRunning(pid) {
			m.currentProcess.Status = domain.ProcessStatusStopped
		} else {
			// Убедимся, что статус running, если процесс действительно запущен
			if m.currentProcess.Status == domain.ProcessStatusStarting {
				m.currentProcess.Status = domain.ProcessStatusRunning
			}
		}
	} else if m.currentProcess == nil {
		// Если внутреннее состояние отсутствует, но процесс запущен в системе
		if m.monitor.IsWinwsRunning() {
			slog.Debug("Процесс winws.exe запущен в системе, но отсутствует внутреннее состояние")
			return &domain.ProcessInfo{
				PID:       0,
				Strategy:  "",
				StartedAt: time.Time{},
				Status:    domain.ProcessStatusRunning,
				Command:   nil,
			}
		}
	}
	return m.currentProcess
}

// RestartStrategy перезапускает стратегию
func (m *Manager) RestartStrategy(strategy *domain.Strategy, assetsPath domain.AssetsPath, gameFilterEnabled bool) error {
	// Останавливаем процесс, если он запущен (проверяем и внутреннее состояние, и системный процесс)
	if m.IsRunning() || m.monitor.IsWinwsRunning() {
		if err := m.StopProcess(); err != nil && err != domain.ErrProcessNotRunning {
			return fmt.Errorf("ошибка остановки текущего процесса: %w", err)
		}

		// Ждём фактического завершения процесса в системе
		maxWaitTime := 5 * time.Second
		checkInterval := 100 * time.Millisecond
		waited := time.Duration(0)

		for m.monitor.IsWinwsRunning() && waited < maxWaitTime {
			time.Sleep(checkInterval)
			waited += checkInterval
		}

		// Если процесс всё ещё запущен после ожидания, принудительно убиваем
		if m.monitor.IsWinwsRunning() {
			slog.Warn("Процесс не завершился за отведённое время, принудительное завершение")
			if err := m.executor.KillAllWinws(); err != nil {
				return fmt.Errorf("ошибка принудительного завершения процесса: %w", err)
			}
			time.Sleep(500 * time.Millisecond)
		}
	}

	// Сбрасываем внутреннее состояние
	m.currentProcess = nil

	if err := m.StartStrategy(strategy, assetsPath, gameFilterEnabled); err != nil {
		return fmt.Errorf("ошибка запуска стратегии: %w", err)
	}

	return nil
}

// IsWinwsProcessRunning проверяет, запущен ли процесс winws.exe в системе
func (m *Manager) IsWinwsProcessRunning() bool {
	return m.monitor.IsWinwsRunning()
}

// SetErrorCallback устанавливает callback для уведомления об ошибках процесса
func (m *Manager) SetErrorCallback(callback ErrorCallback) {
	m.errorCallback = callback
}

// GetLastError возвращает последнюю ошибку процесса
func (m *Manager) GetLastError() string {
	if m.currentProcess != nil {
		return m.currentProcess.LastError
	}
	return ""
}

// getWorkingBinDir возвращает путь к директории bin в рабочей директории
func (m *Manager) getWorkingBinDir() string {
	if m.configManager == nil {
		return ""
	}

	workingDir := m.configManager.GetWorkingDir()
	if workingDir == "" {
		return ""
	}

	return filepath.Join(workingDir, "bin")
}

// isSystemShutdownError проверяет, является ли ошибка результатом системного завершения
// (выход из системы, перезагрузка, спящий режим)
func (m *Manager) isSystemShutdownError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	// 0x40010004 - STATUS_THREAD_IS_TERMINATING (системное завершение сессии)
	// 0xC000013A - STATUS_CONTROL_C_EXIT (Ctrl+C или SIGTERM)
	// exit status 1 при корректном завершении через taskkill
	systemExitCodes := []string{
		"exit status 0x40010004",
		"exit status 0xc000013a",
		"exit status 1073807364", // десятичное значение 0x40010004
	}

	for _, code := range systemExitCodes {
		if errStr == code {
			return true
		}
	}

	return false
}

// monitorProcess мониторит процесс в отдельной горутине
func (m *Manager) monitorProcess(result *StartResult) {
	err := result.Cmd.Wait()

	if m.currentProcess != nil {
		if err != nil {
			// Проверяем, является ли это системным завершением (не ошибкой)
			if m.isSystemShutdownError(err) {
				slog.Debug("Процесс winws завершён системой (выход из системы/перезагрузка/спящий режим)", "exit_code", err.Error())
				m.currentProcess.Status = domain.ProcessStatusStopped
				return
			}

			m.currentProcess.Status = domain.ProcessStatusFailed

			// Получаем вывод stderr
			stderrOutput := result.Stderr.String()
			if stderrOutput != "" {
				m.currentProcess.LastError = stderrOutput
				slog.Error("Процесс winws завершился с ошибкой", "error", err, "stderr", stderrOutput)
			} else {
				m.currentProcess.LastError = err.Error()
				slog.Error("Процесс winws завершился с ошибкой", "error", err)
			}

			// Вызываем callback для уведомления UI
			if m.errorCallback != nil {
				errorMsg := m.currentProcess.LastError
				strategyName := string(m.currentProcess.Strategy)
				go m.errorCallback(strategyName, errorMsg)
			}
		} else {
			m.currentProcess.Status = domain.ProcessStatusStopped
		}
	}
}
