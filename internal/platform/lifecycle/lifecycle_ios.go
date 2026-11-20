package lifecycle

/*
#cgo CFLAGS: -Werror -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework UIKit

void giodeWatchLifecycle(void);
*/
import "C"

import "sync"

// iOS reports going to the background through UIApplication, not the
// window state.
const FromWindowState = false

var handlers struct {
	sync.Mutex
	background, foreground func()
}

// Watch reports UIApplication's background and foreground
// notifications.
func Watch(background, foreground func()) error {
	handlers.Lock()
	handlers.background, handlers.foreground = background, foreground
	handlers.Unlock()
	C.giodeWatchLifecycle()
	return nil
}

//export giodeOnBackground
func giodeOnBackground() {
	handlers.Lock()
	f := handlers.background
	handlers.Unlock()
	if f != nil {
		f()
	}
}

//export giodeOnForeground
func giodeOnForeground() {
	handlers.Lock()
	f := handlers.foreground
	handlers.Unlock()
	if f != nil {
		f()
	}
}
