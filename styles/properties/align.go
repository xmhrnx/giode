package properties

import "gioui.org/layout"

// Align positions children on the cross axis of a flex container
// (CSS align-items).
type Align string

const (
	AlignStart  Align = "start"
	AlignCenter Align = "center"
	AlignEnd    Align = "end"
	// AlignStretch makes every child as tall as the tallest in a row (as
	// wide as the widest in a column), like CSS's align-items: stretch:
	// cards side by side get one bottom edge. A container with a fixed
	// cross size stretches its children to it.
	AlignStretch Align = "stretch"
)

// FlexAlignment maps the value to a cross-axis layout alignment.
// Stretched children start at the start; Box sizes them (see
// AlignStretch).
func (a Align) FlexAlignment() layout.Alignment {
	switch a {
	case AlignCenter:
		return layout.Middle
	case AlignEnd:
		return layout.End
	default:
		return layout.Start
	}
}
