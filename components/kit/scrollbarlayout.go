package kit

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"github.com/xmhrnx/giode/styles/properties"
)

// Layout draws the scrollbar over a viewport of the given size showing
// contentH pixels of content scrolled by *offset pixels, and applies
// thumb drags to *offset (clamped to the content). sb customizes it in
// dp; zero fields use the defaults (8dp wide, 20dp minimum thumb). It
// draws nothing when the content fits, or when sb hides it on this
// screen (Hidden, HiddenMaxWidth, HiddenOn).
func (s *Scrollbar) Layout(gtx layout.Context, viewport image.Point, contentH int, offset *float32, sb properties.ScrollBar) {
	viewH := viewport.Y
	if contentH <= viewH || viewH <= 0 || scrollbarHidden(gtx, sb) {
		s.shown = false
		return
	}
	s.shown = true
	barWidth := gtx.Dp(unit.Dp(sb.Width))
	if sb.Width <= 0 {
		barWidth = gtx.Dp(8)
	}
	minThumb := gtx.Dp(unit.Dp(sb.MinThumbLength))
	if sb.MinThumbLength <= 0 {
		minThumb = gtx.Dp(20)
	}
	radius := gtx.Dp(unit.Dp(sb.Radius))
	if sb.Radius <= 0 {
		radius = barWidth / 2
	}
	trackColor := defaultScrollTrack
	if sb.TrackColor != "" {
		trackColor = properties.CalcColor(sb.TrackColor)
	}
	inset := gtx.Dp(unit.Dp(sb.Inset))
	thumb := sb.ThumbColor
	if thumb == "" {
		thumb = defaultScrollThumb
	}
	maxOff := float32(contentH - viewH)

	x := viewport.X - barWidth - inset
	trackH := viewH - 2*inset
	if trackH <= 0 {
		s.shown = false
		return
	}
	defer op.Offset(image.Pt(0, inset)).Push(gtx.Ops).Pop()
	thumbH := min(trackH, max(minThumb, trackH*viewH/contentH))
	maxThumbOff := trackH - thumbH
	thumbOff := int(*offset) * maxThumbOff / max(contentH-viewH, 1)

	if sb.AutoHide {
		// The whole track shows the bar when the pointer is over it.
		tr := clip.Rect(image.Rect(x, 0, x+barWidth, trackH)).Push(gtx.Ops)
		s.trackHover.Add(gtx.Ops)
		tr.Pop()
	}

	thumbRect := image.Rect(x, thumbOff, x+barWidth, min(thumbOff+thumbH, trackH))
	thumbClip := clip.Rect(thumbRect).Push(gtx.Ops)
	BlockDragHere(gtx)
	s.drag.Add(gtx.Ops)
	s.hover.Add(gtx.Ops)
	s.maxOff, s.maxThumbOff = maxOff, maxThumbOff
	s.handle(gtx, offset)
	hovered := s.hovered
	switch {
	case s.drag.Dragging():
		thumb = Pressed(thumb)
		properties.CursorGrabbing.CursorFor().Add(gtx.Ops)
		MarkLive()
	case hovered:
		thumb = Hovered(thumb)
		properties.CursorGrab.CursorFor().Add(gtx.Ops)
	}
	thumbClip.Pop()

	if sb.AutoHide {
		alpha := s.autoHideAlpha(gtx, *offset)
		if alpha <= 0 {
			return
		}
		if alpha < 1 {
			defer paint.PushOpacity(gtx.Ops, alpha).Pop()
		}
	}
	barOff := op.Offset(image.Pt(x, 0)).Push(gtx.Ops)
	if trackColor.A != 0 {
		trackClip := clip.UniformRRect(image.Rect(0, 0, barWidth, trackH), radius).Push(gtx.Ops)
		paint.Fill(gtx.Ops, trackColor)
		trackClip.Pop()
	}
	thumbOff = int(*offset) * maxThumbOff / max(contentH-viewH, 1)
	tc := clip.UniformRRect(image.Rect(0, thumbOff, barWidth, thumbOff+thumbH), radius).Push(gtx.Ops)
	paint.Fill(gtx.Ops, properties.CalcColor(thumb))
	tc.Pop()
	barOff.Pop()
}

// autoHideAlpha is an AutoHide bar's opacity: whole while the content
// scrolls, the pointer is over the bar or it's dragged and for
// scrollbarLinger after, then fading to nothing over scrollbarFade. It
// asks for the frames the fade needs.
func (s *Scrollbar) autoHideAlpha(gtx layout.Context, offset float32) float32 {
	if offset != s.lastOff || s.hovered || s.trackHovered || s.drag.Dragging() {
		s.lastActive = gtx.Now
	}
	s.lastOff = offset
	idle := gtx.Now.Sub(s.lastActive)
	switch {
	case s.lastActive.IsZero() || idle >= scrollbarLinger+scrollbarFade:
		return 0
	case idle < scrollbarLinger:
		// Laid out again when it's time to fade, not replayed from a
		// cache.
		MarkLive()
		gtx.Execute(op.InvalidateCmd{At: s.lastActive.Add(scrollbarLinger)})
		return 1
	default:
		MarkLive()
		gtx.Execute(op.InvalidateCmd{})
		return 1 - float32(idle-scrollbarLinger)/float32(scrollbarFade)
	}
}
