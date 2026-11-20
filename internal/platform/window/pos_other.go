//go:build android || ios || (!windows && !linux && !darwin)

package window

import (
	"runtime"

	"gioui.org/app"

	"github.com/xmhrnx/giode/internal/platform"
)

// On Android, iOS and the web the window is the screen; other systems
// aren't supported.
func mobileWindow() bool {
	return runtime.GOOS == "android" || runtime.GOOS == "ios" || runtime.GOOS == "js"
}

func ScreenOf(view app.ViewEvent, scale float32) (Screen, error) {
	if !mobileWindow() {
		return Screen{}, platform.ErrNotSupported
	}
	return Screen{}, ErrFillsScreen
}

func Frame(view app.ViewEvent, scale float32) (Rect, error) {
	s, err := ScreenOf(view, scale)
	return s.Bounds, err
}

func SetPosition(view app.ViewEvent, x, y, scale float32) error { return platform.ErrNotSupported }

func SetTopMost(view app.ViewEvent, on bool) bool { return false }

func Placed(view app.ViewEvent, topMost bool) {}

func ClientOrigin(view app.ViewEvent) (x, y int, ok bool) { return 0, 0, false }
