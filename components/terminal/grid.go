package terminal

import (
	"image"
	"image/color"
	"io"
	"math"
	"strings"
	"time"

	"gioui.org/font"
	"gioui.org/io/clipboard"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/io/transfer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"golang.org/x/image/math/fixed"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/pty"
	"github.com/xmhrnx/giode/styles/properties"
)

func ptySize(cols, rows int) pty.Size { return pty.Size{Cols: cols, Rows: rows} }

// metrics are the cell size for the font in use.
type metrics struct {
	px       int
	family   font.Typeface
	cellW    fixed.Int26_6
	cellH    int
	baseline int
	pad      int
}

// x returns where column c starts, from the cells' left edge.
func (m metrics) x(c int) int { return (fixed.Int26_6(c) * m.cellW).Round() }

type paletteKey struct {
	palette *Palette
	mode    properties.Mode
	theme   uint64
}

// rowCache is a row's drawing, kept while the row doesn't change.
type rowCache struct {
	key  uint64
	ops  op.Ops
	call op.CallOp
}

// gridEl is the terminal's content: the cells, inside its KeyboardInput.
type gridEl struct{ t *Terminal }

// measure works out the cell size, and the terminal's size in cells.
func (t *Terminal) measure(gtx layout.Context) {
	px := gtx.Sp(t.fontSize)
	pad := gtx.Dp(t.padding)
	if px != t.m.px || t.family != t.m.family || pad != t.m.pad {
		sh := fonts.Shaper()
		f := font.Font{Typeface: t.family}
		sh.LayoutString(text.Parameters{Font: fonts.Resolve(f), PxPerEm: fixed.I(px), MaxWidth: 1 << 20, MaxLines: 1}, "M")
		m := metrics{px: px, family: t.family, pad: pad, cellW: fixed.I(max(1, px*6/10))}
		asc, desc := px*8/10, px*2/10
		if g, ok := sh.NextGlyph(); ok {
			if g.Advance > 0 {
				m.cellW = g.Advance
			}
			asc, desc = g.Ascent.Ceil(), g.Descent.Ceil()
		}
		for {
			if _, ok := sh.NextGlyph(); !ok {
				break
			}
		}
		m.cellH = max(asc+desc, int(math.Ceil(float64(px)*1.2)))
		m.baseline = asc + (m.cellH-asc-desc)/2
		t.m = m
		t.rowCache = nil
	}
	w, h := gtx.Constraints.Max.X, gtx.Constraints.Max.Y
	if w >= 1<<20 {
		w = t.m.x(80) + 2*pad
	}
	if h >= 1<<20 {
		h = 24*t.m.cellH + 2*pad
	}
	// Dividing two 26.6 numbers gives a plain one.
	cols := max(1, int(fixed.I(w-2*pad)/t.m.cellW))
	rows := max(1, (h-2*pad)/t.m.cellH)
	if cols != t.cols || rows != t.rows || !t.resized {
		t.cols, t.rows, t.resized = cols, rows, true
		if c, r := t.emu.Size(); c != cols || r != rows {
			t.emu.Resize(cols, rows)
			t.mu.Lock()
			p := t.pty
			t.mu.Unlock()
			if p != nil {
				p.Resize(ptySize(cols, rows))
			}
			if t.onResize != nil {
				t.onResize(cols, rows)
			}
		}
	}
}

// Layout draws the cells and handles the pointer.
func (g *gridEl) Layout(gtx layout.Context) layout.Dimensions {
	t := g.t
	size := gtx.Constraints.Max
	if size.X >= 1<<20 {
		size.X = t.m.x(t.cols) + 2*t.m.pad
	}
	if size.Y >= 1<<20 {
		size.Y = t.rows*t.m.cellH + 2*t.m.pad
	}
	gtx.Constraints = layout.Exact(size)
	t.pointer(gtx)
	t.clipboard(gtx)

	pk := paletteKey{palette: t.palette, mode: properties.CurrentMode(), theme: properties.ThemeVersion()}
	if t.pal == nil || pk != t.palKey {
		p := defaultPalette()
		if t.palette != nil {
			p = *t.palette
		}
		t.pal, t.palKey = p.resolve(), pk
		t.rowCache = nil
	}
	snap := t.emu.Snapshot(t.scroll)
	t.scroll = min(t.scroll, snap.Scrollback)
	bg := t.pal.bg
	if snap.Reverse {
		bg = t.pal.fg
	}

	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	paint.Fill(gtx.Ops, bg)
	if len(t.rowCache) != len(snap.Lines) {
		t.rowCache = make([]rowCache, len(snap.Lines))
	}
	for i, l := range snap.Lines {
		rc := &t.rowCache[i]
		k := rowKey(l, snap.Reverse)
		if rc.key != k || rc.key == 0 {
			rc.ops.Reset()
			m := op.Record(&rc.ops)
			t.drawRow(&rc.ops, l, snap.Reverse)
			rc.call, rc.key = m.Stop(), k
		}
		st := op.Offset(image.Pt(t.m.pad, t.m.pad+i*t.m.cellH)).Push(gtx.Ops)
		rc.call.Add(gtx.Ops)
		st.Pop()
	}
	t.drawSelection(gtx, snap)
	t.drawCursor(gtx, snap)
	return layout.Dimensions{Size: size}
}

// rowKey hashes a row's cells (FNV-1a), never 0.
func rowKey(l Line, reverse bool) uint64 {
	h := uint64(14695981039346656037)
	mix := func(v uint64) {
		h ^= v
		h *= 1099511628211
	}
	for _, c := range l.Cells {
		mix(uint64(c.Rune) | uint64(c.Width)<<32 | uint64(c.Style.Attr)<<40)
		mix(uint64(c.Style.FG) | uint64(c.Style.BG)<<32)
	}
	if reverse {
		mix(1)
	}
	return h | 1
}

// drawRow draws a row's backgrounds, text and lines, at the origin.
func (t *Terminal) drawRow(ops *op.Ops, l Line, reverse bool) {
	m := t.m
	cells := l.Cells
	_, defBG := t.pal.colors(Style{}, reverse)
	// Backgrounds, in runs of one color.
	for i := 0; i < len(cells); {
		_, bg := t.pal.colors(cells[i].Style, reverse)
		j := i + 1
		for j < len(cells) {
			_, b := t.pal.colors(cells[j].Style, reverse)
			if b != bg {
				break
			}
			j++
		}
		if bg != defBG {
			fill(ops, image.Rect(m.x(i), 0, m.x(j), m.cellH), bg)
		}
		i = j
	}
	// Text, in runs of plain ASCII of one style; other characters on
	// their own, at their cell, as their font's advance may differ.
	var run strings.Builder
	for i := 0; i < len(cells); {
		c := cells[i]
		if c.Rune == 0 || c.Rune == ' ' || c.Width == 0 {
			i++
			continue
		}
		fg, _ := t.pal.colors(c.Style, reverse)
		j := i + 1
		run.Reset()
		run.WriteRune(c.Rune)
		if c.Rune < 0x7f && c.Width == 1 {
			for j < len(cells) {
				d := cells[j]
				if d.Width != 1 || d.Rune >= 0x7f || d.Style.Attr != c.Style.Attr || d.Style.FG != c.Style.FG || d.Style.BG != c.Style.BG {
					break
				}
				if d.Rune == 0 {
					run.WriteByte(' ')
				} else {
					run.WriteRune(d.Rune)
				}
				j++
			}
		} else if c.Width == 2 {
			j = i + 2
		}
		if j == i+1 && drawBlock(ops, c.Rune, m.x(i), m.x(j), m.cellH, m.px, fg) {
			i = j
			continue
		}
		s := strings.TrimRight(run.String(), " ")
		f := font.Font{Typeface: t.family}
		if c.Style.Attr&AttrBold != 0 {
			f.Weight = font.Bold
		}
		if c.Style.Attr&AttrItalic != 0 {
			f.Style = font.Italic
		}
		t.paintText(ops, s, m.x(i), f, fg)
		if c.Style.Attr&(AttrUnderline|AttrStrike) != 0 {
			th := max(1, m.px/14)
			x0, x1 := m.x(i), m.x(j)
			if c.Style.Attr&AttrUnderline != 0 {
				y := min(m.baseline+th+1, m.cellH-th)
				fill(ops, image.Rect(x0, y, x1, y+th), fg)
			}
			if c.Style.Attr&AttrStrike != 0 {
				y := m.baseline - m.px*3/10
				fill(ops, image.Rect(x0, y, x1, y+th), fg)
			}
		}
		i = j
	}
}

// paintText draws s with its first character's cell at x.
func (t *Terminal) paintText(ops *op.Ops, s string, x int, f font.Font, c color.NRGBA) {
	if s == "" {
		return
	}
	sh := fonts.Shaper()
	sh.LayoutString(text.Parameters{
		Font: fonts.ForText(fonts.Resolve(f), s), PxPerEm: fixed.I(t.m.px),
		MaxWidth: 1 << 20, MaxLines: 1, Locale: system.Locale{Direction: system.LTR},
	}, s)
	var gs []text.Glyph
	for {
		g, ok := sh.NextGlyph()
		if !ok {
			break
		}
		gs = append(gs, g)
	}
	if len(gs) == 0 {
		return
	}
	st := op.Offset(image.Pt(x+gs[0].X.Round(), t.m.baseline)).Push(ops)
	outline := clip.Outline{Path: sh.Shape(gs)}.Op().Push(ops)
	paint.ColorOp{Color: c}.Add(ops)
	paint.PaintOp{}.Add(ops)
	outline.Pop()
	if call := sh.Bitmaps(gs); call != (op.CallOp{}) {
		call.Add(ops)
	}
	st.Pop()
}

func fill(ops *op.Ops, r image.Rectangle, c color.NRGBA) {
	st := clip.Rect(r).Push(ops)
	paint.ColorOp{Color: c}.Add(ops)
	paint.PaintOp{}.Add(ops)
	st.Pop()
}

// drawCursor draws the cursor: filled while focused, outlined while
// not.
func (t *Terminal) drawCursor(gtx layout.Context, s Snapshot) {
	m := t.m
	cx, cy := s.CursorX, s.CursorY
	if cy < 0 || cy >= len(s.Lines) || cx < 0 || cx >= len(s.Lines[cy].Cells) {
		return
	}
	cell := s.Lines[cy].Cells[cx]
	w := 1
	if cell.Width == 2 {
		w = 2
	}
	r := image.Rect(m.pad+m.x(cx), m.pad+cy*m.cellH, m.pad+m.x(cx+w), m.pad+(cy+1)*m.cellH)
	// The input method's candidate window opens at the cursor.
	t.kb.Caret(r)
	if !s.CursorVisible {
		return
	}
	c := t.pal.cursor
	th := max(1, gtx.Dp(1))
	if !t.focused {
		for _, e := range []image.Rectangle{
			{Min: r.Min, Max: image.Pt(r.Max.X, r.Min.Y+th)},
			{Min: image.Pt(r.Min.X, r.Max.Y-th), Max: r.Max},
			{Min: r.Min, Max: image.Pt(r.Min.X+th, r.Max.Y)},
			{Min: image.Pt(r.Max.X-th, r.Min.Y), Max: r.Max},
		} {
			fill(gtx.Ops, e, c)
		}
		return
	}
	switch s.CursorShape {
	case CursorBar:
		fill(gtx.Ops, image.Rectangle{Min: r.Min, Max: image.Pt(r.Min.X+2*th, r.Max.Y)}, c)
	case CursorUnderline:
		fill(gtx.Ops, image.Rectangle{Min: image.Pt(r.Min.X, r.Max.Y-2*th), Max: r.Max}, c)
	default:
		fill(gtx.Ops, r, c)
		if cell.Rune != 0 && cell.Rune != ' ' {
			_, bg := t.pal.colors(cell.Style, s.Reverse)
			f := font.Font{Typeface: t.family}
			if cell.Style.Attr&AttrBold != 0 {
				f.Weight = font.Bold
			}
			st := op.Offset(image.Pt(m.pad, m.pad+cy*m.cellH)).Push(gtx.Ops)
			t.paintText(gtx.Ops, string(cell.Rune), m.x(cx), f, bg)
			st.Pop()
		}
	}
	if comp := t.kb.Composition(); comp != "" {
		// The text an input method is composing, underlined at the
		// cursor.
		st := op.Offset(image.Pt(m.pad, m.pad+cy*m.cellH)).Push(gtx.Ops)
		cw := 0
		for _, r := range comp {
			cw += runeWidth(r)
		}
		fill(gtx.Ops, image.Rect(m.x(cx), 0, m.x(cx+cw), m.cellH), t.pal.bg)
		t.paintText(gtx.Ops, comp, m.x(cx), font.Font{Typeface: t.family}, t.pal.fg)
		fill(gtx.Ops, image.Rect(m.x(cx), m.cellH-th, m.x(cx+cw), m.cellH), t.pal.fg)
		st.Pop()
	}
}

// drawSelection highlights the selected cells.
func (t *Terminal) drawSelection(gtx layout.Context, s Snapshot) {
	if !t.sel.active || t.sel.anchor == t.sel.head {
		return
	}
	from, to := t.sel.anchor, t.sel.head
	if to.Before(from) {
		from, to = to, from
	}
	m := t.m
	for i := range s.Lines {
		line := s.Top + i
		if line < from.Line || line > to.Line {
			continue
		}
		c0, c1 := 0, len(s.Lines[i].Cells)
		if line == from.Line {
			c0 = from.Col
		}
		if line == to.Line {
			c1 = to.Col
		}
		if c1 <= c0 {
			continue
		}
		fill(gtx.Ops, image.Rect(m.pad+m.x(c0), m.pad+i*m.cellH, m.pad+m.x(c1), m.pad+(i+1)*m.cellH), t.pal.selection)
	}
}

// cellAt returns the cell under p, and the history position of the
// edge between cells nearest to it, for selections.
func (t *Terminal) cellAt(p image.Point, top int) (col, row int, edge Pos) {
	m := t.m
	x := fixed.I(p.X - m.pad)
	col = min(max(int(x/m.cellW), 0), t.cols-1)
	if x < 0 {
		col = 0
	}
	ecol := min(max(int((x+m.cellW/2)/m.cellW), 0), t.cols)
	row = min(max((p.Y-m.pad)/m.cellH, 0), t.rows-1)
	return col, row, Pos{Line: top + row, Col: ecol}
}

// pointer handles presses, drags, the wheel and selections.
func (t *Terminal) pointer(gtx layout.Context) {
	modes := t.emu.Modes()
	kinds := pointer.Press | pointer.Drag | pointer.Release | pointer.Scroll
	if modes.Mouse == MouseMoves {
		kinds |= pointer.Move
	}
	top := t.emu.History() - t.rows - t.scroll
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: t, Kinds: kinds, ScrollY: pointer.ScrollRange{Min: -1 << 20, Max: 1 << 20}})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		col, row, edge := t.cellAt(e.Position.Round(), top)
		report := modes.Mouse != MouseOff && !e.Modifiers.Contain(key.ModShift) && t.scroll == 0
		switch e.Kind {
		case pointer.Press:
			btn := MouseLeft
			switch {
			case e.Buttons.Contain(pointer.ButtonSecondary):
				btn = MouseRight
			case e.Buttons.Contain(pointer.ButtonTertiary):
				btn = MouseMiddle
			}
			t.mouseBtn, t.reporting = btn, report
			if report {
				t.Send(EncodeMouse(modes, btn, MousePress, col, row, e.Modifiers))
				break
			}
			if btn != MouseLeft {
				break
			}
			if gtx.Now.Sub(t.lastPress) < 400*time.Millisecond {
				t.clicks++
			} else {
				t.clicks = 1
			}
			t.lastPress = gtx.Now
			switch t.clicks {
			case 1:
				t.sel = selection{active: true, dragging: true, anchor: edge, head: edge}
			case 2:
				a, b := t.emu.WordAt(Pos{Line: top + row, Col: col})
				t.sel = selection{active: true, anchor: a, head: b}
			default:
				t.sel = selection{active: true, anchor: Pos{top + row, 0}, head: Pos{top + row, t.cols}}
			}
		case pointer.Drag:
			if t.reporting {
				t.Send(EncodeMouse(modes, t.mouseBtn, MouseMove, col, row, e.Modifiers))
			} else if t.sel.dragging {
				t.sel.head = edge
			}
		case pointer.Move:
			if report {
				t.Send(EncodeMouse(modes, MouseNone, MouseMove, col, row, e.Modifiers))
			}
		case pointer.Release, pointer.Cancel:
			if t.reporting {
				t.Send(EncodeMouse(modes, t.mouseBtn, MouseRelease, col, row, e.Modifiers))
				t.reporting = false
			}
			if t.sel.dragging {
				t.sel.dragging = false
				if t.sel.anchor == t.sel.head {
					t.sel = selection{}
				}
			}
		case pointer.Scroll:
			t.scrollAcc += e.Scroll.Y
			lines := int(t.scrollAcc / float32(t.m.cellH))
			if lines == 0 {
				break
			}
			t.scrollAcc -= float32(lines * t.m.cellH)
			n := lines
			if n < 0 {
				n = -n
			}
			n = min(n, 10)
			switch {
			case report:
				b := MouseWheelDown
				if lines < 0 {
					b = MouseWheelUp
				}
				for range n {
					t.Send(EncodeMouse(modes, b, MousePress, col, row, e.Modifiers))
				}
			case modes.AltScreen:
				// Full-screen programs without the mouse scroll with the
				// arrow keys, as in other terminals.
				name := key.NameDownArrow
				if lines < 0 {
					name = key.NameUpArrow
				}
				for range n {
					t.Send(EncodeKey(key.Event{Name: name, State: key.Press}, modes))
				}
			default:
				t.ScrollBy(-lines)
			}
		}
		kit.RequestFrame(gtx)
	}
	area := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
	if modes.Mouse == MouseOff {
		pointer.CursorText.Add(gtx.Ops)
	}
	event.Op(gtx.Ops, t)
	area.Pop()
}

// clipboard copies and pastes.
func (t *Terminal) clipboard(gtx layout.Context) {
	if t.wantCopy {
		t.wantCopy = false
		if s := t.Selection(); s != "" {
			gtx.Execute(clipboard.WriteCmd{Type: "application/text", Data: io.NopCloser(strings.NewReader(s))})
		}
	}
	if t.wantPaste {
		t.wantPaste = false
		gtx.Execute(clipboard.ReadCmd{Tag: t})
	}
	for {
		ev, ok := gtx.Event(transfer.TargetFilter{Target: t, Type: "application/text"})
		if !ok {
			break
		}
		if e, ok := ev.(transfer.DataEvent); ok {
			r := e.Open()
			b, _ := io.ReadAll(io.LimitReader(r, 16<<20))
			r.Close()
			t.Paste(string(b))
		}
	}
}
