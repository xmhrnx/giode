// Package iconbutton provides a stateful clickable icon-only button
// built on the stateless elements.
package iconbutton

import (
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/mehran9675/giode/components/kit"
	"github.com/mehran9675/giode/elements"
	"github.com/mehran9675/giode/styles"
	"github.com/mehran9675/giode/styles/properties"
)

// Variant selects the button's visual style.
type Variant string

const (
	// VariantContained fills the button with Background (or black if
	// unset) at all times - the default.
	VariantContained Variant = "contained"
	// VariantOutlined is transparent with a visible border, filling
	// faintly with the border color on hover/press.
	VariantOutlined Variant = "outlined"
	// VariantGhost is fully transparent at rest - only the icon is
	// visible - filling faintly with Color on hover/press.
	VariantGhost Variant = "ghost"
)

// Button is a stateful clickable icon-only button. Create it once with
// New and lay it out every frame; the icon, styles and handler may
// change at any time.
type Button struct {
	icon      elements.Element
	st        styles.Styles
	variant   Variant
	onClick   func()
	clickable widget.Clickable
	pass      func(gtx layout.Context) bool
	label     string
}

// New returns a Button laying out icon. The styles argument is
// optional: Background is the fill, Color the hover/press tint of the
// transparent variants. The icon carries its own color and size.
func New(icon elements.Element, st ...styles.Styles) *Button {
	b := &Button{icon: icon}
	if len(st) > 0 {
		b.st = st[0]
	}
	return b
}

// OnClick sets the handler invoked on every click. It returns b for
// chaining.
func (b *Button) OnClick(fn func()) *Button {
	b.onClick = fn
	return b
}

// Icon replaces the icon. It returns b for chaining.
func (b *Button) Icon(icon elements.Element) *Button {
	b.icon = icon
	return b
}

// Styles replaces the button styles. It returns b for chaining.
func (b *Button) Styles(st styles.Styles) *Button {
	b.st = st
	return b
}

// Variant sets the button's visual style (VariantContained, the
// default, VariantOutlined or VariantGhost). It returns b for
// chaining.
func (b *Button) Variant(v Variant) *Button {
	b.variant = v
	return b
}

// Label sets the accessible name reported to automation (the
// semantic tree, the MCP server), since the button shows no text. It
// returns b for chaining.
func (b *Button) Label(label string) *Button {
	b.label = label
	return b
}

// Click simulates a click programmatically.
func (b *Button) Click() {
	b.clickable.Click()
}

// update handles clicks, reporting whether one called the handler.
func (b *Button) update(gtx layout.Context) bool {
	if b.clickable.Clicked(gtx) && b.onClick != nil {
		b.onClick()
		// The handler may have changed state that this frame's view
		// already read before this click was processed; request
		// another frame so the change shows up right away instead of
		// waiting for an unrelated event to trigger a repaint.
		kit.RequestFrame(gtx)
		return true
	}
	return false
}

// Layout lays out and updates the button.
func (b *Button) Layout(gtx layout.Context) layout.Dimensions {
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

// Unstyled icon buttons fill with the theme's primary color.
var defaultBackground = "theme.primary"

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
