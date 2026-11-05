package input

import (
	"image"
	"sort"
	"unicode"

	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"
	"github.com/xmhrnx/giode/styles/properties"
)

// handle is a draggable selection handle.
type handle struct {
	drag gesture.Drag
	// at is the handle's window position in the last frame: the caret
	// point it hangs from.
	at image.Point
	// grab is the offset from the finger to that point.
	grab image.Point
}

// dragHandles moves the selection ends with their handles.
func (m *editMenu) dragHandles(gtx layout.Context, in *Input) bool {
	if !m.handles {
		return false
	}
	ed := &in.editor
	changed := false
	for i, h := range []*handle{&m.start, &m.end} {
		for {
			ev, ok := h.drag.Update(gtx.Metric, gtx.Source, gesture.Both)
			if !ok {
				break
			}
			// Positions are relative to where the handle was drawn.
			finger := h.at.Add(ev.Position.Round())
			switch ev.Kind {
			case pointer.Press:
				h.grab = h.at.Sub(finger)
			case pointer.Drag:
				p := finger.Add(h.grab).Sub(m.origin)
				// Aim at the line above the handle's point.
				p.Y -= gtx.Dp(4)
				c := caretAt(ed, p)
				s, e := ed.Selection()
				lo, hi := min(s, e), max(s, e)
				if i == 0 {
					lo = min(c, hi)
				} else {
					hi = max(c, lo)
				}
				if lo != min(s, e) || hi != max(s, e) {
					ed.SetCaret(hi, lo)
					changed = true
				}
			}
		}
	}
	return changed
}

// layoutHandle draws a selection handle hanging from h.at and registers
// its drag area.
func (m *editMenu) layoutHandle(gtx layout.Context, h *handle) {
	r := gtx.Dp(9)
	hit := gtx.Dp(20)
	defer op.Offset(h.at).Push(gtx.Ops).Pop()
	col := properties.ResolveColor("theme.primary")
	// A stem from the text down to a round knob.
	stem := clip.Rect{Min: image.Pt(-gtx.Dp(1), 0), Max: image.Pt(gtx.Dp(1), r)}.Push(gtx.Ops)
	paint.Fill(gtx.Ops, col)
	stem.Pop()
	knob := clip.Ellipse{Min: image.Pt(-r, r), Max: image.Pt(r, 3*r)}.Op(gtx.Ops).Push(gtx.Ops)
	paint.Fill(gtx.Ops, col)
	knob.Pop()
	area := clip.Rect{Min: image.Pt(-hit, 0), Max: image.Pt(hit, 2*r+2*hit)}.Push(gtx.Ops)
	properties.CursorGrab.CursorFor().Add(gtx.Ops)
	h.drag.Add(gtx.Ops)
	area.Pop()
}

// caretAt returns the caret position in ed nearest p, in the editor's
// coordinates, among the runes in view.
func caretAt(ed *widget.Editor, p image.Point) int {
	n := ed.Len()
	if n == 0 {
		return 0
	}
	var buf []widget.Region
	visible := func(a, b int) bool {
		buf = ed.Regions(a, b, buf[:0])
		return len(buf) > 0
	}
	if !visible(0, n) {
		s, _ := ed.Selection()
		return s
	}
	// The runes in view, found by bisection: whether any rune up to i
	// (from i) is in view only changes once.
	first := sort.Search(n, func(i int) bool { return visible(0, i+1) })
	last := sort.Search(n, func(i int) bool { return !visible(i, n) }) - 1
	best, bestD := first, -1
	for i := first; i <= last; i++ {
		buf = ed.Regions(i, i+1, buf[:0])
		if len(buf) == 0 {
			continue
		}
		r := buf[0].Bounds
		dy := 0
		switch {
		case p.Y < r.Min.Y:
			dy = r.Min.Y - p.Y
		case p.Y >= r.Max.Y:
			dy = p.Y - r.Max.Y + 1
		}
		for k, x := range [2]int{r.Min.X, r.Max.X} {
			dx := p.X - x
			if dx < 0 {
				dx = -dx
			}
			// The line matters first, then the column.
			if d := dy<<20 + dx; bestD < 0 || d < bestD {
				best, bestD = i+k, d
			}
		}
	}
	return best
}

// wordAt returns the word around caret position c in rs.
func wordAt(rs []rune, c int) (start, end int, ok bool) {
	isWord := func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r) || r == '_' }
	i := min(c, len(rs))
	switch {
	case i < len(rs) && isWord(rs[i]):
	case i > 0 && isWord(rs[i-1]):
		i--
	default:
		return 0, 0, false
	}
	start, end = i, i
	for start > 0 && isWord(rs[start-1]) {
		start--
	}
	for end < len(rs) && isWord(rs[end]) {
		end++
	}
	return start, end, true
}
