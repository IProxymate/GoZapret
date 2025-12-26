package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/IProxymate/GoZapret/internal/domain"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Config содержит настройки логирования
type Config struct {
	// Level уровень логирования (debug, info, warn, error)
	Level string
	// Format формат вывода (text, json)
	Format string
	// Output куда писать логи (stdout, file, both)
	Output string
	// FilePath путь к файлу логов
	FilePath string
	// MaxSize максимальный размер файла в мегабайтах
	MaxSize int
	// MaxBackups максимальное количество старых файлов
	MaxBackups int
	// MaxAge максимальный возраст файлов в днях
	MaxAge int
	// Compress сжимать ли старые файлы
	Compress bool
	// AddSource добавлять ли информацию об источнике
	AddSource bool
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() *Config {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	logPath := filepath.Join(configDir, domain.AppName, domain.LogsDirName, domain.AppLogFile)

	return &Config{
		Level:      "debug",
		Format:     "text",
		Output:     "both",
		FilePath:   logPath,
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
		AddSource:  false,
	}
}

// LoadConfig загружает конфигурацию из файла
func LoadConfig(path string) (*Config, error) {
	// Если путь пустой, используем значения по умолчанию
	if path == "" {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("ошибка чтения файла конфигурации: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("ошибка парсинга конфигурации: %w", err)
	}

	// Если FilePath пустой, используем путь по умолчанию
	if cfg.FilePath == "" {
		configDir, err := os.UserConfigDir()
		if err != nil {
			configDir = "."
		}
		cfg.FilePath = filepath.Join(configDir, domain.AppName, domain.LogsDirName, domain.AppLogFile)
	}

	return &cfg, nil
}

// New создает новый логгер с заданной конфигурацией
func New(cfg *Config) (*slog.Logger, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Определяем уровень логирования
	level := parseLevel(cfg.Level)

	// Создаем writer в зависимости от настроек
	var writer io.Writer
	switch cfg.Output {
	case "stdout":
		writer = os.Stdout
	case "file":
		writer = createFileWriter(cfg)
	case "both":
		fileWriter := createFileWriter(cfg)
		// Проверяем доступность stdout
		if isStdoutAvailable() {
			writer = io.MultiWriter(os.Stdout, fileWriter)
		} else {
			// Если stdout недоступен (например, при запуске через планировщик задач),
			// используем только файловый вывод
			writer = fileWriter
		}
	default:
		writer = os.Stdout
	}

	// Создаем handler в зависимости от формата
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.AddSource,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Форматируем время в читаемый вид
			if a.Key == slog.TimeKey {
				if t, ok := a.Value.Any().(time.Time); ok {
					a.Value = slog.StringValue(t.Format("2006-01-02 15:04:05"))
				}
			}
			return a
		},
	}

	switch cfg.Format {
	case "json":
		handler = slog.NewJSONHandler(writer, opts)
	case "text":
		handler = slog.NewTextHandler(writer, opts)
	default:
		handler = slog.NewTextHandler(writer, opts)
	}

	return slog.New(handler), nil
}

// isStdoutAvailable проверяет, доступен ли stdout для записи
func isStdoutAvailable() bool {
	// Пробуем получить информацию о файле stdout
	_, err := os.Stdout.Stat()
	return err == nil
}

// createFileWriter создает writer для записи в файл с ротацией
func createFileWriter(cfg *Config) io.Writer {
	// Создаем директорию для логов, если её нет
	logDir := filepath.Dir(cfg.FilePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		// Если не удалось создать директорию, пишем в stdout
		return os.Stdout
	}

	return &lumberjack.Logger{
		Filename:   cfg.FilePath,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
		LocalTime:  true,
	}
}

// parseLevel парсит строку уровня логирования
func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
