package box

import (
	"gioui.org/layout"
)

// unstretchCross lets a child size itself on the cross axis.
func unstretchCross(w layout.Widget, axis layout.Axis) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		if axis == layout.Horizontal {
			gtx.Constraints.Min.Y = 0
		} else {
			gtx.Constraints.Min.X = 0
		}
		return w(gtx)
	}
}
