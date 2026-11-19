package datagrid

import (
	"errors"
	"image"
	"strconv"
	"strings"
	"time"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/components/input"
	"github.com/xmhrnx/giode/components/menu"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// editState is a cell being edited.
type editState struct {
	on       bool
	row, col int // display row and column
	in       *input.Input
	err      error
	options  *menu.Menu
	refocus  bool
	// next is the move a Tab commit makes: +1, -1, or 0.
	next int
}

// editor returns display column c's editor.
func (g *DataGrid) editor(c int) EditorKind {
	if c < 0 || c >= len(g.order) || g.onEdit == nil {
		return EditNone
	}
	return g.cols[g.order[c]].Editor
}

// startEdit edits the cell at display row and column: a checkbox flips,
// options open in a menu, other kinds in a field over the cell.
func (g *DataGrid) startEdit(row, col int) {
	src, _ := g.at(row)
	kind := g.editor(col)
	if src < 0 || kind == EditNone {
		return
	}
	c := g.cols[g.order[col]]
	v := g.src.Value(src, c.Key)
	g.focus = cell{row, col}
	switch kind {
	case EditCheckbox:
		b, _ := v.(bool)
		g.onEdit(src, c.Key, !b)
		return
	case EditSelect:
		g.openOptions(row, col, c, v)
		return
	}
	if g.edit.in == nil {
		g.edit.in = input.New("", styles.Styles{FontSize: g.fontSize(), Padding: properties.SymmetricInset(7, 0), BorderRadius: 0, BorderWidth: 2, BorderColor: "theme.primary", Background: "theme.background"})
		g.edit.in.Submit(func(text string) { g.commitEdit(text) })
		g.edit.in.OnKeys(func(e key.Event) {
			if e.State != key.Press {
				return
			}
			switch e.Name {
			case key.NameEscape:
				g.cancelEdit()
			case key.NameTab:
				g.edit.next = 1
				if e.Modifiers.Contain(key.ModShift) {
					g.edit.next = -1
				}
				g.commitEdit(g.edit.in.Text())
			}
		}, key.NameEscape, key.NameTab)
	}
	s := editText(kind, v)
	g.edit = editState{on: true, row: row, col: col, in: g.edit.in, options: g.edit.options}
	g.edit.in.SetText(s)
	g.edit.in.SetSelection(0, len([]rune(s)))
	g.edit.in.Focus()
	g.reveal = true
}

// editText is a value as its editor starts with.
func editText(kind EditorKind, v any) string {
	switch kind {
	case EditDate:
		if t, ok := v.(time.Time); ok && !t.IsZero() {
			return t.Format("2006-01-02")
		}
		return ""
	case EditNumber:
		if x, ok := number(v); ok {
			return strconv.FormatFloat(x, 'f', -1, 64)
		}
	}
	return text(v)
}

// parseEdit reads an editor's text as kind's value.
func parseEdit(kind EditorKind, s string) (any, error) {
	s = strings.TrimSpace(s)
	switch kind {
	case EditNumber:
		if s == "" {
			return nil, nil
		}
		x, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", ""), 64)
		if err != nil {
			return nil, errors.New("Enter a number")
		}
		return x, nil
	case EditDate:
		if s == "" {
			return time.Time{}, nil
		}
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return nil, errors.New("Enter a date as YYYY-MM-DD")
		}
		return t, nil
	}
	return s, nil
}

// commitEdit saves the field's text: closed when OnEdit accepts it, a
// Tab moving on to the next editable cell.
func (g *DataGrid) commitEdit(s string) {
	if !g.edit.on {
		return
	}
	src, _ := g.at(g.edit.row)
	c := g.cols[g.order[g.edit.col]]
	v, err := parseEdit(c.Editor, s)
	if err == nil {
		err = g.onEdit(src, c.Key, v)
	}
	if err != nil {
		g.edit.err = err
		g.edit.next = 0
		return
	}
	next := g.edit.next
	row, col := g.edit.row, g.edit.col
	g.endEdit()
	g.Refresh()
	if next != 0 {
		if r, c, ok := g.nextEditable(row, col, next); ok {
			g.startEdit(r, c)
		}
	}
}

// nextEditable finds the editable cell after (dir 1) or before (-1)
// row, col, in reading order.
func (g *DataGrid) nextEditable(row, col, dir int) (int, int, bool) {
	n := g.count()
	for steps := 0; steps < len(g.order)*4; steps++ {
		col += dir
		if col >= len(g.order) {
			col, row = 0, row+1
		} else if col < 0 {
			col, row = len(g.order)-1, row-1
		}
		if row < 0 || row >= n {
			return 0, 0, false
		}
		if k := g.editor(col); k != EditNone && k != EditCheckbox && k != EditSelect {
			if src, _ := g.at(row); src >= 0 {
				return row, col, true
			}
		}
	}
	return 0, 0, false
}

func (g *DataGrid) cancelEdit() {
	g.endEdit()
}

func (g *DataGrid) endEdit() {
	g.edit.on, g.edit.err, g.edit.next = false, nil, 0
	if g.edit.in != nil {
		g.edit.in.Blur()
	}
	g.edit.refocus = true
}

// openOptions shows a select column's options under the cell.
func (g *DataGrid) openOptions(row, col int, c Column, v any) {
	if g.edit.options == nil {
		g.edit.options = menu.New()
	}
	m := g.edit.options.Clear()
	src, _ := g.at(row)
	cur := text(v)
	for _, o := range c.Options {
		m.RadioItem(o, func() bool { return o == cur }, func() {
			if err := g.onEdit(src, c.Key, o); err == nil {
				g.Refresh()
			}
			g.edit.refocus = true
		})
	}
	l := &g.lay
	x := l.cellX(col, g.offX)
	y := l.rowY(row, g.offY) + l.rowH
	m.OpenAtWindow(g.origin.Add(image.Pt(x, y)))
}

// layoutEditor draws the field over the cell being edited, and its
// error under it.
func (g *DataGrid) layoutEditor(gtx layout.Context) {
	if g.edit.options != nil && g.edit.options.Opened() {
		g.edit.options.Layout(gtx)
	}
	if !g.edit.on {
		return
	}
	l := &g.lay
	row, col := g.edit.row, g.edit.col
	if col >= len(g.order) || row >= g.count() {
		g.endEdit()
		return
	}
	x := l.cellX(col, g.offX)
	y := l.rowY(row, g.offY)
	w := l.colX[col+1] - l.colX[col]
	r := image.Rect(x, y, x+w, y+l.rowH)
	cgtx := gtx
	cgtx.Constraints = layout.Exact(r.Size())
	off := op.Offset(r.Min).Push(gtx.Ops)
	pop := elements.PushOffset(r.Min)
	g.edit.in.Layout(cgtx)
	pop()
	off.Pop()
	if g.edit.err != nil {
		msg := elements.Box(styles.Styles{Background: "theme.danger", Padding: properties.SymmetricInset(8, 4), BorderRadius: 4},
			elements.Text(g.edit.err.Error(), styles.Styles{Color: "#ffffff", FontSize: 12}))
		mgtx := gtx
		mgtx.Constraints = layout.Constraints{Max: image.Pt(max(w, gtx.Dp(200)), gtx.Dp(80))}
		at := image.Pt(x, y+l.rowH+gtx.Dp(2))
		o := op.Offset(at).Push(gtx.Ops)
		msg.Layout(mgtx)
		o.Pop()
	}
}
