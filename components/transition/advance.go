// Package transition animates a component in and out: fade, slide,
// enlarge, shrink, rotate, collapse - alone or combined - with a
// configurable duration and CSS-style easing.

package transition

import (
	"gioui.org/layout"
	"github.com/xmhrnx/giode/components/kit"
)

// advance moves the animation to gtx.Now.
func (t *Transition) advance(gtx layout.Context) {
	target := float32(0)
	if t.visible {
		target = 1
	}
	if t.progress == target {
		t.animating = false
		t.lastFrame = gtx.Now
		return
	}
	// An animation starting after the component sat idle measures
	// from this frame, not from the last frame drawn.
	if !t.animating || t.lastFrame.IsZero() {
		t.lastFrame = gtx.Now
	}
	t.animating = true
	d := t.duration
	if d <= 0 {
		d = Normal
	}
	step := float32(gtx.Now.Sub(t.lastFrame)) / float32(d)
	t.lastFrame = gtx.Now
	if t.progress < target {
		t.progress = min(target, t.progress+step)
	} else {
		t.progress = max(target, t.progress-step)
	}
	if t.progress == target {
		t.animating = false
		if t.onDone != nil {
			t.onDone(t.visible)
		}
	}
	// Keep the frames coming, including the one that paints the final
	// state.
	kit.RequestFrame(gtx)
}

func clamp01(v float32) float32 {
	return max(0, min(1, v))
}
