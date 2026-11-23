package kit

import (
	"gioui.org/f32"
	"github.com/xmhrnx/giode/styles/properties"
)

// Margin is the default distance kept between an overlay and the
// window edges.
const Margin = 8

// ClampPos adjusts the position of an overlay of the given size so it
// stays inside a window of the given size. Overlays that overflow the
// bottom edge flip above the anchor point, like a context menu.
func ClampPos(win, size, pos f32.Point, margin float32) f32.Point {
	p := pos
	if p.Y+size.Y+margin > win.Y {
		p.Y = pos.Y - size.Y
	}
	if p.X+size.X+margin > win.X {
		p.X = win.X - size.X - margin
	}
	if p.X < margin {
		p.X = margin
	}
	if p.Y < margin {
		p.Y = margin
	}
	if p.Y+size.Y > win.Y-margin {
		p.Y = win.Y - size.Y - margin
	}
	return p
}

// PopupBorder and PopupShadow outline floating panels - menus, select
// and autocomplete lists - so they stand out over panels of their own
// color.
const PopupBorder = "theme.border"

var PopupShadow = properties.BoxShadow{Y: 4, Blur: 16, Color: "#00000059"}
