package domain

import "fmt"

// IpsetMode представляет режим работы IPset
type IpsetMode string

// IPset режимы работы
const (
	// IpsetModeAny - использовать все IP-адреса
	IpsetModeAny IpsetMode = "any"
	// IpsetModeNone - не использовать IPset
	IpsetModeNone IpsetMode = "none"
	// IpsetModeLoaded - использовать только загруженные IP-адреса
	IpsetModeLoaded IpsetMode = "loaded"
)

// String возвращает строковое представление режима
func (m IpsetMode) String() string {
	return string(m)
}

// AllIpsetModeStrings возвращает все доступные режимы IPset как строки
func AllIpsetModeStrings() []string {
	return []string{
		IpsetModeAny.String(),
		IpsetModeNone.String(),
		IpsetModeLoaded.String(),
	}
}


// StatusMessage представляет сообщение о статусе
type StatusMessage string

// Статусы приложения
const (
	StatusRunning  StatusMessage = "Статус: Запущено"
	StatusStopped  StatusMessage = "Статус: Остановлено"
	StatusStarting StatusMessage = "Статус: Запускается"
	StatusStopping StatusMessage = "Статус: Останавливается"
	StatusError    StatusMessage = "Статус: Ошибка"
)

// StatusRunningWith содержит методы для форматирования статуса "Запущено"
var StatusRunningWith statusRunningWith

type statusRunningWith struct{}

// Format форматирует статус с именем стратегии
func (statusRunningWith) Format(strategy interface{}) string {
	return fmt.Sprintf("Статус: Запущено (%v)", strategy)
}

// StatusStoppedLastUsed содержит методы для форматирования статуса "Остановлено"
var StatusStoppedLastUsed statusStoppedLastUsed

type statusStoppedLastUsed struct{}

// Format форматирует статус с последней стратегией
func (statusStoppedLastUsed) Format(strategy interface{}) string {
	return fmt.Sprintf("Статус: Остановлено (последняя: %v)", strategy)
}

// Настройки по умолчанию
const (
	DefaultAutoStart  = true
	DefaultGameFilter = false
	DefaultIpsetMode  = IpsetModeNone
)

// Имена файлов
const (
	WinwsExecutable = "winws.exe"
	ServiceBatFile  = "service.bat"
	ListExtraFile   = "list-extra.txt"
	ListExcludeFile = "list-extra-exclude.txt"
	IpsetCustomFile = "ipset-custom.txt"
	IpsetAllFile    = "ipset-all.txt"
	ConfigFileName  = "config.json"
	HostsDirName    = "hosts"
	BinDirName      = "bin"
	ListsDirName    = "lists"
	WorkingDirName  = "working"
	LogsDirName     = "logs"
	AppLogFile      = "app.log"
)

// Имя приложения
const (
	AppName   = "GoZapret"
	AppID     = "com.zapret.gui"
	MutexName = "GoZapret_Mutex"
	PipeName  = `\\.\pipe\GoZapret_IPC_Pipe`
)

// Команды IPC
const (
	IPCActivateCommand = "ACTIVATE"
)

// Аргументы командной строки
const (
	ArgAutostart = "/autostart"
)
