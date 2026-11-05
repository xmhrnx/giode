// Package iconbutton provides a stateful clickable icon-only button
// built on the stateless elements.

package iconbutton

// Variant selects the button's visual style.
type Variant string

const (
	// VariantContained fills the button with Background (or black if
	// unset) at all times - the default.
	VariantContained Variant = "contained"
	// VariantOutlined is transparent with a visible border, filling
	// faintly with the border color on hover/press.
	VariantOutlined Variant = "outlined"
	// VariantGhost is fully transparent at rest - only the icon is
	// visible - filling faintly with Color on hover/press.
	VariantGhost Variant = "ghost"
)
