// Package swipeable provides list rows that reveal actions when swiped
// sideways, like the delete and pin actions of mobile mail and chat
// apps.
package swipeable

import (
	"image"
	"time"

	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Action is a button revealed behind a swiped row.
type Action struct {
	// Label is the button text; Icon, when set, shows above it.
	Label string
	Icon  elements.Element
	// Background fills the button (default theme.primary) and Color
	// its label (default theme.onprimary).
	Background properties.Color
	Color      properties.Color
	// OnTap runs when the button is tapped; the row then closes.
	OnTap func()
}

// actionWidth is each action button's width, in dp.
const actionWidth = 80

const animDuration = 180 * time.Millisecond

// Swipeable is a row that slides sideways to reveal actions: leading
// ones on the left when swiped right, trailing ones on the right when
// swiped left (the other way round in a right-to-left locale). A swipe past half the actions' width opens the row; a
// tap on the row or an action closes it. Vertical drags still scroll
// the list around it.
//
// Create it once per row (in a dynamic list, with Keyed) and lay it out
// every frame. The content should paint an opaque background, since
// the actions sit behind it.
type Swipeable struct {
	content           func() elements.Element
	leading, trailing []Action
	leadClicks        []widget.Clickable
	trailClicks       []widget.Clickable
	closeClick        widget.Clickable

	// offset is how far the content is moved right (positive) or left,
	// in pixels; target is where it settles.
	offset, target float32
	last           time.Time
	drag           gesture.Drag
	dragFrom       float32
	dragStart      float32
	dragging       bool
	pass           func(gtx layout.Context) bool
}

// New returns a row showing content, with leading and trailing actions
// (either may be nil).
func New(content func() elements.Element, leading, trailing []Action) *Swipeable {
	return &Swipeable{
		content: content, leading: leading, trailing: trailing,
		leadClicks: make([]widget.Clickable, len(leading)), trailClicks: make([]widget.Clickable, len(trailing)),
	}
}

// SetActions replaces the row's leading and trailing actions (either
// may be nil), for labels that follow state, a Pin that becomes Unpin,
// say. Call it before Layout whenever they change; an open row stays
// open as long as its side still has actions, resized to fit them.
func (s *Swipeable) SetActions(leading, trailing []Action) {
	s.leading, s.trailing = leading, trailing
	s.leadClicks = resize(s.leadClicks, len(leading))
	s.trailClicks = resize(s.trailClicks, len(trailing))
}

// resize returns clicks with n entries, keeping the existing ones so a
// press in progress survives a relabel.
func resize(clicks []widget.Clickable, n int) []widget.Clickable {
	if n <= len(clicks) {
		return clicks[:n]
	}
	return append(clicks, make([]widget.Clickable, n-len(clicks))...)
}

// Close slides the row back.
func (s *Swipeable) Close() {
	s.target = 0
}

// Opened reports whether the row is open, or opening.
func (s *Swipeable) Opened() bool {
	return s.target != 0
}

// Layout lays out the row and its revealed actions.
// update handles drags and taps.
func (s *Swipeable) update(gtx layout.Context, leadW, trailW float32) {
	// Drags move the row; releasing settles it open or closed.
	for {
		e, ok := s.drag.Update(gtx.Metric, gtx.Source, gesture.Horizontal)
		if !ok {
			break
		}
		switch e.Kind {
		case pointer.Press:
			s.dragFrom, s.dragStart = e.Position.X, s.offset
		case pointer.Drag:
			dx := e.Position.X - s.dragFrom
			if !s.dragging && abs(dx) < float32(gtx.Dp(8)) {
				continue
			}
			s.dragging = true
			s.offset = max(-trailW, min(leadW, s.dragStart+dx))
			s.target = s.offset
		case pointer.Release, pointer.Cancel:
			if !s.dragging {
				continue
			}
			s.dragging = false
			switch {
			case s.offset > leadW/2 && leadW > 0:
				s.target = leadW
			case s.offset < -trailW/2 && trailW > 0:
				s.target = -trailW
			default:
				s.target = 0
			}
		}
		kit.RequestFrame(gtx)
	}

	// Taps on actions, or on the open row to close it.
	for i := range s.leading {
		if s.leadClicks[i].Clicked(gtx) {
			s.tap(s.leading[i])
		}
	}
	for i := range s.trailing {
		if s.trailClicks[i].Clicked(gtx) {
			s.tap(s.trailing[i])
		}
	}
	if s.closeClick.Clicked(gtx) {
		s.target = 0
	}

}

func (s *Swipeable) Layout(gtx layout.Context) layout.Dimensions {
	aw := gtx.Dp(unit.Dp(actionWidth))
	left, leftClicks, right, rightClicks := s.sides(gtx)
	leftW, rightW := float32(aw*len(left)), float32(aw*len(right))
	// SetActions may have changed the widths under an open row.
	switch {
	case s.target > 0:
		s.target = leftW
	case s.target < 0:
		s.target = -rightW
	}

	s.update(gtx, leftW, rightW)
	if s.pass == nil {
		s.pass = func(gtx layout.Context) bool {
			aw := gtx.Dp(unit.Dp(actionWidth))
			left, _, right, _ := s.sides(gtx)
			off, target := s.offset, s.target
			s.update(gtx, float32(aw*len(left)), float32(aw*len(right)))
			return s.offset != off || s.target != target
		}
	}
	kit.HandleInput(s.pass)

	// Animate toward the target.
	dt := gtx.Now.Sub(s.last)
	if s.last.IsZero() || dt < 0 || dt > animDuration {
		dt = animDuration / 10
	}
	s.last = gtx.Now
	if !s.dragging && s.offset != s.target {
		step := float32(aw*max(len(s.leading), len(s.trailing), 1)) * float32(dt) / float32(animDuration)
		if s.offset < s.target {
			s.offset = min(s.target, s.offset+step)
		} else {
			s.offset = max(s.target, s.offset-step)
		}
	}
	if s.offset != s.target || s.dragging {
		kit.RequestFrame(gtx)
	}
	if s.offset != 0 {
		kit.MarkLive()
	}

	// The content, recorded so the actions can be sized to its height.
	off := image.Pt(int(s.offset), 0)
	track := elements.TrackChild()
	macro := op.Record(gtx.Ops)
	dims := s.content().Layout(gtx)
	call := macro.Stop()
	track.End()
	size := dims.Size

	// The drag area is the row's root, so the content's own clicks keep
	// working inside it.
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	s.drag.Add(gtx.Ops)

	rtl := kit.RTL(gtx)
	if s.offset > 0 {
		s.layoutActions(gtx, left, leftClicks, image.Rect(0, 0, int(s.offset), size.Y), 0, aw, rtl)
	}
	if s.offset < 0 {
		s.layoutActions(gtx, right, rightClicks, image.Rect(size.X+int(s.offset), 0, size.X, size.Y), size.X-int(rightW), aw, rtl)
	}

	track.Place(off)
	func() {
		defer op.Offset(off).Push(gtx.Ops).Pop()
		call.Add(gtx.Ops)
		if s.target != 0 && !s.dragging {
			// While open, a tap on the row closes it instead of
			// reaching the content.
			s.closeClick.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Dimensions{Size: size}
			})
		}
	}()
	return dims
}

func (s *Swipeable) tap(a Action) {
	if a.OnTap != nil {
		a.OnTap()
	}
	s.target = 0
}

// sides returns the actions revealed on the left (by swiping right)
// and on the right: leading and trailing, swapped right to left, where
// lines start at the right edge.
func (s *Swipeable) sides(gtx layout.Context) (left []Action, leftClicks []widget.Clickable, right []Action, rightClicks []widget.Clickable) {
	if kit.RTL(gtx) {
		return s.trailing, s.trailClicks, s.leading, s.leadClicks
	}
	return s.leading, s.leadClicks, s.trailing, s.trailClicks
}

// layoutActions paints actions side by side from x0, clipped to the
// revealed area; reversed, the first is the rightmost.
func (s *Swipeable) layoutActions(gtx layout.Context, actions []Action, clicks []widget.Clickable, revealed image.Rectangle, x0, aw int, reversed bool) {
	defer clip.Rect(revealed).Push(gtx.Ops).Pop()
	for i, a := range actions {
		bg, fg := a.Background, a.Color
		if bg == "" {
			bg = "theme.primary"
		}
		if fg == "" {
			fg = "theme.onprimary"
		}
		var kids []elements.Element
		if a.Icon != nil {
			kids = append(kids, a.Icon)
		}
		if a.Label != "" {
			kids = append(kids, elements.Text(a.Label, styles.Styles{Color: fg, FontSize: 13}))
		}
		box := elements.Stack(styles.Styles{Width: -1, Height: -1, Background: bg, Color: fg, Justify: properties.JustifyCenter, Align: properties.AlignCenter, Gap: 4}, kids...)
		func() {
			slot := i
			if reversed {
				slot = len(actions) - 1 - i
			}
			pos := image.Pt(x0+slot*aw, 0)
			defer op.Offset(pos).Push(gtx.Ops).Pop()
			pop := elements.PushOffset(pos)
			defer pop()
			agtx := gtx
			agtx.Constraints = layout.Exact(image.Pt(aw, revealed.Dy()))
			clicks[i].Layout(agtx, box.Layout)
		}()
	}
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
