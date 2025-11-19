package services

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/IProxymate/GoZapret/internal/utils"

	"golang.org/x/sys/windows/registry"
)

const autostartKey = `Software\Microsoft\Windows\CurrentVersion\Run`
const autostartValueName = "GoZapret"
const taskName = "GoZapret_AutoStart"

// AutostartCommand предоставляет команды для управления автозапуском
type AutostartService struct{}

// NewAutostartCommand создает новую команду автозапуска
func NewAutostartService() *AutostartService {
	return &AutostartService{}
}

// IsTaskSchedulerEnabled проверяет существование задачи в планировщике
func (a *AutostartService) IsTaskSchedulerEnabled() (bool, error) {
	err := utils.RunHidden("schtasks", "/query", "/tn", taskName)
	return err == nil, nil
}

// CreateScheduledTask создает задачу в планировщике Windows
func (a *AutostartService) CreateScheduledTask() error {
	slog.Debug("Создание задачи автозапуска в планировщике")

	execPath, err := os.Executable()
	if err != nil {
		slog.Error("Не удалось получить путь к исполняемому файлу", "error", err)
		return fmt.Errorf("не удалось получить путь к исполняемому файлу: %v", err)
	}

	// Удаляем существующую задачу, если есть
	_ = a.RemoveScheduledTask()

	// Создаем XML для задачи
	taskXML := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-16"?>
<Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo>
    <Description>GoZapret автозапуск с правами администратора</Description>
  </RegistrationInfo>
  <Triggers>
    <LogonTrigger>
      <Enabled>true</Enabled>
    </LogonTrigger>
  </Triggers>
  <Principals>
    <Principal id="Author">
      <RunLevel>HighestAvailable</RunLevel>
    </Principal>
  </Principals>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>
    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>
    <AllowHardTerminate>true</AllowHardTerminate>
    <StartWhenAvailable>false</StartWhenAvailable>
    <RunOnlyIfNetworkAvailable>false</RunOnlyIfNetworkAvailable>
    <IdleSettings>
      <StopOnIdleEnd>true</StopOnIdleEnd>
      <RestartOnIdle>false</RestartOnIdle>
    </IdleSettings>
    <AllowStartOnDemand>true</AllowStartOnDemand>
    <Enabled>true</Enabled>
    <Hidden>false</Hidden>
    <RunOnlyIfIdle>false</RunOnlyIfIdle>
    <WakeToRun>false</WakeToRun>
    <ExecutionTimeLimit>PT0S</ExecutionTimeLimit>
    <Priority>7</Priority>
  </Settings>
  <Actions Context="Author">
    <Exec>
      <Command>%s</Command>
      <Arguments>/autostart</Arguments>
    </Exec>
  </Actions>
</Task>`, execPath)

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

// RemoveScheduledTask удаляет задачу из планировщика
func (a *AutostartService) RemoveScheduledTask() error {
	return utils.RunHidden("schtasks", "/delete", "/tn", taskName, "/f")
}

// IsRegistryEnabled проверяет старый способ через реестр
func (a *AutostartService) IsRegistryEnabled() (bool, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, autostartKey, registry.READ)
	if err != nil {
		return false, err
	}
	defer key.Close()

	value, _, err := key.GetStringValue(autostartValueName)
	if err != nil {
		if err == registry.ErrNotExist {
			return false, nil
		}
		return false, err
	}

	execPath, err := os.Executable()
	if err != nil {
		return false, fmt.Errorf("не удалось получить путь к исполняемому файлу: %v", err)
	}
	expectedValue := fmt.Sprintf(`"%s" /autostart`, execPath)

	return value == expectedValue, nil
}

// RemoveRegistryAutostart удаляет старую запись из реестра
func (a *AutostartService) RemoveRegistryAutostart() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, autostartKey, registry.WRITE)
	if err != nil {
		return err
	}
	defer key.Close()

	return key.DeleteValue(autostartValueName)
}

// IsAutoStartEnabled проверяет, включен ли автозапуск приложения
func (a *AutostartService) IsAutoStartEnabled() (bool, error) {
	// Сначала проверяем планировщик задач (новый способ)
	if taskExists, err := a.IsTaskSchedulerEnabled(); err == nil && taskExists {
		return true, nil
	}

	// Если задача не найдена, проверяем старый способ через реестр
	return a.IsRegistryEnabled()
}

// SetAutoStart устанавливает или убирает автозапуск приложения
func (a *AutostartService) SetAutoStart(enabled bool) error {
	slog.Info("Изменение настроек автозапуска", "enabled", enabled)

	if enabled {
		// Удаляем старую запись из реестра, если есть
		_ = a.RemoveRegistryAutostart()

		// Создаем задачу в планировщике
		return a.CreateScheduledTask()
	} else {
		slog.Debug("Удаление автозапуска")
		// Удаляем и задачу, и запись в реестре
		_ = a.RemoveScheduledTask()
		_ = a.RemoveRegistryAutostart()
		slog.Debug("Автозапуск отключен")
		return nil
	}
}
