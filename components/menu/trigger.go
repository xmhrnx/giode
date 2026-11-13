package menu

import (
	"image"

	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
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
	rtl           bool
	// bar is the menu bar it belongs to (Bar.Add), or nil; hover tracks
	// the pointer for a bar that opens on hover.
	bar     *Bar
	hover   *gesture.Hover
	hovered bool
}

func (t *menuTrigger) Layout(gtx layout.Context) layout.Dimensions {
	// Handles its input without an input pass; see kit.OpaqueInput.
	kit.OpaqueInput()
	t.hoverOpens(gtx)
	if t.click.Clicked(gtx) {
		if t.menu.open {
			t.menu.Close()
		} else if t.bar != nil {
			t.bar.switchTo(t, false)
		} else {
			t.menu.open = true
		}
		// The menu's panel was already laid out (or not) this frame:
		// draw it opened or closed now, not on the next input.
		kit.RequestFrame(gtx)
	}
	dims := t.click.Layout(gtx, kit.Unselectable(func(gtx layout.Context) layout.Dimensions {
		return t.trigger.Layout(gtx)
	}))
	t.rtl = kit.RTL(gtx)
	if t.menu.open {
		// Placed from the trigger's position in the last frame, which
		// is exact even inside a row of several children (this frame's
		// is only final once the enclosing containers have placed the
		// trigger), and recomputed every frame so the menu follows it.
		t.placeMenu()
	}
	elements.TrackPosition(&t.origin)
	t.height, t.width = dims.Size.Y, dims.Size.X
	if t.hover != nil {
		// Passes presses through to the clickable beneath.
		pass := pointer.PassOp{}.Push(gtx.Ops)
		area := clip.Rect{Max: dims.Size}.Push(gtx.Ops)
		t.hover.Add(gtx.Ops)
		area.Pop()
		pass.Pop()
	}
	// Keep the trigger clickable even when an ancestor is a frameless
	// window's whole-window drag area - see button.Layout's identical
	// comment.
	kit.BlockDrag(gtx, dims.Size)
	return dims
}
