//go:build giode_gio3

package giopatch

import (
	"image/color"

	"gioui.org/app"
)

// SetBackground sets the color a new window shows until its first
// frame (Windows would show white), or stops it once the first frame is
// drawn. It needs the giode CLI's Gio patches (revision 9 on).
func SetBackground(c color.NRGBA, on bool) {
	app.GiodeSetBackground(c.R, c.G, c.B, on)
}
