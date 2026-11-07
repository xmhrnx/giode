package menu

import (
	"image"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"

	"github.com/xmhrnx/giode/components/kit"

	"github.com/xmhrnx/giode/elements"
)

// Layout renders the open menu: a window-wide dismissal layer, plus
// the panel itself queued as a portal (see elements.RegisterPortalAt)
// so it paints above everything else regardless of where Layout is
// called from. It takes no space where it's laid out. A submenu (see
// SubMenu) skips the dismissal layer - its own parent row's hover
// state controls it instead.
func (m *Menu) Layout(gtx layout.Context) layout.Dimensions {
	// Handles its input without an input pass; see kit.OpaqueInput.
	kit.OpaqueInput()
	if !m.open {
		return layout.Dimensions{}
	}
	m.laidOut = elements.FrameNumber()

	if m.isSubmenu {
		// Placed by its parent row (see submenuX), which moves with the
		// panel.
		elements.RegisterPortal(gtx, image.Pt(int(m.pos.X), int(m.pos.Y)), func() elements.Element {
			return elements.Raw(m.panel)
		})
		return layout.Dimensions{}
	}

	// Back and Escape close the menu.
	elements.HandleDismiss(m.Close)
	if gtx.Source.Enabled() {
		m.handleKeys(gtx)
	}
	// The dismissal layer covers the whole window, wherever the menu is
	// laid out, and the panel opens at the window position it was given:
	// neither follows the containers around the menu.
	elements.RegisterOverlay(func() elements.Element { return elements.Raw(m.layoutScrim) })
	// Right to left, the menu extends leftward from where it opens (the
	// pointer, or its trigger's right edge).
	elements.RegisterPortalAt(gtx, image.Pt(int(m.pos.X), int(m.pos.Y)), kit.RTL(gtx), func() elements.Element {
		return elements.Raw(m.panel)
	})
	return layout.Dimensions{}
}

// LastLaidOut is the frame (elements.FrameNumber) the menu was last
// laid out open in: the frame doesn't lay out an element's context
// menu again when the view already did.
func (m *Menu) LastLaidOut() uint64 {
	return m.laidOut
}

// layoutScrim covers the window under the open menu: a press of any
// button there closes the menu and goes no further, so a right-click
// elsewhere only closes it rather than opening another under it.
func (m *Menu) layoutScrim(gtx layout.Context) layout.Dimensions {
	for {
		ev, ok := gtx.Source.Event(pointer.Filter{Target: &m.scrim, Kinds: pointer.Press})
		if !ok {
			break
		}
		if _, ok := ev.(pointer.Event); ok && m.open {
			m.Close()
			kit.RequestFrame(gtx)
		}
	}
	defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, &m.scrim)
	return layout.Dimensions{}
}

// OutOfFlow marks the menu as taking no space in a Box: it paints over
// the window; see elements.OutOfFlow.
func (m *Menu) OutOfFlow() {}
