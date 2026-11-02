// Package treeview provides a virtualized, keyboard-navigable tree of
// expandable rows, owning expansion and selection while the caller
// only draws each row's content.
package treeview

import (
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/components/spinner"
	"github.com/xmhrnx/giode/components/virtuallist"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Node is one item of the tree.
type Node interface {
	// Key identifies the node across frames; it must be comparable and
	// unique in the tree (a path, an ID).
	Key() any
	// Children returns the loaded children. It is only called for
	// expanded nodes.
	Children() []Node
	// HasChildren reports whether the node can expand, before its
	// children are loaded - so a folder shows its arrow and loads its
	// contents when first expanded (see OnExpand).
	HasChildren() bool
}

// RowState is what a row is drawn with.
type RowState struct {
	// Depth is the nesting level, 0 for roots.
	Depth int
	// Expanded reports whether the node is open.
	Expanded bool
	// Selected reports whether the node is the selection.
	Selected bool
	// Hovered reports whether the pointer is over the row.
	Hovered bool
	// Loading reports an expanded node whose children haven't loaded
	// yet (HasChildren but no Children).
	Loading bool
	// Focused reports whether the tree has keyboard focus.
	Focused bool
}

const (
	defaultRowHeight = 28 // dp
	indent           = 16 // dp per level
	arrowBox         = 20 // dp
)

// row is one visible node in display order.
type row struct {
	node   Node
	depth  int
	parent int // index of the parent row, -1 for roots
}

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

type rowWidgets struct {
	row   *elements.Pressable
	arrow *elements.Pressable
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

// find returns the loaded node with key, searching visible rows first.
func (t *Tree) find(k any) Node {
	if i, ok := t.byKey[k]; ok && i < len(t.rows) {
		return t.rows[i].node
	}
	if p := t.path(k); p != nil {
		return p[len(p)-1]
	}
	return nil
}

// path returns the chain of loaded nodes from a root to key.
func (t *Tree) path(k any) []Node {
	var walk func(nodes []Node, prefix []Node) []Node
	walk = func(nodes []Node, prefix []Node) []Node {
		for _, n := range nodes {
			p := append(prefix[:len(prefix):len(prefix)], n)
			if n.Key() == k {
				return p
			}
			if n.HasChildren() {
				if r := walk(n.Children(), p); r != nil {
					return r
				}
			}
		}
		return nil
	}
	if t.roots == nil {
		return nil
	}
	return walk(t.roots(), nil)
}

// flatten lists the visible rows in display order.
func (t *Tree) flatten() {
	t.rows = t.rows[:0]
	clear(t.byKey)
	var walk func(nodes []Node, depth, parent int)
	walk = func(nodes []Node, depth, parent int) {
		for _, n := range nodes {
			i := len(t.rows)
			t.rows = append(t.rows, row{node: n, depth: depth, parent: parent})
			t.byKey[n.Key()] = i
			if t.expand[n.Key()] && n.HasChildren() {
				walk(n.Children(), depth+1, i)
			}
		}
	}
	if t.roots != nil {
		walk(t.roots(), 0, -1)
	}
}

// handleKey implements Left and Right.
func (t *Tree) handleKey(e key.Event, focused int) {
	if focused < 0 || focused >= len(t.rows) {
		return
	}
	r := t.rows[focused]
	k := r.node.Key()
	switch e.Name {
	case key.NameRightArrow:
		switch {
		case r.node.HasChildren() && !t.expand[k]:
			t.Expand(k, true)
		case t.expand[k] && focused+1 < len(t.rows) && t.rows[focused+1].parent == focused:
			t.list.SetFocusedIndex(focused + 1)
		}
	case key.NameLeftArrow:
		switch {
		case t.expand[k]:
			t.Expand(k, false)
		case r.parent >= 0:
			t.list.SetFocusedIndex(r.parent)
		}
	}
	elements.InvalidateCaches()
}

// widgets returns a node's persistent row and arrow pressables.
func (t *Tree) widgets(n Node) *rowWidgets {
	k := n.Key()
	return t.presses.Get(k, func() *rowWidgets {
		w := &rowWidgets{}
		w.arrow = elements.NewPressable(nil).Focusable(false).OnClick(func(elements.ClickEvent) {
			t.Toggle(k)
		})
		w.row = elements.NewPressable(nil).Focusable(false).OnClick(func(e elements.ClickEvent) {
			if i, ok := t.byKey[k]; ok {
				t.list.SetFocusedIndex(i)
				t.list.Focus()
				t.selectNode(t.rows[i].node)
			}
			if e.Clicks == 2 && e.Button == elements.MousePrimary && t.onActivate != nil {
				if i, ok := t.byKey[k]; ok {
					t.onActivate(t.rows[i].node)
				}
			}
		})
		return w
	})
}

// rowView builds visible row i.
func (t *Tree) rowView(i int) elements.Element {
	if i >= len(t.rows) {
		return nil
	}
	r := t.rows[i]
	n := r.node
	k := n.Key()
	w := t.widgets(n)
	expanded := t.expand[k]
	hasKids := n.HasChildren()
	state := RowState{
		Depth:    r.depth,
		Expanded: expanded && hasKids,
		Selected: t.hasSelected && t.selected == k,
		Hovered:  w.row.Hovered(),
		Loading:  expanded && hasKids && len(n.Children()) == 0,
		Focused:  t.list.HasFocus(),
	}
	w.row.Label(labelOf(n)).Content(func(s elements.PressState) elements.Element {
		state.Hovered = s.Hovered
		bg := ""
		sel := t.st.BorderColor
		if sel == "" {
			sel = "theme.primary"
		}
		switch {
		case state.Selected && state.Focused:
			bg = kit.Alpha(sel, 0.30)
		case state.Selected:
			bg = kit.Alpha(sel, 0.16)
		case s.Hovered:
			bg = "theme.text/0.06"
		}
		arrowColor := t.st.Color
		if arrowColor == "" {
			arrowColor = "theme.text"
		}
		var arrow elements.Element = elements.Box(styles.Styles{Width: arrowBox})
		switch {
		case state.Loading:
			arrow = elements.Box(styles.Styles{Width: arrowBox, Justify: properties.JustifyCenter, Align: properties.AlignCenter},
				spinner.Spinner(styles.Styles{Color: arrowColor, Width: 12}))
		case hasKids:
			g := kit.GlyphChevronForward
			if expanded {
				g = kit.GlyphChevronDown
			}
			arrow = w.arrow.Content(func(as elements.PressState) elements.Element {
				st := styles.Styles{Width: arrowBox, Height: arrowBox, Justify: properties.JustifyCenter, Align: properties.AlignCenter, BorderRadius: 4}
				if as.Hovered {
					st.Background = "theme.text/0.1"
				}
				return elements.Box(st, elements.Glyph(g, styles.Styles{Color: arrowColor, Width: 14}))
			})
		}
		var content elements.Element = elements.Box(styles.Styles{})
		if t.draw != nil {
			if c := t.draw(n, state); c != nil {
				content = c
			}
		}
		return elements.Row(styles.Styles{
			Width: -1, Height: -1, Background: bg, Align: properties.AlignCenter, Wrap: properties.WrapNoWrap,
			Padding: properties.Inset{Start: 4 + r.depth*indent, End: 8}, Gap: 4,
		}, arrow, elements.Box(styles.Styles{FlexGrow: 1}, content))
	})
	return w.row
}

// labelOf names a node for automation: its Label or String method, or
// its key.
func labelOf(n Node) string {
	switch v := n.(type) {
	case interface{ Label() string }:
		return v.Label()
	case interface{ String() string }:
		return v.String()
	}
	if s, ok := n.Key().(string); ok {
		return s
	}
	return ""
}

// Layout lays out the visible rows.
func (t *Tree) Layout(gtx layout.Context) layout.Dimensions {
	// Handles its input without an input pass; see kit.OpaqueInput.
	kit.OpaqueInput()
	t.flatten()
	if t.hasReveal {
		if i, ok := t.byKey[t.reveal]; ok {
			t.list.SetFocusedIndex(i)
			t.list.ScrollToIndex(i, virtuallist.AlignNearest)
		}
		t.hasReveal = false
	}
	// Keep the list's focused row on the selection.
	if t.hasSelected {
		if i, ok := t.byKey[t.selected]; ok && t.list.FocusedIndex() != i {
			t.list.SetFocusedIndex(i)
		}
	}
	return t.list.Layout(gtx)
}
