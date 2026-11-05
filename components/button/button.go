// Package button provides a stateful clickable button component built
// on the stateless elements.
package button

import (
	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
)

// Variant selects the button's visual style.
type Variant string

const (
	// VariantContained fills the button with Background (or black if
	// unset) at all times - the default, and today's only look before
	// Variant existed.
	VariantContained Variant = "contained"
	// VariantOutlined is transparent with a visible border, filling
	// faintly with the border color on hover/press.
	VariantOutlined Variant = "outlined"
	// VariantGhost is fully transparent at rest - only the label is
	// visible - filling faintly with Background on hover/press.
	VariantGhost Variant = "ghost"
)

// Button is a stateful clickable button. Create it once with New and
// lay it out every frame; label, styles and handler may change at any
// time.
type Button struct {
	label     string
	st        styles.Styles
	variant   Variant
	onClick   func()
	clickable widget.Clickable
	pass      func(gtx layout.Context) bool

	icon, trailing elements.Element
	disabled       bool
	loading        bool

	// cache keeps the button's output between frames; see Layout.
	cache *elements.MemoCache
	build func() elements.Element
}

// CacheSafe reports whether the button can cache its output: its icons,
// if any, must be CacheSafe too.
func (b *Button) CacheSafe() bool {
	return (b.icon == nil || elements.IsCacheSafe(b.icon)) && (b.trailing == nil || elements.IsCacheSafe(b.trailing))
}

// watch is what the button draws from its own fields.
func (b *Button) watch() []any {
	return []any{b.label, b.st, b.variant, b.disabled, b.loading, b.icon, b.trailing}
}

// New returns a Button with the given label. The styles argument is
// optional.
func New(label string, st ...styles.Styles) *Button {
	b := &Button{label: label}
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

// Label replaces the button label. It returns b for chaining.
func (b *Button) Label(label string) *Button {
	b.label = label
	return b
}

// Variant sets the button's visual style (VariantContained, the
// default, VariantOutlined or VariantGhost). It returns b for
// chaining.
func (b *Button) Variant(v Variant) *Button {
	b.variant = v
	return b
}

// Styles replaces the button styles. It returns b for chaining.
func (b *Button) Styles(st styles.Styles) *Button {
	b.st = st
	return b
}

// Icon sets an element shown before the label, typically an icon.
// Icons that set no Color of their own follow the label color. It
// returns b for chaining; Icon(nil) removes it.
func (b *Button) Icon(el elements.Element) *Button {
	b.icon = el
	return b
}

// TrailingIcon sets an element shown after the label - a chevron, an
// external-link mark. It follows the label color like Icon. It returns
// b for chaining.
func (b *Button) TrailingIcon(el elements.Element) *Button {
	b.trailing = el
	return b
}

// Disabled dims the button and ignores input: no clicks, no hover,
// default cursor. It returns b for chaining.
func (b *Button) Disabled(v bool) *Button {
	b.disabled = v
	return b
}

// Loading shows a spinner in place of the icon and ignores clicks
// while v is true - for actions in progress. It returns b for
// chaining.
func (b *Button) Loading(v bool) *Button {
	b.loading = v
	return b
}

// IsDisabled reports whether the button is disabled.
func (b *Button) IsDisabled() bool { return b.disabled }

// IsLoading reports whether the button shows its loading state.
func (b *Button) IsLoading() bool { return b.loading }

// Click simulates a click programmatically. It does nothing while the
// button is disabled or loading.
func (b *Button) Click() {
	if b.disabled || b.loading {
		return
	}
	b.clickable.Click()
}

// inert reports whether input is ignored.
func (b *Button) inert() bool {
	return b.disabled || b.loading
}

// update handles clicks, reporting whether one called the handler.
func (b *Button) update(gtx layout.Context) bool {
	if b.clickable.Clicked(gtx) && b.onClick != nil && !b.inert() {
		b.onClick()
		// The handler may have changed state that this frame's view
		// already read before this click was processed (e.g. a
		// counter formatted into a Text earlier in the tree); request
		// another frame so the change shows up right away instead of
		// waiting for an unrelated event to trigger a repaint.
		kit.RequestFrame(gtx)
		return true
	}
	return false
}

// Layout lays out and updates the button. It replays the output
// recorded last time until its label, styles, state, hover or press,
// constraints or direction change.
func (b *Button) Layout(gtx layout.Context) layout.Dimensions {
	if b.CacheSafe() {
		if b.cache == nil {
			b.cache = elements.NewMemoCache()
			uncached := elements.Raw(b.layoutUncached)
			b.build = func() elements.Element { return uncached }
		}
		return b.cache.LayoutOwn(gtx, b.build, b.watch)
	}
	return b.layoutUncached(gtx)
}

func (b *Button) layoutUncached(gtx layout.Context) layout.Dimensions {
	if b.inert() {
		// No events reach a disabled or loading button.
		gtx = gtx.Disabled()
	}
	b.update(gtx)
	// On touch screens the hit area grows to 48dp around the visual.
	dims := b.clickable.Layout(gtx, kit.Unselectable(func(gtx layout.Context) layout.Dimensions {
		return elements.MinTouchTarget(gtx, b.layoutVisual)
	}))
	// Keep the button clickable even when an ancestor is a frameless
	// window's whole-window drag area (see the root package's
	// layoutDrag): without this, pressing the button would be
	// hijacked into a native window-move instead of a click.
	kit.BlockDrag(gtx, dims.Size)
	if b.pass == nil {
		b.pass = kit.ClickablePass(&b.clickable, func(gtx layout.Context) bool {
			if b.inert() {
				gtx = gtx.Disabled()
			}
			return b.update(gtx)
		})
	}
	if !kit.HandleInput(b.pass) && !b.inert() && (gtx.Source.Focused(&b.clickable) || b.clickable.Pressed()) {
		// Keyboard activation must reach the clickable even inside a
		// cached subtree.
		kit.MarkLive()
	}
	elements.RecordSemantic(gtx, elements.SemanticNode{Kind: elements.SemanticButton, Label: b.label, Disabled: b.inert(), Target: b}, dims.Size)
	return dims
}
