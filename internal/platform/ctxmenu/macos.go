//go:build darwin && !ios

package ctxmenu

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework AppKit

void giodeRegisterServices(void);
*/
import "C"

import "strings"

// On macOS, context actions are Services: the giode CLI declares them
// in the app's Info.plist (NSServices), Finder lists them under Quick
// Actions, and macOS calls the provider registered here with the
// selected files.

func Install(string, string, []Action) error { return nil }
func Uninstall(string) error                 { return nil }

// RegisterServices makes the app the provider of its Services.
func RegisterServices() { C.giodeRegisterServices() }

//export giodeServiceInvoked
func giodeServiceInvoked(id, paths *C.char) {
	if OnService == nil {
		return
	}
	var ps []string
	if p := C.GoString(paths); p != "" {
		ps = strings.Split(p, "\n")
	}
	OnService(C.GoString(id), ps)
}
