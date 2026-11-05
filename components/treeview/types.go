// Package treeview provides a virtualized, keyboard-navigable tree of
// expandable rows, owning expansion and selection while the caller
// only draws each row's content.

package treeview

import (
	"github.com/xmhrnx/giode/elements"
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

type rowWidgets struct {
	row   *elements.Pressable
	arrow *elements.Pressable
}
