package kit

import (
	"image"
	"image/color"
	"strings"
	"time"

	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/styles/properties"
)

var (
	defaultScrollTrack = color.NRGBA{R: 0x47, G: 0x55, B: 0x69, A: 0x99}
	defaultScrollThumb = "#94a3b8cc"
)

// Scrollbar is the vertical scrollbar shared by scrolling components:
// a track and a draggable thumb along the right edge of a viewport.
// Embed one per scrolling component and lay it out after the content.
type Scrollbar struct {
	drag  gesture.Drag
	hover gesture.Hover
	dragY float32

	// For AutoHide: the pointer over the whole track, the offset at
	// the last layout, and when the bar was last in use.
	trackHover   gesture.Hover
	trackHovered bool
	lastOff      float32
	lastActive   time.Time

	// hovered, and the geometry of the last layout, for Update.
	hovered     bool
	shown       bool
	maxOff      float32
	maxThumbOff int
}

// AutoHide timing: how long the bar stays after its last use, and how
// long it takes to fade out.
const (
	scrollbarLinger = time.Second
	scrollbarFade   = 300 * time.Millisecond
)

// Update handles thumb drags and hover with the geometry of the last
// Layout, applying drags to *offset, and reports whether anything
// changed - for a scrolling component's input pass (see HandleInput).
func (s *Scrollbar) Update(gtx layout.Context, offset *float32) bool {
	if !s.shown {
		return false
	}
	h, th, d, o := s.hovered, s.trackHovered, s.drag.Dragging(), *offset
	s.handle(gtx, offset)
	return h != s.hovered || th != s.trackHovered || d != s.drag.Dragging() || o != *offset
}

// handle processes the thumb's events.
func (s *Scrollbar) handle(gtx layout.Context, offset *float32) {
	s.hovered = s.hover.Update(gtx.Source)
	s.trackHovered = s.trackHover.Update(gtx.Source)
	for {
		ev, ok := s.drag.Update(gtx.Metric, gtx.Source, gesture.Both)
		if !ok {
			break
		}
		switch ev.Kind {
		case pointer.Press:
			if ev.Priority == pointer.Grabbed {
				s.dragY = ev.Position.Y
			}
		case pointer.Drag:
			if ev.Priority == pointer.Grabbed {
				dy := ev.Position.Y - s.dragY
				s.dragY = ev.Position.Y
				if dy != 0 && s.maxThumbOff > 0 {
					*offset = max(0, min(s.maxOff, *offset+dy*s.maxOff/float32(s.maxThumbOff)))
				}
			}
		}
	}
}

// Dragging reports whether the thumb is being dragged.
func (s *Scrollbar) Dragging() bool {
	return s.drag.Dragging()
}

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

// Screen reports the window's width in pixels and the name of the
// breakpoint the innermost Responsive matched, for scrollbars hidden on
// some screens. The elements package sets it.
var Screen func() (width int, breakpoint string)

// scrollbarHidden reports whether sb hides the bar on this screen.
func scrollbarHidden(gtx layout.Context, sb properties.ScrollBar) bool {
	if sb.Hidden {
		return true
	}
	if (sb.HiddenMaxWidth <= 0 && sb.HiddenOn == "") || Screen == nil {
		return false
	}
	width, bp := Screen()
	if sb.HiddenMaxWidth > 0 && width > 0 && width <= gtx.Dp(unit.Dp(sb.HiddenMaxWidth)) {
		return true
	}
	if sb.HiddenOn != "" && bp != "" {
		for _, name := range strings.Split(sb.HiddenOn, ",") {
			if strings.TrimSpace(name) == bp {
				return true
			}
		}
	}
	return false
}
