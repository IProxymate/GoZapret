package app

import (
	"sync"

	"fyne.io/fyne/v2"
)

// EventType определяет тип события
type EventType string

const (
	// События стратегий
	EventStrategyStarted  EventType = "strategy.started"
	EventStrategyStopped  EventType = "strategy.stopped"
	EventStrategyRestart  EventType = "strategy.restart"
	EventStrategyError    EventType = "strategy.error"
	EventStrategiesLoaded EventType = "strategies.loaded"

	// События статуса
	EventStatusChanged EventType = "status.changed"

	// События конфигурации
	EventConfigChanged   EventType = "config.changed"
	EventAssetsPathSet   EventType = "config.assets_path_set"
	EventGameFilterSet   EventType = "config.game_filter_set"
	EventIpsetModeSet    EventType = "config.ipset_mode_set"
	EventAutoStartSet    EventType = "config.autostart_set"
)

// Event представляет событие в системе
type Event struct {
	Type EventType
	Data any
}

// StrategyEventData данные события стратегии
type StrategyEventData struct {
	StrategyName string
	GameFilter   bool
	Error        error
}

// StatusEventData данные события статуса
type StatusEventData struct {
	IsRunning    bool
	StatusMessage string
}

// StrategiesLoadedData данные события загрузки стратегий
type StrategiesLoadedData struct {
	StrategyNames    []string
	SelectedStrategy string
}

// EventHandler функция-обработчик события
type EventHandler func(event Event)

// EventBus шина событий для связи между компонентами приложения
type EventBus struct {
	mu       sync.RWMutex
	handlers map[EventType][]EventHandler
}

// NewEventBus создаёт новую шину событий
func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make(map[EventType][]EventHandler),
	}
}

// Subscribe подписывает обработчик на событие
// Возвращает функцию отписки
func (eb *EventBus) Subscribe(eventType EventType, handler EventHandler) func() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.handlers[eventType] = append(eb.handlers[eventType], handler)

	// Возвращаем функцию отписки
	return func() {
		eb.Unsubscribe(eventType, handler)
	}
}

// SubscribeMultiple подписывает обработчик на несколько событий
func (eb *EventBus) SubscribeMultiple(eventTypes []EventType, handler EventHandler) func() {
	unsubscribers := make([]func(), len(eventTypes))
	for i, et := range eventTypes {
		unsubscribers[i] = eb.Subscribe(et, handler)
	}

	return func() {
		for _, unsub := range unsubscribers {
			unsub()
		}
	}
}

// Unsubscribe отписывает обработчик от события
func (eb *EventBus) Unsubscribe(eventType EventType, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	handlers := eb.handlers[eventType]
	for i, h := range handlers {
		// Сравниваем указатели функций
		if &h == &handler {
			eb.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}
}

// Publish публикует событие всем подписчикам
func (eb *EventBus) Publish(event Event) {
	eb.mu.RLock()
	handlers := make([]EventHandler, len(eb.handlers[event.Type]))
	copy(handlers, eb.handlers[event.Type])
	eb.mu.RUnlock()

	for _, handler := range handlers {
		handler(event)
	}
}

// PublishAsync публикует событие асинхронно
func (eb *EventBus) PublishAsync(event Event) {
	go eb.Publish(event)
}

// PublishOnMainThread публикует событие в главном потоке UI (Fyne)
func (eb *EventBus) PublishOnMainThread(event Event) {
	fyne.Do(func() {
		eb.Publish(event)
	})
}

// Helper методы для публикации типизированных событий

// PublishStrategyStarted публикует событие запуска стратегии
func (eb *EventBus) PublishStrategyStarted(strategyName string, gameFilter bool) {
	eb.PublishOnMainThread(Event{
		Type: EventStrategyStarted,
		Data: StrategyEventData{
			StrategyName: strategyName,
			GameFilter:   gameFilter,
		},
	})
}

// PublishStrategyStopped публикует событие остановки стратегии
func (eb *EventBus) PublishStrategyStopped() {
	eb.PublishOnMainThread(Event{
		Type: EventStrategyStopped,
		Data: StrategyEventData{},
	})
}

// PublishStrategyError публикует событие ошибки стратегии
func (eb *EventBus) PublishStrategyError(strategyName string, err error) {
	eb.PublishOnMainThread(Event{
		Type: EventStrategyError,
		Data: StrategyEventData{
			StrategyName: strategyName,
			Error:        err,
		},
	})
}

// PublishStatusChanged публикует событие изменения статуса
func (eb *EventBus) PublishStatusChanged(isRunning bool, statusMessage string) {
	eb.PublishOnMainThread(Event{
		Type: EventStatusChanged,
		Data: StatusEventData{
			IsRunning:     isRunning,
			StatusMessage: statusMessage,
		},
	})
}

// PublishStrategiesLoaded публикует событие загрузки стратегий
func (eb *EventBus) PublishStrategiesLoaded(strategyNames []string, selected string) {
	eb.PublishOnMainThread(Event{
		Type: EventStrategiesLoaded,
		Data: StrategiesLoadedData{
			StrategyNames:    strategyNames,
			SelectedStrategy: selected,
		},
	})
}

