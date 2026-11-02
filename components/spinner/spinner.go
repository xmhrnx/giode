// Package spinner provides an indeterminate circular loading
// indicator.
package spinner

import (
	"image/color"

	"github.com/xmhrnx/giode/styles"
)

var defaultColor = color.NRGBA{R: 0x3b, G: 0x82, B: 0xf6, A: 0xff}

// Spinner is a stateless animated loading indicator. Color sets the
// arc color and Width/Height the size (default 32px). Chain OnHover
// to add hover styles.
func Spinner(st ...styles.Styles) *SpinnerElement {
	var s styles.Styles
	if len(st) > 0 {
		s = st[0]
	}
	return &SpinnerElement{st: s}
}
