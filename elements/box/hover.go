package box

import (
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
	b.hover = &boxHover{st: st}
	return b
}

// Hovered reports whether the pointer was over the element on the last
// frame.
func (b *BoxElement) Hovered() bool {
	return b.hover != nil && b.hover.hovered
}

// resolveStyles processes hover events and returns the styles to lay
// out and paint with: the base styles, with the hover styles merged
// on top while hovered - blended over the base styles' Transition when
// one is set.
func (b *BoxElement) resolveStyles(gtx layout.Context) styles.Styles {
	h := b.hover
	if h == nil {
		return b.st
	}
	wasHovered := h.hovered
	h.hovered = h.Update(gtx.Source)
	if h.hovered != wasHovered {
		kit.MarkLive()
	}
	if h.pass == nil {
		h.pass = func(gtx layout.Context) bool {
			was := h.hovered
			h.hovered = h.Update(gtx.Source)
			return was != h.hovered
		}
	}
	kit.HandleInput(h.pass)
	tr := b.st.Transition
	if tr.Duration <= 0 {
		if !h.hovered {
			return b.st
		}
		return styles.Merge(b.st, h.st)
	}

	target := float32(0)
	if h.hovered {
		target = 1
	}
	// A transition starting after the element sat idle measures from
	// this frame, not from the last frame drawn, which may be long ago.
	if !h.animating {
		h.lastFrame = gtx.Now
	}
	dt := gtx.Now.Sub(h.lastFrame)
	h.lastFrame = gtx.Now
	step := float32(dt) / float32(tr.Duration)
	switch {
	case h.progress < target:
		h.progress = min(target, h.progress+step)
	case h.progress > target:
		h.progress = max(target, h.progress-step)
	}
	h.animating = h.progress != target
	if h.animating {
		kit.RequestFrame(gtx)
	}

	switch h.progress {
	case 0:
		return b.st
	case 1:
		return styles.Merge(b.st, h.st)
	}
	return blendStyles(b.st, styles.Merge(b.st, h.st), tr.Easing.Apply(h.progress))
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
