package main

import (
	"embed"
	"log/slog"
	"os"

	"github.com/IProxymate/GoZapret/internal/app"
	"github.com/IProxymate/GoZapret/internal/domain"
	"github.com/IProxymate/GoZapret/internal/logger"
	"github.com/IProxymate/GoZapret/internal/ui"
	"github.com/IProxymate/GoZapret/internal/utils"
)

// Версия приложения, устанавливается при сборке через ldflags:
// go build -ldflags "-X main.version=1.0.4"
var version = "dev"

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

	log.Debug("Запуск приложения", "app", domain.AppName)

	// Создаем менеджер единственного экземпляра
	singleInstance := utils.NewSingleInstance(domain.AppName)
	singleInstance.SetLogger(log)

	// Пытаемся захватить мьютекс
	isFirst, err := singleInstance.TryAcquire()
	if err != nil {
		log.Error("Ошибка проверки единственного экземпляра", "error", err)
		os.Exit(1)
	}

	if !isFirst {
		// Уже запущен другой экземпляр - отправляем команду активации
		log.Info("Приложение уже запущено, отправка команды активации")
		if err := singleInstance.SendActivateCommand(); err != nil {
			log.Warn("Не удалось отправить команду активации", "error", err)
		}
		os.Exit(0)
	}

	// Освобождаем мьютекс при завершении
	defer singleInstance.Release()

	// Создаем приложение
	application := app.NewApp(Assets, log, singleInstance, version)

	// Регистрируем фабрику MainView из пакета ui
	application.SetMainViewFactory(func(a *app.App) app.MainViewInterface {
		return ui.NewMainView(a)
	})

	application.Run()
}
