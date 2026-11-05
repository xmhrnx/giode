package box

import (
	"github.com/xmhrnx/giode/styles/properties"
)

// justifySpacing splits leftover main-axis space into a leading
// offset and extra per-gap spacing, mirroring Gio's Spacing modes.
func justifySpacing(justify properties.Justify, leftover, n int) (lead, between int) {
	if leftover <= 0 || n <= 0 {
		return 0, 0
	}
	switch justify {
	case properties.JustifyEnd:
		return leftover, 0
	case properties.JustifyCenter:
		return leftover / 2, 0
	case properties.JustifySpaceBetween:
		if n <= 1 {
			return 0, 0
		}
		return 0, leftover / (n - 1)
	case properties.JustifySpaceAround:
		g := leftover / n
		return g / 2, g
	case properties.JustifySpaceEvenly:
		g := leftover / (n + 1)
		return g, g
	default: // JustifyStart
		return 0, 0
	}
}

// alignOffset positions an item of the given cross-axis extent within
// a line of the given thickness.
func alignOffset(align properties.Align, thickness, cross int) int {
	switch align {
	case properties.AlignCenter:
		return (thickness - cross) / 2
	case properties.AlignEnd:
		return thickness - cross
	default:
		return 0
	}
}
