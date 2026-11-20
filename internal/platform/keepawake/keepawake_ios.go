package keepawake

/*
#cgo CFLAGS: -Werror -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework UIKit

#include <stdint.h>
#include <stdlib.h>

void giodeHaptic(int kind);
void giodeIdleTimerDisabled(int disabled);
uintptr_t giodeBeginBackgroundTask(const char *name);
void giodeEndBackgroundTask(uintptr_t task);
*/
import "C"

import (
	"context"
	"sync"
	"unsafe"

	"github.com/xmhrnx/giode/internal/platform"
)

// Haptic plays the feedback generators (keepawake_ios.m).
func Haptic(h platform.Host, kind int) {
	C.giodeHaptic(C.int(kind))
}

// KeepScreenOn turns UIApplication's idle timer off while on.
func KeepScreenOn(ctx context.Context, h platform.Host, on bool) error {
	d := C.int(0)
	if on {
		d = 1
	}
	C.giodeIdleTimerDisabled(d)
	return nil
}

// KeepAlive begins a background task.
func KeepAlive(h platform.Host, reason string, onStart func()) (func(), error) {
	cr := C.CString(reason)
	defer C.free(unsafe.Pointer(cr))
	task := C.giodeBeginBackgroundTask(cr)
	var once sync.Once
	return func() { once.Do(func() { C.giodeEndBackgroundTask(task) }) }, nil
}
