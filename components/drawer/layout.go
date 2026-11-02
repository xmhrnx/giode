package drawer

import (
	"image"
	"image/color"
	"time"

	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
)

const animDuration = 200 * time.Millisecond

// edgeWidth is the strip along the window edge that starts a swipe to
// open, in dp.
const edgeWidth = 20

// Layout updates the drawer and, while it's visible, registers the
// overlay that paints it. It takes no space in the layout.
func (d *Drawer) Layout(gtx layout.Context) layout.Dimensions {
	// Handles its input without an input pass; see kit.OpaqueInput.
	kit.OpaqueInput()
	isOpen := d.Opened()
	if isOpen && !d.fixed {
		elements.HandleDismiss(func() { *d.open = false })
	}

	// Advance the animation toward the open state, unless a drag holds
	// the panel.
	dur := d.duration
	dt := gtx.Now.Sub(d.last)
	if d.last.IsZero() || dt < 0 || dt > dur {
		dt = dur / 10
	}
	d.last = gtx.Now
	target := float32(0)
	if isOpen {
		target = 1
	}
	if !d.dragging && d.progress != target && dur <= 0 {
		// No animation: straight to open or closed.
		d.progress = target
	}
	if !d.dragging && d.progress != target {
		step := float32(dt) / float32(dur)
		if d.progress < target {
			d.progress = min(target, d.progress+step)
		} else {
			d.progress = max(target, d.progress-step)
		}
	}
	if d.progress != target || d.dragging {
		kit.RequestFrame(gtx)
	}

	switch {
	case d.progress > 0 || d.dragging:
		elements.RegisterOverlay(func() elements.Element { return elements.Raw(d.layoutOverlay) })
	case d.edgeSwipe && !d.fixed:
		elements.RegisterOverlay(func() elements.Element { return elements.Raw(d.layoutEdge) })
	}
	return layout.Dimensions{}
}

// panelWidth is the panel's width in pixels, at most the window's.
func (d *Drawer) panelWidth(gtx layout.Context) int {
	return min(gtx.Dp(unit.Dp(d.width)), gtx.Constraints.Max.X)
}

// updateDrag moves the panel with a drag and, when it ends, settles it
// open or closed. dir is +1 when dragging away from the drawer's edge
// opens it, as seen from the drag's coordinates.
func (d *Drawer) updateDrag(gtx layout.Context, width int) {
	if d.fixed || d.open == nil {
		return
	}
	dir := float32(1)
	if d.edge(gtx) == Right {
		dir = -1
	}
	for {
		e, ok := d.drag.Update(gtx.Metric, gtx.Source, gesture.Horizontal)
		if !ok {
			break
		}
		switch e.Kind {
		case pointer.Press:
			d.dragFrom = e.Position.X
			d.dragStart = d.progress
		case pointer.Drag:
			dx := (e.Position.X - d.dragFrom) * dir
			if !d.dragging && abs(dx) < float32(gtx.Dp(8)) {
				continue
			}
			d.dragging = true
			d.progress = clamp01(d.dragStart + dx/float32(width))
		case pointer.Release, pointer.Cancel:
			if !d.dragging {
				continue
			}
			d.dragging = false
			*d.open = d.progress >= 0.5
		}
		kit.RequestFrame(gtx)
	}
}

// layoutEdge registers the strip along the window edge that drags a
// closed drawer open. It passes presses through, so content under it
// stays tappable until a drag starts.
func (d *Drawer) layoutEdge(gtx layout.Context) layout.Dimensions {
	win := gtx.Constraints.Max
	width := d.panelWidth(gtx)
	d.updateDrag(gtx, width)
	strip := image.Rect(0, 0, gtx.Dp(edgeWidth), win.Y)
	if d.edge(gtx) == Right {
		strip = image.Rect(win.X-gtx.Dp(edgeWidth), 0, win.X, win.Y)
	}
	defer pointer.PassOp{}.Push(gtx.Ops).Pop()
	defer clip.Rect(strip).Push(gtx.Ops).Pop()
	d.drag.Add(gtx.Ops)
	return layout.Dimensions{}
}

// layoutOverlay paints the scrim and the panel over the window.
func (d *Drawer) layoutOverlay(gtx layout.Context) layout.Dimensions {
	win := gtx.Constraints.Max
	width := d.panelWidth(gtx)
	d.updateDrag(gtx, width)
	if d.scrim.Clicked(gtx) && !d.fixed && d.open != nil {
		*d.open = false
		kit.RequestFrame(gtx)
	}

	// The drag area is the overlay's root, so the scrim and the panel's
	// content, laid out inside it, keep their own clicks.
	defer clip.Rect{Max: win}.Push(gtx.Ops).Pop()
	d.drag.Add(gtx.Ops)

	d.scrim.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		paint.Fill(gtx.Ops, color.NRGBA{A: uint8(0.4 * 0xff * d.progress)})
		return layout.Dimensions{Size: win}
	})

	var x int
	if d.edge(gtx) == Left {
		x = -width + int(float32(width)*d.progress)
	} else {
		x = win.X - int(float32(width)*d.progress)
	}
	off := op.Offset(image.Pt(x, 0)).Push(gtx.Ops)
	pop := elements.PushOffset(image.Pt(x, 0))
	cl := clip.Rect{Max: image.Pt(width, win.Y)}.Push(gtx.Ops)
	st := d.st
	if st.Background == "" {
		st.Background = defaultBackground
	}
	panelGtx := gtx
	panelGtx.Constraints = layout.Exact(image.Pt(width, win.Y))
	elements.Box(st, d.content()).Layout(panelGtx)
	cl.Pop()
	pop()
	off.Pop()
	return layout.Dimensions{Size: win}
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func clamp01(v float32) float32 {
	return max(0, min(1, v))
}

// OutOfFlow marks the drawer as taking no space in a Box, so the box's
// Gap leaves it out; see elements.OutOfFlow.
func (d *Drawer) OutOfFlow() {}
