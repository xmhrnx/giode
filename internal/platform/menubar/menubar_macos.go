//go:build darwin && !ios

package menubar

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework AppKit
#include <stdlib.h>

void giodeSetMenuBar(const char *specJSON, const char *appName);
*/
import "C"

import "unsafe"

// Set installs the menu bar described by specJSON, under the app menu
// named appName.
func Set(specJSON, appName string) {
	cs, cn := C.CString(specJSON), C.CString(appName)
	defer C.free(unsafe.Pointer(cs))
	defer C.free(unsafe.Pointer(cn))
	C.giodeSetMenuBar(cs, cn)
}

//export giodeMenuBarClick
func giodeMenuBarClick(id C.int) {
	if f := OnClick; f != nil {
		f(int(id))
	}
}

//export giodeMenuQuit
func giodeMenuQuit() {
	if f := OnQuit; f != nil {
		f()
	}
}
