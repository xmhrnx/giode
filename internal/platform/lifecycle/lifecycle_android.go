package lifecycle

import (
	"gioui.org/app"

	"github.com/xmhrnx/giode/internal/mobile"
)

// Android reports going to the background through the activity
// lifecycle, not the window state.
const FromWindowState = false

// Watch reports activity stops and starts.
func Watch(background, foreground func()) error {
	mobile.SetLifecycleHandlers(background, foreground)
	return mobile.WatchLifecycle(app.JavaVM(), app.AppContext())
}
