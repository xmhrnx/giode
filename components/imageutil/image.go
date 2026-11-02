// Package imageutil renders plain images with fit modes.
package imageutil

import (
	"image"

	"github.com/xmhrnx/giode/styles"
)

// New renders img as a stateless element. Width and Height fix the
// box size (the natural size is used when unset) and Fit controls the
// scaling: none, contain, cover or stretch. Chain OnHover to add
// hover styles (Opacity and BorderRadius).
func New(img image.Image, st ...styles.Styles) *ImageElement {
	var s styles.Styles
	if len(st) > 0 {
		s = st[0]
	}
	return &ImageElement{img: img, st: s}
}
