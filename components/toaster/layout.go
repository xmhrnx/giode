// Package toaster shows small notifications that pop up in a corner of
// the window and disappear on their own.

package toaster

import (
	"image"
	"time"

	"gioui.org/layout"
	"gioui.org/unit"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
)

// Layout queues the toasts above everything else. It takes no space.
func (t *Toaster) Layout(gtx layout.Context) layout.Dimensions {
	// Handles its input without an input pass; see kit.OpaqueInput.
	kit.OpaqueInput()
	t.sync()

	// Start the timers of toasts coming into view, pause hovered ones,
	// and retire expired ones.
	shown := 0
	var next time.Time
	alive := t.toasts[:0]
	for _, ts := range t.toasts {
		if ts.leaving && !ts.trans.Visible() && !ts.trans.Animating() {
			continue // gone
		}
		alive = append(alive, ts)
		if ts.leaving {
			continue
		}
		if shown >= t.max {
			continue // waiting its turn
		}
		shown++
		if !ts.started {
			ts.started = true
			ts.remaining = t.duration
			ts.deadline = gtx.Now.Add(t.duration)
		}
		if t.duration <= 0 {
			continue
		}
		hovered := ts.body.Hovered()
		switch {
		case hovered && !ts.paused:
			ts.paused = true
			ts.remaining = max(0, ts.deadline.Sub(gtx.Now))
		case !hovered && ts.paused:
			ts.paused = false
			ts.deadline = gtx.Now.Add(ts.remaining)
		}
		if ts.paused {
			continue
		}
		if !gtx.Now.Before(ts.deadline) {
			t.leave(ts)
			continue
		}
		if next.IsZero() || ts.deadline.Before(next) {
			next = ts.deadline
		}
	}
	clear(t.toasts[len(alive):])
	t.toasts = alive
	if !next.IsZero() {
		kit.RequestFrameAt(gtx, next)
	}
	if len(t.toasts) == 0 {
		return layout.Dimensions{}
	}
	kit.MarkLive()
	elements.RegisterPortal(gtx, image.Point{}, func() elements.Element {
		return elements.RawFunc(t.layoutStack)
	})
	return layout.Dimensions{}
}

// layoutStack lays the visible toasts out in their corner. It runs as
// a portal covering the window; only the toasts take input.
// insetPx is the inset in pixels, its left and top in Min, its right
// and bottom in Max. Not image.Rect: it swaps the coordinates when the
// top is more than the bottom, and the top inset was taken for the
// bottom.
func (t *Toaster) insetPx(gtx layout.Context) image.Rectangle {
	return image.Rectangle{
		Min: image.Pt(gtx.Dp(unit.Dp(t.inset.Left)), gtx.Dp(unit.Dp(t.inset.Top))),
		Max: image.Pt(gtx.Dp(unit.Dp(t.inset.Right)), gtx.Dp(unit.Dp(t.inset.Bottom))),
	}
}

func (t *Toaster) layoutStack(gtx layout.Context) layout.Dimensions {
	win := gtx.Constraints.Max
	w := gtx.Dp(toastWidth)
	if t.st.Width > 0 {
		w = gtx.Dp(unit.Dp(t.st.Width))
	}
	m := gtx.Dp(margin)
	in := t.insetPx(gtx)
	w = min(w, win.X-2*m-in.Min.X-in.Max.X)
	if w <= 0 {
		return layout.Dimensions{}
	}
	// Newest nearest the edge the toasts come from.
	var visible []*toast
	n := 0
	for _, ts := range t.toasts {
		if !ts.started {
			continue
		}
		visible = append(visible, ts)
		n++
	}
	if t.placement.top() {
		for i, j := 0, len(visible)-1; i < j; i, j = i+1, j-1 {
			visible[i], visible[j] = visible[j], visible[i]
		}
	}
	children := make([]elements.Element, 0, n)
	for _, ts := range visible {
		children = append(children, elements.SizedPx(w, 0, ts.trans))
	}
	stack := elements.Stack(styles.Styles{Gap: gap}, children...)

	// Measure, then place in the corner.
	cgtx := gtx
	cgtx.Constraints = layout.Constraints{Max: image.Pt(w, max(0, win.Y-2*m-in.Min.Y-in.Max.Y))}
	dims := elements.Measure(cgtx, stack)
	var x, y int
	switch t.placement.physical(kit.RTL(gtx)) {
	case BottomLeft, TopLeft:
		x = m + in.Min.X
	case BottomCenter, TopCenter:
		x = in.Min.X + (win.X-in.Min.X-in.Max.X-w)/2
	default:
		x = win.X - w - m - in.Max.X
	}
	if t.placement.top() {
		y = m + in.Min.Y
	} else {
		y = win.Y - m - in.Max.Y - dims.Size.Y
	}
	return elements.OffsetPx(image.Pt(x, y), elements.RawFunc(func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints = cgtx.Constraints
		return stack.Layout(gtx)
	})).Layout(gtx)
}
