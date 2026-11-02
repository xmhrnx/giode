package main

import "github.com/xmhrnx/giode"

// dialogShowcase holds the dialog and its own open state (owned by
// the caller, per the dialog's external-state contract), created once
// so they persist across frames.
type dialogShowcase struct {
	open   bool
	dialog giode.Element
	toggle giode.Element
}

func NewDialogShowcase() *dialogShowcase {
	d := &dialogShowcase{}
	d.toggle = giode.Button("Open dialog", giode.Styles{Background: "gray", BorderRadius: 10}).
		OnClick(func() { d.open = true })
	d.dialog = giode.Dialog(&d.open, func() giode.Element {
		return giode.Text("This is a dialog", giode.Styles{Color: "white"})
	}, giode.Styles{Background: "#1e293b"})
	return d
}

// View returns the button that opens the dialog, to place inline with
// the other showcase pages.
func (d *dialogShowcase) View() giode.Element {
	return giode.Stack(boxStyle, d.toggle)
}

// Overlay returns the dialog itself. Lay it out last in the tree
// (alongside drawerPage.Overlay) so its scrim covers the whole
// window rather than just the card it was opened from; it is a
// no-op while closed.
func (d *dialogShowcase) Overlay() giode.Element {
	return d.dialog
}
