package box

import (
	"image"

	"gioui.org/layout"
)

// axisName names the length along axis, for diagnostics.
func axisName(axis layout.Axis) string {
	if axis == layout.Horizontal {
		return "width"
	}
	return "height"
}

// axisMain and axisCross read a size along and across axis.
func axisMain(p image.Point, axis layout.Axis) int {
	if axis == layout.Horizontal {
		return p.X
	}
	return p.Y
}

func axisCross(p image.Point, axis layout.Axis) int {
	if axis == layout.Horizontal {
		return p.Y
	}
	return p.X
}

// axisPoint builds a point from a main/cross pair for axis.
func axisPoint(axis layout.Axis, main, cross int) image.Point {
	if axis == layout.Horizontal {
		return image.Pt(main, cross)
	}
	return image.Pt(cross, main)
}
