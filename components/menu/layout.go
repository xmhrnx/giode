package menu

import (
	"image"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"

	"github.com/mehran9675/giode/components/kit"

	"github.com/mehran9675/giode/elements"
)

// Layout renders the open menu: a window-wide dismissal layer, plus
// the panel itself queued as a portal (see elements.RegisterPortal)
// so it paints above everything else regardless of where Layout is
// called from. A submenu (see SubMenu) skips the dismissal layer -
// its own parent row's hover state controls it instead.
func (m *Menu) Layout(gtx layout.Context) layout.Dimensions {
	// Handles its input without an input pass; see kit.OpaqueInput.
	kit.OpaqueInput()
	if !m.open {
		return layout.Dimensions{}
	}

	if !m.isSubmenu {
		// A press of any button outside the panel closes it, and goes
		// no further: a right-click elsewhere only closes the menu
		// rather than opening another under it.
		closed := false
		for {
			ev, ok := gtx.Source.Event(pointer.Filter{Target: &m.scrim, Kinds: pointer.Press})
			if !ok {
				break
			}
			if _, ok := ev.(pointer.Event); ok {
				closed = true
			}
		}
		if closed {
			m.Close()
			kit.RequestFrame(gtx)
			return layout.Dimensions{}
		}
		area := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
		event.Op(gtx.Ops, &m.scrim)
		area.Pop()

		// Back and Escape close the menu.
		elements.HandleDismiss(m.Close)
		if gtx.Source.Enabled() {
			m.handleKeys(gtx)
		}
	}

	register := elements.RegisterPortal
	if !m.isSubmenu && kit.RTL(gtx) {
		// Right to left, the menu extends leftward from where it opens
		// (the pointer, or its trigger's right edge); submenus place
		// themselves (see submenuX).
		register = elements.RegisterPortalEnd
	}
	register(gtx, image.Pt(int(m.pos.X), int(m.pos.Y)), func() elements.Element {
		return elements.Raw(m.panel)
	})
	return layout.Dimensions{Size: gtx.Constraints.Max}
}
