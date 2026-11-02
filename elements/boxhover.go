package elements

import (
	"time"

	"gioui.org/gesture"
	"gioui.org/layout"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// OnHover sets styles applied while the pointer is over the element,
// merged on top of its base styles for both layout and painting:
//
//	card := giode.Box(base, children...).OnHover(giode.Styles{
//		Background: "#1e40af",
//		Cursor:     giode.CursorPointer,
//	})
//
// Set Transition on the base styles to animate the change.
//
// Hover state lives in the element, so create it once and reuse it
// every frame; rebuilding it in the view function resets the hover on
// each frame.
func (b *BoxElement) OnHover(st styles.Styles) *BoxElement {
	b.hover = new(gesture.Hover)
	b.hoverSt = st
	return b
}

// Hovered reports whether the pointer was over the element on the last
// frame.
func (b *BoxElement) Hovered() bool {
	return b.hovered
}

// resolveStyles processes hover events and returns the styles to lay
// out and paint with: the base styles, with the hover styles merged
// on top while hovered - blended over the base styles' Transition when
// one is set.
func (b *BoxElement) resolveStyles(gtx layout.Context) styles.Styles {
	if b.hover == nil {
		return b.st
	}
	wasHovered := b.hovered
	b.hovered = b.hover.Update(gtx.Source)
	if b.hovered != wasHovered {
		kit.MarkLive()
	}
	if b.hoverPass == nil {
		b.hoverPass = func(gtx layout.Context) bool {
			was := b.hovered
			b.hovered = b.hover.Update(gtx.Source)
			return was != b.hovered
		}
	}
	kit.HandleInput(b.hoverPass)
	tr := b.st.Transition
	if tr.Duration <= 0 {
		if !b.hovered {
			return b.st
		}
		return styles.Merge(b.st, b.hoverSt)
	}

	target := float32(0)
	if b.hovered {
		target = 1
	}
	// A transition starting after the element sat idle measures from
	// this frame, not from the last frame drawn, which may be long ago.
	if !b.animating {
		b.lastFrame = gtx.Now
	}
	dt := gtx.Now.Sub(b.lastFrame)
	b.lastFrame = gtx.Now
	step := float32(dt) / float32(tr.Duration)
	switch {
	case b.progress < target:
		b.progress = min(target, b.progress+step)
	case b.progress > target:
		b.progress = max(target, b.progress-step)
	}
	b.animating = b.progress != target
	if b.animating {
		kit.RequestFrame(gtx)
	}

	switch b.progress {
	case 0:
		return b.st
	case 1:
		return styles.Merge(b.st, b.hoverSt)
	}
	return blendStyles(b.st, styles.Merge(b.st, b.hoverSt), tr.Easing.Apply(b.progress))
}

// blendStyles returns to with its animatable properties - colors and
// opacity - interpolated from from by t.
func blendStyles(from, to styles.Styles, t float32) styles.Styles {
	out := to
	out.Background = properties.LerpColor(from.Background, to.Background, t)
	out.Color = properties.LerpColor(from.Color, to.Color, t)
	out.BorderColor = properties.LerpColor(from.BorderColor, to.BorderColor, t)
	out.BoxShadow.Color = properties.LerpColor(from.BoxShadow.Color, to.BoxShadow.Color, t)
	op := func(o properties.Opacity) float32 {
		if o == 0 {
			return 1
		}
		return float32(o)
	}
	out.Opacity = properties.Opacity(op(from.Opacity) + (op(to.Opacity)-op(from.Opacity))*t)
	return out
}

// hoverAnim is the transition state of a hoverable Box.
type hoverAnim struct {
	progress  float32
	animating bool
	lastFrame time.Time
}
