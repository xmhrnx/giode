package input

import (
	"image"
	"sort"
	"time"
	"unicode"

	"gioui.org/f32"
	"gioui.org/gesture"
	"gioui.org/io/clipboard"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/i18n"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// longPress is how long a finger rests on a field to open its menu.
const longPress = 500 * time.Millisecond

// editMenu is a field's touch editing: a long-press places the caret or
// selects the word under the finger and opens a menu - Cut, Copy,
// Paste, Select all - with handles to adjust the selection. A
// right-click opens the same menu with a mouse.
type editMenu struct {
	// tag receives presses over the editor, passed through to it.
	tag int

	pressing bool
	pressAt  time.Time
	pressPos f32.Point
	pressID  pointer.ID

	// open is set while the menu shows; handles while the selection
	// handles do (touch only).
	open, handles bool
	// hadFocus is set once the field had focus while the menu showed,
	// so losing it closes the menu.
	hadFocus bool
	// origin is the editor's window position, exact from the last
	// frame.
	origin image.Point

	// outside observes presses anywhere, to close the menu.
	outside int
	buttons map[string]*elements.Pressable
	action  string

	start, end handle
}

// handle is a draggable selection handle.
type handle struct {
	drag gesture.Drag
	// at is the handle's window position in the last frame: the caret
	// point it hangs from.
	at image.Point
	// grab is the offset from the finger to that point.
	grab image.Point
}

// update processes the menu's events and reports whether the field's
// look changed.
func (m *editMenu) update(gtx layout.Context, in *Input) bool {
	changed := false
	ed := &in.editor
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &m.tag, Kinds: pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		switch {
		case e.Kind == pointer.Cancel:
			// The editor or a list took the pointer to scroll. Gio's
			// cancel carries no source.
			m.pressing = false
		case e.Kind == pointer.Press && e.Source == pointer.Touch:
			if m.open {
				// A tap in the field closes the menu; the editor
				// places the caret.
				m.close()
				changed = true
			}
			m.pressing, m.pressAt, m.pressPos, m.pressID = true, gtx.Now, e.Position, e.PointerID
			kit.RequestFrameAt(gtx, gtx.Now.Add(longPress))
		case e.Kind == pointer.Press && e.Buttons.Contain(pointer.ButtonSecondary):
			// The desktop context menu: keep a selection the click is
			// in, otherwise place the caret.
			p := e.Position.Round()
			c := caretAt(ed, p)
			if s, t := ed.Selection(); c < min(s, t) || c > max(s, t) || s == t {
				ed.SetCaret(c, c)
			}
			m.focus(gtx, in)
			m.open, m.handles = true, false
			changed = true
		case e.Kind == pointer.Drag && m.pressing:
			d := e.Position.Sub(m.pressPos)
			if slop := float32(gtx.Dp(8)); d.X*d.X+d.Y*d.Y > slop*slop {
				m.pressing = false
			}
		case e.Kind == pointer.Release:
			m.pressing = false
		}
	}
	if m.pressing && kit.Scrolling() {
		m.pressing = false
	}
	if m.pressing {
		if gtx.Now.Sub(m.pressAt) >= longPress {
			m.pressing = false
			m.longPress(gtx, in)
			changed = true
		} else {
			kit.RequestFrameAt(gtx, m.pressAt.Add(longPress))
		}
	}

	if m.open {
		// A press anywhere outside the menu and handles closes it.
		for {
			ev, ok := gtx.Event(pointer.Filter{Target: &m.outside, Kinds: pointer.Press})
			if !ok {
				break
			}
			if _, ok := ev.(pointer.Event); ok {
				m.close()
				changed = true
			}
		}
		if m.dragHandles(gtx, in) {
			changed = true
		}
		if m.action != "" {
			m.do(gtx, in, m.action)
			m.action = ""
			changed = true
		}
		if gtx.Source.Focused(ed) {
			m.hadFocus = true
		} else if m.hadFocus {
			m.close()
			changed = true
		}
	}
	return changed
}

// longPress selects the word under the finger, or places the caret
// where there's none, and opens the menu.
func (m *editMenu) longPress(gtx layout.Context, in *Input) {
	ed := &in.editor
	c := caretAt(ed, m.pressPos.Round())
	if ed.Mask == 0 {
		if s, e, ok := wordAt([]rune(ed.Text()), c); ok {
			ed.SetCaret(s, e)
		} else {
			ed.SetCaret(c, c)
		}
	} else {
		// A password field shows no words.
		ed.SetCaret(c, c)
	}
	m.focus(gtx, in)
	// Keep the finger's release from reaching the editor as a tap,
	// which would collapse the selection.
	gtx.Execute(pointer.GrabCmd{Tag: &m.tag, ID: m.pressID})
	m.open, m.handles = true, true
}

func (m *editMenu) focus(gtx layout.Context, in *Input) {
	gtx.Execute(key.FocusCmd{Tag: &in.editor})
	if !in.editor.ReadOnly {
		gtx.Execute(key.SoftKeyboardCmd{Show: true})
	}
}

func (m *editMenu) close() {
	m.open, m.handles, m.hadFocus = false, false, false
}

// do runs a menu action.
func (m *editMenu) do(gtx layout.Context, in *Input, action string) {
	ed := &in.editor
	switch action {
	case "cut":
		if t := ed.SelectedText(); t != "" && ed.Mask == 0 {
			kit.WriteClipboard(gtx, t)
			ed.Delete(1)
		}
		m.close()
	case "copy":
		if t := ed.SelectedText(); t != "" && ed.Mask == 0 {
			kit.WriteClipboard(gtx, t)
		}
		s, e := ed.Selection()
		c := max(s, e)
		ed.SetCaret(c, c)
		m.close()
	case "paste":
		// The editor inserts the clipboard's text when it arrives.
		gtx.Execute(clipboard.ReadCmd{Tag: ed})
		m.close()
	case "select_all":
		ed.SetCaret(ed.Len(), 0)
		// Stay open for Cut or Copy.
	}
	kit.RequestFrame(gtx)
}

// actions returns the menu's actions for the field's state.
func (m *editMenu) actions(in *Input) []string {
	ed := &in.editor
	var out []string
	sel := ed.SelectionLen() > 0
	if sel && !ed.ReadOnly && ed.Mask == 0 {
		out = append(out, "cut")
	}
	if sel && ed.Mask == 0 {
		out = append(out, "copy")
	}
	if !ed.ReadOnly {
		out = append(out, "paste")
	}
	if n := ed.Len(); n > 0 && ed.SelectionLen() < n {
		out = append(out, "select_all")
	}
	return out
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

// layout registers the long-press area over the editor, size dims, and
// the menu and handles while they show. Call it right after laying the
// editor out.
func (m *editMenu) layout(gtx layout.Context, in *Input, size image.Point) {
	elements.TrackPosition(&m.origin)
	ps := pointer.PassOp{}.Push(gtx.Ops)
	cl := clip.Rect{Max: size}.Push(gtx.Ops)
	event.Op(gtx.Ops, &m.tag)
	cl.Pop()
	ps.Pop()
	if !m.open {
		return
	}
	ed := &in.editor
	acts := m.actions(in)
	s, e := ed.Selection()
	lo, hi := min(s, e), max(s, e)
	// The selection's (or caret's) rectangles, in window coordinates.
	origin := m.origin
	var bounds image.Rectangle
	var startPt, endPt image.Point
	var have bool
	regions := ed.Regions(lo, hi, nil)
	for i, r := range regions {
		b := r.Bounds.Add(origin)
		if i == 0 {
			bounds = b
			startPt = image.Pt(b.Min.X, b.Max.Y)
		} else {
			bounds = bounds.Union(b)
		}
		endPt = image.Pt(b.Max.X, b.Max.Y)
		have = true
	}
	if !have {
		c := ed.CaretCoords().Round().Add(origin)
		lh := gtx.Sp(properties.ResolveFontSize(in.st.FontSize))
		bounds = image.Rect(c.X, c.Y-lh, c.X+1, c.Y+lh/4)
		startPt, endPt = image.Pt(c.X, bounds.Max.Y), image.Pt(c.X, bounds.Max.Y)
	}
	showHandles := m.handles && lo != hi && len(regions) > 0
	dragging := m.start.drag.Dragging() || m.end.drag.Dragging()
	if m.buttons == nil {
		m.buttons = map[string]*elements.Pressable{}
	}
	elements.RegisterOverlay(func() elements.Element {
		return elements.Raw(func(gtx layout.Context) layout.Dimensions {
			win := gtx.Constraints.Max
			// Watch presses everywhere without taking them.
			ps := pointer.PassOp{}.Push(gtx.Ops)
			cl := clip.Rect{Max: win}.Push(gtx.Ops)
			event.Op(gtx.Ops, &m.outside)
			cl.Pop()
			ps.Pop()

			if showHandles {
				m.start.at, m.end.at = startPt, endPt
				m.layoutHandle(gtx, &m.start)
				m.layoutHandle(gtx, &m.end)
			}
			if len(acts) > 0 && !dragging {
				m.layoutBar(gtx, acts, bounds, showHandles)
			}
			if m.action != "" {
				m.do(gtx, in, m.action)
				m.action = ""
			}
			return layout.Dimensions{Size: win}
		})
	})
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
	pointer.CursorGrab.Add(gtx.Ops)
	h.drag.Add(gtx.Ops)
	area.Pop()
}

// layoutBar draws the action bar above the selection, or below it when
// there's no room above.
func (m *editMenu) layoutBar(gtx layout.Context, acts []string, sel image.Rectangle, handles bool) {
	win := gtx.Constraints.Max
	items := make([]elements.Element, 0, len(acts))
	for _, a := range acts {
		btn, ok := m.buttons[a]
		if !ok {
			action := a
			btn = elements.NewPressable(func(p elements.PressState) elements.Element {
				bg := ""
				if p.Pressed || p.Hovered {
					bg = "#374151"
				}
				return elements.Box(styles.Styles{Background: properties.Color(bg), BorderRadius: 6, Padding: properties.SymmetricInset(12, 8)},
					elements.Text(i18n.T("giode."+action), styles.Styles{Color: "#f9fafb", FontSize: 14}))
			}).Label(i18n.T("giode." + a)).OnClick(func(elements.ClickEvent) { m.action = action })
			m.buttons[a] = btn
		}
		items = append(items, btn)
	}
	bar := elements.Box(styles.Styles{Background: "#1f2937", BorderRadius: 8, Padding: properties.UniformInset(2),
		BoxShadow: properties.BoxShadow{Y: 2, Blur: 8, Color: "#00000040"}}, items...)

	track := elements.TrackChild()
	macro := op.Record(gtx.Ops)
	cgtx := gtx
	cgtx.Constraints.Min = image.Point{}
	dims := bar.Layout(cgtx)
	call := macro.Stop()
	track.End()

	gap := gtx.Dp(8)
	pos := image.Pt((sel.Min.X+sel.Max.X)/2-dims.Size.X/2, sel.Min.Y-dims.Size.Y-gap)
	if pos.Y < gap {
		below := sel.Max.Y + gap
		if handles {
			below += gtx.Dp(30) // clear of the handles
		}
		pos.Y = below
	}
	pos.X = max(gap, min(pos.X, win.X-dims.Size.X-gap))
	track.Place(pos)
	defer op.Offset(pos).Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
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
