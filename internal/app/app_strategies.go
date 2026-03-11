package app

import (
	"fmt"

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

	if _, err := a.Services.StrategyController.LoadStrategies(assetsPath, currentSelected); err != nil {
		return err
	}

	a.updateVersionFromServiceBat(assetsPath)
	return nil
}

// LoadStrategies загружает стратегии из указанного пути (первая загрузка)
func (a *App) LoadStrategies(assetsPath domain.AssetsPath) {
	if _, err := a.Services.StrategyController.LoadStrategies(assetsPath, ""); err != nil {
		a.State.Strategies.Set([]string{})
		dialog.ShowError(fmt.Errorf("ошибка загрузки стратегий: %v", err), a.Window)
		return
	}
	a.updateVersionFromServiceBat(assetsPath)
}

// LoadStrategiesAndRefreshUI загружает стратегии и обновляет UI (для использования после обновления)
func (a *App) LoadStrategiesAndRefreshUI(assetsPath domain.AssetsPath) {
	a.LoadStrategies(assetsPath)
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
	strategyName, _ := a.State.SelectedStrategy.Get()
	if strategyName == "" {
		return
	}

	gameFilterMode, _ := a.State.GameFilterMode.Get()

	go func() {
		if err := a.Services.StrategyController.RestartCurrentStrategy(strategyName, gameFilterMode); err != nil {
			a.Logger.Error("Ошибка перезапуска стратегии", "strategy", strategyName, "error", err)
		}
	}()
}
