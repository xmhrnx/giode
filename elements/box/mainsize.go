package box

import (
	"gioui.org/layout"

	"github.com/xmhrnx/giode/elements/internal/base"
)

// mainSized wraps an element to force its size on the flex main axis,
// used for percentage-sized flex children.
type mainSized struct {
	el   base.Element
	axis layout.Axis
	size int
}

func (s *mainSized) Layout(gtx layout.Context) layout.Dimensions {
	if s.axis == layout.Horizontal {
		gtx.Constraints.Min.X = s.size
		gtx.Constraints.Max.X = s.size
	} else {
		gtx.Constraints.Min.Y = s.size
		gtx.Constraints.Max.Y = s.size
	}
	return s.el.Layout(gtx)
}
