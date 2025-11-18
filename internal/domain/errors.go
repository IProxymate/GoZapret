package domain

import "errors"

// Определение ошибок домена
var (
	ErrInvalidStrategy       = errors.New("некорректная стратегия")
	ErrStrategyNotFound      = errors.New("стратегия не найдена")
	ErrNoStrategiesFound     = errors.New("стратегии не найдены")
	ErrAssetsPathInvalid     = errors.New("некорректный путь к ресурсам")
	ErrWinwsNotFound         = errors.New("winws.exe не найден")
	ErrBatFileNotFound       = errors.New("bat файл не найден")
	ErrBatFileParseFailed    = errors.New("ошибка парсинга bat файла")
	ErrProcessAlreadyRunning = errors.New("процесс уже запущен")
	ErrProcessNotRunning     = errors.New("процесс не запущен")
	ErrProcessStartFailed    = errors.New("ошибка запуска процесса")
	ErrProcessStopFailed     = errors.New("ошибка остановки процесса")
	ErrAdminRightsRequired   = errors.New("требуются права администратора")
	ErrConfigNotFound        = errors.New("конфигурация не найдена")
	ErrConfigInvalid         = errors.New("некорректная конфигурация")
	ErrConfigSaveFailed      = errors.New("ошибка сохранения конфигурации")
	ErrTempDirCreateFailed   = errors.New("ошибка создания временной директории")
)
