package box

import (
	"image"

	"gioui.org/layout"

	"github.com/xmhrnx/giode/elements/internal/base"
)

// wrapStretch measures a wrapping container's children, breaks them
// into lines as the layout will, and returns each child's cross size:
// its line's thickest.
func wrapStretch(gtx layout.Context, children []base.Element, axis layout.Axis, mainMax, gap int) []int {
	mgtx := gtx
	if axis == layout.Horizontal {
		mgtx.Constraints = layout.Constraints{Max: image.Pt(mainMax, gtx.Constraints.Max.Y)}
	} else {
		mgtx.Constraints = layout.Constraints{Max: image.Pt(gtx.Constraints.Max.X, mainMax)}
	}
	out := make([]int, len(children))
	start, used, thick := 0, 0, 0
	end := func(i int) {
		for j := start; j < i; j++ {
			out[j] = thick
		}
	}
	for i, c := range children {
		el := c
		if pct := base.FlexPercentOf(c, axis); pct > 0 {
			el = &mainSized{el: c, axis: axis, size: mainMax * pct / 100}
		}
		d := base.Measure(mgtx, el)
		m, cr := axisMain(d.Size, axis), axisCross(d.Size, axis)
		add := m
		if i > start {
			add += gap
		}
		if i > start && used+add > mainMax {
			end(i)
			start, used, thick, add = i, 0, 0, m
		}
		used += add
		thick = max(thick, cr)
	}
	end(len(children))
	return out
}
