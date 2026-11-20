//go:build darwin && !ios

package giode

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework AppKit

void giodeWatchSleep(void);
*/
import "C"

func (a *App) watchPower(done <-chan struct{}, event func(suspend bool)) {
	sleepHandler = event
	C.giodeWatchSleep()
}

var sleepHandler func(suspend bool)

//export giodeSleepEvent
func giodeSleepEvent(sleeping C.int) {
	if f := sleepHandler; f != nil {
		go f(sleeping != 0)
	}
}
