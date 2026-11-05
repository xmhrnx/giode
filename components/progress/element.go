package progress

import (
	"image"

	"math"

	"gioui.org/layout"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// ProgressElement is the bar: a rounded track box holding a row of the
// fill and the rest, sized by flex growth in proportion to the value.
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

// growScale is the flex growth the fill and the rest share: the value
// in thousandths.
const growScale = 1000

// Layout lays the bar out.
func (p *ProgressElement) Layout(gtx layout.Context) layout.Dimensions {
	p.Update(gtx)
	st := p.st
	if p.Enabled() && p.Hovered() {
		st = styles.Merge(p.st, p.hoverSt)
	}
	dims := p.bar(st).Layout(gtx)
	p.RegisterArea(gtx, image.Point{}, dims.Size, st.Cursor)
	return dims
}

// bar builds the elements for st: the track, and the fill at the start
// of a row, so right to left it fills from the right.
func (p *ProgressElement) bar(st styles.Styles) elements.Element {
	height := st.Height
	if height <= 0 {
		height = 8
	}
	track := st.Background
	if properties.CalcColor(track).A == 0 {
		track = properties.CalcColorReverse(defaultTrack)
	}
	fill := st.Color
	if properties.CalcColor(fill).A == 0 {
		fill = properties.CalcColorReverse(defaultFill)
	}
	v := float64(min(max(p.value, 0), 1))
	grow := int(math.Round(v * growScale))
	radius := properties.BorderRadius(height / 2)
	return elements.Box(styles.Styles{
		Width:        -1,
		Height:       height,
		Background:   track,
		BorderRadius: radius,
		Overflow:     properties.OverflowHidden,
	},
		elements.Row(styles.Styles{Width: -1, Height: -1, Wrap: properties.WrapNoWrap},
			elements.Box(styles.Styles{Height: -1, FlexGrow: properties.FlexGrow(grow), Background: fill, BorderRadius: radius}),
			elements.Box(styles.Styles{Height: -1, FlexGrow: properties.FlexGrow(growScale - grow)}),
		),
	)
}
