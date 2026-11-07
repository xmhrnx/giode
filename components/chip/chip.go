// Package chip provides chips: small rounded labels for filters,
// tags and choices, optionally selectable and removable.
package chip

import (
	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// chipColor is the default fill: a faint tint of the text, mixed opaque
// so it stays faint on dark themes.
var chipColor = properties.Mix("theme.text", "theme.background", 0.08)

// Chip is a rounded label with an optional icon, a selected state and a
// remove button. Without OnClick or OnRemove it's a plain element that
// can be built every frame; with them, create it once (or with Keyed),
// since it then holds press state.
type Chip struct {
	// Contextual gives it a context menu (Context).
	elements.Contextual

	label    string
	icon     elements.Element
	selected bool
	onClick  func()
	onRemove func()
	st       styles.Styles
	selSt    styles.Styles

	press      *elements.Pressable
	remove     widget.Clickable
	removePass func(gtx layout.Context) bool
}

// New returns a chip. The styles argument is optional: Background
// (default a faint tint of the text), Color (default theme.text),
// BorderColor, FontSize. They style the unselected chip; a selected
// chip keeps its theme.primary fill and theme.onprimary text unless
// SelectedStyles sets others.
func New(label string, st ...styles.Styles) *Chip {
	c := &Chip{label: label}
	if len(st) > 0 {
		c.st = st[0]
	}
	return c
}

// Icon shows el before the label. It returns c for chaining.
func (c *Chip) Icon(el elements.Element) *Chip {
	c.icon = el
	return c
}

// Selected sets the selected state, drawn filled. It returns c for
// chaining.
func (c *Chip) Selected(on bool) *Chip {
	c.selected = on
	return c
}

// SelectedStyles sets the styles merged over st when the chip is
// selected (default Background theme.primary, Color theme.onprimary,
// BorderColor theme.primary). It returns c for chaining.
func (c *Chip) SelectedStyles(st styles.Styles) *Chip {
	c.selSt = st
	return c
}

// IsSelected reports the selected state.
func (c *Chip) IsSelected() bool { return c.selected }

// OnClick makes the chip tappable. It returns c for chaining.
func (c *Chip) OnClick(fn func()) *Chip {
	c.onClick = fn
	return c
}

// OnRemove adds a remove (x) button after the label. It returns c for
// chaining.
func (c *Chip) OnRemove(fn func()) *Chip {
	c.onRemove = fn
	return c
}

// Click runs the click handler, as a tap would.
func (c *Chip) Click() {
	if c.onClick != nil {
		c.onClick()
	}
}

// styles returns the chip's styles for its selected state.
func (c *Chip) styles() styles.Styles {
	st := styles.Merge(styles.Styles{
		Background: chipColor, Color: "theme.text", FontSize: 13, BorderRadius: 999,
		Padding: properties.SymmetricInset(12, 6), Align: properties.AlignCenter, Gap: 6,
	}, c.st)
	if c.selected {
		st.Background, st.Color = "theme.primary", "theme.onprimary"
		if st.BorderColor != "" {
			st.BorderColor = "theme.primary"
		}
		st = styles.Merge(st, c.selSt)
	}
	if st.BorderColor != "" && st.BorderWidth == 0 {
		st.BorderWidth = 1
	}
	return st
}

func (c *Chip) content(p elements.PressState) elements.Element {
	st := c.styles()
	if p.Pressed || p.Hovered {
		st.Background = kit.Hovered(st.Background)
	}
	var kids []elements.Element
	if c.icon != nil {
		kids = append(kids, c.icon)
	}
	kids = append(kids, elements.Text(c.label, styles.Styles{Color: st.Color, FontSize: st.FontSize}))
	if c.onRemove != nil {
		kids = append(kids, elements.Raw(func(gtx layout.Context) layout.Dimensions {
			c.updateRemove(gtx)
			if c.removePass == nil {
				c.removePass = kit.ClickablePass(&c.remove, c.updateRemove)
			}
			kit.HandleInput(c.removePass)
			return c.remove.Layout(gtx, elements.Glyph(kit.GlyphClose, styles.Styles{Color: st.Color, Width: 14}).Layout)
		}))
	}
	return elements.Row(st, kids...)
}

// updateRemove handles the remove button's clicks.
func (c *Chip) updateRemove(gtx layout.Context) bool {
	if c.remove.Clicked(gtx) {
		c.onRemove()
		kit.RequestFrame(gtx)
		return true
	}
	return false
}

// Layout lays out the chip.
func (c *Chip) Layout(gtx layout.Context) layout.Dimensions {
	dims := c.layoutSelf(gtx)
	// A right-click or long-press anywhere on it opens its context menu.
	c.AnchorContext(gtx, dims.Size)
	return dims
}

// layoutSelf lays it out without its context menu area.
func (c *Chip) layoutSelf(gtx layout.Context) layout.Dimensions {
	if c.onClick == nil {
		return c.content(elements.PressState{}).Layout(gtx)
	}
	if c.press == nil {
		c.press = elements.NewPressable(c.content).Label(c.label).OnClick(func(elements.ClickEvent) { c.Click() })
	}
	return c.press.Layout(gtx)
}
