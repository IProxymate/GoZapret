package autostart

import (
	"log/slog"
	"os"
)

// Service управляет автозапуском приложения
type Service struct {
	taskScheduler *TaskScheduler
	registry      *RegistryManager
}

// NewService создает новый сервис автозапуска
func NewService() *Service {
	return &Service{
		taskScheduler: NewTaskScheduler(),
		registry:      NewRegistryManager(),
	}
}

// IsAutoStartEnabled проверяет, включен ли автозапуск приложения
func (s *Service) IsAutoStartEnabled() (bool, error) {
	// Сначала проверяем планировщик задач (новый способ)
	if taskExists, err := s.taskScheduler.IsTaskExists(); err == nil && taskExists {
		return true, nil
	}

	// Если задача не найдена, проверяем старый способ через реестр
	return s.registry.IsEnabled()
}

// SetAutoStart устанавливает или убирает автозапуск приложения
func (s *Service) SetAutoStart(enabled bool) error {
	slog.Info("Изменение настроек автозапуска", "enabled", enabled)

	if enabled {
		// Удаляем старую запись из реестра, если есть
		_ = s.registry.Remove()

		// Получаем путь к исполняемому файлу
		execPath, err := os.Executable()
		if err != nil {
			slog.Error("Не удалось получить путь к исполняемому файлу", "error", err)
			return err
		}

		// Создаем задачу в планировщике
		return s.taskScheduler.CreateTask(execPath)
	}

	slog.Debug("Удаление автозапуска")
	// Удаляем и задачу, и запись в реестре
	_ = s.taskScheduler.RemoveTask()
	_ = s.registry.Remove()
	slog.Debug("Автозапуск отключен")
	return nil
}
