package kit

import "gioui.org/gesture"

// Scroll gestures in progress, so a touch that's scrolling isn't also
// taken for a long-press.

var (
	frames      uint64
	scrollFrame uint64
)

// NextFrame advances the frame counter. The app calls it once per
// frame (elements.BeginFrame does).
func NextFrame() {
	frames++
	frameCount.Add(1)
}

// NoteScrolling records that a scroll gesture is moving content this
// frame: a touch drag past the slop, or a fling. Scrolling components
// call it.
func NoteScrolling() {
	scrollFrame = frames + 1
}

// Scrolling reports whether a scroll gesture moved content this frame
// or the last - the last, since a component handling touches may run
// before the list it's in.
func Scrolling() bool {
	return scrollFrame != 0 && frames+1-scrollFrame <= 1
}

// ScrollGesture tells a scroll gesture moving content from a finger
// merely resting on it: Gio's gesture.Scroll reports dragging from the
// moment a finger touches down. Scrolling components embed one and call
// Track after each Update.
type ScrollGesture struct {
	moving bool
}

// Track updates the state from the gesture and the distance it just
// returned. started is set when a touch drag starts moving content,
// ended when a drag or fling that moved content is over. While content
// moves it calls NoteScrolling.
func (g *ScrollGesture) Track(s *gesture.Scroll, dist int) (started, ended bool) {
	switch s.State() {
	case gesture.StateIdle:
		ended = g.moving
		g.moving = false
	case gesture.StateFlinging:
		g.moving = true
	case gesture.StateDragging:
		if dist != 0 && !g.moving {
			g.moving, started = true, true
		}
	}
	if g.moving {
		NoteScrolling()
	}
	return started, ended
}

// Moving reports whether a touch drag or fling is moving content.
func (g *ScrollGesture) Moving() bool {
	return g.moving
}
