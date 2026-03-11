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

// GameFilterMode представляет режим работы Game Filter
type GameFilterMode string

// Режимы Game Filter
const (
	// GameFilterDisabled - Game Filter отключён
	GameFilterDisabled GameFilterMode = "disabled"
	// GameFilterTCP - обрабатывать только TCP трафик на портах 1024-65535
	GameFilterTCP GameFilterMode = "tcp"
	// GameFilterUDP - обрабатывать только UDP трафик на портах 1024-65535
	GameFilterUDP GameFilterMode = "udp"
	// GameFilterAll - обрабатывать TCP и UDP трафик на портах 1024-65535
	GameFilterAll GameFilterMode = "all"
)

// String возвращает строковое представление режима
func (m GameFilterMode) String() string {
	return string(m)
}

// IsEnabled возвращает true, если Game Filter включён в любом режиме
func (m GameFilterMode) IsEnabled() bool {
	return m != GameFilterDisabled && m != ""
}

// TCPEnabled возвращает true, если TCP порты должны быть открыты
func (m GameFilterMode) TCPEnabled() bool {
	return m == GameFilterTCP || m == GameFilterAll
}

// UDPEnabled возвращает true, если UDP порты должны быть открыты
func (m GameFilterMode) UDPEnabled() bool {
	return m == GameFilterUDP || m == GameFilterAll
}

// TCPValue возвращает значение для подстановки в %GameFilterTCP%
func (m GameFilterMode) TCPValue() string {
	if m.TCPEnabled() {
		return "1024-65535"
	}
	return "12"
}

// UDPValue возвращает значение для подстановки в %GameFilterUDP%
func (m GameFilterMode) UDPValue() string {
	if m.UDPEnabled() {
		return "1024-65535"
	}
	return "12"
}

// LegacyValue возвращает значение для подстановки в старый %GameFilter%
func (m GameFilterMode) LegacyValue() string {
	if m.IsEnabled() {
		return "1024-65535"
	}
	return "12"
}

// AllGameFilterModeStrings возвращает все доступные режимы Game Filter как строки
func AllGameFilterModeStrings() []string {
	return []string{
		GameFilterDisabled.String(),
		GameFilterTCP.String(),
		GameFilterUDP.String(),
		GameFilterAll.String(),
	}
}

// GameFilterModeLabels возвращает человекочитаемые метки для режимов
func GameFilterModeLabels() map[string]string {
	return map[string]string{
		GameFilterDisabled.String(): "Выключен",
		GameFilterTCP.String():      "TCP",
		GameFilterUDP.String():      "UDP",
		GameFilterAll.String():      "TCP + UDP",
	}
}

// GameFilterModeFromBool конвертирует старый bool формат в GameFilterMode (для миграции)
func GameFilterModeFromBool(enabled bool) GameFilterMode {
	if enabled {
		return GameFilterUDP // по умолчанию UDP, как в zapret v1.9.7
	}
	return GameFilterDisabled
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
	DefaultAutoStart      = true
	DefaultGameFilterMode = GameFilterDisabled
	DefaultIpsetMode      = IpsetModeNone
)

// Имена файлов
const (
	WinwsExecutable      = "winws.exe"
	ServiceBatFile       = "service.bat"
	ListExtraFile        = "list-general-user.txt"
	ListExcludeFile      = "list-exclude-user.txt"
	IpsetExcludeUserFile = "ipset-exclude-user.txt"
	// Устаревшие имена файлов (для миграции)
	LegacyListExtraFile   = "list-extra.txt"
	LegacyListExcludeFile = "list-extra-exclude.txt"
	LegacyIpsetCustomFile = "ipset-custom.txt"
	IpsetAllFile          = "ipset-all.txt"
	ConfigFileName        = "config.json"
	HostsDirName          = "hosts"
	BinDirName            = "bin"
	ListsDirName          = "lists"
	WorkingDirName        = "working"
	LogsDirName           = "logs"
	AppLogFile            = "app.log"
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
