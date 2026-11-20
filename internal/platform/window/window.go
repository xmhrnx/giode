// Package window places the app's window and sets its native chrome:
// position and screen, always on top, transparency and title bar
// buttons.
package window

import "errors"

// Rect is an area of the desktop in dp, as giode.Rect.
type Rect struct {
	X, Y, Width, Height float32
}

// Screen is the screen showing the window, as giode.Screen.
type Screen struct {
	Bounds, WorkArea Rect
	Scale            float32
}

// ErrNoWindowYet is returned for window placement before the window
// exists.
var ErrNoWindowYet = errors.New("giode: the window isn't shown yet")

// ErrFillsScreen means the window is the screen (Android, iOS, the
// web): the caller works the screen out from its frames.
var ErrFillsScreen = errors.New("giode: the window fills the screen")
