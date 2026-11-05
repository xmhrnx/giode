package menu

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles/properties"
)

// menuItemActor lets automation activate a menu item as a click
// would.
type menuItemActor struct {
	menu *Menu
	item *Item
}

// radioDot is a selected radio item's indicator: a dot in the 16dp
// leading column.
func radioDot(c properties.Color) elements.Element {
	return elements.RawFunc(func(gtx layout.Context) layout.Dimensions {
		sz := gtx.Dp(16)
		d := gtx.Dp(6)
		o := (sz - d) / 2
		defer clip.Ellipse(image.Rect(o, o, o+d, o+d)).Push(gtx.Ops).Pop()
		paint.Fill(gtx.Ops, elements.ResolveColor(c))
		return layout.Dimensions{Size: image.Pt(sz, sz)}
	})
}

func (a menuItemActor) Click() {
	if a.item.submenu != nil {
		return
	}
	if a.item.action != nil {
		a.item.action()
	}
	a.menu.root().Close()
}

// disabledText is the text color of disabled rows.
func (m *Menu) disabledText() properties.Color {
	if m.disabledColor != "" {
		return m.disabledColor
	}
	return properties.CalcColorReverse(defaultDisabled)
}
