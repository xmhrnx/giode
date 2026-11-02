// Package kit provides the small building blocks used to create
// custom components: color state shading, hover tracking and overlay
// positioning (used by elements.RegisterPortal to keep click/hover
// popups inside the window).
//
// See the "Extending the library" documentation for a full recipe.
package kit

import "github.com/xmhrnx/giode/styles/properties"

// Scale multiplies each channel of c by f, clamping to the valid
// range.
func Scale(c properties.Color, f float32) properties.Color {
	nc := properties.CalcColor(c)
	sc := func(v uint8) uint8 {
		n := int(float32(v) * f)
		if n > 0xff {
			n = 0xff
		}
		return uint8(n)
	}
	nc.R, nc.G, nc.B = sc(nc.R), sc(nc.G), sc(nc.B)
	return properties.CalcColorReverse(nc)
}
