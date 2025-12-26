package app

import (
	"fmt"
	"log/slog"

	"github.com/IProxymate/GoZapret/internal/domain"
	"github.com/IProxymate/GoZapret/internal/services/process"
)

// StrategyController управляет бизнес-логикой стратегий.
// Это промежуточный слой между UI и сервисами.
type StrategyController struct {
	logger          *slog.Logger
	configService   ConfigService
	strategyService StrategyService
	processManager  *process.Manager
	eventBus        *EventBus
}

// NewStrategyController создаёт новый контроллер стратегий
func NewStrategyController(
	logger *slog.Logger,
	configService ConfigService,
	strategyService StrategyService,
	processManager *process.Manager,
	eventBus *EventBus,
) *StrategyController {
	return &StrategyController{
		logger:          logger,
		configService:   configService,
		strategyService: strategyService,
		processManager:  processManager,
		eventBus:        eventBus,
	}
}

// StartStrategy запускает указанную стратегию
func (c *StrategyController) StartStrategy(strategyName string, gameFilter bool) error {
	if strategyName == "" {
		return fmt.Errorf("стратегия не выбрана")
	}

	if c.processManager.IsRunning() {
		return fmt.Errorf("процесс уже запущен")
	}

	strategy, err := c.strategyService.GetByName(domain.StrategyName(strategyName))
	if err != nil {
		return fmt.Errorf("ошибка получения стратегии: %w", err)
	}

	assetsPath := c.configService.GetAssetsPath()
	if assetsPath == "" {
		return fmt.Errorf("путь к ресурсам не установлен")
	}

	c.logger.Info("Запуск стратегии", "strategy", strategyName, "gameFilter", gameFilter)

	if err := c.processManager.StartStrategy(strategy, assetsPath, gameFilter); err != nil {
		c.eventBus.PublishStrategyError(strategyName, err)
		return fmt.Errorf("ошибка запуска стратегии: %w", err)
	}

	// Сохраняем последнюю стратегию
	c.configService.SetLastStrategyName(domain.StrategyName(strategyName))

	// Публикуем события
	c.eventBus.PublishStrategyStarted(strategyName, gameFilter)
	c.publishStatusChanged()

	return nil
}

// StopStrategy останавливает текущую стратегию
func (c *StrategyController) StopStrategy() error {
	if !c.processManager.IsRunning() && !c.processManager.IsWinwsProcessRunning() {
		return fmt.Errorf("процесс не запущен")
	}

	c.logger.Info("Остановка стратегии")

	if err := c.processManager.StopProcess(); err != nil {
		return fmt.Errorf("ошибка остановки стратегии: %w", err)
	}

	// Публикуем события
	c.eventBus.PublishStrategyStopped()
	c.publishStatusChanged()

	return nil
}

// RestartCurrentStrategy перезапускает текущую стратегию
func (c *StrategyController) RestartCurrentStrategy(strategyName string, gameFilter bool) error {
	if !c.processManager.IsRunning() && !c.processManager.IsWinwsProcessRunning() {
		return nil // Не запущена — нечего перезапускать
	}

	if strategyName == "" {
		return nil
	}

	strategy, err := c.strategyService.GetByName(domain.StrategyName(strategyName))
	if err != nil {
		return fmt.Errorf("ошибка получения стратегии для перезапуска: %w", err)
	}

	assetsPath := c.configService.GetAssetsPath()
	if assetsPath == "" {
		return fmt.Errorf("путь к ресурсам не установлен")
	}

	c.logger.Info("Перезапуск стратегии", "strategy", strategyName)

	if err := c.processManager.RestartStrategy(strategy, assetsPath, gameFilter); err != nil {
		c.eventBus.PublishStrategyError(strategyName, err)
		return fmt.Errorf("ошибка перезапуска стратегии: %w", err)
	}

	// Публикуем событие
	c.eventBus.Publish(Event{Type: EventStrategyRestart, Data: StrategyEventData{
		StrategyName: strategyName,
		GameFilter:   gameFilter,
	}})
	c.publishStatusChanged()

	return nil
}

// IsRunning проверяет, запущен ли процесс
func (c *StrategyController) IsRunning() bool {
	return c.processManager.IsRunning() || c.processManager.IsWinwsProcessRunning()
}

// GetCurrentStrategyName возвращает имя текущей запущенной стратегии
func (c *StrategyController) GetCurrentStrategyName() string {
	if processInfo := c.processManager.GetCurrentProcess(); processInfo != nil {
		return string(processInfo.Strategy)
	}
	return ""
}

// GetLastStrategyName возвращает имя последней использованной стратегии
func (c *StrategyController) GetLastStrategyName() string {
	return string(c.configService.GetLastStrategyName())
}

// LoadStrategies загружает стратегии из указанного пути
func (c *StrategyController) LoadStrategies(assetsPath domain.AssetsPath) ([]string, error) {
	if err := c.strategyService.LoadFromPath(assetsPath); err != nil {
		return nil, fmt.Errorf("ошибка загрузки стратегий: %w", err)
	}

	strategyList := c.strategyService.GetAll()
	strategyNames := make([]string, len(strategyList))
	for i, s := range strategyList {
		strategyNames[i] = s.Name.String()
	}

	c.logger.Info("Загружены стратегии", "count", len(strategyNames))

	// Определяем выбранную стратегию
	selected := c.GetLastStrategyName()
	if selected == "" && len(strategyNames) > 0 {
		selected = strategyNames[0]
	}

	// Публикуем событие
	c.eventBus.PublishStrategiesLoaded(strategyNames, selected)

	return strategyNames, nil
}

// GetStatusMessage возвращает текущее сообщение о статусе
func (c *StrategyController) GetStatusMessage() string {
	if c.IsRunning() {
		if currentName := c.GetCurrentStrategyName(); currentName != "" {
			return domain.StatusRunningWith.Format(currentName)
		}
		if lastName := c.GetLastStrategyName(); lastName != "" {
			return domain.StatusRunningWith.Format(lastName)
		}
		return string(domain.StatusRunning)
	}

	if lastName := c.GetLastStrategyName(); lastName != "" {
		return domain.StatusStoppedLastUsed.Format(lastName)
	}
	return string(domain.StatusStopped)
}

// publishStatusChanged публикует событие изменения статуса
func (c *StrategyController) publishStatusChanged() {
	isRunning := c.IsRunning()
	statusMessage := c.GetStatusMessage()
	c.eventBus.PublishStatusChanged(isRunning, statusMessage)
}
