package menu

import (
	"image"

	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles/properties"
)

// scrollState scrolls a menu whose rows are taller than the window
// leaves room for: with the wheel, a touch drag, and the keyboard
// (see keys.go), which scrolls the highlighted row into view.
type scrollState struct {
	gesture gesture.Scroll
	// offset is how far the rows are scrolled, in pixels.
	offset int
	// tops and heights are each row's position and height in the last
	// frame, in content pixels; contentH is their total and viewH the
	// visible height.
	tops, heights   []int
	contentH, viewH int
	// reveal asks the next frame to scroll the active row into view.
	reveal bool
}

// layoutRows lays the rows out one under the other, scrolled and
// clipped to the height the panel has, with a thin indicator of the
// position when they don't fit.
func (m *Menu) layoutRows(gtx layout.Context, rows []elements.Element, thumb properties.Color) layout.Dimensions {
	s := &m.scroll
	avail := gtx.Constraints.Max.Y
	s.viewH = avail
	maxOff := max(0, s.contentH-avail)

	if maxOff > 0 && gtx.Source.Enabled() {
		dist := s.gesture.Update(gtx.Metric, gtx.Source, gtx.Now, gesture.Vertical,
			pointer.ScrollRange{}, pointer.ScrollRange{Min: -s.offset, Max: maxOff - s.offset})
		s.offset += dist
		if s.gesture.State() != gesture.StateIdle {
			kit.RequestFrame(gtx)
		}
	}
	if s.reveal && m.active >= 0 && m.active < len(s.tops) {
		top, bottom := s.tops[m.active], s.tops[m.active]+s.heights[m.active]
		switch {
		case top < s.offset:
			s.offset = top
		case bottom > s.offset+avail:
			s.offset = bottom - avail
		}
	}
	s.reveal = false
	s.offset = max(0, min(s.offset, maxOff))

	// The rows, recorded at their scrolled positions so the clip below
	// hides the ones outside, and so submenus anchor at their rows'
	// true positions.
	macro := op.Record(gtx.Ops)
	rgtx := gtx
	rgtx.Constraints.Min.Y = 0
	rgtx.Constraints.Max.Y = 1 << 24
	s.tops, s.heights = s.tops[:0], s.heights[:0]
	var size image.Point
	for _, row := range rows {
		off := image.Pt(0, size.Y-s.offset)
		trans := op.Offset(off).Push(gtx.Ops)
		pop := elements.PushOffset(off)
		dims := row.Layout(rgtx)
		pop()
		trans.Pop()
		s.tops = append(s.tops, size.Y)
		s.heights = append(s.heights, dims.Size.Y)
		size.X = max(size.X, dims.Size.X)
		size.Y += dims.Size.Y
	}
	call := macro.Stop()
	if s.contentH != size.Y {
		s.contentH = size.Y
		kit.RequestFrame(gtx)
	}
	view := image.Pt(size.X, min(size.Y, avail))

	defer clip.Rect{Max: view}.Push(gtx.Ops).Pop()
	if size.Y > avail {
		// Under the rows: the wheel and drags scroll, while the rows
		// keep their own hover and clicks.
		s.gesture.Add(gtx.Ops)
	}
	call.Add(gtx.Ops)
	if size.Y > avail && avail > 0 {
		paintThumb(gtx, view, s.offset, size.Y, thumb)
	}
	return layout.Dimensions{Size: view}
}

// paintThumb draws a thin position indicator along the end edge.
func paintThumb(gtx layout.Context, view image.Point, offset, contentH int, c properties.Color) {
	w := gtx.Dp(3)
	h := max(gtx.Dp(24), view.Y*view.Y/contentH)
	y := offset * (view.Y - h) / max(1, contentH-view.Y)
	x := view.X - w - gtx.Dp(1)
	if kit.RTL(gtx) {
		x = gtx.Dp(1)
	}
	r := image.Rect(x, y, x+w, y+h)
	defer clip.UniformRRect(r, w/2).Push(gtx.Ops).Pop()
	paint.Fill(gtx.Ops, properties.CalcColor(kit.Alpha(c, 0.35)))
}
