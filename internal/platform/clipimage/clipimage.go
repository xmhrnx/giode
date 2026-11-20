// Package clipimage reads images from the system clipboard.
package clipimage

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// maxImage bounds the encoded image Read reads.
const maxImage = 64 << 20

// decode decodes an image read from the clipboard in any
// format Go can: PNG, JPEG, GIF, BMP, TIFF, WebP.
func decode(data []byte) (image.Image, error) {
	if len(data) == 0 {
		return nil, nil
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("giode: decoding the clipboard image: %w", err)
	}
	return img, nil
}
