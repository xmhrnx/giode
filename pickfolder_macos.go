//go:build darwin && !ios

package giode

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

func pickFolder(a *App, opts FolderOptions) (string, error) {
	var view uintptr
	if v, ok := a.NativeView().(app.AppKitViewEvent); ok {
		view = v.View
	}
	title, dir := C.CString(opts.Title), C.CString(opts.Directory)
	defer C.free(unsafe.Pointer(title))
	defer C.free(unsafe.Pointer(dir))
	C.giodePickFolder(C.uintptr_t(view), title, dir)
	path := <-folderPicked
	if path == "\x00" {
		return "", ErrCanceled
	}
	// In the sandbox, the folder stays reachable after the app quits.
	rememberFolderAccess(path)
	return path, nil
}
