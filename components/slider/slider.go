// Package slider provides a stateful horizontal slider.
package slider

import (
	"image"
	"strconv"

	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"

	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Slider is a stateful horizontal slider in the range [0, 1]. Create
// it once with New and lay it out every frame.
type Slider struct {
	// Contextual gives it a context menu (Context).
	elements.Contextual

	value    float32
	drag     gesture.Drag
	hover    gesture.Hover
	onChange func(v float32)
	st       styles.Styles

	// hovered, and the geometry of the last layout, for handling input
	// in the input pass (see kit.HandleInput).
	hovered     bool
	lastWidth   float32
	lastPadding int
	pass        func(gtx layout.Context) bool
	// cache keeps the output between frames; see Layout.
	cache   *elements.MemoCache
	build   func() elements.Element
	watched elements.Watch[drawn]
}

// drawn is what a Slider draws from its own fields.
type drawn struct {
	value float32
	st    styles.Styles
}

// CacheSafe reports true: what it draws comes from its own fields,
// which the cache watches.
func (s *Slider) CacheSafe() bool { return true }

// watch is what it draws from its own fields.
func (s *Slider) watch() drawn {
	return drawn{s.value, s.st}
}

// New returns a Slider. The styles argument is optional: Color is
// the fill, Background is the track, BorderColor is the thumb.
func New(st ...styles.Styles) *Slider {
	s := &Slider{value: 0}
	if len(st) > 0 {
		s.st = st[0]
	}
	return s
}

// Value returns the current value in the range [0, 1].
func (s *Slider) Value() float32 {
	return s.value
}

// Dragging reports whether the thumb is being dragged this frame.
// Callers that sync the slider to an external value (a player's
// position, say) use it to avoid overwriting the value under the
// user's finger.
func (s *Slider) Dragging() bool {
	return s.drag.Dragging()
}

// Change sets the value and invokes OnChange, as a user drag would.
func (s *Slider) Change(v float32) {
	s.value = clamp01(v)
	if s.onChange != nil {
		s.onChange(s.value)
	}
}

// SetValue sets the value without invoking OnChange.
func (s *Slider) SetValue(v float32) {
	s.value = clamp01(v)
}

// OnChange sets the handler invoked while dragging. It returns s for
// chaining.
func (s *Slider) OnChange(fn func(v float32)) *Slider {
	s.onChange = fn
	return s
}

// update handles hover and drags, with the geometry of the last
// layout.
func (s *Slider) update(gtx layout.Context) {
	s.hovered = s.hover.Update(gtx.Source)
	changed := false
	for {
		ev, ok := s.drag.Update(gtx.Metric, gtx.Source, gesture.Horizontal)
		if !ok {
			break
		}
		switch ev.Kind {
		case pointer.Press, pointer.Drag:
			// No Priority check: a plain click never grabs the pointer
			// (the drag only grabs once it moves past the touch slop),
			// so requiring a grab made clicking the track do nothing.
			// Press only arrives when the click lands on the slider, and
			// a drag another handler grabs arrives as Cancel instead.
			v := s.valueFromX(float32(ev.Position.X), s.lastWidth, s.lastPadding)
			if v != s.value {
				s.value = v
				changed = true
			}
		}
	}
	if changed && s.onChange != nil {
		s.onChange(s.value)
		kit.RequestFrame(gtx)
	}
}

// Layout lays out and updates the slider. It expands to the available
// width.
func (s *Slider) Layout(gtx layout.Context) layout.Dimensions {
	dims := s.layoutSelf(gtx)
	// A right-click or long-press anywhere on it opens its context menu.
	s.AnchorContext(gtx, dims.Size)
	return dims
}

// layoutSelf lays it out without its context menu area.
func (s *Slider) layoutSelf(gtx layout.Context) layout.Dimensions {
	// The output recorded last time is replayed until the value, the
	// styles, hover, a drag, constraints or direction change.
	if s.cache == nil {
		s.cache = elements.NewMemoCache()
		uncached := elements.Raw(s.layoutUncached)
		s.build = func() elements.Element { return uncached }
		s.watched.Snap = s.watch
	}
	return s.cache.LayoutOwn(gtx, s.build, &s.watched)
}

// layoutUncached lays it out without the cache.
func (s *Slider) layoutUncached(gtx layout.Context) layout.Dimensions {
	m := metricsFor(gtx)
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	size := image.Pt(gtx.Constraints.Min.X, m.height)

	// The drag and hover regions must be clipped to the slider itself,
	// otherwise they would span whatever outer area it is laid out in
	// and steal events from siblings.
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	// Right to left, the value grows leftward; the flip also mirrors
	// the pointer positions the drag reports.
	defer kit.MirrorRTL(gtx, size.X)()
	// Keep the slider draggable even inside a frameless window's
	// whole-window drag area (see the root package's layoutDrag):
	// without this, dragging the thumb would be hijacked into a
	// native window-move instead.
	kit.BlockDragHere(gtx)
	s.drag.Add(gtx.Ops)
	s.hover.Add(gtx.Ops)
	s.lastWidth, s.lastPadding = float32(size.X), m.padding
	s.update(gtx)
	hovered := s.hovered
	if s.pass == nil {
		s.pass = func(gtx layout.Context) bool {
			h, d, v := s.hovered, s.drag.Dragging(), s.value
			s.update(gtx)
			return h != s.hovered || d != s.drag.Dragging() || v != s.value
		}
	}
	kit.HandleInput(s.pass)

	thumb := s.st.BorderColor
	switch {
	case s.drag.Dragging():
		thumb = kit.Pressed(resolve(thumb, defaultThumb))
		properties.CursorGrabbing.CursorFor().Add(gtx.Ops)
	case hovered:
		thumb = kit.Hovered(resolve(thumb, defaultThumb))
		properties.CursorGrab.CursorFor().Add(gtx.Ops)
	}

	paintTrack(gtx, size, s.st.Background, m)
	paintFill(gtx, size, s.value, s.st.Color, m)
	paintThumb(gtx, size, s.value, thumb, m)
	if s.drag.Dragging() {
		kit.MarkLive()
	}
	elements.RecordSemantic(gtx, elements.SemanticNode{
		Kind: elements.SemanticSlider, Value: strconv.FormatFloat(float64(s.value), 'f', 3, 32), Target: s,
	}, size)
	return layout.Dimensions{Size: size}
}

func (s *Slider) valueFromX(x, width float32, padding int) float32 {
	if width <= float32(2*padding) {
		return 0
	}
	return clamp01((x - float32(padding)) / (width - float32(2*padding)))
}

func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
