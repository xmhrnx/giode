//go:build (linux && !android) || windows

package camera

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/jpeg"
	"time"
)

// captureLatest encodes the newest preview frame as a JPEG, waiting a
// moment for the first.
func captureLatest(ctx context.Context, c *Camera) ([]byte, error) {
	for {
		c.mu.Lock()
		i := c.latest
		var img *image.RGBA
		if i >= 0 && c.bufs[i] != nil {
			// A copy: the converter reuses the buffers.
			src := c.bufs[i]
			img = &image.RGBA{Pix: append([]byte(nil), src.Pix...), Stride: src.Stride, Rect: src.Rect}
		}
		c.mu.Unlock()
		if img != nil {
			var b bytes.Buffer
			if err := jpeg.Encode(&b, img, &jpeg.Options{Quality: 92}); err != nil {
				return nil, err
			}
			return b.Bytes(), nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(30 * time.Millisecond):
		}
		if c.closed.Load() {
			return nil, errors.New("camera: closed")
		}
	}
}
