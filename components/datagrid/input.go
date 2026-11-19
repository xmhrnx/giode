package datagrid

import (
	"image"
	"strings"
	"time"

	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"

	"github.com/xmhrnx/giode/components/kit"
)

// navKeys are the keys the grid handles while focused.
var navKeys = []key.Name{
	key.NameUpArrow, key.NameDownArrow, key.NameLeftArrow, key.NameRightArrow,
	key.NameHome, key.NameEnd, key.NamePageUp, key.NamePageDown,
	key.NameReturn, key.NameEnter, key.NameSpace, "A", "C",
}

// doubleClick is the longest time between a double click's clicks.
const doubleClick = 400 * time.Millisecond

// update handles input since the last frame, and reports whether the
// grid must be laid out again.
func (g *DataGrid) update(gtx layout.Context) bool {
	changed := false
	filters := []event.Filter{
		pointer.Filter{Target: g, Kinds: pointer.Press | pointer.Release | pointer.Drag | pointer.Move | pointer.Leave | pointer.Cancel},
		key.FocusFilter{Target: g},
	}
	for _, n := range navKeys {
		filters = append(filters, key.Filter{Focus: g, Name: n, Optional: key.ModShift | key.ModShortcut})
	}
	for {
		ev, ok := gtx.Event(filters...)
		if !ok {
			break
		}
		changed = true
		switch e := ev.(type) {
		case key.FocusEvent:
			g.focused = e.Focus
		case key.Event:
			if e.State == key.Press {
				g.onKey(gtx, e)
			}
		case pointer.Event:
			g.onPointer(gtx, e)
		}
	}
	return changed
}

// applyScroll applies wheel and touch scrolling.
func (g *DataGrid) applyScroll(gtx layout.Context) {
	l := &g.lay
	rangeY := pointer.ScrollRange{Min: -int(g.offY), Max: l.maxY - int(g.offY)}
	if dy := g.scrollY.Update(gtx.Metric, gtx.Source, gtx.Now, gesture.Vertical, pointer.ScrollRange{}, rangeY); dy != 0 {
		g.offY += float32(dy)
	}
	rangeX := pointer.ScrollRange{Min: -int(g.offX), Max: l.maxX - int(g.offX)}
	if dx := g.scrollX.Update(gtx.Metric, gtx.Source, gtx.Now, gesture.Horizontal, rangeX, pointer.ScrollRange{}); dx != 0 {
		g.offX += float32(dx)
	}
	if g.scrollY.State() != gesture.StateIdle || g.scrollX.State() != gesture.StateIdle {
		kit.MarkLive()
	}
}

// revealFocus scrolls the focused cell into view.
func (g *DataGrid) revealFocus() {
	l := &g.lay
	if g.focus.row >= 0 {
		top := g.focus.row * l.rowH
		switch {
		case float32(top) < g.offY:
			g.offY = float32(top)
		case float32(top+l.rowH) > g.offY+float32(l.bodyH):
			g.offY = float32(top + l.rowH - l.bodyH)
		}
	}
	if c := g.focus.col; c >= l.frozenCols && c < len(g.order) {
		left, right := l.colX[c]-l.frozenW, l.colX[c+1]-l.frozenW
		view := l.size.X - l.frozenW
		switch {
		case float32(left) < g.offX:
			g.offX = float32(left)
		case float32(right) > g.offX+float32(view):
			g.offX = float32(right - view)
		}
	}
}

// hit is what's under a point.
type hit struct {
	header, check bool
	row, col      int // -1 when none
	resize        int // the display column whose right edge is under it, or -1
}

func (g *DataGrid) hitTest(gtx layout.Context, p image.Point) hit {
	l := &g.lay
	h := hit{row: -1, col: -1, resize: -1}
	if !p.In(image.Rectangle{Max: l.size}) {
		return h
	}
	h.header = p.Y < l.headerH
	if !h.header {
		row := (p.Y - l.headerH + int(g.offY)) / l.rowH
		if row < g.src.Len() {
			h.row = row
		}
	}
	if p.X < l.checkW {
		h.check = true
		return h
	}
	x := p.X
	if p.X >= l.frozenW {
		x += int(g.offX)
	}
	slop := gtx.Dp(4)
	for c := range g.order {
		if p.X < l.frozenW && c >= l.frozenCols || p.X >= l.frozenW && c < l.frozenCols {
			continue
		}
		left, right := l.colX[c], l.colX[c+1]
		if h.header && x >= right-slop && x < right+slop {
			h.resize = c
		}
		if x >= left && x < right {
			h.col = c
		}
	}
	return h
}

func (g *DataGrid) onPointer(gtx layout.Context, e pointer.Event) {
	p := e.Position.Round()
	switch e.Kind {
	case pointer.Move:
		h := g.hitTest(gtx, p)
		g.hoverRow = h.row
	case pointer.Leave, pointer.Cancel:
		g.hoverRow = -1
		g.resizing = -1
	case pointer.Press:
		h := g.hitTest(gtx, p)
		gtx.Execute(key.FocusCmd{Tag: g})
		g.pressRow, g.pressCol = -1, -1
		if e.Buttons.Contain(pointer.ButtonSecondary) {
			if h.row >= 0 {
				if !g.Selected(h.row) {
					g.Select(h.row)
				}
				g.focus = cell{h.row, max(h.col, 0)}
				g.buildMenu(h.row, h.col)
			}
			return
		}
		if !e.Buttons.Contain(pointer.ButtonPrimary) && e.Source == pointer.Mouse {
			return
		}
		switch {
		case h.header && h.resize >= 0:
			g.resizing = h.resize
			g.resizeX = e.Position.X
			g.resizeW = g.widths[g.order[h.resize]]
		case h.header:
			g.pressRow, g.pressCol = -1, h.col
			if h.check {
				g.pressCol = -2
			}
		case h.row >= 0:
			g.pressRow, g.pressCol = h.row, h.col
			if h.check {
				g.toggle(h.row)
				g.anchor = h.row
				g.focus.row = h.row
				return
			}
			g.clickRow(h.row, max(h.col, 0), e.Modifiers, e.Time)
		}
	case pointer.Drag:
		if g.resizing >= 0 {
			ci := g.order[g.resizing]
			minW := float32(g.cols[ci].MinWidth)
			if minW <= 0 {
				minW = 48
			}
			g.widths[ci] = max(g.resizeW+(e.Position.X-g.resizeX)/g.lay.pxPerDp, minW)
		}
	case pointer.Release:
		if g.resizing >= 0 {
			g.resizing = -1
			return
		}
		h := g.hitTest(gtx, p)
		if h.header && g.pressRow == -1 {
			switch {
			case g.pressCol == -2 && h.check:
				g.SelectAll(g.sel.count(g.src.Len()) < g.src.Len())
			case g.pressCol >= 0 && h.col == g.pressCol:
				g.toggleSort(g.order[h.col], e.Modifiers.Contain(key.ModShift))
			}
		}
	}
}

// clickRow selects as a click on row does, with Shift extending from
// the anchor and Ctrl/Cmd toggling; a second click soon after on the
// same row activates it.
func (g *DataGrid) clickRow(row, col int, mods key.Modifiers, t time.Duration) {
	g.focus = cell{row, col}
	switch {
	case mods.Contain(key.ModShift):
		g.selectRange(g.anchor, row, mods.Contain(key.ModShortcut))
	case mods.Contain(key.ModShortcut):
		g.toggle(row)
		g.anchor = row
	default:
		double := g.lastClick == row && t-time.Duration(g.clickedAt) < doubleClick && g.clickedAt != 0
		g.sel = selection{}
		g.sel.set(g.keyOf(row), true)
		g.anchor = row
		g.selected()
		if double && g.onActivate != nil {
			g.onActivate(row)
			g.clickedAt = 0
			return
		}
	}
	g.lastClick, g.clickedAt = row, int64(t)
}

func (g *DataGrid) toggle(row int) {
	k := g.keyOf(row)
	g.sel.set(k, !g.sel.has(k))
	g.selected()
}

// selectRange selects rows a to b, adding to the selection with keep.
func (g *DataGrid) selectRange(a, b int, keep bool) {
	if !keep {
		g.sel = selection{}
	}
	lo, hi := min(a, b), max(a, b)
	for r := lo; r <= hi && r < g.src.Len(); r++ {
		g.sel.set(g.keyOf(r), true)
	}
	g.selected()
}

// toggleSort sorts by column ci: ascending, descending, then not; with
// add, as another level after the current ones.
func (g *DataGrid) toggleSort(ci int, add bool) {
	col := g.cols[ci]
	if !col.Sortable {
		return
	}
	idx := -1
	for i, k := range g.sort {
		if k.Key == col.Key {
			idx = i
		}
	}
	var next []SortKey
	if add {
		next = append([]SortKey(nil), g.sort...)
	}
	switch {
	case idx < 0:
		next = append(next, SortKey{Key: col.Key})
	case !g.sort[idx].Descending:
		if add {
			next[idx].Descending = true
		} else {
			next = []SortKey{{Key: col.Key, Descending: true}}
		}
	default:
		if add {
			next = append(next[:idx], next[idx+1:]...)
		} else {
			next = nil
		}
	}
	g.Sort(next...)
	if g.onSort != nil {
		g.onSort(g.SortKeys())
	}
}

func (g *DataGrid) onKey(gtx layout.Context, e key.Event) {
	n := g.src.Len()
	if n == 0 {
		return
	}
	shift, mod := e.Modifiers.Contain(key.ModShift), e.Modifiers.Contain(key.ModShortcut)
	row, col := g.focus.row, g.focus.col
	page := max(g.lay.bodyH/max(g.lay.rowH, 1)-1, 1)
	moved := true
	switch e.Name {
	case key.NameUpArrow:
		row--
	case key.NameDownArrow:
		row++
	case key.NamePageUp:
		row -= page
	case key.NamePageDown:
		row += page
	case key.NameLeftArrow:
		col--
	case key.NameRightArrow:
		col++
	case key.NameHome:
		if mod {
			row = 0
		} else {
			col = 0
		}
	case key.NameEnd:
		if mod {
			row = n - 1
		} else {
			col = len(g.order) - 1
		}
	case key.NameReturn, key.NameEnter:
		if g.onActivate != nil && row >= 0 && row < n {
			g.onActivate(row)
		}
		return
	case key.NameSpace:
		if row >= 0 && row < n {
			g.toggle(row)
			g.anchor = row
		}
		return
	case "A":
		if mod {
			g.SelectAll(true)
		}
		return
	case "C":
		if mod {
			kit.WriteClipboard(gtx, g.CopyText())
		}
		return
	default:
		moved = false
	}
	if !moved {
		return
	}
	row = min(max(row, 0), n-1)
	col = min(max(col, 0), max(len(g.order)-1, 0))
	vertical := row != g.focus.row
	g.focus = cell{row, col}
	g.reveal = true
	if vertical {
		if shift {
			g.selectRange(g.anchor, row, false)
		} else {
			g.Select(row)
		}
	}
}

// CopyText returns the selected rows - or the focused row when none is
// selected - as tab-separated text, a line per row, with the visible
// columns' text: what Ctrl+C copies, for pasting into a spreadsheet.
func (g *DataGrid) CopyText() string {
	rows := g.SelectedRows()
	if len(rows) == 0 && g.focus.row >= 0 && g.focus.row < g.src.Len() {
		rows = []int{g.focus.row}
	}
	var b strings.Builder
	for i, r := range rows {
		if i > 0 {
			b.WriteByte('\n')
		}
		for j, ci := range g.order {
			if j > 0 {
				b.WriteByte('\t')
			}
			s := formatValue(g.cols[ci], g.src.Value(r, g.cols[ci].Key))
			b.WriteString(strings.NewReplacer("\t", " ", "\n", " ").Replace(s))
		}
	}
	return b.String()
}

// buildMenu fills the row menu for row and display column col.
func (g *DataGrid) buildMenu(row, col int) {
	if g.rowMenu == nil || g.menu.Opened() {
		return
	}
	key := ""
	if col >= 0 && col < len(g.order) {
		key = g.cols[g.order[col]].Key
	}
	g.menuRow, g.menuCol = row, key
	g.rowMenu(row, key, g.menu.Clear())
}
