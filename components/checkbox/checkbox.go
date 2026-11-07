// Package checkbox provides a stateful checkbox with an optional
// label.
package checkbox

import (
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/xmhrnx/giode/components/icon"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Checkbox is a stateful checkbox with an optional label. Create it
// once with New and lay it out every frame.
type Checkbox struct {
	// Contextual gives it a context menu (Context).
	elements.Contextual

	label    string
	bool     widget.Bool
	pass     func(gtx layout.Context) bool
	onChange func(checked bool)
	st       styles.Styles
	icon     icon.Source
	iconSt   styles.Styles
	outlined bool
	// cache keeps the output between frames; see Layout.
	cache   *elements.MemoCache
	build   func() elements.Element
	watched elements.Watch[drawn]
}

// drawn is what a Checkbox draws from its own fields.
type drawn struct {
	label    string
	checked  bool
	st       styles.Styles
	icon     icon.Source
	iconSt   styles.Styles
	outlined bool
}

// CacheSafe reports true: what it draws comes from its own fields,
// which the cache watches.
func (c *Checkbox) CacheSafe() bool { return true }

// watch is what it draws from its own fields.
func (c *Checkbox) watch() drawn {
	return drawn{c.label, c.bool.Value, c.st, c.icon, c.iconSt, c.outlined}
}

// New returns a Checkbox. label is optional: pass "" for a bare
// checkbox with no label. The styles argument is optional: Color is
// the label color, Background is the checked fill, BorderColor is
// the unchecked border, BorderWidth and BorderRadius the border, and
// Width/Height size the box (default 18x18).
func New(label string, st ...styles.Styles) *Checkbox {
	c := &Checkbox{label: label}
	if len(st) > 0 {
		c.st = st[0]
	}
	return c
}

// Icon replaces the check mark shown when checked with any icon.Source
// (a loaded SVG, a registered icon, path data). The styles argument is
// optional and overrides the icon's Color (default white) and size
// (defaults to the box size). Icon(nil) restores the built-in check
// mark.
func (c *Checkbox) Icon(src icon.Source, st ...styles.Styles) *Checkbox {
	c.icon = src
	if len(st) > 0 {
		c.iconSt = st[0]
	}
	return c
}

// Outlined draws a checked box as an outline in the Background colour
// with a check mark of the same colour, instead of filling it: for
// outline-only designs. It returns c for chaining.
func (c *Checkbox) Outlined(on bool) *Checkbox {
	c.outlined = on
	return c
}

// Checked reports the current state.
func (c *Checkbox) Checked() bool {
	return c.bool.Value
}

// SetChecked sets the state without invoking OnChange.
func (c *Checkbox) SetChecked(v bool) {
	c.bool.Value = v
}

// OnChange sets the handler invoked when the state changes. It
// returns c for chaining.
func (c *Checkbox) OnChange(fn func(checked bool)) *Checkbox {
	c.onChange = fn
	return c
}

// update handles toggles, reporting whether the value changed.
func (c *Checkbox) update(gtx layout.Context) bool {
	if !c.bool.Update(gtx) {
		return false
	}
	if c.onChange != nil {
		c.onChange(c.bool.Value)
	}
	kit.RequestFrame(gtx)
	return true
}

// Layout lays out and updates the checkbox.
func (c *Checkbox) Layout(gtx layout.Context) layout.Dimensions {
	dims := c.layoutSelf(gtx)
	// A right-click or long-press anywhere on it opens its context menu.
	c.AnchorContext(gtx, dims.Size)
	return dims
}

// layoutSelf lays it out without its context menu area.
func (c *Checkbox) layoutSelf(gtx layout.Context) layout.Dimensions {
	// The output recorded last time is replayed until the label, the
	// value, the styles, hover, press, constraints or direction change.
	if c.cache == nil {
		c.cache = elements.NewMemoCache()
		uncached := elements.Raw(c.layoutUncached)
		c.build = func() elements.Element { return uncached }
		c.watched.Snap = c.watch
	}
	return c.cache.LayoutOwn(gtx, c.build, &c.watched)
}

// layoutUncached lays it out without the cache.
func (c *Checkbox) layoutUncached(gtx layout.Context) layout.Dimensions {
	c.update(gtx)
	st := styles.Merge(styles.Styles{Color: defaultText}, c.st)
	// On touch screens the hit area grows to 48dp around the visual.
	dims := c.bool.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		if c.bool.Hovered() {
			pointer.CursorPointer.Add(gtx.Ops)
		}
		box := c.box()
		if c.label == "" {
			return elements.MinTouchTarget(gtx, box.Layout)
		}
		return elements.MinTouchTarget(gtx, elements.Row(styles.Styles{Align: properties.AlignCenter, Gap: 8},
			box,
			elements.Text(c.label, styles.Styles{Color: st.Color, FontSize: st.FontSize}),
		).Layout)
	})
	// Keep the checkbox clickable even inside a frameless window's
	// whole-window drag area - see button.Layout's identical comment.
	kit.BlockDrag(gtx, dims.Size)
	if c.pass == nil {
		c.pass = func(gtx layout.Context) bool {
			h, p := c.bool.Hovered(), c.bool.Pressed()
			changed := c.update(gtx)
			return changed || h != c.bool.Hovered() || p != c.bool.Pressed()
		}
	}
	if !kit.HandleInput(c.pass) && gtx.Source.Focused(&c.bool) {
		kit.MarkLive()
	}
	value := "false"
	if c.bool.Value {
		value = "true"
	}
	elements.RecordSemantic(gtx, elements.SemanticNode{Kind: elements.SemanticCheckbox, Label: c.label, Value: value, Target: c}, dims.Size)
	return dims
}

// Toggle flips the state and invokes OnChange, as a click would.
func (c *Checkbox) Toggle() {
	c.bool.Value = !c.bool.Value
	if c.onChange != nil {
		c.onChange(c.bool.Value)
	}
}
