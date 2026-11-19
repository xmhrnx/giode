// Package dnd is in-app drag and drop: Draggable sources, DropTarget
// areas that accept them, insertion slots in lists (ListItem), a
// preview that follows the pointer or finger, auto-scrolling of the
// scroll containers under it, and a keyboard way to do the same.
//
// One drag runs at a time. While it runs, the window owns the pointer:
// the source can scroll out of view or be removed from the tree, and
// the drag goes on. The app draws the drag layer (preview, insertion
// line) and finishes drops after everything else in the frame (see
// Drain).
package dnd

import (
	"image"
	"time"

	"gioui.org/f32"
	"gioui.org/io/pointer"
	"gioui.org/op"

	"github.com/xmhrnx/giode/elements/internal/base"
)

// session is the drag in progress.
type session struct {
	src     *Draggable
	payload any
	id      pointer.ID
	touch   bool
	// keyboard is set for a drag started with the keyboard: arrows move
	// between drop slots, there's no pointer.
	keyboard bool
	// pos is the pointer's window position: from the source's events
	// until the window takes them over, then from the window's.
	pos f32.Point
	// grab is where the source was taken, relative to its origin, and
	// size its size then.
	grab, size image.Point
	// srcOrigin is where the source was when the drag started.
	srcOrigin image.Point

	// over is the target under the pointer that accepts the payload,
	// and index the insertion slot in it (-1 when it's not a list).
	over  *DropTarget
	index int
	// release and cancel are a release or cancel seen by the source in
	// the frame the drag started, finished by Drain.
	release, cancel bool

	// srcTarget and srcIndex are where the source sits: the list target
	// holding it and its item index, -1 when it's in none (found on the
	// first Drain, for keyboard moves).
	srcTarget *DropTarget
	srcIndex  int
	located   bool

	started time.Time
	// previewEl is the custom preview, built when the drag starts.
	previewEl base.Element
	// scrolled holds each auto-scrolling container's last tick and the
	// fraction of a pixel it hasn't moved yet.
	scrolled map[any]*scrollTick
}

type scrollTick struct {
	at   time.Time
	frac float32
}

var (
	active *session
	// frame counts Drain calls, to tell this frame's source recording
	// from an old one.
	frame uint64
	// targets are the drop targets laid out this frame, in layout
	// order.
	targets []*DropTarget
	// lastTargets are the last frame's.
	lastTargets []*DropTarget
	// listStack holds the list targets being laid out, innermost last.
	listStack []*DropTarget
)

// Dragging reports whether a drag is in progress.
func Dragging() bool { return active != nil }

// Payload returns the payload of the drag in progress.
func Payload() (any, bool) {
	if active == nil {
		return nil, false
	}
	return active.payload, true
}

// Cancel ends the drag in progress, if any, as cancelled.
func Cancel() {
	if active != nil {
		active.cancel = true
	}
}

// As returns payload as a T.
func As[T any](payload any) (T, bool) {
	v, ok := payload.(T)
	return v, ok
}

// AcceptType returns an accept predicate for payloads of type T.
func AcceptType[T any]() func(payload any) bool {
	return func(p any) bool {
		_, ok := p.(T)
		return ok
	}
}

// Move moves s[from] to insertion slot to, a slot between items as a
// DropEvent's Index gives it (0 before the first, len(s) after the
// last), and returns the index it lands at.
func Move[T any](s []T, from, to int) int {
	if from < 0 || from >= len(s) {
		return from
	}
	to = max(0, min(to, len(s)))
	if to > from {
		to--
	}
	if to == from {
		return to
	}
	v := s[from]
	if to < from {
		copy(s[to+1:from+1], s[to:from])
	} else {
		copy(s[from:to], s[from+1:to+1])
	}
	s[to] = v
	return to
}

// Insert inserts v at slot i (clamped to s) and returns the slice.
func Insert[T any](s []T, i int, v T) []T {
	i = max(0, min(i, len(s)))
	var zero T
	s = append(s, zero)
	copy(s[i+1:], s[i:])
	s[i] = v
	return s
}

// Remove removes s[i] and returns the slice.
func Remove[T any](s []T, i int) []T {
	if i < 0 || i >= len(s) {
		return s
	}
	return append(s[:i], s[i+1:]...)
}

// start begins a drag of d.
func start(d *Draggable, s *session) {
	if active != nil {
		finish(false)
	}
	s.src = d
	s.payload = d.payload
	s.index = -1
	s.srcIndex = -1
	s.scrolled = map[any]*scrollTick{}
	active = s
	base.SetDragActive(true)
	if d.onStart != nil {
		d.onStart()
	}
}

// finish ends the drag: dropped on the target under it when drop is
// set and there is one, cancelled otherwise.
func finish(drop bool) {
	s := active
	if s == nil {
		return
	}
	active = nil
	base.SetDragActive(false)
	res := DragResult{Cancelled: true, Index: -1}
	if drop && s.over != nil {
		t := s.over
		res = DragResult{Dropped: true, Target: t, Index: s.index}
		if t.onLeave != nil {
			t.onLeave(s.payload)
		}
		if t.onDrop != nil {
			t.onDrop(DropEvent{
				Payload: s.payload, Index: s.index, Source: s.src,
				Position: s.pos.Round().Sub(t.origin), Keyboard: s.keyboard,
			})
		}
	} else if s.over != nil && s.over.onLeave != nil {
		s.over.onLeave(s.payload)
	}
	if s.src != nil && s.src.onEnd != nil {
		s.src.onEnd(res)
	}
}

// ExportData is what a Draggable gives other applications when it's
// dragged out of the window (Draggable.Export).
type ExportData struct {
	// Files are paths of local files, for a file manager, a mail
	// composer, an editor.
	Files []string
	// Text is plain text, for a text field or an editor.
	Text string
}

func (e ExportData) empty() bool { return len(e.Files) == 0 && e.Text == "" }

// Exporter starts a drag of data in the operating system's drag and
// drop, from the pointer's position, while its button is still held.
// It reports whether it started; done is called on the UI goroutine
// when the drag ends, with whether another application took the drop.
// The app sets it (SetExporter) where the platform supports it.
type Exporter func(data ExportData, done func(dropped bool)) bool

var exporter Exporter

// SetExporter sets the function that hands drags leaving the window to
// the operating system. The app calls it; component authors don't need
// to.
func SetExporter(fn Exporter) { exporter = fn }

// CanExport reports whether drags can leave the window on this
// platform.
func CanExport() bool { return exporter != nil }

// export hands the drag to the operating system, if its source exports
// something. It reports whether it did: the in-app drag is then over,
// and the source's OnDragEnd runs when the system's drag ends.
func export(s *session) bool {
	d := s.src
	if exporter == nil || d == nil || d.export == nil || s.keyboard || s.touch {
		return false
	}
	data := d.export()
	if data.empty() {
		return false
	}
	ended := false
	if !exporter(data, func(dropped bool) {
		if ended {
			return
		}
		ended = true
		if d.onEnd != nil {
			d.onEnd(DragResult{Exported: true, Dropped: dropped, Cancelled: !dropped, Index: -1})
		}
	}) {
		return false
	}
	active = nil
	base.SetDragActive(false)
	if s.over != nil && s.over.onLeave != nil {
		s.over.onLeave(s.payload)
	}
	return true
}

// Files is the payload of files dragged in from another application:
// a DropTarget that accepts it (Accept(AcceptType[Files]())) takes them.
// Paths is empty while the drag is over the window - not every platform
// tells what's dragged before the drop - and filled in OnDrop.
type Files struct {
	Paths []string
}

// ExternalKind is the stage of a drag from another application.
type ExternalKind int

const (
	ExternalEnter ExternalKind = iota + 1
	ExternalMove
	ExternalLeave
	ExternalDrop
)

// External reports files dragged over the window from another
// application, at pos in window pixels, on the UI goroutine: targets
// accepting Files highlight and take them as with an in-app drag. For
// ExternalDrop, paths are the files, and it reports whether a target
// took them. The app calls it; component authors don't need to.
func External(kind ExternalKind, pos f32.Point, paths []string) (taken bool) {
	s := active
	if s != nil && s.src != nil {
		// An in-app drag owns the pointer.
		return false
	}
	switch kind {
	case ExternalEnter, ExternalMove:
		if s == nil {
			s = &session{payload: Files{}, index: -1, srcIndex: -1, located: true, scrolled: map[any]*scrollTick{}}
			active = s
			base.SetDragActive(true)
		}
		s.pos = pos
	case ExternalLeave:
		finish(false)
	case ExternalDrop:
		if s == nil {
			// A drop with no drag reported before it (Windows tells an
			// app running as administrator only of the drop): find the
			// target among the last frame's.
			s = &session{payload: Files{}, pos: pos, index: -1, srcIndex: -1, located: true}
			s.over, s.index = hitTestIn(lastTargets, s)
			active = s
			base.SetDragActive(true)
		}
		s.pos = pos
		s.payload = Files{Paths: paths}
		taken = s.over != nil
		finish(true)
	}
	return taken
}

// callOf returns d's recording from this frame, if it was laid out.
func callOf(d *Draggable) (op.CallOp, bool) {
	return d.call, d.callFrame == frame+1
}
