package giode

import (
	"errors"
	"image"

	"gioui.org/app"
	"github.com/xmhrnx/giode/internal/platform/window"
)

// Rect is an area of the desktop in dp: X and Y are its top left
// corner, from the primary screen's top left.
type Rect struct {
	X, Y, Width, Height float32
}

// Screen describes the screen (monitor) showing the window.
type Screen struct {
	// Bounds is the whole screen.
	Bounds Rect
	// WorkArea is the part of it windows go in: without the taskbar,
	// the Dock and menu bar, or desktop panels.
	WorkArea Rect
	// Scale is the screen's pixels per dp.
	Scale float32
}

// Corner names a place in the screen's work area for MoveTo.
type Corner int

const (
	CornerTopLeft Corner = iota
	CornerTopRight
	CornerBottomLeft
	CornerBottomRight
	// CornerCenter centers the window.
	CornerCenter
)

// errNoWindowYet is returned for window placement before the window
// exists.
var errNoWindowYet = window.ErrNoWindowYet

// frameGeom is a frame's size in pixels and scale.
type frameGeom struct {
	pxPerDp float32
	size    image.Point
}

// scale returns the last frame's pixels per dp, 1 before the first.
func (a *App) scale() float32 {
	if g := a.frameGeom.Load(); g != nil && g.pxPerDp > 0 {
		return g.pxPerDp
	}
	return 1
}

// Screen returns the screen showing the window, in dp. It's supported
// on Windows, macOS and X11; on Android, iOS and the web the window is
// the screen, which it returns with the system bars taken out of the
// work area. Wayland doesn't tell apps where their windows are:
// there it returns ErrNotSupported. Safe to call from any goroutine.
func (a *App) Screen() (Screen, error) {
	s, err := window.ScreenOf(a.NativeView(), a.scale())
	if errors.Is(err, window.ErrFillsScreen) {
		return a.windowScreen()
	}
	return Screen{Bounds: Rect(s.Bounds), WorkArea: Rect(s.WorkArea), Scale: s.Scale}, err
}

// Frame returns the window's area on the desktop in dp, its title bar
// and borders included. See Screen for where it's supported.
func (a *App) Frame() (Rect, error) {
	r, err := window.Frame(a.NativeView(), a.scale())
	if errors.Is(err, window.ErrFillsScreen) {
		s, err := a.windowScreen()
		return s.Bounds, err
	}
	return Rect(r), err
}

// SetPosition moves the window so its top left corner (of the title
// bar and border) is at x, y in dp on the desktop, from the primary
// screen's top left. It's supported on Windows, macOS and X11, and
// returns ErrNotSupported elsewhere: Wayland compositors place windows
// themselves. Safe to call from any goroutine; the move happens
// shortly after.
func (a *App) SetPosition(x, y float32) error {
	return window.SetPosition(a.NativeView(), x, y, a.scale())
}

// MoveTo moves the window to a corner of its screen's work area (or
// its center), margin dp from the edges - a mini player in the bottom
// right, say. Where it's supported is as for SetPosition.
func (a *App) MoveTo(c Corner, margin float32) error {
	s, err := a.Screen()
	if err != nil {
		return err
	}
	f, err := a.Frame()
	if err != nil {
		return err
	}
	w := s.WorkArea
	x, y := w.X+margin, w.Y+margin
	switch c {
	case CornerTopRight:
		x = w.X + w.Width - f.Width - margin
	case CornerBottomLeft:
		y = w.Y + w.Height - f.Height - margin
	case CornerBottomRight:
		x = w.X + w.Width - f.Width - margin
		y = w.Y + w.Height - f.Height - margin
	case CornerCenter:
		x = w.X + (w.Width-f.Width)/2
		y = w.Y + (w.Height-f.Height)/2
	}
	return a.SetPosition(x, y)
}

// SetAlwaysOnTop keeps the window above other windows, or stops it,
// as Config.AlwaysOnTop does at start. It works on Windows, macOS and
// X11; Wayland leaves the choice to the user. Safe to call from any
// goroutine.
func (a *App) SetAlwaysOnTop(on bool) {
	a.topMost.Store(on)
	if !window.SetTopMost(a.NativeView(), on) {
		a.window.Option(app.TopMost(on))
	}
}

// AlwaysOnTop reports whether the window was set to stay above other
// windows, by Config.AlwaysOnTop or SetAlwaysOnTop.
func (a *App) AlwaysOnTop() bool {
	return a.topMost.Load()
}

// windowScreen is Screen for platforms whose window fills the screen:
// the last frame, less the system bars.
func (a *App) windowScreen() (Screen, error) {
	g := a.frameGeom.Load()
	if g == nil {
		return Screen{}, errNoWindowYet
	}
	s := a.scale()
	b := Rect{Width: float32(g.size.X) / s, Height: float32(g.size.Y) / s}
	in := SafeArea()
	w := Rect{X: float32(in.Left), Y: float32(in.Top),
		Width: b.Width - float32(in.Left+in.Right), Height: b.Height - float32(in.Top+in.Bottom)}
	return Screen{Bounds: b, WorkArea: w, Scale: s}, nil
}
