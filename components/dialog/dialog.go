// Package dialog provides a stateful centered modal dialog.
package dialog

import (
	"image"

	"gioui.org/widget"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

var defaultBackground = "#1e293b"

// Dialog is a modal dialog with a dimmed scrim. It implements
// elements.Element, so it can be placed directly in the tree like any
// other element — lay it out last (e.g. in Stack) so it paints above
// the rest of the UI. It is a no-op (zero size, nothing painted)
// whenever *open is false.
//
// Create it once with New, outside the view function.
type Dialog struct {
	open    *bool
	content func() elements.Element
	st      styles.Styles
	// fixed disables closing from the scrim, Back and Escape.
	fixed bool
	scrim widget.Clickable
	// panel takes the clicks on the panel's own empty space, which
	// would otherwise fall through to the scrim and close the dialog.
	panel widget.Clickable
	// padSet records an explicit Padding, which may be zero.
	padSet bool
	// size is the panel's size last frame, used to center it before
	// laying it out (see Layout).
	size image.Point
}

// New returns a Dialog bound to open: it is visible whenever *open is
// true, and clicking the scrim sets *open back to false. content
// builds the panel body fresh every frame the dialog is shown, so it
// can reflect state that changed since the dialog opened. The styles
// argument is optional and applies to the panel.
func New(open *bool, content func() elements.Element, st ...styles.Styles) *Dialog {
	d := &Dialog{open: open, content: content}
	if len(st) > 0 {
		d.st = st[0]
	}
	return d
}

// Padding sets the panel's padding, zero included: the styles' Padding
// can't say zero, since an all-zero inset means "use the default"
// (20dp). Padding(giode.Inset{}) lets the content reach the panel's
// edges, as for a settings panel with a sidebar. It returns d for
// chaining.
func (d *Dialog) Padding(p properties.Inset) *Dialog {
	d.st.Padding, d.padSet = p, true
	return d
}

// Opened reports whether the dialog is currently visible.
func (d *Dialog) Opened() bool {
	return d.open != nil && *d.open
}

// Dismissible controls whether the dialog closes itself when the scrim
// is clicked or Back (Android) or Escape is pressed. It's on by
// default; turn it off for a dialog the user must answer through its
// own controls. It returns d for chaining.
func (d *Dialog) Dismissible(on bool) *Dialog {
	d.fixed = !on
	return d
}
