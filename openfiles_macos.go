//go:build darwin && !ios

package giode

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework AppKit

#include <stdint.h>
#include <stdlib.h>

void giodeOpenFiles(uintptr_t view, char *exts, int multiple);
*/
import "C"

import (
	"strings"
	"unsafe"

	"gioui.org/app"
)

// filesPicked receives the paths giodeOpenFiles' panel returns, nil
// when the user cancels it.
var filesPicked = make(chan []string, 1)

//export giodeFilesPicked
func giodeFilesPicked(paths *C.char, n C.int) {
	if paths == nil {
		filesPicked <- nil
		return
	}
	filesPicked <- splitPaths(C.GoBytes(unsafe.Pointer(paths), n))
}

// openFileDialog shows an NSOpenPanel for files (several when multiple
// is set), as a sheet on the app's window when there is one, and returns
// their paths. Gio's explorer can't choose several files on macOS.
func (a *App) openFileDialog(exts []string, multiple bool) ([]string, error) {
	fileDialogMu.Lock()
	defer fileDialogMu.Unlock()
	var view uintptr
	if v, ok := a.NativeView().(app.AppKitViewEvent); ok {
		view = v.View
	}
	// The extensions without dots, one per line.
	clean := make([]string, 0, len(exts))
	for _, e := range exts {
		if e = strings.TrimLeft(strings.TrimSpace(e), "*."); e != "" {
			clean = append(clean, e)
		}
	}
	cexts := C.CString(strings.Join(clean, "\n"))
	defer C.free(unsafe.Pointer(cexts))
	m := C.int(0)
	if multiple {
		m = 1
	}
	C.giodeOpenFiles(C.uintptr_t(view), cexts, m)
	paths := <-filesPicked
	if len(paths) == 0 {
		return nil, ErrCanceled
	}
	return paths, nil
}
