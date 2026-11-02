package kit

import "github.com/xmhrnx/giode/styles/properties"

// Alpha returns c with its alpha multiplied by a.
func Alpha(c properties.Color, a float32) properties.Color {
	nc := properties.CalcColor(c)
	nc.A = uint8(float32(nc.A) * a)
	return properties.CalcColorReverse(nc)
}
