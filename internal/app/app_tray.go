package app

import (
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"

	"github.com/IProxymate/GoZapret/internal/domain"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

// waitForShell ожидает готовности оболочки Explorer (наличие окна Shell_TrayWnd).
// Вызывается при автозапуске, чтобы системный трей был доступен для регистрации иконки.
func (a *App) waitForShell() {
	user32 := syscall.NewLazyDLL("user32.dll")
	findWindow := user32.NewProc("FindWindowW")
	className, _ := syscall.UTF16PtrFromString("Shell_TrayWnd")

	const (
		timeout      = 60 * time.Second
		pollInterval = 500 * time.Millisecond
		settleDelay  = 3 * time.Second
	)

	a.Logger.Info("Ожидание готовности оболочки Explorer...")
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		hwnd, _, _ := findWindow.Call(
			uintptr(unsafe.Pointer(className)),
			0,
		)
		if hwnd != 0 {
			a.Logger.Info("Оболочка Explorer готова, ожидание стабилизации", "settleDelay", settleDelay)
			time.Sleep(settleDelay)
			return
		}
		time.Sleep(pollInterval)
	}

	a.Logger.Warn("Таймаут ожидания оболочки Explorer", "timeout", timeout)
}

// setupTray настраивает сворачивание приложения в системный трей
func (a *App) setupTray(icon []byte) {
	deskApp, ok := a.FyneApp.(desktop.App)
	if !ok {
		return
	}

	menu := fyne.NewMenu(domain.AppName,
		fyne.NewMenuItem("Показать", func() {
			a.Window.Show()
			a.Window.Resize(fyne.NewSize(900, 650))
			a.Window.CenterOnScreen()
			a.Window.RequestFocus()
		}),
	)

	deskApp.SetSystemTrayMenu(menu)
	a.Window.SetCloseIntercept(func() {
		a.Window.Hide()
	})

	go a.setTrayIconWithRetry(deskApp, icon)
}

// setTrayIconWithRetry устанавливает иконку в трее с повторными попытками
func (a *App) setTrayIconWithRetry(deskApp desktop.App, icon []byte) {
	time.Sleep(2 * time.Second)

	trayIcon := fyne.NewStaticResource("tray-icon.png", icon)
	const maxAttempts = 5
	baseDelay := time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		a.Logger.Debug("Попытка установки иконки в трее", "attempt", attempt, "max", maxAttempts)

		if a.trySetTrayIcon(deskApp, trayIcon) {
			a.Logger.Info("Иконка в трее успешно установлена", "attempt", attempt)
			return
		}

		if attempt < maxAttempts {
			delay := time.Duration(attempt) * baseDelay
			a.Logger.Debug("Ожидание перед следующей попыткой", "delay", delay)
			time.Sleep(delay)
		}
	}

	a.Logger.Error("Не удалось установить иконку в трее после всех попыток", "attempts", maxAttempts)
}

// trySetTrayIcon пытается установить иконку в трее
func (a *App) trySetTrayIcon(deskApp desktop.App, trayIcon *fyne.StaticResource) bool {
	success := make(chan bool, 1)

	fyne.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				a.Logger.Warn("Паника при установке иконки в трее", "error", r)
				success <- false
			}
		}()
		deskApp.SetSystemTrayIcon(trayIcon)
		success <- true
	})

	select {
	case ok := <-success:
		return ok
	case <-time.After(3 * time.Second):
		a.Logger.Warn("Таймаут при установке иконки в трее")
		return false
	}
}

// loadIcon загружает иконку приложения
func (a *App) loadIcon() []byte {
	iconData, err := a.loadIconData()
	if err == nil {
		a.FyneApp.SetIcon(fyne.NewStaticResource("icon256.png", iconData))
	}
	return iconData
}

// loadIconData загружает иконку из embedded ресурсов или файла
func (a *App) loadIconData() ([]byte, error) {
	if iconData, err := a.Assets.ReadFile("assets/icon256.png"); err == nil {
		return iconData, nil
	}

	if iconData, err := os.ReadFile("assets/icon256.png"); err == nil {
		return iconData, nil
	}

	return nil, fmt.Errorf("иконка не найдена")
}
