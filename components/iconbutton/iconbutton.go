// Package iconbutton provides a stateful clickable icon-only button
// built on the stateless elements.

package iconbutton

import (
	"gioui.org/layout"
	"gioui.org/widget"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
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

// Unstyled icon buttons fill with the theme's primary color.
var defaultBackground = "theme.primary"
