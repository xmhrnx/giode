// Package progress provides a determinate progress bar.
package progress

import (
	"image/color"

	"github.com/xmhrnx/giode/styles"
)

var (
	defaultTrack = color.NRGBA{R: 0x33, G: 0x41, B: 0x55, A: 0xff}
	defaultFill  = color.NRGBA{R: 0x3b, G: 0x82, B: 0xf6, A: 0xff}
)

// Progress is a stateless determinate progress bar. It expands to the
// available width; the height defaults to 8px. Color is the fill and
// Background is the track. Chain OnHover to add hover styles.
func Progress(value float32, st ...styles.Styles) *ProgressElement {
	var s styles.Styles
	if len(st) > 0 {
		s = st[0]
	}
	return &ProgressElement{value: value, st: s}
}
