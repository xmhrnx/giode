package mcpbridge

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"

	"gioui.org/gpu/headless"
	"gioui.org/op"
	"github.com/xmhrnx/giode/elements"
)

// CaptureUnderlay reads back a frame drawn over an underlay when a
// screenshot asked for one. It runs on the render thread.
func (b *Bridge) CaptureUnderlay(gl bool, read func() *image.RGBA) {
	b.underlayGL.Store(gl)
	if !gl || !b.wantCapture.Swap(false) {
		return
	}
	img := read()
	b.mu.Lock()
	b.captured = img
	b.mu.Unlock()
}

// takeCapture returns the frame CaptureUnderlay read back, if any.
func (b *Bridge) takeCapture() *image.RGBA {
	b.mu.Lock()
	defer b.mu.Unlock()
	img := b.captured
	b.captured = nil
	return img
}

// FrameDone records the frame's semantic tree and runs after-frame
// callbacks; the frame loop calls it after presenting a frame.
func (b *Bridge) FrameDone(nodes []elements.SemanticNode, ops *op.Ops, size image.Point) {
	b.mu.Lock()
	b.nodes = nodes
	b.winSize = size
	after := b.after
	b.after = nil
	b.mu.Unlock()
	for _, fn := range after {
		fn(ops, size)
	}
}

// afterFrame queues fn to run on the UI goroutine right after the next
// frame, requests that frame and waits.
func (b *Bridge) afterFrame(ctx context.Context, fn func(ops *op.Ops, size image.Point)) error {
	done := make(chan struct{})
	b.mu.Lock()
	b.after = append(b.after, func(ops *op.Ops, size image.Point) { fn(ops, size); close(done) })
	b.mu.Unlock()
	b.h.Invalidate()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("giode: no frame was drawn (window minimized or closed?): %w", ctx.Err())
	}
}

// snapshot waits for a fresh frame and returns its semantic tree.
func (b *Bridge) snapshot(ctx context.Context) ([]elements.SemanticNode, image.Point, error) {
	if err := b.afterFrame(ctx, func(*op.Ops, image.Point) {}); err != nil {
		return nil, image.Point{}, err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.nodes, b.winSize, nil
}

// node finds a node of the last frame by id.
func (b *Bridge) node(id string) (elements.SemanticNode, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, n := range b.nodes {
		if n.ID == id {
			return n, true
		}
	}
	return elements.SemanticNode{}, false
}

// screenshot renders the next frame's ops offscreen. It runs on the UI
// goroutine.
func (b *Bridge) screenshot(ops *op.Ops, size image.Point) (*image.RGBA, error) {
	if size.X <= 0 || size.Y <= 0 {
		return nil, errors.New("window has no size")
	}
	if b.headless == nil || b.headlessSz != size {
		if b.headless != nil {
			b.headless.Release()
		}
		w, err := headless.NewWindow(size.X, size.Y)
		if err != nil {
			b.headless = nil
			return nil, fmt.Errorf("offscreen renderer unavailable: %w", err)
		}
		b.headless, b.headlessSz = w, size
	}
	if err := b.headless.Frame(ops); err != nil {
		return nil, err
	}
	img := image.NewRGBA(image.Rectangle{Max: size})
	if err := b.headless.Screenshot(img); err != nil {
		return nil, err
	}
	return img, nil
}

func encodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
