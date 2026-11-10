//go:build giode_gio3

package giode

import (
	"image/color"

	"gioui.org/app"
)

// setNativeBackground sets the color a new window shows until its first
// frame (Windows would show white), or stops it once the first frame is
// drawn. It needs the giode CLI's Gio patches (revision 9 on).
func setNativeBackground(c color.NRGBA, on bool) {
	app.GiodeSetBackground(c.R, c.G, c.B, on)
}
