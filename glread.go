package giode

import (
	"image"

	"github.com/xmhrnx/giode/internal/platform/gl"
)

// readFramebuffer reads an OpenGL target's pixels into an image, top
// row first. It runs on the render thread with the context current.
func readFramebuffer(t GPUTarget) *image.RGBA {
	w, h := t.Size.X, t.Size.Y
	if t.API != GPUOpenGL || w <= 0 || h <= 0 {
		return nil
	}
	buf := make([]byte, w*h*4)
	if !gl.ReadPixels(t.Framebuffer, w, h, buf) {
		return nil
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		copy(img.Pix[y*img.Stride:y*img.Stride+w*4], buf[(h-1-y)*w*4:(h-y)*w*4])
	}
	// The window is opaque; GL may leave alpha as the underlay wrote it.
	for i := 3; i < len(img.Pix); i += 4 {
		img.Pix[i] = 0xff
	}
	return img
}
