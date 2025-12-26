package app

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"

	"github.com/IProxymate/GoZapret/internal/domain"
)

// ReloadStrategies перечитывает стратегии и обновляет файлы в рабочей директории
func (a *App) ReloadStrategies() error {
	assetsPath := a.Services.Config.GetAssetsPath()
	if assetsPath == "" {
		return fmt.Errorf("путь к ресурсам не установлен")
	}

	currentSelected, _ := a.State.SelectedStrategy.Get()

	if err := a.Services.Config.PrepareWorkingDirectory(); err != nil {
		return fmt.Errorf("ошибка подготовки рабочей директории: %w", err)
	}

	a.reloadStrategiesWithSelection(assetsPath, currentSelected)
	return nil
}

// LoadStrategies загружает стратегии из указанного пути (первая загрузка)
func (a *App) LoadStrategies(assetsPath domain.AssetsPath) {
	strategyNames := a.loadStrategiesInternal(assetsPath)
	if strategyNames == nil {
		return
	}

	a.State.Strategies.Set(strategyNames)

	if lastStrategy := a.Services.Config.GetLastStrategyName(); lastStrategy != "" {
		a.State.SelectedStrategy.Set(lastStrategy.String())
	} else if len(strategyNames) > 0 {
		a.State.SelectedStrategy.Set(strategyNames[0])
	}
}

// LoadStrategiesAndRefreshUI загружает стратегии и обновляет UI (для использования после обновления)
func (a *App) LoadStrategiesAndRefreshUI(assetsPath domain.AssetsPath) {
	strategyNames := a.loadStrategiesInternal(assetsPath)
	if strategyNames == nil {
		return
	}

	// Определяем выбранную стратегию
	selectedStrategy := ""
	if lastStrategy := a.Services.Config.GetLastStrategyName(); lastStrategy != "" {
		// Проверяем, существует ли последняя стратегия в новом списке
		for _, name := range strategyNames {
			if name == lastStrategy.String() {
				selectedStrategy = lastStrategy.String()
				break
			}
		}
	}
	if selectedStrategy == "" && len(strategyNames) > 0 {
		selectedStrategy = strategyNames[0]
	}

	// Обновляем состояние и UI в главном потоке
	fyne.Do(func() {
		a.State.Strategies.Set(strategyNames)
		a.State.SelectedStrategy.Set(selectedStrategy)

		// Обновляем виджет выбора стратегий если MainView существует
		if a.mainView != nil {
			a.mainView.UpdateStrategyOptions(strategyNames, selectedStrategy)
		}

		a.Logger.Info("Стратегии загружены и UI обновлён после обновления", "count", len(strategyNames), "selected", selectedStrategy)
	})
}

// reloadStrategiesWithSelection перезагружает стратегии с сохранением текущего выбора
func (a *App) reloadStrategiesWithSelection(assetsPath domain.AssetsPath, currentSelected string) {
	strategyNames := a.loadStrategiesInternal(assetsPath)
	if strategyNames == nil {
		return
	}

	a.Logger.Info("Перезагрузка стратегий", "count", len(strategyNames), "currentSelected", currentSelected)

	selectedStrategy := a.findSelectedStrategy(strategyNames, currentSelected)

	fyne.Do(func() {
		a.State.Strategies.Set(strategyNames)
		a.State.SelectedStrategy.Set(selectedStrategy)

		if a.mainView != nil {
			a.mainView.UpdateStrategyOptions(strategyNames, selectedStrategy)
		}
	})
}

// findSelectedStrategy находит выбранную стратегию в списке или возвращает первую
func (a *App) findSelectedStrategy(strategyNames []string, currentSelected string) string {
	if currentSelected != "" {
		for _, name := range strategyNames {
			if name == currentSelected {
				return currentSelected
			}
		}
	}

	if len(strategyNames) > 0 {
		return strategyNames[0]
	}
	return ""
}

// loadStrategiesInternal загружает стратегии и возвращает список имён
func (a *App) loadStrategiesInternal(assetsPath domain.AssetsPath) []string {
	if err := a.Services.Strategy.LoadFromPath(assetsPath); err != nil {
		a.State.Strategies.Set([]string{})
		dialog.ShowError(fmt.Errorf("ошибка загрузки стратегий: %v", err), a.Window)
		return nil
	}

	a.updateVersionFromServiceBat(assetsPath)

	strategyList := a.Services.Strategy.GetAll()
	strategyNames := make([]string, len(strategyList))
	for i, s := range strategyList {
		strategyNames[i] = s.Name.String()
	}

	a.Logger.Info("Загружены стратегии", "count", len(strategyNames), "names", strategyNames)
	return strategyNames
}

// updateVersionFromServiceBat обновляет версию из service.bat
func (a *App) updateVersionFromServiceBat(assetsPath domain.AssetsPath) {
	newVersion, err := a.Services.Strategy.ReadVersionFromServiceBat(assetsPath)
	if err != nil {
		a.Logger.Warn("Не удалось прочитать версию из service.bat", "error", err)
		return
	}

	currentVersion := a.Services.Config.GetVersion()
	if currentVersion == newVersion {
		return
	}

	if err := a.Services.Config.SetVersion(newVersion); err != nil {
		a.Logger.Warn("Не удалось сохранить версию в конфиг", "version", newVersion, "error", err)
		return
	}

	a.Logger.Debug("Версия Zapret обновлена в конфиге", "old_version", currentVersion, "new_version", newVersion)
	a.State.Version.Set(newVersion)
}

// RestartCurrentStrategy перезапускает текущую стратегию, если процесс запущен.
// Этот метод предназначен для использования из views при изменении настроек.
func (a *App) RestartCurrentStrategy() {
	// Проверяем, запущен ли процесс
	if !a.Services.Process.IsRunning() && !a.Services.Process.IsWinwsProcessRunning() {
		return
	}

	// Получаем текущую стратегию
	strategyName, _ := a.State.SelectedStrategy.Get()
	if strategyName == "" {
		return
	}

	strategy, err := a.Services.Strategy.GetByName(domain.StrategyName(strategyName))
	if err != nil {
		a.Logger.Error("Ошибка получения стратегии для перезапуска", "strategy", strategyName, "error", err)
		return
	}

	assetsPath := a.Services.Config.GetAssetsPath()
	if assetsPath == "" {
		a.Logger.Error("Путь к ресурсам не установлен для перезапуска стратегии")
		return
	}

	// Получаем текущее состояние GameFilter
	gameFilter, _ := a.State.GameFilter.Get()

	// Перезапускаем стратегию в фоне
	go func() {
		a.Logger.Info("Перезапуск стратегии", "strategy", strategyName)
		err := a.Services.Process.RestartStrategy(strategy, assetsPath, gameFilter)
		if err != nil {
			a.Logger.Error("Ошибка перезапуска стратегии", "strategy", strategyName, "error", err)
			fyne.Do(func() {
				dialog.ShowError(fmt.Errorf("ошибка перезапуска стратегии: %v", err), a.Window)
			})
		}
		// Обновляем статус после перезапуска
		a.UpdateStatus()
	}()
}

