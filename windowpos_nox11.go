//go:build linux && !android && nox11

package giode

import "gioui.org/app"

func x11Frame(app.ViewEvent, float32) (Rect, error)                 { return Rect{}, ErrNotSupported }
func x11SetPosition(app.ViewEvent, float32, float32, float32) error { return ErrNotSupported }
func x11Screen(app.ViewEvent, float32) (Screen, error)              { return Screen{}, ErrNotSupported }
func x11SetAbove(app.ViewEvent, bool)                               {}

func x11ClientOrigin(app.ViewEvent) (int, int, bool) { return 0, 0, false }
