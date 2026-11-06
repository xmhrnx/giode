package kit

import (
	"image/color"
	"strings"
	"time"

	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/unit"
	"github.com/xmhrnx/giode/styles/properties"
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

var (
	defaultScrollTrack = color.NRGBA{R: 0x47, G: 0x55, B: 0x69, A: 0x99}
	defaultScrollThumb = "#94a3b8cc"
)

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
		// Every press on the thumb starts a drag and every drag moves
		// it, whatever their priority: the thumb shares its events with
		// whatever lies under it (the list, hover, a window-wide
		// listener), so a press is rarely the thumb's alone, and Gio
		// grabs the pointer for the thumb only once it has moved a few
		// pixels - the drags before that count too.
		switch ev.Kind {
		case pointer.Press:
			s.dragY = ev.Position.Y
		case pointer.Drag:
			dy := ev.Position.Y - s.dragY
			s.dragY = ev.Position.Y
			if dy != 0 && s.maxThumbOff > 0 {
				*offset = max(0, min(s.maxOff, *offset+dy*s.maxOff/float32(s.maxThumbOff)))
			}
		}
	}
}

// Dragging reports whether the thumb is being dragged.
func (s *Scrollbar) Dragging() bool {
	return s.drag.Dragging()
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
