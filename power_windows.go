package giode

import (
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Sleep on Windows: PowerRegisterSuspendResumeNotification calls back
// on a system thread before suspending and after resuming, with no
// window needed.

var (
	procPowerRegisterSuspendResume = windows.NewLazySystemDLL("powrprof.dll").NewProc("PowerRegisterSuspendResumeNotification")
	powerCallbackOnce              sync.Once
	powerCallback                  uintptr
)

const (
	pbtSuspend         = 0x4
	pbtResumeAutomatic = 0x12
)

type deviceNotifySubscribe struct {
	callback uintptr
	context  uintptr
}

var (
	powerSubscription deviceNotifySubscribe
	powerHandler      func(suspend bool)
)

func (a *App) watchPower(done <-chan struct{}, event func(suspend bool)) {
	if procPowerRegisterSuspendResume.Find() != nil {
		return
	}
	powerCallbackOnce.Do(func() {
		powerCallback = windows.NewCallback(func(context, kind, setting uintptr) uintptr {
			f := powerHandler
			if f == nil {
				return 0
			}
			switch kind {
			case pbtSuspend:
				go f(true)
			case pbtResumeAutomatic:
				// Resume comes as automatic, then (with a user present)
				// as suspend-resume: report the first.
				go f(false)
			}
			return 0
		})
	})
	powerHandler = event
	powerSubscription = deviceNotifySubscribe{callback: powerCallback}
	var handle uintptr
	const deviceNotifyCallback = 2
	if r, _, _ := procPowerRegisterSuspendResume.Call(deviceNotifyCallback, uintptr(unsafe.Pointer(&powerSubscription)), uintptr(unsafe.Pointer(&handle))); r != 0 {
		a.logf(LogLevelDebug, "giode: no sleep notices (%d)", r)
	}
}
