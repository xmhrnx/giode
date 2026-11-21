//go:build giode_gio2

package giopatch

import "gioui.org/app"

// Target is a frame's GPU target, as Gio's patches report it.
type Target = app.GiodeTarget

// SetUnderlay hands the underlay and its hooks to giode's patched Gio:
// draw runs before each frame's UI, after once the frame is drawn,
// release when the GPU context goes away.
func SetUnderlay(draw, after func(Target), release func(), forceGL bool) error {
	app.GiodeSetUnderlay(draw, forceGL)
	app.GiodeSetUnderlayHooks(release, after)
	return nil
}
