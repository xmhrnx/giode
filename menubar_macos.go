//go:build darwin && !ios

package giode

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework AppKit
#include <stdlib.h>

void giodeSetMenuBar(const char *specJSON, const char *appName);
*/
import "C"

import (
	"encoding/json"
	"unsafe"
)

func (a *App) setNativeMenuBar(specs []menuBarSpec) {
	if a.cfg.MenuBarApp {
		return // no app menu in the menu bar
	}
	js, _ := json.Marshal(specs)
	a.menuBarJSON.Store(string(js))
	a.applyNativeMenuBar()
}

// applyNativeMenuBar installs the menu bar; again once the window
// exists, after Gio's launch installed its own.
func (a *App) applyNativeMenuBar() {
	js, _ := a.menuBarJSON.Load().(string)
	if js == "" {
		return
	}
	cs, cn := C.CString(js), C.CString(a.Title())
	defer C.free(unsafe.Pointer(cs))
	defer C.free(unsafe.Pointer(cn))
	C.giodeSetMenuBar(cs, cn)
}

//export giodeMenuBarClick
func giodeMenuBarClick(id C.int) {
	if a := currentApp.Load(); a != nil {
		a.menuBarClicked(int(id))
	}
}

//export giodeMenuQuit
func giodeMenuQuit() {
	if a := currentApp.Load(); a != nil {
		a.menuQuit()
	}
}

// menuBarView puts the menu bar back once the window exists: Gio's
// launch installs its own menu.
func (a *App) menuBarView() { a.applyNativeMenuBar() }
