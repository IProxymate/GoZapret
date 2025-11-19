package utils

import (
	"context"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// SetupHiddenCommand настраивает существующую команду для запуска без отображения окна консоли.
// Используйте эту функцию, если команда уже создана и нужно только добавить скрытие окна.
//
// Параметры:
//   - cmd: существующая команда для настройки
func SetupHiddenCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
}

// NewHiddenCommand создает новую команду с скрытым окном консоли.
// Это полезно для фоновых процессов в Windows GUI приложениях.
//
// Параметры:
//   - name: имя исполняемого файла или путь к нему
//   - args: аргументы командной строки
//
// Возвращает настроенную команду *exec.Cmd
func NewHiddenCommand(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	SetupHiddenCommand(cmd)
	return cmd
}

// NewHiddenCommandContext создает новую команду с контекстом и скрытым окном консоли.
//
// Параметры:
//   - ctx: контекст для управления временем жизни команды
//   - name: имя исполняемого файла или путь к нему
//   - args: аргументы командной строки
//
// Возвращает настроенную команду *exec.Cmd
func NewHiddenCommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	SetupHiddenCommand(cmd)
	return cmd
}

// NewHiddenCommandWithDir создает новую команду с скрытым окном и рабочей директорией.
//
// Параметры:
//   - ctx: контекст для управления временем жизни команды
//   - workDir: рабочая директория для выполнения команды
//   - name: имя исполняемого файла или путь к нему
//   - args: аргументы командной строки
//
// Возвращает настроенную команду *exec.Cmd
func NewHiddenCommandWithDir(ctx context.Context, workDir, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = workDir
	SetupHiddenCommand(cmd)
	return cmd
}

// RunHidden выполняет команду со скрытым окном и возвращает ошибку.
// Используйте когда вам нужно только выполнить команду без получения вывода.
//
// Параметры:
//   - name: имя исполняемого файла или путь к нему
//   - args: аргументы командной строки
//
// Возвращает ошибку выполнения команды
func RunHidden(name string, args ...string) error {
	cmd := NewHiddenCommand(name, args...)
	return cmd.Run()
}

// OutputHidden выполняет команду со скрытым окном и возвращает stdout.
// Используйте когда нужен только стандартный вывод команды.
//
// Параметры:
//   - name: имя исполняемого файла или путь к нему
//   - args: аргументы командной строки
//
// Возвращает stdout и ошибку
func OutputHidden(name string, args ...string) ([]byte, error) {
	cmd := NewHiddenCommand(name, args...)
	return cmd.Output()
}

// CombinedOutputHidden выполняет команду со скрытым окном и возвращает объединенный stdout и stderr.
// Используйте когда нужен полный вывод команды (stdout + stderr).
//
// Параметры:
//   - name: имя исполняемого файла или путь к нему
//   - args: аргументы командной строки
//
// Возвращает объединенный вывод и ошибку
func CombinedOutputHidden(name string, args ...string) ([]byte, error) {
	cmd := NewHiddenCommand(name, args...)
	return cmd.CombinedOutput()
}

// OutputHiddenContext выполняет команду с контекстом и скрытым окном, возвращает stdout.
// Используйте когда нужен контроль времени выполнения команды.
//
// Параметры:
//   - ctx: контекст для управления временем жизни команды
//   - name: имя исполняемого файла или путь к нему
//   - args: аргументы командной строки
//
// Возвращает stdout и ошибку
func OutputHiddenContext(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := NewHiddenCommandContext(ctx, name, args...)
	return cmd.Output()
}
