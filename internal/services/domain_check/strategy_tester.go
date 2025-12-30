package domain_check

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/IProxymate/GoZapret/internal/domain"
	"github.com/IProxymate/GoZapret/internal/services/process"
	"github.com/IProxymate/GoZapret/internal/utils"
)

// StrategyTestMode режим тестирования
type StrategyTestMode string

const (
	TestModeStandard StrategyTestMode = "standard"
	TestModeDPI      StrategyTestMode = "dpi"
)

// StrategyTestProgress прогресс тестирования
type StrategyTestProgress struct {
	CurrentStrategy string
	CurrentIndex    int
	TotalStrategies int
	Phase           string // "starting", "testing", "stopping"
}

// AllStrategiesTestResult результат тестирования всех стратегий
type AllStrategiesTestResult struct {
	Results      []StrategyTestResult
	BestStrategy string
	BestScore    int
	TotalTime    time.Duration
}

// StrategyTester тестер стратегий
type StrategyTester struct {
	batParser     *process.BatParser
	argsBuilder   *process.ArgsBuilder
	multiChecker  *MultiChecker
	workingBinDir string
	assetsPath    domain.AssetsPath
	gameFilter    bool

	// Для отслеживания прогресса
	progressCallback func(StrategyTestProgress)
	cancelFunc       context.CancelFunc
	mu               sync.Mutex
}

// NewStrategyTester создает новый тестер стратегий
func NewStrategyTester(
	batParser *process.BatParser,
	argsBuilder *process.ArgsBuilder,
	workingBinDir string,
	assetsPath domain.AssetsPath,
	gameFilter bool,
) *StrategyTester {
	return &StrategyTester{
		batParser:     batParser,
		argsBuilder:   argsBuilder,
		multiChecker:  NewMultiChecker(),
		workingBinDir: workingBinDir,
		assetsPath:    assetsPath,
		gameFilter:    gameFilter,
	}
}

// SetProgressCallback устанавливает callback для отслеживания прогресса
func (t *StrategyTester) SetProgressCallback(callback func(StrategyTestProgress)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.progressCallback = callback
}

// Cancel отменяет текущее тестирование
func (t *StrategyTester) Cancel() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cancelFunc != nil {
		t.cancelFunc()
	}
}

// TestAllStrategies тестирует все стратегии последовательно
func (t *StrategyTester) TestAllStrategies(
	ctx context.Context,
	strategies []*domain.Strategy,
	mode StrategyTestMode,
	targets []Target,
) *AllStrategiesTestResult {
	startTime := time.Now()

	// Создаем контекст с отменой
	ctx, cancel := context.WithCancel(ctx)
	t.mu.Lock()
	t.cancelFunc = cancel
	t.mu.Unlock()
	defer cancel()

	result := &AllStrategiesTestResult{
		Results: make([]StrategyTestResult, len(strategies)),
	}

	totalStrategies := len(strategies)

	for i, strategy := range strategies {
		// Проверяем отмену
		select {
		case <-ctx.Done():
			slog.Info("Тестирование стратегий отменено")
			result.TotalTime = time.Since(startTime)
			return result
		default:
		}

		currentIndex := i + 1

		// Уведомляем о начале тестирования стратегии
		t.notifyProgress(StrategyTestProgress{
			CurrentStrategy: strategy.Name.String(),
			CurrentIndex:    currentIndex,
			TotalStrategies: totalStrategies,
			Phase:           "starting",
		})

		// Тестируем стратегию
		result.Results[i] = t.testSingleStrategy(ctx, strategy, mode, targets, currentIndex, totalStrategies)
	}

	// Определяем лучшую стратегию
	for _, res := range result.Results {
		if res.Error == nil && res.Score > result.BestScore {
			result.BestScore = res.Score
			result.BestStrategy = res.StrategyName
		}
	}

	result.TotalTime = time.Since(startTime)
	return result
}

// testSingleStrategy тестирует одну стратегию
func (t *StrategyTester) testSingleStrategy(
	ctx context.Context,
	strategy *domain.Strategy,
	mode StrategyTestMode,
	targets []Target,
	currentIndex int,
	totalStrategies int,
) StrategyTestResult {
	startTime := time.Now()
	result := StrategyTestResult{
		StrategyName: strategy.Name.String(),
	}

	// Запускаем winws с этой стратегией
	cmd, err := t.startWinws(ctx, strategy)
	if err != nil {
		result.Error = fmt.Errorf("ошибка запуска стратегии: %w", err)
		result.Duration = time.Since(startTime)
		return result
	}

	// Ждём инициализации (5 секунд как в PowerShell скрипте)
	select {
	case <-ctx.Done():
		t.stopWinws(cmd)
		result.Error = ctx.Err()
		result.Duration = time.Since(startTime)
		return result
	case <-time.After(5 * time.Second):
	}

	// Уведомляем о фазе тестирования
	t.notifyProgress(StrategyTestProgress{
		CurrentStrategy: strategy.Name.String(),
		CurrentIndex:    currentIndex,
		TotalStrategies: totalStrategies,
		Phase:           "testing",
	})

	// Выполняем тесты
	switch mode {
	case TestModeStandard:
		if len(targets) == 0 {
			targets = GetDefaultTargets()
		}
		result.StandardResult = t.multiChecker.CheckAll(targets)
		result.Score, result.PingScore = CalculateScore(result.StandardResult)
	case TestModeDPI:
		dpiTargets := GetDPITargets()
		result.DPIResult = t.multiChecker.CheckDPI(dpiTargets)
		result.Score = CalculateDPIScore(result.DPIResult)
	}

	// Уведомляем о фазе остановки
	t.notifyProgress(StrategyTestProgress{
		CurrentStrategy: strategy.Name.String(),
		CurrentIndex:    currentIndex,
		TotalStrategies: totalStrategies,
		Phase:           "stopping",
	})

	// Останавливаем winws
	t.stopWinws(cmd)

	result.Duration = time.Since(startTime)
	return result
}

// startWinws запускает winws с указанной стратегией
func (t *StrategyTester) startWinws(ctx context.Context, strategy *domain.Strategy) (*exec.Cmd, error) {
	// Парсим bat файл
	parsedArgs, err := t.batParser.Parse(strategy.BatFile, t.assetsPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга bat файла: %w", err)
	}

	// Строим финальные аргументы
	args := t.argsBuilder.Build(parsedArgs, t.workingBinDir, t.gameFilter)

	// Запускаем процесс
	winwsPath := filepath.Join(t.workingBinDir, "winws.exe")

	cmd := exec.CommandContext(ctx, winwsPath, args...)
	cmd.Dir = t.workingBinDir

	// Запускаем в фоне (не ждём завершения)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("ошибка запуска winws: %w", err)
	}

	slog.Debug("Запущен winws для тестирования", "strategy", strategy.Name, "pid", cmd.Process.Pid)
	return cmd, nil
}

// stopWinws останавливает winws
func (t *StrategyTester) stopWinws(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}

	// Пытаемся корректно завершить
	_ = cmd.Process.Kill()

	// Ждём завершения
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		// Принудительно убиваем все winws
		t.killAllWinws()
	}
}

// killAllWinws убивает все процессы winws
func (t *StrategyTester) killAllWinws() {
	_, _ = utils.OutputHidden("taskkill", "/F", "/IM", "winws.exe")
}

// notifyProgress уведомляет о прогрессе
func (t *StrategyTester) notifyProgress(progress StrategyTestProgress) {
	t.mu.Lock()
	callback := t.progressCallback
	t.mu.Unlock()

	if callback != nil {
		callback(progress)
	}
}

// TestSingleStrategySync синхронно тестирует одну стратегию (для простых случаев)
func (t *StrategyTester) TestSingleStrategySync(
	strategy *domain.Strategy,
	mode StrategyTestMode,
	targets []Target,
) StrategyTestResult {
	ctx := context.Background()
	return t.testSingleStrategy(ctx, strategy, mode, targets, 1, 1)
}

