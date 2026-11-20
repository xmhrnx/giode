//go:build linux && !android

package window

import (
	"gioui.org/app"

	"github.com/xmhrnx/giode/internal/platform"
)

// Window placement on Linux works on X11. Wayland gives apps no
// positions: the compositor places windows.

func Frame(view app.ViewEvent, scale float32) (Rect, error) {
	if _, ok := view.(app.X11ViewEvent); ok {
		return x11Frame(view, scale)
	}
	return Rect{}, placementErr(view)
}

func SetPosition(view app.ViewEvent, x, y, scale float32) error {
	if _, ok := view.(app.X11ViewEvent); ok {
		return x11SetPosition(view, x, y, scale)
	}
	return placementErr(view)
}

func ScreenOf(view app.ViewEvent, scale float32) (Screen, error) {
	if _, ok := view.(app.X11ViewEvent); ok {
		return x11Screen(view, scale)
	}
	return Screen{}, placementErr(view)
}

// SetTopMost keeps an X11 window above others, reporting false where
// Gio's option applies instead (a no-op on Wayland).
func SetTopMost(view app.ViewEvent, on bool) bool {
	if _, ok := view.(app.X11ViewEvent); ok {
		x11SetAbove(view, on)
		return true
	}
	return false
}

// Placed keeps the window above others on X11 when Config.AlwaysOnTop
// asks: Gio's option doesn't reach X11 windows.
func Placed(view app.ViewEvent, topMost bool) {
	if _, ok := view.(app.X11ViewEvent); ok && view.Valid() && topMost {
		x11SetAbove(view, true)
	}
}

// ClientOrigin is the X11 window's top left on the screen.
func ClientOrigin(view app.ViewEvent) (x, y int, ok bool) { return x11ClientOrigin(view) }

func placementErr(view app.ViewEvent) error {
	if view == nil {
		return ErrNoWindowYet
	}
	return platform.ErrNotSupported
}
