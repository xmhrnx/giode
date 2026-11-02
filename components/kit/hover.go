package kit

import (
	"image"

	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"

	"github.com/xmhrnx/giode/styles/properties"
)

// HoverTracker is the hover-tracking building block behind every
// stateless element's OnHover (Box, Text, Divider, Image, ...).
// Embed it in a custom stateless element to add the same capability:
// the zero value has hover tracking disabled and costs nothing until
// Enable is called.
type HoverTracker struct {
	hover   *gesture.Hover
	hovered bool
	pass    func(gtx layout.Context) bool
}

// Enable turns on hover tracking. Call from the element's own
// OnHover.
func (h *HoverTracker) Enable() {
	h.hover = new(gesture.Hover)
}

// Enabled reports whether Enable was called.
func (h *HoverTracker) Enabled() bool {
	return h.hover != nil
}

// Hovered reports whether the pointer was over the element on the
// last frame.
func (h *HoverTracker) Hovered() bool {
	return h.hovered
}

// Update processes pending hover events, queued from the area
// RegisterArea added on a previous frame, and caches the result for
// Hovered. Call before deciding which styles to lay out and paint
// with - it reflects last frame's pointer position, not this one. A
// no-op, returning false, when Enable hasn't been called.
func (h *HoverTracker) Update(gtx layout.Context) bool {
	if h.hover == nil {
		return false
	}
	h.hovered = h.hover.Update(gtx.Source)
	return h.hovered
}

// RegisterArea registers a pass-through hover area of size, offset by
// off, for the region to be queried on the next frame, and applies
// cur if currently hovered. A pass-through region sees pointer events
// without blocking children painted above it. Call after painting. A
// no-op when Enable hasn't been called.
func (h *HoverTracker) RegisterArea(gtx layout.Context, off, size image.Point, cur properties.Cursor) {
	if h.hover == nil {
		return
	}
	if h.pass == nil {
		h.pass = func(gtx layout.Context) bool {
			was := h.hovered
			return h.Update(gtx) != was
		}
	}
	HandleInput(h.pass)
	o := op.Offset(off).Push(gtx.Ops)
	ps := pointer.PassOp{}.Push(gtx.Ops)
	cl := clip.Rect{Max: size}.Push(gtx.Ops)
	h.hover.Add(gtx.Ops)
	if h.hovered && cur != "" {
		cur.CursorFor().Add(gtx.Ops)
	}
	cl.Pop()
	ps.Pop()
	o.Pop()
}
