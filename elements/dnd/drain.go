package dnd

import (
	"image"
	"image/color"
	"time"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/elements/internal/base"
	"github.com/xmhrnx/giode/styles/properties"
)

// Drain runs the drag in progress after the frame's layout: it takes the
// pointer's events (window-wide, so the drag goes on when the source
// scrolls away), finds the target under it, finishes drops, and draws
// the insertion line and the preview over everything. The app calls it
// after the view and its portals; component authors don't need to.
func Drain(gtx layout.Context) {
	defer endFrame()
	s := active
	if s == nil {
		return
	}
	win := base.WindowSize()
	if win == (image.Point{}) {
		win = gtx.Constraints.Max
	}
	if s.src == nil {
		drainExternal(gtx, s)
		return
	}
	// The source's handler, window-wide and registered last: its events
	// keep coming, in window coordinates, wherever the source is.
	cl := clip.Rect{Max: win}.Push(gtx.Ops)
	event.Op(gtx.Ops, s.src)
	if !s.keyboard {
		properties.CursorGrabbing.CursorFor().Add(gtx.Ops)
	}
	cl.Pop()

	drop := s.release
	moveTarget, moveSlot := 0, 0
	for {
		ev, ok := gtx.Event(
			pointer.Filter{Target: s.src, Kinds: pointer.Drag | pointer.Release | pointer.Cancel},
			key.FocusFilter{Target: s.src},
			key.Filter{Focus: s.src, Name: key.NameEscape},
			key.Filter{Focus: s.src, Name: key.NameSpace},
			key.Filter{Focus: s.src, Name: key.NameReturn},
			key.Filter{Focus: s.src, Name: key.NameEnter},
			key.Filter{Focus: s.src, Name: key.NameUpArrow},
			key.Filter{Focus: s.src, Name: key.NameDownArrow},
			key.Filter{Focus: s.src, Name: key.NameLeftArrow},
			key.Filter{Focus: s.src, Name: key.NameRightArrow},
		)
		if !ok {
			break
		}
		switch e := ev.(type) {
		case pointer.Event:
			if s.keyboard || e.PointerID != s.id {
				if e.Kind == pointer.Cancel && !s.keyboard {
					s.cancel = true
				}
				break
			}
			switch e.Kind {
			case pointer.Drag:
				s.pos = e.Position
			case pointer.Release:
				s.pos = e.Position
				drop = true
			case pointer.Cancel:
				s.cancel = true
			}
		case key.FocusEvent:
			if !e.Focus && s.keyboard {
				s.cancel = true
			}
		case key.Event:
			if e.State != key.Press {
				break
			}
			switch e.Name {
			case key.NameEscape:
				s.cancel = true
			case key.NameSpace, key.NameReturn, key.NameEnter:
				if s.keyboard {
					drop = true
				}
			case key.NameUpArrow:
				moveSlot--
			case key.NameDownArrow:
				moveSlot++
			case key.NameLeftArrow:
				moveTarget--
			case key.NameRightArrow:
				moveTarget++
			}
		}
	}
	if !s.located {
		s.located = true
		locateSource(s)
	}
	if s.cancel {
		finish(false)
		gtx.Execute(op.InvalidateCmd{})
		return
	}

	over, index := s.over, s.index
	if s.keyboard {
		over, index = keyboardSlot(s, moveTarget, moveSlot)
		if p, ok := slotPoint(over, index); ok {
			s.pos = f32.Pt(float32(p.X), float32(p.Y))
		}
	} else {
		over, index = hitTest(s)
	}
	if over != s.over || index != s.index {
		if s.over != nil && s.over != over && s.over.onLeave != nil {
			s.over.onLeave(s.payload)
		}
		if over != nil && over != s.over && over.onEnter != nil {
			over.onEnter(s.payload)
		}
		s.over, s.index = over, index
		gtx.Execute(op.InvalidateCmd{})
	}
	if drop {
		finish(true)
		gtx.Execute(op.InvalidateCmd{})
		return
	}
	if !s.pos.Round().In(image.Rectangle{Max: win}) && export(s) {
		// Dragged out of the window: the system's drag now.
		gtx.Execute(op.InvalidateCmd{})
		return
	}
	// The line over the preview: the pointer, and so the preview, is
	// usually right on it.
	paintPreview(gtx, s, win)
	paintIndicator(gtx, s)
}

// drainExternal follows a drag from another application: it has no
// source here, and the system draws its preview.
func drainExternal(gtx layout.Context, s *session) {
	over, index := hitTest(s)
	if over != s.over || index != s.index {
		if s.over != nil && s.over != over && s.over.onLeave != nil {
			s.over.onLeave(s.payload)
		}
		if over != nil && over != s.over && over.onEnter != nil {
			over.onEnter(s.payload)
		}
		s.over, s.index = over, index
		gtx.Execute(op.InvalidateCmd{})
	}
	paintIndicator(gtx, s)
}

// endFrame keeps the frame's targets as the last frame's, for a drop
// arriving between frames without a drag before it, and starts the next
// frame's list.
func endFrame() {
	clear(lastTargets)
	lastTargets, targets = targets, lastTargets[:0]
	listStack = listStack[:0]
	frame++
}

// accepting returns the targets laid out this frame that take the
// payload and show.
func accepting(s *session) []*DropTarget {
	var out []*DropTarget
	for _, t := range targets {
		if t.accepts(s.payload) && !t.visible().Empty() {
			out = append(out, t)
		}
	}
	return out
}

// hitTest finds the topmost target under the pointer that takes the
// payload, and the slot in it.
func hitTest(s *session) (*DropTarget, int) {
	return hitTestIn(targets, s)
}

// hitTestIn is hitTest among ts.
func hitTestIn(ts []*DropTarget, s *session) (*DropTarget, int) {
	p := s.pos.Round()
	for i := len(ts) - 1; i >= 0; i-- {
		t := ts[i]
		if !t.accepts(s.payload) || !p.In(t.visible()) {
			continue
		}
		if t.list {
			return t, t.slotAt(p)
		}
		return t, -1
	}
	return nil, -1
}

// locateSource finds the list item holding the source, for keyboard
// moves to start from.
func locateSource(s *session) {
	c := s.src.origin.Add(s.src.size.Div(2))
	for i := len(targets) - 1; i >= 0; i-- {
		t := targets[i]
		if !t.list {
			continue
		}
		for _, it := range t.items {
			if c.In(image.Rectangle{Min: it.origin, Max: it.origin.Add(it.size)}) {
				s.srcTarget, s.srcIndex = t, it.index
				return
			}
		}
	}
}

// keyboardSlot moves the keyboard drag's slot: dt targets on (left and
// right arrows), ds slots on (up and down).
func keyboardSlot(s *session, dt, ds int) (*DropTarget, int) {
	acc := accepting(s)
	if len(acc) == 0 {
		return nil, -1
	}
	cur := -1
	for i, t := range acc {
		if t == s.over {
			cur = i
		}
	}
	index := s.index
	if cur < 0 {
		// Start where the source is, or at the first target.
		cur = 0
		index = -1
		for i, t := range acc {
			if t == s.srcTarget {
				cur, index = i, s.srcIndex
			}
		}
		if acc[cur].list && index < 0 {
			index, _ = acc[cur].slotRange()
		}
	}
	t := acc[cur]
	if !t.list && ds != 0 {
		// Up and down step through targets that aren't lists.
		dt, ds = dt+ds, 0
	}
	if dt != 0 {
		cur = max(0, min(len(acc)-1, cur+dt))
		t = acc[cur]
		if t.list {
			lo, hi := t.slotRange()
			if index < 0 {
				index = lo
			}
			index = max(lo, min(hi, index))
		} else {
			index = -1
		}
	}
	for ; ds != 0 && t.list; ds -= sign(ds) {
		lo, hi := t.slotRange()
		next := index + sign(ds)
		// The slots on either side of the source in its own list are
		// the same place: step over the second.
		if t == s.srcTarget && next == s.srcIndex+1 {
			next += sign(ds)
		}
		if next < lo || next > hi {
			break
		}
		index = next
	}
	return t, index
}

func sign(v int) int {
	if v < 0 {
		return -1
	}
	return 1
}

// slotPoint is where a keyboard drag points: the middle of slot index's
// line, or of the target.
func slotPoint(t *DropTarget, index int) (image.Point, bool) {
	if t == nil {
		return image.Point{}, false
	}
	if t.list {
		if at, from, to, ok := t.slotLine(index); ok {
			if t.axis == AxisHorizontal {
				return image.Pt(at, (from+to)/2), true
			}
			return image.Pt((from+to)/2, at), true
		}
	}
	v := t.visible()
	return v.Min.Add(v.Size().Div(2)), true
}

// paintIndicator draws the insertion line of the list under the drag.
func paintIndicator(gtx layout.Context, s *session) {
	t := s.over
	if t == nil || !t.list || !t.indicator {
		return
	}
	at, from, to, ok := t.slotLine(s.index)
	if !ok {
		return
	}
	th := max(2, gtx.Dp(2))
	dot := max(6, gtx.Dp(8))
	vis := t.visible()
	// Inside the target's visible part, with room for the line at its
	// edges.
	defer clip.Rect(vis.Inset(-dot)).Push(gtx.Ops).Pop()
	col := properties.CalcColor("theme.primary")
	var line, ring image.Rectangle
	if t.axis == AxisHorizontal {
		at = max(vis.Min.X+th/2, min(vis.Max.X-th/2, at))
		line = image.Rect(at-th/2, from+dot/2, at-th/2+th, to)
		ring = image.Rect(at-dot/2, from, at-dot/2+dot, from+dot)
	} else {
		at = max(vis.Min.Y+th/2, min(vis.Max.Y-th/2, at))
		line = image.Rect(from+dot/2, at-th/2, to, at-th/2+th)
		ring = image.Rect(from, at-dot/2, from+dot, at-dot/2+dot)
	}
	paint.FillShape(gtx.Ops, col, clip.UniformRRect(line, th/2).Op(gtx.Ops))
	paint.FillShape(gtx.Ops, col, clip.Stroke{Path: clip.Ellipse(ring).Path(gtx.Ops), Width: float32(th)}.Op())
}

// paintPreview draws what follows the pointer.
func paintPreview(gtx layout.Context, s *session, win image.Point) {
	d := s.src
	var call op.CallOp
	size := s.size
	switch {
	case d.preview != nil:
		if s.previewEl == nil {
			s.previewEl = d.preview()
		}
		pgtx := gtx
		pgtx.Constraints = layout.Constraints{Max: win}
		m := op.Record(gtx.Ops)
		size = s.previewEl.Layout(pgtx).Size
		call = m.Stop()
	default:
		var ok bool
		if call, ok = callOf(d); ok {
			size = d.size
		} else if d.content != nil {
			// The source isn't laid out (scrolled away): draw its content
			// afresh.
			pgtx := gtx
			pgtx.Constraints = layout.Exact(s.size)
			m := op.Record(gtx.Ops)
			if el := d.content(DragState{Dragging: true}); el != nil {
				el.Layout(pgtx)
			}
			call = m.Stop()
		} else {
			return
		}
	}
	grab := image.Pt(min(s.grab.X, size.X), min(s.grab.Y, size.Y))
	at := s.pos.Round().Sub(grab)
	if s.keyboard {
		at = keyboardPreviewAt(gtx, s, size)
	}
	switch d.axis {
	case AxisVertical:
		at.X = d.origin.X
	case AxisHorizontal:
		at.Y = d.origin.Y
	}
	off := op.Offset(at).Push(gtx.Ops)
	paintShadow(gtx, size, gtx.Dp(8))
	o := paint.PushOpacity(gtx.Ops, 0.9)
	call.Add(gtx.Ops)
	o.Pop()
	off.Pop()
}

// keyboardPreviewAt places a keyboard drag's preview just past its slot,
// or over the middle of its target.
func keyboardPreviewAt(gtx layout.Context, s *session, size image.Point) image.Point {
	t := s.over
	if t == nil {
		return s.srcOrigin
	}
	gap := gtx.Dp(4)
	if t.list {
		if at, from, _, ok := t.slotLine(s.index); ok {
			if t.axis == AxisHorizontal {
				return image.Pt(at+gap, from)
			}
			return image.Pt(from, at+gap)
		}
	}
	v := t.visible()
	return v.Min.Add(v.Size().Sub(size).Div(2))
}

// paintShadow draws a soft shadow under a size box with radius corners,
// the way a lifted card casts one.
func paintShadow(gtx layout.Context, size image.Point, radius int) {
	spread := gtx.Dp(12)
	drop := gtx.Dp(4)
	const steps = 6
	for i := steps; i >= 1; i-- {
		g := spread * i / steps
		r := image.Rect(-g, -g+drop, size.X+g, size.Y+g+drop)
		a := uint8(10 + 6*(steps-i))
		paint.FillShape(gtx.Ops, color.NRGBA{A: a}, clip.UniformRRect(r, radius+g).Op(gtx.Ops))
	}
}

// AutoScroll is for scroll containers: it returns how far, in pixels,
// to scroll this frame for the drag in progress - the pointer near an
// edge of r, the container's viewport in window coordinates, moves the
// content in that direction, faster the closer it is. room says which
// ways the container can still scroll. Positive values scroll toward
// the bottom and right. key identifies the container across frames.
func AutoScroll(gtx layout.Context, key any, r image.Rectangle, room Room) image.Point {
	s := active
	if s == nil || r.Empty() {
		return image.Point{}
	}
	p := s.pos
	zone := float32(gtx.Dp(autoScrollZone))
	maxSpeed := float32(gtx.Dp(autoScrollSpeed))
	outside := zone
	if s.keyboard {
		outside = 0
	}
	depth := func(c, lo, hi float32, back, fwd bool) float32 {
		z := min(zone, (hi-lo)/4)
		if z <= 0 {
			return 0
		}
		switch {
		case back && c < lo+z && c >= lo-outside:
			d := min(1, (lo+z-c)/z)
			return -d * d
		case fwd && c > hi-z && c <= hi+outside:
			d := min(1, (c-(hi-z))/z)
			return d * d
		}
		return 0
	}
	var vx, vy float32
	if p.Y >= float32(r.Min.Y)-outside && p.Y <= float32(r.Max.Y)+outside && p.X >= float32(r.Min.X) && p.X <= float32(r.Max.X) {
		vy = depth(p.Y, float32(r.Min.Y), float32(r.Max.Y), room.Up, room.Down)
	}
	if p.X >= float32(r.Min.X)-outside && p.X <= float32(r.Max.X)+outside && p.Y >= float32(r.Min.Y) && p.Y <= float32(r.Max.Y) {
		vx = depth(p.X, float32(r.Min.X), float32(r.Max.X), room.Left, room.Right)
	}
	if vx == 0 && vy == 0 {
		delete(s.scrolled, key)
		return image.Point{}
	}
	gtx.Execute(op.InvalidateCmd{})
	tick := s.scrolled[key]
	if tick == nil {
		s.scrolled[key] = &scrollTick{at: gtx.Now}
		return image.Point{}
	}
	dt := float32(min(gtx.Now.Sub(tick.at), 50*time.Millisecond).Seconds())
	tick.at = gtx.Now
	// One axis at a time keeps the fraction simple; corners favor the
	// vertical one.
	if vy != 0 {
		d := vy*maxSpeed*dt + tick.frac
		tick.frac = d - float32(int(d))
		return image.Pt(0, int(d))
	}
	d := vx*maxSpeed*dt + tick.frac
	tick.frac = d - float32(int(d))
	return image.Pt(int(d), 0)
}

// Room says which ways a scroll container can still scroll.
type Room struct {
	Up, Down, Left, Right bool
}

const (
	// autoScrollZone is how close to an edge the pointer starts
	// scrolling, and autoScrollSpeed the fastest it scrolls, per second.
	autoScrollZone  = unit.Dp(56)
	autoScrollSpeed = unit.Dp(1400)
)
