package elements

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/styles/properties"

	"github.com/xmhrnx/giode/styles"
)

var defaultDividerColor = color.NRGBA{R: 0x33, G: 0x41, B: 0x55, A: 0xff}

// DividerElement is a thin horizontal line. Chain OnHover to add
// hover styles.
type DividerElement struct {
	st styles.Styles

	kit.HoverTracker
	hoverSt styles.Styles
}

// Divider is a thin horizontal line. Its color defaults to a subtle
// gray and its height to 1px; both can be overridden with styles.
func Divider(st ...styles.Styles) *DividerElement {
	var s styles.Styles
	if len(st) > 0 {
		s = st[0]
	}
	return &DividerElement{st: s}
}

// OnHover sets styles applied while the pointer is over the divider,
// merged on top of its base styles. Hover state lives in the element,
// so create it once and reuse it every frame; rebuilding it in the
// view function resets the hover on each frame.
func (d *DividerElement) OnHover(st styles.Styles) *DividerElement {
	d.Enable()
	d.hoverSt = st
	return d
}

func (d *DividerElement) Layout(gtx layout.Context) layout.Dimensions {
	d.Update(gtx)
	st := d.st
	if d.Enabled() && d.Hovered() {
		st = styles.Merge(d.st, d.hoverSt)
	}
	if st.Display == properties.DisplayNone {
		return layout.Dimensions{}
	}
	st = st.Px(gtx.Metric)

	height := int(st.Height)
	if height <= 0 {
		height = max(1, gtx.Dp(1))
	}
	width := gtx.Constraints.Max.X
	if width < gtx.Constraints.Min.X {
		width = gtx.Constraints.Min.X
	}
	size := image.Pt(width, height)
	c := properties.CalcColor(st.Color)
	if c.A == 0 {
		c = defaultDividerColor
	}
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	paint.Fill(gtx.Ops, c)
	d.RegisterArea(gtx, image.Pt(0, 0), size, st.Cursor)
	return layout.Dimensions{Size: size}
}
