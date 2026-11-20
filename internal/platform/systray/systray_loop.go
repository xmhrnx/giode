//go:build windows || (linux && !android)

package systray

import (
	"runtime"

	"fyne.io/systray"
)

// Start runs the tray's event loop on a goroutine of its own: on
// Windows its window's messages arrive on the thread that created it,
// so the loop stays on that thread. ready runs once the tray exists.
func Start(ready func()) {
	go func() {
		runtime.LockOSThread()
		systray.Run(ready, nil)
	}()
}

// Remove takes the icon out of the tray.
func Remove() {
	systray.Quit()
}
