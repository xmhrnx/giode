package markdown

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/xmhrnx/giode/components/scroll"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// quote draws a block quote: content indented past a bar as tall as
// the content.
func quote(content elements.Element) elements.Element {
	return elements.Raw(func(gtx layout.Context) layout.Dimensions {
		bar, gap := gtx.Dp(3), gtx.Dp(10)
		cgtx := gtx
		cgtx.Constraints.Min.X = max(0, cgtx.Constraints.Min.X-bar-gap)
		cgtx.Constraints.Max.X = max(0, cgtx.Constraints.Max.X-bar-gap)
		off := image.Pt(bar+gap, 0)
		track := elements.TrackChild()
		macro := op.Record(gtx.Ops)
		dims := content.Layout(cgtx)
		call := macro.Stop()
		track.End()
		track.Place(off)

		func() {
			defer clip.UniformRRect(image.Rect(0, 0, bar, dims.Size.Y), bar/2).Push(gtx.Ops).Pop()
			paint.Fill(gtx.Ops, properties.CalcColor("theme.border"))
		}()
		func() {
			defer op.Offset(off).Push(gtx.Ops).Pop()
			call.Add(gtx.Ops)
		}()
		return layout.Dimensions{Size: image.Pt(dims.Size.X+bar+gap, dims.Size.Y), Baseline: dims.Baseline}
	})
}

// hscroll is a horizontal scroll kept across frames for one code block
// or table; el is what it shows this frame.
type hscroll struct {
	s  *scroll.Scroll
	el elements.Element
}

// sideways lets el scroll horizontally when it's wider than the
// space, as code and tables are on a phone, instead of wrapping or
// squeezing. key identifies the block across frames.
func sideways(key string, el elements.Element) elements.Element {
	h := elements.Keyed("giode.markdown.scroll", key, func() *hscroll {
		h := &hscroll{}
		h.s = scroll.New(func() elements.Element { return h.el }, styles.Styles{ScrollBar: properties.ScrollBar{Width: 4}}).Axis(scroll.Horizontal)
		return h
	})
	h.el = el
	return h.s
}
