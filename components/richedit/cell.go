package richedit

import (
	"image"
	"strings"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/layout"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/internal/textrender"
	"github.com/xmhrnx/giode/richdoc"
)

// cellCaret is the caret inside a table cell.
type cellCaret struct {
	on              bool
	block, row, col int
	anchor, head    int
}

func (c cellCaret) lo() int { return min(c.anchor, c.head) }
func (c cellCaret) hi() int { return max(c.anchor, c.head) }

// cellSpans returns the cell's content, or nil when the caret's cell is
// gone (undone, or changed by a collaborator).
func (e *RichEditor) cellSpans() ([]richdoc.Span, bool) {
	c := &e.cell
	if !c.on || c.block >= len(e.st.Doc.Blocks) {
		return nil, false
	}
	t := &e.st.Doc.Blocks[c.block]
	if t.Kind != richdoc.Table || c.row >= len(t.Cells) || c.col >= len(t.Cells[c.row]) {
		return nil, false
	}
	return t.Cells[c.row][c.col], true
}

// enterCell puts the caret in a cell, at off (-1 for its end).
func (e *RichEditor) enterCell(block, row, col, off int) {
	e.cell = cellCaret{on: true, block: block, row: row, col: col}
	spans, ok := e.cellSpans()
	if !ok {
		e.cell.on = false
		return
	}
	n := richdoc.SpansLen(spans)
	if off < 0 || off > n {
		off = n
	}
	e.cell.anchor, e.cell.head = off, off
	e.st.Select(richdoc.Caret(richdoc.Pos{Block: block}))
	e.sugg.open = false
	e.ime.block = -1
}

// leaveCell takes the caret out of the table, to before or after it.
func (e *RichEditor) leaveCell(after bool) {
	b := e.cell.block
	e.cell.on = false
	e.ime.block = -1
	p := richdoc.Pos{Block: b}
	switch {
	case after && b+1 < len(e.st.Doc.Blocks):
		p = richdoc.Pos{Block: b + 1}
	case !after && b > 0:
		p = richdoc.Pos{Block: b - 1, Offset: e.st.Doc.BlockLen(b - 1)}
	case after:
		p.Offset = 1
	}
	e.moveTo(p, false)
}

func (e *RichEditor) tableSize(block int) (rows, cols int) {
	t := &e.st.Doc.Blocks[block]
	for _, r := range t.Cells {
		cols = max(cols, len(r))
	}
	return len(t.Cells), cols
}

// editCell replaces [from, to) of the cell's text with ins and puts the
// caret after it.
func (e *RichEditor) editCell(from, to int, ins []richdoc.Span) {
	spans, ok := e.cellSpans()
	if !ok || e.readOnly {
		return
	}
	c := &e.cell
	e.run(e.st.SetCell(c.block, c.row, c.col, richdoc.ReplaceSpans(spans, from, to, ins)))
	p := from + richdoc.SpansLen(ins)
	c.anchor, c.head = p, p
}

// cellText inserts typed text in the cell, replacing its selection.
func (e *RichEditor) cellText(text string) {
	spans, ok := e.cellSpans()
	if !ok {
		return
	}
	text = strings.NewReplacer("\r\n", " ", "\n", " ", "\t", " ").Replace(text)
	c := e.cell
	m := richdoc.MarksAt(spans, c.lo())
	m.Link = ""
	e.editCell(c.lo(), c.hi(), []richdoc.Span{{Text: text, Marks: m}})
}

// cellEdit applies an input method's edit in the cell.
func (e *RichEditor) cellEdit(ev key.EditEvent) {
	spans, ok := e.cellSpans()
	if !ok {
		return
	}
	n := richdoc.SpansLen(spans)
	from, to := min(max(ev.Range.Start, 0), n), min(max(ev.Range.End, 0), n)
	if key.Range(ev.Range) == e.ime.sel {
		from, to = e.cell.lo(), e.cell.hi()
	}
	m := richdoc.MarksAt(spans, from)
	m.Link = ""
	text := strings.NewReplacer("\n", " ", "\t", " ").Replace(ev.Text)
	var ins []richdoc.Span
	if text != "" {
		ins = []richdoc.Span{{Text: text, Marks: m}}
	}
	e.editCell(min(from, to), max(from, to), ins)
}

// cellKey handles a key in a cell, reporting whether it took it.
func (e *RichEditor) cellKey(gtx layout.Context, ev key.Event) bool {
	spans, ok := e.cellSpans()
	if !ok {
		e.cell.on = false
		return false
	}
	c := &e.cell
	text := []rune(blockText(&richdoc.Block{Spans: spans}))
	n := len(text)
	shift := ev.Modifiers.Contain(key.ModShift)
	word := ev.Modifiers.Contain(key.ModShortcutAlt)
	short := ev.Modifiers.Contain(key.ModShortcut)
	rows, cols := e.tableSize(c.block)
	move := func(p int) {
		c.head = p
		if !shift {
			c.anchor = p
		}
	}
	switch ev.Name {
	case key.NameEscape:
		b := c.block
		c.on = false
		e.st.Select(richdoc.Selection{Anchor: richdoc.Pos{Block: b}, Head: richdoc.Pos{Block: b, Offset: 1}})
	case key.NameTab, key.NameReturn, key.NameEnter:
		next := ev.Name != key.NameTab || !shift
		r, col := c.row, c.col
		if ev.Name == key.NameTab {
			if next {
				col++
				if col >= cols {
					col, r = 0, r+1
				}
			} else {
				col--
				if col < 0 {
					col, r = cols-1, r-1
				}
			}
		} else {
			r++
		}
		if r < 0 {
			e.leaveCell(false)
			return true
		}
		if r >= rows && !e.readOnly {
			e.run(e.st.InsertRow(c.block, rows))
		}
		e.enterCell(c.block, min(r, rows), col, -1)
	case key.NameLeftArrow, key.NameRightArrow:
		fwd := ev.Name == key.NameRightArrow
		if kit.RTL(gtx) {
			fwd = !fwd
		}
		switch {
		case !shift && c.anchor != c.head:
			if fwd {
				move(c.hi())
			} else {
				move(c.lo())
			}
		case fwd && c.head >= n && !shift:
			if c.col+1 < cols {
				e.enterCell(c.block, c.row, c.col+1, 0)
			} else if c.row+1 < rows {
				e.enterCell(c.block, c.row+1, 0, 0)
			} else {
				e.leaveCell(true)
			}
		case !fwd && c.head == 0 && !shift:
			if c.col > 0 {
				e.enterCell(c.block, c.row, c.col-1, -1)
			} else if c.row > 0 {
				e.enterCell(c.block, c.row-1, cols-1, -1)
			} else {
				e.leaveCell(false)
			}
		case fwd && word:
			move(richdoc.WordEnd(text, c.head))
		case fwd:
			move(richdoc.NextStop(text, c.head))
		case word:
			move(richdoc.WordStart(text, c.head))
		default:
			move(richdoc.PrevStop(text, c.head))
		}
	case key.NameUpArrow, key.NameDownArrow:
		r := c.row - 1
		if ev.Name == key.NameDownArrow {
			r = c.row + 1
		}
		switch {
		case r < 0:
			e.leaveCell(false)
		case r >= rows:
			e.leaveCell(true)
		default:
			e.enterCell(c.block, r, c.col, min(c.head, -1))
		}
	case key.NameHome:
		move(0)
	case key.NameEnd:
		move(n)
	case key.NameDeleteBackward, key.NameDeleteForward:
		if e.readOnly {
			return true
		}
		from, to := c.lo(), c.hi()
		if from == to {
			if ev.Name == key.NameDeleteBackward {
				from = richdoc.PrevStop(text, from)
				if word {
					from = richdoc.WordStart(text, to)
				}
			} else {
				to = richdoc.NextStop(text, to)
				if word {
					to = richdoc.WordEnd(text, from)
				}
			}
		}
		e.editCell(from, to, nil)
	default:
		if !short {
			return false
		}
		switch ev.Name {
		case "A":
			c.anchor, c.head = 0, n
		case "C", "X":
			if ev.Name == "X" && shift {
				e.cellMark(richdoc.Strike)
				break
			}
			if c.lo() < c.hi() {
				kit.WriteClipboard(gtx, string(text[c.lo():c.hi()]))
				if ev.Name == "X" {
					e.editCell(c.lo(), c.hi(), nil)
				}
			}
		case "B":
			e.cellMark(richdoc.Bold)
		case "I":
			e.cellMark(richdoc.Italic)
		case "U":
			e.cellMark(richdoc.Underline)
		case "E":
			e.cellMark(richdoc.InlineCode)
		default:
			return false
		}
	}
	return true
}

// cellMark toggles a mark on the cell's selection.
func (e *RichEditor) cellMark(k richdoc.MarkKind) {
	spans, ok := e.cellSpans()
	c := e.cell
	if !ok || c.lo() == c.hi() || e.readOnly {
		return
	}
	part := richdoc.SliceSpans(spans, c.lo(), c.hi())
	tmp := richdoc.NewState(&richdoc.Doc{Blocks: []richdoc.Block{{Spans: part}}})
	tmp.Select(richdoc.Selection{Head: tmp.Doc.End()})
	tmp.ToggleMark(k)
	e.editCell(c.lo(), c.hi(), tmp.Doc.Blocks[0].Spans)
	e.cell.anchor, e.cell.head = c.anchor, c.head
}

// cellAt returns the table cell under content point p, if any.
func (e *RichEditor) cellAt(p image.Point) (block, row, col, off int, ok bool) {
	if len(e.tops) == 0 {
		return
	}
	i := e.blockAt(p.Y)
	if i >= len(e.laid) || e.laid[i] == nil || e.st.Doc.Blocks[i].Kind != richdoc.Table {
		return
	}
	v := e.laid[i]
	q := p.Sub(image.Pt(0, e.tops[i]))
	for r, rowv := range v.cells {
		for c, cv := range rowv {
			if q.In(cv.rect) {
				pad := cellPad(e, v)
				o := textrender.HitRune(cv.lines, q.Sub(cv.rect.Min.Add(image.Pt(pad, pad))))
				return i, r, c, o, true
			}
		}
	}
	return
}

func cellPad(_ *RichEditor, v *blockView) int { return v.cellPad }

// paintCellCaret draws the cell's caret or selection, in the table's
// coordinates.
func (e *RichEditor) paintCellCaret(gtx layout.Context, v *blockView, pal palette, pad int) {
	c := e.cell
	if !c.on || c.row >= len(v.cells) || c.col >= len(v.cells[c.row]) {
		return
	}
	cv := v.cells[c.row][c.col]
	o := cv.rect.Min.Add(image.Pt(pad, pad))
	thick := max(gtx.Dp(2), 1)
	r := cv.rect
	// The cell being edited is outlined.
	for _, side := range []image.Rectangle{
		{r.Min, image.Pt(r.Max.X, r.Min.Y+thick)}, {image.Pt(r.Min.X, r.Max.Y-thick), r.Max},
		{r.Min, image.Pt(r.Min.X+thick, r.Max.Y)}, {image.Pt(r.Max.X-thick, r.Min.Y), r.Max},
	} {
		fillRect(gtx, side, pal.caret)
	}
	if c.lo() < c.hi() {
		for _, ln := range cv.lines {
			for k := 0; k+1 < len(ln.Stops); k++ {
				a, b := ln.Stops[k], ln.Stops[k+1]
				if min(a.Rune, b.Rune) >= c.lo() && max(a.Rune, b.Rune) <= c.hi() && a.Rune != b.Rune {
					fillRect(gtx, image.Rect(o.X+min(a.X, b.X), o.Y+ln.Top, o.X+max(a.X, b.X), o.Y+ln.Bottom), pal.sel)
				}
			}
		}
		return
	}
	if len(cv.lines) == 0 || !e.focused {
		return
	}
	li, s := stopFor(cv.lines, c.head)
	ln := cv.lines[li]
	w := max(gtx.Dp(1.5), 1)
	fillRect(gtx, image.Rect(o.X+s.X, o.Y+ln.Top, o.X+s.X+w, o.Y+ln.Bottom), pal.caret)
}

// cellCaretRect is the cell caret's box in content coordinates.
func (e *RichEditor) cellCaretRect() (image.Rectangle, bool) {
	c := e.cell
	if !c.on || c.block >= len(e.laid) || e.laid[c.block] == nil {
		return image.Rectangle{}, false
	}
	v := e.laid[c.block]
	if c.row >= len(v.cells) || c.col >= len(v.cells[c.row]) {
		return image.Rectangle{}, false
	}
	cv := v.cells[c.row][c.col]
	pad := cellPad(e, v)
	o := cv.rect.Min.Add(image.Pt(pad, pad+e.tops[c.block]))
	if len(cv.lines) == 0 {
		return image.Rectangle{Min: o, Max: o.Add(image.Pt(1, cv.laid.Dims.Size.Y))}, true
	}
	li, s := stopFor(cv.lines, c.head)
	ln := cv.lines[li]
	return image.Rect(o.X+s.X, o.Y+ln.Top, o.X+s.X+1, o.Y+ln.Bottom), true
}

// cellIME tells the input method the cell's text.
func (e *RichEditor) cellIME(gtx layout.Context) {
	spans, ok := e.cellSpans()
	if !ok {
		return
	}
	text := blockText(&richdoc.Block{Spans: spans})
	snip := key.Snippet{Range: key.Range{Start: 0, End: len([]rune(text))}, Text: text}
	if e.ime.block != -2 || snip != e.ime.snippet {
		e.ime.block = -2
		e.ime.snippet = snip
		gtx.Execute(key.SnippetCmd{Tag: e, Snippet: snip})
	}
	rng := key.Range{Start: e.cell.anchor, End: e.cell.head}
	var caret key.Caret
	if r, ok := e.cellCaretRect(); ok {
		r = r.Add(image.Pt(0, -int(e.offY)))
		base := r.Min.Y + r.Dy()*4/5
		caret = key.Caret{Pos: f32.Pt(float32(r.Min.X), float32(base)), Ascent: float32(base - r.Min.Y), Descent: float32(r.Max.Y - base)}
	}
	if rng != e.ime.sel || caret != e.ime.caret {
		e.ime.sel, e.ime.caret = rng, caret
		gtx.Execute(key.SelectionCmd{Tag: e, Range: rng, Caret: caret})
	}
}
