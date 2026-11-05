package box

import (
	"gioui.org/layout"

	"github.com/xmhrnx/giode/styles/properties"
)

// insetConstraints shrinks c by the inset on each side. Min shrinks by
// the same total inset as Max (clamped at 0), so an exact constraint
// (Min == Max, from a fixed or filled size) stays exact after the
// inset is removed instead of drifting.
func insetConstraints(c layout.Constraints, i properties.Inset) layout.Constraints {
	c.Min.X -= i.Left + i.Right
	c.Max.X -= i.Left + i.Right
	c.Min.Y -= i.Top + i.Bottom
	c.Max.Y -= i.Top + i.Bottom
	if c.Min.X < 0 {
		c.Min.X = 0
	}
	if c.Min.Y < 0 {
		c.Min.Y = 0
	}
	if c.Max.X < c.Min.X {
		c.Max.X = c.Min.X
	}
	if c.Max.Y < c.Min.Y {
		c.Max.Y = c.Min.Y
	}
	return c
}

// minConstraint returns the main-axis minimum of c.
func minConstraint(c layout.Constraints, axis layout.Axis) int {
	if axis == layout.Horizontal {
		return c.Min.X
	}
	return c.Min.Y
}

// mainConstraint returns the main-axis maximum of c.
func mainConstraint(c layout.Constraints, axis layout.Axis) int {
	if axis == layout.Horizontal {
		return c.Max.X
	}
	return c.Max.Y
}
