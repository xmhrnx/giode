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
	// AlignBaseline lines a row's children up on their first baselines,
	// like CSS's align-items: baseline: text in different sizes, or a
	// formula beside text.
	AlignBaseline Align = "baseline"
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
	case AlignBaseline:
		return layout.Baseline
	default:
		return layout.Start
	}
}
