package box

import (
	"image"

	"gioui.org/layout"

	"github.com/xmhrnx/giode/elements/internal/base"
)

// stretchedElement lays an element out with stretchChild.
type stretchedElement struct {
	child base.Element
	box   base.Element
	axis  layout.Axis
}

// stretchCross raises the cross-axis minimum to the largest child's
// cross size, measured first, for AlignStretch: Gio's flex passes that
// minimum to every child, so all of them grow to the largest.
func stretchCross(gtx layout.Context, children []base.Element, axis layout.Axis, mainMax int) layout.Constraints {
	cs := gtx.Constraints
	mgtx := gtx
	if axis == layout.Horizontal {
		mgtx.Constraints = layout.Constraints{Max: image.Pt(mainMax, cs.Max.Y)}
	} else {
		mgtx.Constraints = layout.Constraints{Max: image.Pt(cs.Max.X, mainMax)}
	}
	cross := 0
	for _, c := range children {
		d := base.Measure(mgtx, c)
		if axis == layout.Horizontal {
			cross = max(cross, d.Size.Y)
		} else {
			cross = max(cross, d.Size.X)
		}
	}
	if axis == layout.Horizontal {
		cs.Min.Y = max(cs.Min.Y, min(cross, cs.Max.Y))
	} else {
		cs.Min.X = max(cs.Min.X, min(cross, cs.Max.X))
	}
	return cs
}

// stretchChild lays a child of an AlignStretch container out at exactly
// the cross size stretchCross settled on: a Box without its own cross
// size fills it, as an element sized by its content wouldn't.
func stretchChild(child base.Element, w layout.Widget, axis layout.Axis) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		box, _ := child.(*BoxElement)
		if axis == layout.Horizontal {
			if gtx.Constraints.Min.Y > 0 {
				gtx.Constraints.Max.Y = gtx.Constraints.Min.Y
				if box != nil {
					box.stretchY = true
					defer func() { box.stretchY = false }()
				}
			}
		} else if gtx.Constraints.Min.X > 0 {
			gtx.Constraints.Max.X = gtx.Constraints.Min.X
			if box != nil {
				box.stretchX = true
				defer func() { box.stretchX = false }()
			}
		}
		return w(gtx)
	}
}

func (s stretchedElement) Layout(gtx layout.Context) layout.Dimensions {
	return stretchChild(s.box, s.child.Layout, s.axis)(gtx)
}
