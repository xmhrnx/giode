package richedit

import (
	"strings"

	"github.com/xmhrnx/giode/components/menu"
	"github.com/xmhrnx/giode/richdoc"
)

// menuKey is what the editor's menu was built for.
type menuKey struct {
	empty, cell, undo, redo, link, ro bool
}

// updateMenu rebuilds the editor's own context menu for the selection
// while it's closed. An app's menu set with Context replaces it.
func (e *RichEditor) updateMenu() {
	if e.menu == nil {
		e.menu = menu.New()
		e.Context(e.menu)
	}
	if e.ContextMenu() != e.menu || e.menu.Opened() {
		return
	}
	k := menuKey{empty: e.st.Sel.Empty() && (!e.cell.on || e.cell.anchor == e.cell.head), cell: e.cell.on,
		undo: e.CanUndo(), redo: e.CanRedo(), link: e.onLinkEdit != nil, ro: e.readOnly}
	if e.menuBuilt && k == e.menuFor {
		return
	}
	e.menuBuilt, e.menuFor = true, k
	m := e.menu.Clear()
	if k.ro {
		if k.empty {
			m.DisabledItem("Copy")
		} else {
			m.Item("Copy", e.copySelection).Shortcut("Ctrl+C")
		}
		m.Item("Select all", e.SelectAll).Shortcut("Ctrl+A")
		return
	}
	if k.empty {
		m.DisabledItem("Cut").DisabledItem("Copy")
	} else {
		m.Item("Cut", e.cutSelection).Shortcut("Ctrl+X")
		m.Item("Copy", e.copySelection).Shortcut("Ctrl+C")
	}
	m.Item("Paste", func() { e.pendingRead = true }).Shortcut("Ctrl+V")
	m.Item("Select all", e.SelectAll).Shortcut("Ctrl+A")
	m.Separator()
	mark := func(label string, k richdoc.MarkKind, sc string) {
		m.CheckItem(label, func() bool { return e.HasMark(k) }, func() {
			if e.cell.on {
				e.cellMark(k)
			} else {
				e.ToggleMark(k)
			}
		}).Shortcut(sc)
	}
	mark("Bold", richdoc.Bold, "Ctrl+B")
	mark("Italic", richdoc.Italic, "Ctrl+I")
	mark("Underline", richdoc.Underline, "Ctrl+U")
	mark("Strikethrough", richdoc.Strike, "Ctrl+Shift+X")
	mark("Code", richdoc.InlineCode, "Ctrl+E")
	if k.link && !k.cell {
		m.Item("Link…", e.editLink).Shortcut("Ctrl+K")
	}
	if k.cell {
		c := &e.cell
		m.Separator()
		m.Item("Insert row above", func() { e.run(e.st.InsertRow(c.block, c.row)) })
		m.Item("Insert row below", func() { e.run(e.st.InsertRow(c.block, c.row+1)) })
		m.Item("Insert column left", func() { e.run(e.st.InsertColumn(c.block, c.col)) })
		m.Item("Insert column right", func() { e.run(e.st.InsertColumn(c.block, c.col+1)) })
		m.Item("Delete row", func() { e.run(e.st.DeleteRow(c.block, c.row)); e.cell.on = false })
		m.Item("Delete column", func() { e.run(e.st.DeleteColumn(c.block, c.col)); e.cell.on = false })
		m.Item("Delete table", func() { e.run(e.st.DeleteBlock(c.block)); e.cell.on = false })
	} else {
		turn := menu.New()
		for _, s := range slashCommands()[:9] {
			cmd := s.Command
			turn.Item(s.Label, func() { cmd(e) })
		}
		ins := menu.New().
			Item("Table", func() { e.InsertTable(3, 3) }).
			Item("Divider", e.InsertRule)
		m.SubMenu("Turn into", turn)
		m.SubMenu("Insert", ins)
	}
	m.Separator()
	if k.undo {
		m.Item("Undo", e.Undo).Shortcut("Ctrl+Z")
	} else {
		m.DisabledItem("Undo")
	}
	if k.redo {
		m.Item("Redo", e.Redo).Shortcut("Ctrl+Shift+Z")
	} else {
		m.DisabledItem("Redo")
	}
}

// copySelection copies the selection with its formatting (from a menu:
// written at the next layout).
func (e *RichEditor) copySelection() {
	if e.cell.on {
		if spans, ok := e.cellSpans(); ok && e.cell.lo() < e.cell.hi() {
			e.copyNext = string([]rune(blockText(&richdoc.Block{Spans: spans}))[e.cell.lo():e.cell.hi()])
		}
		return
	}
	if e.st.Sel.Empty() {
		return
	}
	frag := e.st.Fragment()
	text := strings.TrimSuffix(frag.Text(), "\n")
	lastCopy.Lock()
	lastCopy.text, lastCopy.doc = text, frag
	lastCopy.Unlock()
	e.copyNext = text
}

func (e *RichEditor) cutSelection() {
	e.copySelection()
	if e.cell.on {
		e.editCell(e.cell.lo(), e.cell.hi(), nil)
		return
	}
	e.run(e.st.Delete(true, richdoc.Char))
}
