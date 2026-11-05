package base

import (
	"image/color"

	"github.com/xmhrnx/giode/styles/properties"
)

// Inherited color.
//
// Like CSS's currentColor, a Box with Color set passes it down: Text,
// icons and glyphs inside that set no Color of their own use it, and
// the value "currentColor" refers to it explicitly. A button's icon
// therefore follows the button's label color without being told.

var colorStack []string

// PushColor makes c the inherited color until the returned function
// runs. Empty colors are ignored. Custom containers that set a text
// color for their content call it around laying the content out.
func PushColor(c string) (pop func()) {
	if c == "" || c == "currentColor" {
		return func() {}
	}
	colorStack = append(colorStack, c)
	n := len(colorStack)
	return func() { colorStack = colorStack[:n-1] }
}

// CurrentColor returns the inherited color, or "" outside any colored
// container.
func CurrentColor() string {
	if len(colorStack) == 0 {
		return ""
	}
	return colorStack[len(colorStack)-1]
}

// ResolveColor resolves an element's own color: c when set, otherwise
// the inherited color, otherwise black.
func ResolveColor(c string) color.NRGBA {
	if c == "" || c == "currentColor" {
		c = CurrentColor()
	}
	return properties.ResolveColor(c)
}
