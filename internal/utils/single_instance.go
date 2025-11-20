package utils

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"syscall"
	"time"
	"unsafe"

	"github.com/Microsoft/go-winio"
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procCreateMutex  = kernel32.NewProc("CreateMutexW")
	procReleaseMutex = kernel32.NewProc("ReleaseMutex")
	procCloseHandle  = kernel32.NewProc("CloseHandle")
)

const (
	ERROR_ALREADY_EXISTS = 183
	PIPE_NAME            = `\\.\pipe\GoZapret_IPC_Pipe`
	ACTIVATE_COMMAND     = "ACTIVATE"
)

// ActivateCallback - функция обратного вызова для активации окна
type ActivateCallback func()

// SingleInstance управляет единственным экземпляром приложения
type SingleInstance struct {
	mutexHandle      syscall.Handle
	mutexName        string
	logger           *slog.Logger
	pipeListener     net.Listener
	activateCallback ActivateCallback
	stopChan         chan struct{}
}

// NewSingleInstance создает новый экземпляр менеджера единственного экземпляра
func NewSingleInstance(appName string) *SingleInstance {
	return &SingleInstance{
		mutexName: fmt.Sprintf("Global\\%s_Mutex", appName),
		logger:    slog.Default(),
	}
}

// SetLogger устанавливает логгер для SingleInstance
func (si *SingleInstance) SetLogger(logger *slog.Logger) {
	si.logger = logger
}

// SetActivateCallback устанавливает callback для активации окна
func (si *SingleInstance) SetActivateCallback(callback ActivateCallback) {
	si.activateCallback = callback
}

// TryAcquire пытается захватить мьютекс для обеспечения единственного экземпляра
// Возвращает true, если это первый экземпляр, false - если уже запущен
func (si *SingleInstance) TryAcquire() (bool, error) {
	si.logger.Debug("Попытка захвата мьютекса", "mutex_name", si.mutexName)

	mutexNamePtr, err := syscall.UTF16PtrFromString(si.mutexName)
	if err != nil {
		si.logger.Error("Ошибка преобразования имени мьютекса", "error", err)
		return false, fmt.Errorf("ошибка преобразования имени мьютекса: %w", err)
	}

	// Создаем мьютекс и сразу получаем последнюю ошибку
	ret, _, callErr := procCreateMutex.Call(
		0,                                     // lpMutexAttributes
		0,                                     // bInitialOwner
		uintptr(unsafe.Pointer(mutexNamePtr)), // lpName
	)

	if ret == 0 {
		si.logger.Error("Не удалось создать мьютекс", "error", callErr)
		return false, fmt.Errorf("не удалось создать мьютекс: %v", callErr)
	}

	si.mutexHandle = syscall.Handle(ret)
	si.logger.Debug("Мьютекс создан", "handle", si.mutexHandle)

	// Проверяем код ошибки, возвращенный из CreateMutex
	// В Go syscall.Errno содержит последнюю ошибку Windows
	if errno, ok := callErr.(syscall.Errno); ok {
		si.logger.Debug("Проверка последней ошибки", "errno", errno, "ERROR_ALREADY_EXISTS", ERROR_ALREADY_EXISTS)

		if errno == ERROR_ALREADY_EXISTS {
			// Мьютекс уже существует - другой экземпляр запущен
			si.logger.Info("Обнаружен запущенный экземпляр приложения")
			si.Release()
			return false, nil
		}
	}

	si.logger.Info("Это первый экземпляр приложения")
	return true, nil
}

// StartIPCServer запускает IPC сервер для приема команд активации через Named Pipe
func (si *SingleInstance) StartIPCServer() error {
	// Создаем конфигурацию для Named Pipe
	pipeConfig := &winio.PipeConfig{
		SecurityDescriptor: "", // Используем дефолтные права доступа
		MessageMode:        true,
		InputBufferSize:    1024,
		OutputBufferSize:   1024,
	}

	listener, err := winio.ListenPipe(PIPE_NAME, pipeConfig)
	if err != nil {
		si.logger.Error("Не удалось создать Named Pipe сервер", "pipe", PIPE_NAME, "error", err)
		return fmt.Errorf("не удалось создать Named Pipe сервер: %w", err)
	}

	si.pipeListener = listener
	si.stopChan = make(chan struct{})
	si.logger.Info("IPC сервер запущен", "pipe", PIPE_NAME)

	// Запускаем горутину для обработки входящих соединений
	go si.handleConnections()

	return nil
}

// handleConnections обрабатывает входящие IPC соединения через Named Pipe
func (si *SingleInstance) handleConnections() {
	for {
		select {
		case <-si.stopChan:
			si.logger.Debug("Остановка обработки IPC соединений")
			return
		default:
			conn, err := si.pipeListener.Accept()
			if err != nil {
				select {
				case <-si.stopChan:
					// Нормальное завершение
					return
				default:
					si.logger.Debug("Ошибка принятия соединения", "error", err)
					continue
				}
			}

			go si.handleConnection(conn)
		}
	}
}

// handleConnection обрабатывает одно IPC соединение
func (si *SingleInstance) handleConnection(conn io.ReadWriteCloser) {
	defer conn.Close()

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		si.logger.Debug("Ошибка чтения из соединения", "error", err)
		return
	}

	command := string(buf[:n])
	si.logger.Debug("Получена IPC команда", "command", command)

	if command == ACTIVATE_COMMAND && si.activateCallback != nil {
		si.logger.Info("Активация окна по IPC команде")
		si.activateCallback()
	}
}

// Release освобождает мьютекс и закрывает IPC сервер
func (si *SingleInstance) Release() {
	if si.stopChan != nil {
		si.logger.Debug("Отправка сигнала остановки IPC сервера")
		close(si.stopChan)
		si.stopChan = nil
	}

	if si.pipeListener != nil {
		si.logger.Debug("Закрытие Named Pipe сервера")
		si.pipeListener.Close()
		si.pipeListener = nil
	}

	if si.mutexHandle != 0 {
		si.logger.Debug("Освобождение мьютекса", "handle", si.mutexHandle)
		procReleaseMutex.Call(uintptr(si.mutexHandle))
		procCloseHandle.Call(uintptr(si.mutexHandle))
		si.mutexHandle = 0
	}
}

// SendActivateCommand отправляет команду активации существующему экземпляру через Named Pipe
func (si *SingleInstance) SendActivateCommand() error {
	si.logger.Debug("Отправка команды активации существующему экземпляру")

	// Пробуем подключиться к Named Pipe с несколькими попытками
	var conn io.ReadWriteCloser
	var err error

	for i := 0; i < 10; i++ {
		conn, err = winio.DialPipe(PIPE_NAME, nil)
		if err == nil {
			break
		}

		si.logger.Debug("Попытка подключения к Named Pipe", "attempt", i+1, "pipe", PIPE_NAME, "error", err)
		time.Sleep(200 * time.Millisecond)
	}

	if err != nil {
		si.logger.Error("Не удалось подключиться к Named Pipe", "pipe", PIPE_NAME, "error", err)
		return fmt.Errorf("не удалось подключиться к Named Pipe: %w", err)
	}
	defer conn.Close()

	// Отправляем команду активации
	_, err = conn.Write([]byte(ACTIVATE_COMMAND))
	if err != nil {
		si.logger.Error("Ошибка отправки команды активации", "error", err)
		return fmt.Errorf("ошибка отправки команды активации: %w", err)
	}

	si.logger.Info("Команда активации успешно отправлена")
	return nil
}
