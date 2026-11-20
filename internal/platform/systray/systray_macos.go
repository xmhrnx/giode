//go:build darwin && !ios

package systray

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework AppKit

void giodeTrayStartMain(void);
void giodeTrayHideItem(void);
*/
import "C"

import (
	"sync"

	"fyne.io/systray"
)

var (
	trayStartFn   func()
	trayStartOnce sync.Once
)

// Start creates the status item on the main thread, inside the
// app loop Gio runs: systray mustn't run (or replace the delegate of)
// an NSApplication of its own.
func Start(ready func()) {
	start, _ := systray.RunWithExternalLoop(ready, nil)
	trayStartFn = start
	C.giodeTrayStartMain()
}

//export giodeTrayStart
func giodeTrayStart() {
	trayStartOnce.Do(func() {
		if trayStartFn != nil {
			trayStartFn()
		}
	})
}

// Remove hides the status item: systray's Quit would stop Gio's
// application loop.
func Remove() {
	C.giodeTrayHideItem()
}

func setIcon(icon []byte, template bool) {
	if template {
		systray.SetTemplateIcon(icon, icon)
	} else {
		systray.SetIcon(icon)
	}
}
