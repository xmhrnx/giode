package progress

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/styles/properties"

	"github.com/xmhrnx/giode/styles"
)

// ProgressElement renders the track and fill.
type ProgressElement struct {
	value float32
	st    styles.Styles

	kit.HoverTracker
	hoverSt styles.Styles
}

// OnHover sets styles applied while the pointer is over the bar,
// merged on top of its base styles. Hover state lives in the element,
// so create it once and reuse it every frame; rebuilding it in the
// view function resets the hover on each frame.
func (p *ProgressElement) OnHover(st styles.Styles) *ProgressElement {
	p.Enable()
	p.hoverSt = st
	return p
}

// Layout renders the bar.
func (p *ProgressElement) Layout(gtx layout.Context) layout.Dimensions {
	p.Update(gtx)
	st := p.st
	if p.Enabled() && p.Hovered() {
		st = styles.Merge(p.st, p.hoverSt)
	}

	st = st.Px(gtx.Metric)
	height := int(st.Height)
	if height <= 0 {
		height = gtx.Dp(8)
	}
	track := st.Background
	if properties.CalcColor(track).A == 0 {
		track = properties.CalcColorReverse(defaultTrack)
	}
	fill := st.Color
	if properties.CalcColor(fill).A == 0 {
		fill = properties.CalcColorReverse(defaultFill)
	}
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	size := image.Pt(gtx.Constraints.Min.X, height)

	defer clip.UniformRRect(image.Rectangle{Max: size}, height/2).Push(gtx.Ops).Pop()
	paint.Fill(gtx.Ops, properties.CalcColor(track))

	v := p.value
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	fillW := int(v * float32(size.X))
	if fillW > height/2 {
		// Right to left, the bar fills from the right.
		defer kit.MirrorRTL(gtx, size.X)()
		defer clip.UniformRRect(image.Rect(0, 0, fillW, height), height/2).Push(gtx.Ops).Pop()
		paint.Fill(gtx.Ops, properties.CalcColor(fill))
	}
	p.RegisterArea(gtx, image.Pt(0, 0), size, st.Cursor)
	return layout.Dimensions{Size: size}
}
