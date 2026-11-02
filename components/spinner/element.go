package spinner

import (
	"image"
	"math"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/styles/properties"

	"github.com/xmhrnx/giode/styles"
)

// SpinnerElement renders the rotating arc.
type SpinnerElement struct {
	st styles.Styles

	kit.HoverTracker
	hoverSt styles.Styles
}

// OnHover sets styles applied while the pointer is over the spinner,
// merged on top of its base styles. Hover state lives in the element,
// so create it once and reuse it every frame; rebuilding it in the
// view function resets the hover on each frame.
func (s *SpinnerElement) OnHover(st styles.Styles) *SpinnerElement {
	s.Enable()
	s.hoverSt = st
	return s
}

// Layout draws one frame of the animation and requests the next.
func (s *SpinnerElement) Layout(gtx layout.Context) layout.Dimensions {
	s.Update(gtx)
	st := s.st
	if s.Enabled() && s.Hovered() {
		st = styles.Merge(s.st, s.hoverSt)
	}

	st = st.Px(gtx.Metric)
	sizeX := int(st.Width)
	if sizeX <= 0 {
		sizeX = gtx.Dp(32)
	}
	sizeY := int(st.Height)
	if sizeY <= 0 {
		sizeY = sizeX
	}
	size := gtx.Constraints.Constrain(image.Pt(sizeX, sizeY))
	if size.X <= 0 || size.Y <= 0 {
		return layout.Dimensions{}
	}
	radius := float32(size.X) / 2
	if float32(size.Y)/2 < radius {
		radius = float32(size.Y) / 2
	}
	c := properties.CalcColor(st.Color)
	if c.A == 0 {
		c = defaultColor
	}

	off := op.Offset(image.Pt(size.X/2, size.Y/2)).Push(gtx.Ops)

	dt := float32((time.Duration(gtx.Now.UnixNano()) % time.Second).Seconds())
	startAngle := dt * math.Pi * 2
	endAngle := startAngle + math.Pi*1.5

	arcClip := clipArc(gtx.Ops, startAngle, endAngle, radius).Push(gtx.Ops)
	paintFill(gtx.Ops, c)
	arcClip.Pop()
	off.Pop()

	s.RegisterArea(gtx, image.Pt(0, 0), size, st.Cursor)

	// Keep animating while visible.
	kit.RequestFrame(gtx)
	return layout.Dimensions{Size: size}
}
