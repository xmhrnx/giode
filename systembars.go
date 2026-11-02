package giode

import (
	"image/color"
	"runtime"

	"gioui.org/app"

	"github.com/xmhrnx/giode/styles/properties"
)

// Orientation restricts how a mobile app may be rotated.
type Orientation int

const (
	// OrientationAny lets the app follow the device's rotation.
	OrientationAny Orientation = iota
	// OrientationPortrait keeps the app in portrait.
	OrientationPortrait
	// OrientationLandscape keeps the app in landscape.
	OrientationLandscape
)

func (o Orientation) option() app.Option {
	switch o {
	case OrientationPortrait:
		return app.PortraitOrientation.Option()
	case OrientationLandscape:
		return app.LandscapeOrientation.Option()
	}
	return app.AnyOrientation.Option()
}

// systemBarsSupported reports whether the platform colors its system
// bars from the window options. Gio implements it on Android only.
var systemBarsSupported = runtime.GOOS == "android"

// systemBarColors resolves the status and navigation bar colors:
// StatusBar and NavigationBar when set, the window background
// otherwise. Theme tokens resolve against the active palette.
func (a *App) systemBarColors() (status, nav color.NRGBA) {
	s, n := a.cfg.StatusBar, a.cfg.NavigationBar
	if s == "" {
		s = a.cfg.Background
	}
	if n == "" {
		n = a.cfg.Background
	}
	return properties.CalcColor(s), properties.CalcColor(n)
}

// updateSystemBars recolors the system bars when their resolved
// colors changed since they were last applied, such as after SetMode
// or SetPalette changed the theme a token resolves against. The app
// calls it once per frame.
func (a *App) updateSystemBars() {
	if !systemBarsSupported {
		return
	}
	status, nav := a.systemBarColors()
	if a.barsApplied && status == a.statusColor && nav == a.navColor {
		return
	}
	a.barsApplied, a.statusColor, a.navColor = true, status, nav
	a.window.Option(app.StatusColor(status), app.NavigationColor(nav))
}
