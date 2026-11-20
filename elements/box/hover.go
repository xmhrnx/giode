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
	st := b.st
	if b.animKey != nil {
		st = animated(gtx, b.animKey, st)
	}
	if b.anim != nil {
		st = b.anim.apply(gtx, st)
	}
	return b.hoverStyles(gtx, st)
}

// hoverStyles returns base with the hover styles merged on top while
// hovered, blended over base's Transition when one is set.
func (b *BoxElement) hoverStyles(gtx layout.Context, base styles.Styles) styles.Styles {
	h := b.hover
	if h == nil {
		return base
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
	tr := base.Transition
	if tr.Duration <= 0 || kit.ReducedMotion() {
		if !h.hovered {
			return base
		}
		return styles.Merge(base, h.st)
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
		return base
	case 1:
		return styles.Merge(base, h.st)
	}
	return blendStyles(base, styles.Merge(base, h.st), tr.Easing.Apply(h.progress))
}

// blendStyles returns to with its animatable properties interpolated
// from from by t: colors, opacity, transform, corner radius, border
// width, shadow, padding, gap, and widths and heights fixed at both
// ends. The rest are to's.
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
	li := func(a, b int) int { return a + int(float32(b-a)*t+0.5*sign(float32(b-a)*t)) }
	lf := func(a, b float32) float32 { return a + (b-a)*t }
	one := func(v float32) float32 {
		if v == 0 {
			return 1
		}
		return v
	}
	ft, tt := from.Transform, to.Transform
	out.Transform.Rotate = lf(ft.Rotate, tt.Rotate)
	out.Transform.Scale = lf(one(ft.Scale), one(tt.Scale))
	if ft.ScaleX != 0 || tt.ScaleX != 0 {
		out.Transform.ScaleX = lf(one(ft.ScaleX)*one(ft.Scale), one(tt.ScaleX)*one(tt.Scale)) / one(out.Transform.Scale)
	}
	if ft.ScaleY != 0 || tt.ScaleY != 0 {
		out.Transform.ScaleY = lf(one(ft.ScaleY)*one(ft.Scale), one(tt.ScaleY)*one(tt.Scale)) / one(out.Transform.Scale)
	}
	out.Transform.TranslateX = li(ft.TranslateX, tt.TranslateX)
	out.Transform.TranslateY = li(ft.TranslateY, tt.TranslateY)
	out.BorderRadius = properties.BorderRadius(li(int(from.BorderRadius), int(to.BorderRadius)))
	out.BorderWidth = properties.BorderWidth(li(int(from.BorderWidth), int(to.BorderWidth)))
	out.BoxShadow.X, out.BoxShadow.Y = li(from.BoxShadow.X, to.BoxShadow.X), li(from.BoxShadow.Y, to.BoxShadow.Y)
	out.BoxShadow.Blur, out.BoxShadow.Spread = li(from.BoxShadow.Blur, to.BoxShadow.Blur), li(from.BoxShadow.Spread, to.BoxShadow.Spread)
	out.Gap = properties.Gap(li(int(from.Gap), int(to.Gap)))
	if from.Padding != (properties.Inset{}) || to.Padding != (properties.Inset{}) {
		fp, tp := from.Padding, to.Padding
		out.Padding = properties.Inset{Top: li(fp.Top, tp.Top), Right: li(fp.Right, tp.Right), Bottom: li(fp.Bottom, tp.Bottom),
			Left: li(fp.Left, tp.Left), Start: li(fp.Start, tp.Start), End: li(fp.End, tp.End)}
		if out.Padding == (properties.Inset{}) {
			out.Padding = tp
		}
	}
	if from.Width > 0 && to.Width > 0 {
		out.Width = properties.Width(li(int(from.Width), int(to.Width)))
	}
	if from.Height > 0 && to.Height > 0 {
		out.Height = properties.Height(li(int(from.Height), int(to.Height)))
	}
	return out
}

func sign(v float32) float32 {
	if v < 0 {
		return -1
	}
	return 1
}
