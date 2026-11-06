package text

import (
	"image"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements/internal/base"
	"github.com/xmhrnx/giode/internal/textrender"
	"github.com/xmhrnx/giode/styles/properties"
)

// selAreaStack holds the selection areas being laid out.
var selAreaStack []*selArea

func init() {
	base.SelectionLen = func() int {
		if n := len(selAreaStack); n > 0 && selAreaStack[n-1].collecting {
			return len(selAreaStack[n-1].parts)
		}
		return -1
	}
	base.SelectionReorder = func(spans [][2]int) {
		if n := len(selAreaStack); n > 0 {
			selAreaStack[n-1].reorder(spans)
		}
	}
	base.SelectionVersion = selectionVersion
	base.InSelectionArea = func() bool { return len(selAreaStack) > 0 }
}

// selectionVersion is the version of the innermost selection area's
// selection, for caches inside it to notice a change.
func selectionVersion() uint64 {
	if n := len(selAreaStack); n > 0 {
		return selAreaStack[n-1].version
	}
	return 0
}

// registerSelectable adds a laid out text element of the given size
// to the innermost selection area, and paints the part of it that's
// selected, in the selection color for col. It reports whether there
// is an area.
func registerSelectable(gtx layout.Context, text string, size image.Point, stops func() []textrender.StopLine, col properties.Color) bool {
	if len(selAreaStack) == 0 || !kit.TextSelectable() || base.Measuring() {
		return false
	}
	a := selAreaStack[len(selAreaStack)-1]
	p := &selPart{size: size, text: text, stops: stops}
	base.TrackPosition(&p.origin)
	idx := a.placeOf(len(a.order))
	add := func() {
		if a.collecting {
			a.order = append(a.order, len(a.order))
			a.parts = append(a.parts, p)
		}
	}
	add()
	// A cache replaying this text registers it again.
	base.RetainKeyed(add)
	if lo, hi := a.rangeOf(idx, text); lo < hi {
		c := properties.CalcColor(selectionColor(col))
		for _, r := range textrender.RangeRects(p.stopLines(), lo, hi) {
			cl := clip.Rect(r).Push(gtx.Ops)
			paint.Fill(gtx.Ops, c)
			cl.Pop()
		}
	}
	cl := clip.Rect{Max: size}.Push(gtx.Ops)
	pointer.CursorText.Add(gtx.Ops)
	cl.Pop()
	return true
}

// runeSlice returns the runes [lo, hi) of str.
func runeSlice(str string, lo, hi int) string {
	start, end := len(str), len(str)
	n := 0
	for i := range str {
		if n == lo {
			start = i
		}
		if n == hi {
			end = i
			break
		}
		n++
	}
	if start > end {
		return ""
	}
	return str[start:end]
}
