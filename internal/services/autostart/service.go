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
	var errs []error
	if err := s.taskScheduler.RemoveTask(); err != nil {
		slog.Warn("Ошибка удаления задачи автозапуска", "error", err)
		errs = append(errs, err)
	}
	if err := s.registry.Remove(); err != nil {
		slog.Warn("Ошибка удаления записи реестра автозапуска", "error", err)
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		slog.Warn("Не все компоненты автозапуска удалены", "errors", len(errs))
	}
	slog.Debug("Автозапуск отключен")
	return nil
}
