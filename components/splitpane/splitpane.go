// Package splitpane provides SplitPane: two panes side by side (or one
// above the other) with a divider between them the user drags to share
// the space - a sidebar and an editor, an editor and a terminal.
package splitpane

import (
	"image"
	"strconv"
	"time"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles/properties"
)

// Axis selects how the panes are placed.
type Axis uint8

const (
	// Horizontal places them side by side, the first at the start (the
	// left, or the right in a right-to-left locale). The default.
	Horizontal Axis = iota
	// Vertical places the first above the second.
	Vertical
)

// SplitPane is two panes with a divider between them. Create it once
// with New and lay it out every frame; it fills the space it's given.
//
// By default the first pane takes Ratio of the space, and keeps that
// share as the space changes; FirstSize instead keeps the first pane's
// size in dp (a sidebar), the second taking the rest. Dragging the
// divider resizes; a double-click on it puts back the size it started
// with. The divider takes keyboard focus with Tab, and the arrow keys
// move it.
type SplitPane struct {
	first, second func() elements.Element
	axis          Axis

	ratio, initRatio float32
	fixed            bool
	size, initSize   float32 // dp, with fixed
	minFirst         float32 // dp
	minSecond        float32 // dp
	divider          properties.Color
	onResize         func(ratio, firstDp float32)

	// Layout of the last frame, in pixels along the axis.
	total, firstPx int
	pxPerDp        float32
	rtl            bool

	hovered, dragging bool
	pressID           pointer.ID
	grab              float32 // where on the divider it was taken
	lastPress         time.Time
	focused           bool
}

// New returns a split pane showing first and second, each built every
// frame, the first taking half the space.
func New(first, second func() elements.Element) *SplitPane {
	return &SplitPane{first: first, second: second, ratio: 0.5, initRatio: 0.5, minFirst: 48, minSecond: 48}
}

// Axis sets how the panes are placed: Horizontal (side by side, the
// default) or Vertical. It returns s for chaining.
func (s *SplitPane) Axis(a Axis) *SplitPane {
	s.axis = a
	return s
}

// Ratio sets the first pane's share of the space, 0 to 1, and the
// share a double-click on the divider puts back. It returns s for
// chaining.
func (s *SplitPane) Ratio(r float32) *SplitPane {
	s.ratio, s.initRatio, s.fixed = clamp01(r), clamp01(r), false
	return s
}

// FirstSize keeps the first pane dp wide (or tall), as a sidebar keeps
// its width when the window is resized; dragging changes it. It
// returns s for chaining.
func (s *SplitPane) FirstSize(dp float32) *SplitPane {
	s.size, s.initSize, s.fixed = dp, dp, true
	return s
}

// MinSizes sets how small, in dp, each pane may be dragged; 48dp by
// default. It returns s for chaining.
func (s *SplitPane) MinSizes(first, second float32) *SplitPane {
	s.minFirst, s.minSecond = max(first, 0), max(second, 0)
	return s
}

// DividerColor sets the divider line's color; theme.border by default.
// It returns s for chaining.
func (s *SplitPane) DividerColor(c properties.Color) *SplitPane {
	s.divider = c
	return s
}

// OnResize sets a handler for the user moving the divider, with the
// first pane's share of the space and its size in dp: to remember
// them. It returns s for chaining.
func (s *SplitPane) OnResize(fn func(ratio, firstDp float32)) *SplitPane {
	s.onResize = fn
	return s
}

// Content replaces the panes' contents. It returns s for chaining.
func (s *SplitPane) Content(first, second func() elements.Element) *SplitPane {
	s.first, s.second = first, second
	return s
}

// Share returns the first pane's share of the space in the last frame.
func (s *SplitPane) Share() float32 {
	if s.total <= 0 {
		return s.ratio
	}
	return float32(s.firstPx) / float32(s.total)
}

// Dragging reports whether the divider is being dragged.
func (s *SplitPane) Dragging() bool { return s.dragging }

func clamp01(v float32) float32 { return min(max(v, 0), 1) }

// Layout lays out the panes and the divider.
func (s *SplitPane) Layout(gtx layout.Context) layout.Dimensions {
	kit.OpaqueInput()
	size := gtx.Constraints.Max
	s.pxPerDp = gtx.Metric.PxPerDp
	if s.pxPerDp == 0 {
		s.pxPerDp = 1
	}
	s.rtl = s.axis == Horizontal && kit.RTL(gtx)
	along := func(p image.Point) int {
		if s.axis == Vertical {
			return p.Y
		}
		return p.X
	}
	line := max(gtx.Dp(1), 1)
	s.total = max(along(size)-line, 0)
	s.update(gtx)
	s.firstPx = s.place(s.wanted())

	// The panes, each clipped to its part.
	firstR, divR, secondR := s.rects(size, line)
	s.layoutPane(gtx, s.first, firstR)
	s.layoutPane(gtx, s.second, secondR)

	// The divider: its line, and a wider area to grab it by.
	col := properties.Color("theme.border")
	if s.divider != "" {
		col = s.divider
	}
	if s.hovered || s.dragging || s.focused {
		col = "theme.primary"
	}
	paint.FillShape(gtx.Ops, properties.CalcColor(col), clip.Rect(divR).Op())
	grab := gtx.Dp(6)
	hit := divR
	if s.axis == Vertical {
		hit.Min.Y, hit.Max.Y = divR.Min.Y-grab/2, divR.Max.Y+grab/2
	} else {
		hit.Min.X, hit.Max.X = divR.Min.X-grab/2, divR.Max.X+grab/2
	}
	area := clip.Rect(hit).Push(gtx.Ops)
	if s.axis == Vertical {
		pointer.CursorRowResize.Add(gtx.Ops)
	} else {
		pointer.CursorColResize.Add(gtx.Ops)
	}
	event.Op(gtx.Ops, s)
	area.Pop()
	kit.BlockDrag(gtx, size)
	if kit.FocusVisible(gtx, s) {
		off := op.Offset(hit.Min).Push(gtx.Ops)
		kit.FocusRing(gtx, s, hit.Size(), 0)
		off.Pop()
	}
	pct := 0.0
	if s.total > 0 {
		pct = float64(s.firstPx) / float64(s.total)
	}
	off := op.Offset(hit.Min).Push(gtx.Ops)
	pop := elements.PushOffset(hit.Min)
	elements.RecordSemantic(gtx, elements.SemanticNode{
		Kind: elements.SemanticSlider, Label: "Resize panes", Value: percent(pct), Focused: s.focused,
	}, hit.Size())
	pop()
	off.Pop()
	return layout.Dimensions{Size: size}
}

func percent(v float64) string { return strconv.Itoa(int(v*100+0.5)) + "%" }

// wanted is the first pane's size in pixels before the minimums.
func (s *SplitPane) wanted() int {
	if s.fixed {
		return int(s.size*s.pxPerDp + 0.5)
	}
	return int(s.ratio*float32(s.total) + 0.5)
}

// place applies the minimums to a first pane size in pixels; the
// second pane's minimum wins over the first's when both can't fit.
func (s *SplitPane) place(px int) int {
	minA := int(s.minFirst * s.pxPerDp)
	minB := int(s.minSecond * s.pxPerDp)
	px = max(px, minA)
	px = min(px, s.total-minB)
	return min(max(px, 0), s.total)
}

// rects returns the first pane's, the divider's and the second pane's
// rectangles.
func (s *SplitPane) rects(size image.Point, line int) (a, div, b image.Rectangle) {
	f := s.firstPx
	if s.axis == Vertical {
		return image.Rect(0, 0, size.X, f), image.Rect(0, f, size.X, f+line), image.Rect(0, f+line, size.X, size.Y)
	}
	a, div, b = image.Rect(0, 0, f, size.Y), image.Rect(f, 0, f+line, size.Y), image.Rect(f+line, 0, size.X, size.Y)
	if s.rtl {
		mirror := func(r image.Rectangle) image.Rectangle {
			return image.Rect(size.X-r.Max.X, r.Min.Y, size.X-r.Min.X, r.Max.Y)
		}
		a, div, b = mirror(a), mirror(div), mirror(b)
	}
	return a, div, b
}

func (s *SplitPane) layoutPane(gtx layout.Context, content func() elements.Element, r image.Rectangle) {
	if content == nil || r.Empty() {
		return
	}
	off := op.Offset(r.Min).Push(gtx.Ops)
	pop := elements.PushOffset(r.Min)
	cl := clip.Rect{Max: r.Size()}.Push(gtx.Ops)
	pgtx := gtx
	pgtx.Constraints = layout.Exact(r.Size())
	content().Layout(pgtx)
	cl.Pop()
	pop()
	off.Pop()
}

// setFirst sets the first pane's size, in pixels, as the user moved
// the divider.
func (s *SplitPane) setFirst(px int) {
	px = s.place(px)
	if px == s.firstPx {
		return
	}
	s.firstPx = px
	if s.fixed {
		s.size = float32(px) / s.pxPerDp
	}
	if s.total > 0 {
		s.ratio = float32(px) / float32(s.total)
	}
	if s.onResize != nil {
		s.onResize(s.ratio, float32(px)/s.pxPerDp)
	}
}

// update handles dragging, double-clicks and the keyboard.
func (s *SplitPane) update(gtx layout.Context) {
	along := func(p f32.Point) float32 {
		if s.axis == Vertical {
			return p.Y
		}
		return p.X
	}
	sign := 1
	if s.rtl {
		sign = -1
	}
	// Drags are relative to the divider as the last frame placed it.
	base := s.firstPx
	for {
		ev, ok := gtx.Event(
			pointer.Filter{Target: s, Kinds: pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel | pointer.Enter | pointer.Leave},
			key.FocusFilter{Target: s},
			key.Filter{Focus: s, Name: key.NameLeftArrow, Optional: key.ModShift},
			key.Filter{Focus: s, Name: key.NameRightArrow, Optional: key.ModShift},
			key.Filter{Focus: s, Name: key.NameUpArrow, Optional: key.ModShift},
			key.Filter{Focus: s, Name: key.NameDownArrow, Optional: key.ModShift},
			key.Filter{Focus: s, Name: key.NameHome},
			key.Filter{Focus: s, Name: key.NameEnd},
		)
		if !ok {
			break
		}
		switch e := ev.(type) {
		case pointer.Event:
			switch e.Kind {
			case pointer.Enter:
				s.hovered = true
			case pointer.Leave:
				s.hovered = false
			case pointer.Press:
				if e.Source == pointer.Mouse && !e.Buttons.Contain(pointer.ButtonPrimary) {
					break
				}
				if gtx.Now.Sub(s.lastPress) < 400*time.Millisecond {
					// A double-click: the size it started with.
					s.lastPress = time.Time{}
					if s.fixed {
						s.setFirst(int(s.initSize*s.pxPerDp + 0.5))
					} else {
						s.setFirst(int(s.initRatio*float32(s.total) + 0.5))
					}
					break
				}
				s.lastPress = gtx.Now
				s.dragging, s.pressID = true, e.PointerID
				s.grab = along(e.Position)
				gtx.Execute(pointer.GrabCmd{Tag: s, ID: e.PointerID})
			case pointer.Drag:
				if !s.dragging || e.PointerID != s.pressID {
					break
				}
				d := along(e.Position) - s.grab
				s.setFirst(base + sign*int(d))
			case pointer.Release, pointer.Cancel:
				s.dragging = false
			}
		case key.FocusEvent:
			s.focused = e.Focus
		case key.Event:
			if e.State != key.Press {
				break
			}
			step := gtx.Dp(16)
			if e.Modifiers.Contain(key.ModShift) {
				step = gtx.Dp(64)
			}
			dir := 0
			switch e.Name {
			case key.NameLeftArrow:
				if s.axis == Horizontal {
					dir = -sign
				}
			case key.NameRightArrow:
				if s.axis == Horizontal {
					dir = sign
				}
			case key.NameUpArrow:
				if s.axis == Vertical {
					dir = -1
				}
			case key.NameDownArrow:
				if s.axis == Vertical {
					dir = 1
				}
			case key.NameHome:
				s.setFirst(0)
			case key.NameEnd:
				s.setFirst(s.total)
			}
			if dir != 0 {
				s.setFirst(s.firstPx + dir*step)
			}
		}
		kit.RequestFrame(gtx)
	}
	kit.NoteFocusable(s)
}

var _ elements.Element = (*SplitPane)(nil)
