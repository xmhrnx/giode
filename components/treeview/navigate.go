// Package treeview provides a virtualized, keyboard-navigable tree of
// expandable rows, owning expansion and selection while the caller
// only draws each row's content.

package treeview

import (
	"gioui.org/io/key"
	"github.com/xmhrnx/giode/elements"
)

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
