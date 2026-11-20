//go:build linux && !android && nox11

package window

import (
	"gioui.org/app"
	"github.com/xmhrnx/giode/internal/platform"
)

func x11Frame(app.ViewEvent, float32) (Rect, error)                 { return Rect{}, platform.ErrNotSupported }
func x11SetPosition(app.ViewEvent, float32, float32, float32) error { return platform.ErrNotSupported }
func x11Screen(app.ViewEvent, float32) (Screen, error)              { return Screen{}, platform.ErrNotSupported }
func x11SetAbove(app.ViewEvent, bool)                               {}

func x11ClientOrigin(app.ViewEvent) (int, int, bool) { return 0, 0, false }
