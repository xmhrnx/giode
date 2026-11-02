package menu

import (
	"image"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
)

// Trigger wraps el so clicking it toggles the menu, anchored just
// below it - a widget-anchored, dropdown-style trigger, alongside
// SetContextMenu (right-click anywhere) and OpenAt/Toggle (any other
// custom trigger, e.g. a long-press).
//
// The returned element must be laid out somewhere in the view like el
// would be on its own; m itself must also still be laid out somewhere
// in the tree (its panel renders via elements.RegisterPortal
// regardless of where) for its open state to actually render.
func (m *Menu) Trigger(el elements.Element) elements.Element {
	return &menuTrigger{menu: m, trigger: el, click: new(widget.Clickable)}
}

type menuTrigger struct {
	menu    *Menu
	trigger elements.Element
	click   *widget.Clickable
	// origin is the trigger's position, exact once a frame's layout is
	// done (see elements.TrackPosition).
	origin        image.Point
	height, width int
}

func (t *menuTrigger) Layout(gtx layout.Context) layout.Dimensions {
	// Handles its input without an input pass; see kit.OpaqueInput.
	kit.OpaqueInput()
	if t.click.Clicked(gtx) {
		if t.menu.open {
			t.menu.Close()
		} else {
			t.menu.open = true
		}
	}
	dims := t.click.Layout(gtx, kit.Unselectable(func(gtx layout.Context) layout.Dimensions {
		return t.trigger.Layout(gtx)
	}))
	if t.menu.open {
		// Placed from the trigger's position in the last frame, which
		// is exact even inside a row of several children (this frame's
		// is only final once the enclosing containers have placed the
		// trigger), and recomputed every frame so the menu follows it.
		p := t.origin.Add(image.Pt(0, t.height+4))
		if kit.RTL(gtx) {
			// The menu's right edge under the trigger's.
			p.X += t.width
		}
		t.menu.pos = f32.Pt(float32(p.X), float32(p.Y))
	}
	elements.TrackPosition(&t.origin)
	t.height, t.width = dims.Size.Y, dims.Size.X
	// Keep the trigger clickable even when an ancestor is a frameless
	// window's whole-window drag area - see button.Layout's identical
	// comment.
	kit.BlockDrag(gtx, dims.Size)
	return dims
}
