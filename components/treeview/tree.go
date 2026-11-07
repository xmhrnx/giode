// Package treeview provides a virtualized, keyboard-navigable tree of
// expandable rows, owning expansion and selection while the caller
// only draws each row's content.

package treeview

import (
	"gioui.org/io/key"
	"gioui.org/unit"
	"github.com/xmhrnx/giode/components/virtuallist"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
)

// Tree is a tree of expandable rows. Create it once and lay it out
// every frame:
//
//	tree := giode.TreeView(func() []giode.TreeNode { return roots },
//		func(n giode.TreeNode, s giode.TreeRowState) giode.Element {
//			return giode.Text(n.(*Folder).Name)
//		}).
//		OnExpand(func(n giode.TreeNode) { n.(*Folder).LoadChildren() })
//
// The tree owns expansion and selection, keyed by Node.Key, and draws
// the indentation, expand arrows, selection and hover backgrounds; the
// row function draws only the content. Only visible rows are built,
// so trees of any size stay fast. Keyboard (once focused): Up/Down
// move the selection, Right expands (or moves into a node), Left
// collapses (or moves to the parent), Enter activates,
// PageUp/PageDown and Home/End jump. A click selects, a double click
// activates, a click on the arrow toggles.
type Tree struct {
	// Contextual gives it a context menu (Context).
	elements.Contextual

	roots  func() []Node
	draw   func(n Node, s RowState) elements.Element
	st     styles.Styles
	rowH   unit.Dp
	list   *virtuallist.List
	rows   []row
	byKey  map[any]int
	expand map[any]bool

	selected    any
	hasSelected bool
	reveal      any
	hasReveal   bool

	presses *elements.KeyedCache[any, *rowWidgets]

	onSelect   func(Node)
	onExpand   func(Node)
	onCollapse func(Node)
	onActivate func(Node)
}

// New returns a tree of roots() drawn with draw. The styles argument is
// optional: Background (default transparent), Height (row height in
// dp, default 28), ScrollBar, Color (arrows, default theme.text) and
// BorderColor (selection color, default theme.primary).
func New(roots func() []Node, draw func(n Node, s RowState) elements.Element, st ...styles.Styles) *Tree {
	t := &Tree{
		roots:   roots,
		draw:    draw,
		rowH:    defaultRowHeight,
		byKey:   map[any]int{},
		expand:  map[any]bool{},
		presses: elements.NewKeyedCache[any, *rowWidgets]().Retain(300),
	}
	if len(st) > 0 {
		t.st = st[0]
		if t.st.Height > 0 {
			t.rowH = unit.Dp(t.st.Height)
		}
	}
	t.list = virtuallist.New(func() int { return len(t.rows) }, int(t.rowH), t.rowView,
		styles.Styles{Background: t.st.Background, ScrollBar: t.st.ScrollBar, BorderColor: "theme.primary/0.0"}).
		OnFocusChange(func(i int) {
			if i >= 0 && i < len(t.rows) {
				t.selectNode(t.rows[i].node)
			}
		}).
		OnActivate(func(i int) {
			if i >= 0 && i < len(t.rows) && t.onActivate != nil {
				t.onActivate(t.rows[i].node)
			}
		}).
		OnKey(t.handleKey, key.NameLeftArrow, key.NameRightArrow).
		// Rows select themselves on click; a click on an expand arrow
		// must not select.
		ClickFocusesRow(false)
	return t
}

// OnSelect sets a handler run when the selection changes. It returns
// t for chaining.
func (t *Tree) OnSelect(fn func(Node)) *Tree { t.onSelect = fn; return t }

// OnExpand sets a handler run when a node is expanded - the place to
// load its children lazily. It returns t for chaining.
func (t *Tree) OnExpand(fn func(Node)) *Tree { t.onExpand = fn; return t }

// OnCollapse sets a handler run when a node is collapsed. It returns t
// for chaining.
func (t *Tree) OnCollapse(fn func(Node)) *Tree { t.onCollapse = fn; return t }

// OnActivate sets a handler run on a double click or Enter. It returns
// t for chaining.
func (t *Tree) OnActivate(fn func(Node)) *Tree { t.onActivate = fn; return t }

// Expand opens (or, with open false, closes) the node with key,
// running OnExpand/OnCollapse. Ancestors aren't opened; see Reveal.
func (t *Tree) Expand(k any, open bool) {
	if t.expand[k] == open {
		return
	}
	if open {
		t.expand[k] = true
	} else {
		delete(t.expand, k)
	}
	if n := t.find(k); n != nil {
		if open && t.onExpand != nil {
			t.onExpand(n)
		}
		if !open && t.onCollapse != nil {
			t.onCollapse(n)
		}
	}
}

// Toggle opens a closed node and closes an open one.
func (t *Tree) Toggle(k any) {
	t.Expand(k, !t.expand[k])
}

// Expanded reports whether the node with key is open.
func (t *Tree) Expanded(k any) bool { return t.expand[k] }

// Select makes the node with key the selection, running OnSelect.
func (t *Tree) Select(k any) {
	if n := t.find(k); n != nil {
		t.selectNode(n)
		return
	}
	t.selected, t.hasSelected = k, true
}

// Selected returns the selected key.
func (t *Tree) Selected() (key any, ok bool) { return t.selected, t.hasSelected }

// Reveal opens every ancestor of the node with key, selects it and
// scrolls it into view. Only loaded nodes can be revealed.
func (t *Tree) Reveal(k any) {
	path := t.path(k)
	if path == nil {
		return
	}
	for _, n := range path[:len(path)-1] {
		t.Expand(n.Key(), true)
	}
	t.selectNode(path[len(path)-1])
	t.reveal, t.hasReveal = k, true
}

// Focus gives the tree keyboard focus on the next frame.
func (t *Tree) Focus() { t.list.Focus() }

func (t *Tree) selectNode(n Node) {
	k := n.Key()
	if t.hasSelected && t.selected == k {
		return
	}
	t.selected, t.hasSelected = k, true
	if t.onSelect != nil {
		t.onSelect(n)
	}
}
