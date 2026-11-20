package background

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework Foundation -framework BackgroundTasks

#include <stdlib.h>

void giodeBackgroundDone(int rid, int ok);
char *giodeBackgroundSchedule(const char *name, double interval, double delay, int network, int charging, int isLong);
void giodeBackgroundCancel(const char *name);
*/
import "C"

import (
	"context"
	"errors"
	"sync"
	"unsafe"
)

var runs = struct {
	sync.Mutex
	cancel map[int]context.CancelFunc
}{cancel: map[int]context.CancelFunc{}}

func b2i(b bool) C.int {
	if b {
		return 1
	}
	return 0
}

func schedule(name string, o Options) error {
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	msg := C.giodeBackgroundSchedule(cs, C.double(o.Interval.Seconds()), C.double(o.Delay.Seconds()), b2i(o.Network), b2i(o.Charging), b2i(o.Long))
	if msg != nil {
		defer C.free(unsafe.Pointer(msg))
		return errors.New("background: " + C.GoString(msg))
	}
	return nil
}

func cancel(name string) error {
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	C.giodeBackgroundCancel(cs)
	return nil
}

//export giodeBackgroundRun
func giodeBackgroundRun(rid C.int, name *C.char) {
	n := C.GoString(name)
	ctx, stop := context.WithCancel(context.Background())
	runs.Lock()
	runs.cancel[int(rid)] = stop
	runs.Unlock()
	go func() {
		err := run(ctx, n)
		runs.Lock()
		delete(runs.cancel, int(rid))
		runs.Unlock()
		stop()
		C.giodeBackgroundDone(rid, b2i(err == nil))
	}()
}

//export giodeBackgroundExpire
func giodeBackgroundExpire(rid C.int) {
	runs.Lock()
	stop := runs.cancel[int(rid)]
	runs.Unlock()
	if stop != nil {
		stop()
	}
}
