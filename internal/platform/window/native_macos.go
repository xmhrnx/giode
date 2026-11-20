//go:build darwin && !ios

package window

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework AppKit
#include <stdint.h>

void giodeHideWindow(uintptr_t view);
void giodeShowWindow(uintptr_t view);
void giodeInterceptClose(uintptr_t view);
void giodeHandleReopen(void);
void giodeMenuBarApp(void);
*/
import "C"

import "github.com/xmhrnx/giode/internal/platform"

// CloseNeedsPatch is Linux's.
const CloseNeedsPatch = false

// Init: closing goes through Shown's interception.
func Init(h platform.Host, transparent bool, closeRequest func() bool) {}

// MenuBarApp removes the Dock icon and app menu, before the app
// finishes launching.
func MenuBarApp() { C.giodeMenuBarApp() }

// Hide orders the window out.
func Hide(h platform.Host) bool {
	v := h.NativeHandle()
	if v == 0 {
		return false
	}
	C.giodeHideWindow(C.uintptr_t(v))
	return true
}

// Show orders a hidden window back in.
func Show(h platform.Host) {
	if v := h.NativeHandle(); v != 0 {
		C.giodeShowWindow(C.uintptr_t(v))
	}
}

// Raise reports false: raising goes through Gio here.
func Raise(h platform.Host) bool { return false }

// Shown sets up closing (through Hooks, when interceptClose) and Dock
// re-opening once the window exists.
func Shown(h platform.Host, interceptClose bool) {
	v := h.NativeHandle()
	if v == 0 {
		return
	}
	if interceptClose {
		C.giodeInterceptClose(C.uintptr_t(v))
	}
	C.giodeHandleReopen()
}

// InterceptClose routes closing through Hooks; it asks them each time,
// so turning CloseToTray off needs nothing.
func InterceptClose(h platform.Host) {
	if v := h.NativeHandle(); v != 0 {
		C.giodeInterceptClose(C.uintptr_t(v))
	}
}

func boolC(b bool) C.int {
	if b {
		return 1
	}
	return 0
}

//export giodeShouldHideOnClose
func giodeShouldHideOnClose() C.int {
	return boolC(Hooks.HideOnClose != nil && Hooks.HideOnClose())
}

//export giodeVetoClose
func giodeVetoClose() C.int {
	return boolC(Hooks.VetoClose != nil && Hooks.VetoClose())
}

//export giodeHiddenOnClose
func giodeHiddenOnClose() {
	if Hooks.HiddenOnClose != nil {
		Hooks.HiddenOnClose()
	}
}

//export giodeReopen
func giodeReopen() {
	if Hooks.Reopen != nil {
		Hooks.Reopen()
	}
}
