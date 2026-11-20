//go:build darwin && !ios

package picker

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework AppKit

#include <stdint.h>
#include <stdlib.h>

void giodePickFolder(uintptr_t view, char *title, char *dir);
*/
import "C"

import (
	"unsafe"

	"gioui.org/app"
	"github.com/xmhrnx/giode/internal/platform"
	"github.com/xmhrnx/giode/internal/platform/sandbox"
)

// folderPicked receives the folder giodePickFolder's panel returns.
var folderPicked = make(chan string, 1)

//export giodeFolderPicked
func giodeFolderPicked(path *C.char, ok C.int) {
	if ok == 0 {
		folderPicked <- "\x00"
		return
	}
	folderPicked <- C.GoString(path)
}

// PickFolder shows an NSOpenPanel for folders.
func PickFolder(h platform.Host, title, dir string) (string, error) {
	var view uintptr
	if v, ok := h.NativeView().(app.AppKitViewEvent); ok {
		view = v.View
	}
	ct, cd := C.CString(title), C.CString(dir)
	defer C.free(unsafe.Pointer(ct))
	defer C.free(unsafe.Pointer(cd))
	C.giodePickFolder(C.uintptr_t(view), ct, cd)
	path := <-folderPicked
	if path == "\x00" {
		return "", platform.ErrCanceled
	}
	// In the sandbox, the folder stays reachable after the app quits.
	sandbox.RememberFolder(path)
	return path, nil
}
