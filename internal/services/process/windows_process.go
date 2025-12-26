package process

import (
	"log/slog"
	"strings"

	"github.com/IProxymate/GoZapret/internal/utils"
	"golang.org/x/sys/windows"
)

// STILL_ACTIVE константа Windows для активного процесса
const STILL_ACTIVE = 259

// WindowsProcessMonitor отвечает за мониторинг процессов Windows
type WindowsProcessMonitor struct{}

// NewWindowsProcessMonitor создает новый монитор процессов
func NewWindowsProcessMonitor() *WindowsProcessMonitor {
	return &WindowsProcessMonitor{}
}

// IsProcessRunning проверяет, запущен ли процесс по PID
func (m *WindowsProcessMonitor) IsProcessRunning(pid int) bool {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		slog.Debug("Не удалось открыть процесс", "pid", pid, "error", err)
		return false
	}
	defer windows.CloseHandle(handle)

	var exitCode uint32
	err = windows.GetExitCodeProcess(handle, &exitCode)
	if err != nil {
		slog.Debug("Не удалось получить статус выхода процесса", "pid", pid, "error", err)
		return false
	}

	// Если exitCode != STILL_ACTIVE, процесс завершен
	if exitCode != STILL_ACTIVE {
		slog.Debug("Процесс завершен", "pid", pid, "exitCode", exitCode)
		return false
	}

	slog.Debug("Процесс активен", "pid", pid)
	return true
}

// GetProcessExitCode получает код выхода процесса
func (m *WindowsProcessMonitor) GetProcessExitCode(pid int) (uint32, error) {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		return 0, err
	}
	defer windows.CloseHandle(handle)

	var exitCode uint32
	err = windows.GetExitCodeProcess(handle, &exitCode)
	if err != nil {
		return 0, err
	}

	return exitCode, nil
}

// IsWinwsRunning проверяет, запущен ли процесс winws.exe в системе
func (m *WindowsProcessMonitor) IsWinwsRunning() bool {
	output, err := utils.OutputHidden("tasklist", "/FI", "IMAGENAME eq winws.exe")
	if err != nil {
		return false
	}

	return strings.Contains(string(output), "winws.exe")
}
