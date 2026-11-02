package kit

import "github.com/xmhrnx/giode/styles/properties"

// Pressed returns the pressed shading of c: slightly darker.
func Pressed(c properties.Color) properties.Color {
	return Scale(c, 0.85)
}
