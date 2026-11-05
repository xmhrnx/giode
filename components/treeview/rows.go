// Package treeview provides a virtualized, keyboard-navigable tree of
// expandable rows, owning expansion and selection while the caller
// only draws each row's content.

package treeview

import (
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/components/spinner"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

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
