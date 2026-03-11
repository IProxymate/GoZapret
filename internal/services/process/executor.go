package process

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"syscall"

	"github.com/IProxymate/GoZapret/internal/domain"
	"github.com/IProxymate/GoZapret/internal/utils"
)

// ProcessExecutor отвечает за запуск и остановку процессов
type ProcessExecutor struct{}

// NewProcessExecutor создает новый исполнитель процессов
func NewProcessExecutor() *ProcessExecutor {
	return &ProcessExecutor{}
}

// StartResult содержит результат запуска процесса
type StartResult struct {
	Cmd    *exec.Cmd
	Stderr *bytes.Buffer
}

// Start запускает процесс winws
func (e *ProcessExecutor) Start(winwsPath string, args []string) (*StartResult, error) {
	slog.Debug("Запуск winws.exe", "executable", winwsPath, "args", args)

	ctx := context.Background()
	cmd := utils.NewHiddenCommandContext(ctx, winwsPath, args...)

	// Захватываем stderr для получения ошибок
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return nil, domain.ErrProcessStartFailed
	}

	return &StartResult{
		Cmd:    cmd,
		Stderr: &stderr,
	}, nil
}

// Stop останавливает процесс
func (e *ProcessExecutor) Stop(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}

	// На Windows SIGTERM не работает — сразу используем taskkill для корректного завершения
	killCmd := exec.Command("taskkill", "/PID", fmt.Sprintf("%d", cmd.Process.Pid), "/F")
	killCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := killCmd.Run(); err != nil {
		// Если taskkill не помог, принудительно убиваем
		return cmd.Process.Kill()
	}
	return nil
}

// ForceKill принудительно убивает процесс
func (e *ProcessExecutor) ForceKill(cmd *exec.Cmd) error {
	if cmd.Process != nil {
		return cmd.Process.Kill()
	}
	return nil
}

// KillAllWinws убивает все процессы winws.exe в системе
func (e *ProcessExecutor) KillAllWinws() error {
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
