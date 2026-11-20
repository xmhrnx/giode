//go:build darwin && !ios

package power

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework AppKit

void giodeWatchSleep(void);
*/
import "C"
import "github.com/xmhrnx/giode/internal/platform"

func Watch(h platform.Host, done <-chan struct{}, event func(suspend bool), hold bool) {
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
