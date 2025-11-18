package main

import (
	"embed"
	"log/slog"

	"github.com/IProxymate/GoZapret/internal/logger"
	"github.com/IProxymate/GoZapret/internal/ui"
)

//go:embed assets/*
var Assets embed.FS

func main() {
	// Загружаем конфигурацию логирования из файла
	logConfig, err := logger.LoadConfig("./internal/config/logger.json")
	if err != nil {
		slog.Warn("Ошибка загрузки конфигурации логирования, используются настройки по умолчанию", "error", err)
		logConfig = logger.DefaultConfig()
	}

	// Создаем логгер
	log, err := logger.New(logConfig)
	if err != nil {
		slog.Error("Ошибка создания логгера", "error", err)
		return
	}

	// Устанавливаем как глобальный логгер
	slog.SetDefault(log)

	slog.Debug("Запуск приложения GoZapret")

	app := ui.NewApp(Assets, log)

	app.Run()
}
