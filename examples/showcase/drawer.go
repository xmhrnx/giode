package main

import "github.com/xmhrnx/giode"

// drawerPage holds the drawer and its own open state (owned by the
// caller, per the drawer's external-state contract), created once so
// they persist across frames.
type drawerPage struct {
	open   bool
	drawer giode.Element
	toggle giode.Element
}

func NewDrawerPage() *drawerPage {
	p := &drawerPage{}
	p.drawer = giode.Drawer(&p.open, giode.SideRight, 100, func() giode.Element {
		return giode.Box(
			giode.Styles{Padding: giode.UniformInset(16)},
			giode.Text("Drawer content", giode.Styles{Color: "white"}),
		)
	}, giode.Styles{Background: "purple"})
	p.toggle = giode.Button("Open drawer", giode.Styles{Background: "gray", BorderRadius: 10}).
		OnClick(func() { p.open = true })
	return p
}

// View returns the button that opens the drawer, to place inline with
// the other showcase pages.
func (p *drawerPage) View() giode.Element {
	return giode.Stack(boxStyle, p.toggle)
}

// Overlay returns the drawer itself. Lay it out last in the tree
// (e.g. after the rest of the Stack) so it paints above the UI; it is
// a no-op while closed.
func (p *drawerPage) Overlay() giode.Element {
	return p.drawer
}
