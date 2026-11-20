//go:build darwin && !ios

package filedrag

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework AppKit

#include <stdint.h>
#include <stdlib.h>

void giodeStartExport(uintptr_t view, char *files, char *text);
*/
import "C"

import (
	"strings"
	"sync"
	"unsafe"

	"gioui.org/app"

	"github.com/xmhrnx/giode/elements/dnd"
	"github.com/xmhrnx/giode/internal/platform"
)

// Dragging out of the window on macOS (Draggable.Export): an
// NSDraggingSession begun on Gio's view, with file URLs or a string on
// the pasteboard.

var (
	exportMu   sync.Mutex
	exportDone func(bool)
)

// InstallExporter lets Draggable.Export drag out of the window; done
// runs through post, on the UI goroutine.
func InstallExporter(h platform.Host) {
	dnd.SetExporter(func(data dnd.ExportData, done func(bool)) bool {
		v, ok := h.NativeView().(app.AppKitViewEvent)
		if !ok || v.View == 0 {
			return false
		}
		exportMu.Lock()
		exportDone = func(dropped bool) { h.Post(func() { done(dropped) }) }
		exportMu.Unlock()
		files := C.CString(strings.Join(data.Files, "\n"))
		text := C.CString(data.Text)
		// giodeStartExport copies them before returning.
		C.giodeStartExport(C.uintptr_t(v.View), files, text)
		C.free(unsafe.Pointer(files))
		C.free(unsafe.Pointer(text))
		return true
	})
}

//export giodeExportEnded
func giodeExportEnded(dropped C.int) {
	exportMu.Lock()
	done := exportDone
	exportDone = nil
	exportMu.Unlock()
	if done != nil {
		done(dropped != 0)
	}
}
