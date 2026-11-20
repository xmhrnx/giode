//go:build darwin && !ios

package filedrag

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework AppKit

#include <stdint.h>

void giodeEnableFileDrop(uintptr_t view);
*/
import "C"

import (
	"strings"
	"sync"

	"gioui.org/app"
)

var fileDropOnce sync.Once

func Install() {}

// ViewReady makes Gio's view a dragging destination.
func ViewReady(ev app.ViewEvent) string {
	if v, ok := ev.(app.AppKitViewEvent); ok && v.View != 0 {
		fileDropOnce.Do(func() { C.giodeEnableFileDrop(C.uintptr_t(v.View)) })
	}
	return ""
}

//export giodeFileDragEvent
func giodeFileDragEvent(kind C.int, x, y C.double) {
	drag(Kind(kind), float32(x), float32(y))
}

//export giodeFileDrop
func giodeFileDrop(paths *C.char, x, y C.double) {
	list := strings.Split(strings.TrimSuffix(C.GoString(paths), "\n"), "\n")
	if len(list) == 1 && list[0] == "" {
		list = nil
	}
	drop(list, float32(x), float32(y))
}
