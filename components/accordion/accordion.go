// Package accordion provides a stateful list of expand/collapse
// sections.
package accordion

import (
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

var (
	defaultBorder = "#334155"
	defaultText   = "#e2e8f0"
	defaultHover  = "#334155"
)

// item is one section: a header and its lazily-built content.
type item struct {
	label    string
	content  func() elements.Element
	expanded bool
	click    widget.Clickable
}

// Accordion is a stateful list of expand/collapse sections. Create it
// once with New and lay it out every frame; sections may be added at
// any time.
type Accordion struct {
	// Contextual gives it a context menu (Context).
	elements.Contextual

	items  []*item
	single bool
	st     styles.Styles
}

// New returns an empty accordion. The styles argument is optional:
// Color styles the label and chevron icon, BorderColor the divider
// between sections.
func New(st ...styles.Styles) *Accordion {
	a := &Accordion{}
	if len(st) > 0 {
		a.st = st[0]
	}
	return a
}

// Item appends a section: label is its header, content is built
// lazily, only while the section is expanded. It returns a for
// chaining.
func (a *Accordion) Item(label string, content func() elements.Element) *Accordion {
	a.items = append(a.items, &item{label: label, content: content})
	return a
}

// SingleOpen controls whether opening a section closes any other open
// one (true) or sections expand independently (false, the default).
// It returns a for chaining.
func (a *Accordion) SingleOpen(v bool) *Accordion {
	a.single = v
	return a
}

// Expanded reports whether section i is open.
func (a *Accordion) Expanded(i int) bool {
	if i < 0 || i >= len(a.items) {
		return false
	}
	return a.items[i].expanded
}

// SetExpanded opens or closes section i. Opening one closes every
// other section when SingleOpen(true) is set.
func (a *Accordion) SetExpanded(i int, v bool) {
	if i < 0 || i >= len(a.items) {
		return
	}
	if v && a.single {
		for _, it := range a.items {
			it.expanded = false
		}
	}
	a.items[i].expanded = v
}

// Layout lays out every section header, and the content of whichever
// sections are expanded.
func (a *Accordion) Layout(gtx layout.Context) layout.Dimensions {
	dims := a.layoutSelf(gtx)
	// A right-click or long-press anywhere on it opens its context menu.
	a.AnchorContext(gtx, dims.Size)
	return dims
}

// layoutSelf lays it out without its context menu area.
func (a *Accordion) layoutSelf(gtx layout.Context) layout.Dimensions {
	// Handles its input without an input pass; see kit.OpaqueInput.
	kit.OpaqueInput()
	st := styles.Merge(styles.Styles{
		Color:       defaultText,
		BorderColor: defaultBorder,
	}, a.st)
	st.FlexDirection = properties.FlexDirectionColumn

	children := make([]elements.Element, 0, len(a.items)*2)
	for i, it := range a.items {
		if it.click.Clicked(gtx) {
			a.SetExpanded(i, !it.expanded)
			kit.RequestFrame(gtx)
		}
		children = append(children, a.header(it, st))
		if it.expanded {
			children = append(children, elements.Box(styles.Styles{Padding: properties.SymmetricInset(12, 12)}, it.content()))
		}
		if i < len(a.items)-1 {
			children = append(children, elements.Divider(styles.Styles{Color: st.BorderColor, Height: 1}))
		}
	}
	return elements.Box(st, children...).Layout(gtx)
}

// header renders one section's clickable header row: the label, a
// chevron indicating open/closed, and a hover highlight.
func (a *Accordion) header(it *item, st styles.Styles) elements.Element {
	return elements.Raw(func(gtx layout.Context) layout.Dimensions {
		dims := it.click.Layout(gtx, kit.Unselectable(func(gtx layout.Context) layout.Dimensions {
			hst := styles.Styles{Padding: properties.SymmetricInset(12, 12)}
			if it.click.Hovered() {
				hst.Background = defaultHover
				pointer.CursorPointer.Add(gtx.Ops)
			}
			chevron := kit.GlyphChevronForward
			if it.expanded {
				chevron = kit.GlyphChevronDown
			}
			row := elements.Row(styles.Styles{Align: properties.AlignCenter, Justify: properties.JustifySpaceBetween, Width: -1},
				elements.Text(it.label, styles.Styles{Color: st.Color}),
				elements.Glyph(chevron, styles.Styles{Color: st.Color, Width: 16}),
			)
			return elements.Box(hst, row).Layout(gtx)
		}))
		// Keep the header clickable even when an ancestor is a
		// frameless window's whole-window drag area - see
		// button.Layout's identical comment.
		kit.BlockDrag(gtx, dims.Size)
		return dims
	})
}
