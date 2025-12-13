package autostart

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/IProxymate/GoZapret/internal/utils"
)

const taskName = "GoZapret_AutoStart"

// TaskScheduler управляет задачами в планировщике Windows
type TaskScheduler struct{}

// NewTaskScheduler создает новый менеджер задач планировщика
func NewTaskScheduler() *TaskScheduler {
	return &TaskScheduler{}
}

// IsTaskExists проверяет существование задачи в планировщике
func (t *TaskScheduler) IsTaskExists() (bool, error) {
	err := utils.RunHidden("schtasks", "/query", "/tn", taskName)
	return err == nil, nil
}

// CreateTask создает задачу в планировщике Windows
func (t *TaskScheduler) CreateTask(execPath string) error {
	slog.Debug("Создание задачи автозапуска в планировщике")

	// Удаляем существующую задачу, если есть
	_ = t.RemoveTask()

	// Генерируем XML для задачи
	taskXML := GenerateTaskXML(execPath)

	// Создаем временный файл для XML
	tempFile, err := os.CreateTemp("", "gozapret_task_*.xml")
	if err != nil {
		return fmt.Errorf("не удалось создать временный файл: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Записываем XML в файл
	if _, err := tempFile.WriteString(taskXML); err != nil {
		return fmt.Errorf("не удалось записать XML: %v", err)
	}
	tempFile.Close()

	// Создаем задачу через schtasks
	output, err := utils.CombinedOutputHidden("schtasks", "/create", "/tn", taskName, "/xml", tempFile.Name(), "/f")
	if err != nil {
		slog.Error("Не удалось создать задачу автозапуска", "error", err, "output", string(output))
		return fmt.Errorf("не удалось создать задачу: %v, вывод: %s", err, string(output))
	}

	slog.Debug("Задача автозапуска успешно создана", "task", taskName)
	return nil
}

// RemoveTask удаляет задачу из планировщика
func (t *TaskScheduler) RemoveTask() error {
	return utils.RunHidden("schtasks", "/delete", "/tn", taskName, "/f")
}
