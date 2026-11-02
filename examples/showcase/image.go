package main

import (
	"image"
	"image/color"

	"github.com/xmhrnx/giode"
)

// demoImage generates a small gradient so the example is
// self-contained (no image asset to ship alongside it).
func demoImage() image.Image {
	const size = 64
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(x * 255 / size),
				G: uint8(y * 255 / size),
				B: 200,
				A: 255,
			})
		}
	}
	return img
}

// ImageShowcase is stateless: the image is fixed, so it can be built
// fresh every frame like Text.
func ImageShowcase() giode.Element {
	return giode.Stack(boxStyle,
		giode.Text("Image", textStyle),
		giode.Image(demoImage(), giode.Styles{Width: 64, Height: 64, BorderRadius: 8}),
	)
}
