// Package iconbutton provides a stateful clickable icon-only button
// built on the stateless elements.

package iconbutton

import (
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Layout lays out and updates the button.
func (b *Button) Layout(gtx layout.Context) layout.Dimensions {
	dims := b.layoutSelf(gtx)
	// A right-click or long-press anywhere on it opens its context menu.
	b.AnchorContext(gtx, dims.Size)
	return dims
}

// layoutSelf lays it out without its context menu area.
func (b *Button) layoutSelf(gtx layout.Context) layout.Dimensions {
	b.update(gtx)
	// On touch screens the hit area grows to 48dp around the visual.
	dims := kit.LayoutClickable(gtx, &b.clickable, kit.Unselectable(func(gtx layout.Context) layout.Dimensions {
		return elements.MinTouchTarget(gtx, b.layoutVisual)
	}))
	// Keep the button clickable even when an ancestor is a frameless
	// window's whole-window drag area (see the root package's
	// layoutDrag): without this, pressing the button would be
	// hijacked into a native window-move instead of a click.
	kit.BlockDrag(gtx, dims.Size)
	if b.pass == nil {
		b.pass = kit.ClickablePass(&b.clickable, b.update)
	}
	if !kit.HandleInput(b.pass) && (gtx.Source.Focused(&b.clickable) || b.clickable.Pressed()) {
		kit.MarkLive()
	}
	elements.RecordSemantic(gtx, elements.SemanticNode{Kind: elements.SemanticButton, Label: b.label, Target: b}, dims.Size)
	return dims
}

// layoutVisual renders the button face inside the clickable region.
func (b *Button) layoutVisual(gtx layout.Context) layout.Dimensions {
	st := styles.Merge(styles.Styles{
		Padding: properties.UniformInset(8),
		Justify: properties.JustifyCenter,
		Align:   properties.AlignCenter,
	}, b.st)

	accent := st.Background
	if accent == "" {
		accent = defaultBackground
	}
	hovered := b.clickable.Hovered()
	pressed := b.clickable.Pressed()

	switch b.variant {
	case VariantOutlined:
		border := st.BorderColor
		if border == "" {
			border = accent
		}
		st.BorderColor = border
		if st.BorderWidth == 0 {
			st.BorderWidth = 1
		}
		st.Background = ""
		switch {
		case pressed:
			st.Background = kit.Alpha(border, 0.24)
		case hovered:
			st.Background = kit.Alpha(border, 0.12)
			pointer.CursorPointer.Add(gtx.Ops)
		}
	case VariantGhost:
		// Fully transparent at rest: only the icon is visible until
		// hover reveals the box. The tint follows Color (falling back
		// to Background) so a flat button can echo the icon's color.
		st.BorderWidth = 0
		tint := st.Color
		if tint == "" {
			tint = accent
		}
		st.Background = ""
		switch {
		case pressed:
			st.Background = kit.Alpha(tint, 0.24)
		case hovered:
			st.Background = kit.Alpha(tint, 0.12)
			pointer.CursorPointer.Add(gtx.Ops)
		}
	default: // VariantContained, and the zero value for compatibility.
		switch {
		case pressed:
			accent = kit.Pressed(accent)
		case hovered:
			accent = kit.Hovered(accent)
			pointer.CursorPointer.Add(gtx.Ops)
		}
		st.Background = accent
	}

	content := b.icon
	if content == nil {
		content = elements.Box(styles.Styles{})
	}
	return elements.Box(st, content).Layout(gtx)
}
