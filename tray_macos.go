//go:build darwin && !ios

package giode

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework AppKit

#include <stdint.h>

void giodeTrayStartMain(void);
void giodeTrayHideItem(void);
void giodeHideWindow(uintptr_t view);
void giodeShowWindow(uintptr_t view);
void giodeInterceptClose(uintptr_t view);
void giodeHandleReopen(void);
void giodeMenuBarApp(void);
*/
import "C"

import (
	"sync"

	"gioui.org/app"

	"fyne.io/systray"
)

var (
	trayStartFn   func()
	trayStartOnce sync.Once
)

// startSystray creates the status item on the main thread, inside the
// app loop Gio runs: systray mustn't run (or replace the delegate of)
// an NSApplication of its own.
func startSystray(ready func()) {
	start, _ := systray.RunWithExternalLoop(ready, nil)
	trayStartFn = start
	C.giodeTrayStartMain()
}

//export giodeTrayStart
func giodeTrayStart() {
	trayStartOnce.Do(func() {
		if trayStartFn != nil {
			trayStartFn()
		}
	})
}

// trayRemove hides the status item: systray's Quit would stop Gio's
// application loop.
func trayRemove() {
	C.giodeTrayHideItem()
}

func setTrayIcon(icon []byte, template bool) {
	if template {
		systray.SetTemplateIcon(icon, icon)
	} else {
		systray.SetIcon(icon)
	}
}

func (a *App) hideNativeWindow() bool {
	v := a.NativeHandle()
	if v == 0 {
		return false
	}
	C.giodeHideWindow(C.uintptr_t(v))
	return true
}

func (a *App) showNativeWindow() {
	if v := a.NativeHandle(); v != 0 {
		C.giodeShowWindow(C.uintptr_t(v))
	}
}

func (a *App) warnTrayLimits() {}

// platformInit applies MenuBarApp before the app finishes launching.
func (a *App) platformInit() {
	if a.cfg.MenuBarApp {
		C.giodeMenuBarApp()
	}
	a.registerServices()
}

// trayView sets up close-to-tray and Dock re-opening once the window
// exists.
func (a *App) trayView(app.ViewEvent) {
	v := a.NativeHandle()
	if v == 0 {
		return
	}
	if a.closeToTray.Load() {
		C.giodeInterceptClose(C.uintptr_t(v))
	}
	C.giodeHandleReopen()
}

// applyTrayOptions intercepts closing once CloseToTray is turned on;
// the interception asks closesToTray each time, so turning it off
// needs nothing.
func (a *App) applyTrayOptions() {
	if v := a.NativeHandle(); v != 0 && a.closeToTray.Load() {
		C.giodeInterceptClose(C.uintptr_t(v))
	}
}

//export giodeShouldHideOnClose
func giodeShouldHideOnClose() C.int {
	if a := currentApp.Load(); a != nil && a.closesToTray() {
		return 1
	}
	return 0
}

//export giodeHiddenOnClose
func giodeHiddenOnClose() {
	if a := currentApp.Load(); a != nil {
		a.windowHiddenNatively()
	}
}

//export giodeReopen
func giodeReopen() {
	if a := currentApp.Load(); a != nil && a.windowHidden.Load() {
		a.ShowWindow()
	}
}
