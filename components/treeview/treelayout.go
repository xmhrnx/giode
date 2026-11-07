// Package treeview provides a virtualized, keyboard-navigable tree of
// expandable rows, owning expansion and selection while the caller
// only draws each row's content.

package treeview

import (
	"gioui.org/layout"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/components/virtuallist"
)

// Layout lays out the visible rows.
func (t *Tree) Layout(gtx layout.Context) layout.Dimensions {
	dims := t.layoutSelf(gtx)
	// A right-click or long-press anywhere on it opens its context menu.
	t.AnchorContext(gtx, dims.Size)
	return dims
}

// layoutSelf lays it out without its context menu area.
func (t *Tree) layoutSelf(gtx layout.Context) layout.Dimensions {
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
