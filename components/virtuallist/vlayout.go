// Package virtuallist provides a vertical list that only builds the
// rows in or near its viewport, so it handles any number of items at a
// constant cost per frame.

package virtuallist

import (
	"image"
	"math"

	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles/properties"
)

// Layout lays out the visible rows.
func (l *List) Layout(gtx layout.Context) layout.Dimensions {
	dims := l.layoutSelf(gtx)
	// A right-click or long-press anywhere on it opens its context menu.
	l.AnchorContext(gtx, dims.Size)
	return dims
}

// layoutSelf lays it out without its context menu area.
func (l *List) layoutSelf(gtx layout.Context) layout.Dimensions {
	// Handles its input without an input pass; see kit.OpaqueInput.
	kit.OpaqueInput()
	viewport := gtx.Constraints.Max
	if viewport.X >= 1e6 {
		viewport.X = gtx.Constraints.Min.X
	}
	if viewport.Y >= 1e6 {
		// Unbounded height (e.g. inside a Scroll): show a screenful.
		viewport.Y = max(gtx.Constraints.Min.Y, gtx.Dp(400))
	}
	gtx.Constraints = layout.Exact(viewport)
	l.viewportH = viewport.Y
	l.itemPx = max(1, gtx.Dp(l.itemH))
	l.n = 0
	if l.count != nil {
		l.n = max(0, l.count())
	}
	if l.focusIdx >= l.n {
		l.focusIdx = l.n - 1
	}

	l.handleKeys(gtx)
	l.resolveScroll()

	defer clip.Rect{Max: viewport}.Push(gtx.Ops).Pop()
	if bg := properties.CalcColor(l.st.Background); bg.A > 0 {
		paint.Fill(gtx.Ops, bg)
	}

	// Wheel and touch scrolling, plus press-to-focus, beneath the rows
	// so interactive rows keep priority.
	kit.BlockDragHere(gtx)
	l.scroll.Add(gtx.Ops)
	ps := pointer.PassOp{}.Push(gtx.Ops)
	event.Op(gtx.Ops, &l.focusIdx)
	event.Op(gtx.Ops, l)
	ps.Pop()
	l.offset = l.clamp(l.offset)
	dist := l.scroll.Update(gtx.Metric, gtx.Source, gtx.Now, gesture.Vertical,
		pointer.ScrollRange{},
		pointer.ScrollRange{Min: -int(l.offset), Max: int(l.maxOffset() - l.offset)})
	if dist != 0 {
		l.offset = l.clamp(l.offset + float32(dist))
	}
	l.gest.Track(&l.scroll, dist)
	if dist != 0 || l.scroll.State() != gesture.StateIdle {
		kit.MarkLive()
	}

	// Visible rows.
	l.first, l.last = 0, -1
	if l.n > 0 {
		l.first = int(l.offset) / l.itemPx
		l.last = min(l.n-1, int(math.Ceil(float64(l.offset+float32(viewport.Y))/float64(l.itemPx)))-1)
	}
	from := max(0, l.first-l.overscan)
	to := min(l.n-1, l.last+l.overscan)
	focusColor := l.st.BorderColor
	if focusColor == "" {
		focusColor = defaultFocus
	}
	rowSize := image.Pt(viewport.X, l.itemPx)
	popSem := elements.PushSemanticScroll(l)
	for i := from; i <= to; i++ {
		y := i*l.itemPx - int(l.offset)
		off := op.Offset(image.Pt(0, y)).Push(gtx.Ops)
		pop := elements.PushOffset(image.Pt(0, y))
		if l.focused && i == l.focusIdx {
			paint.FillShape(gtx.Ops, properties.CalcColor(focusColor), clip.Rect{Max: rowSize}.Op())
		}
		rgtx := gtx
		rgtx.Constraints = layout.Exact(rowSize)
		if l.item != nil {
			if el := l.item(i); el != nil {
				el.Layout(rgtx)
			}
		}
		pop()
		off.Pop()
	}
	popSem()
	if l.focused {
		kit.MarkLive()
	}

	atEnd := l.n > 0 && l.last >= l.n-1
	if atEnd && !l.wasAtEnd && l.onReachEnd != nil {
		l.onReachEnd()
	}
	l.wasAtEnd = atEnd

	l.bar.Layout(gtx, viewport, l.n*l.itemPx, &l.offset, l.st.ScrollBar)
	elements.RecordSemantic(gtx, elements.SemanticNode{Kind: elements.SemanticScroll, Target: l}, viewport)
	return layout.Dimensions{Size: viewport}
}
