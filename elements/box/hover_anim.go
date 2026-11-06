package box

import (
	"time"

	"gioui.org/gesture"
	"gioui.org/layout"

	"github.com/xmhrnx/giode/styles"
)

// boxHover is a hoverable Box's hover state: its hover styles, the
// last observed hover, its input pass (see kit.HandleInput) and its
// transition.
type boxHover struct {
	gesture.Hover
	st      styles.Styles
	hovered bool
	pass    func(gtx layout.Context) bool
	hoverAnim
}

// hoverAnim is the transition state of a hoverable Box.
type hoverAnim struct {
	progress  float32
	animating bool
	lastFrame time.Time
}
