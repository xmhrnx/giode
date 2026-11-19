package dnd

import (
	"image"
	"slices"

	"gioui.org/layout"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements/internal/base"
)

// DropState is a DropTarget's state during a drag, for its content to
// style itself.
type DropState struct {
	// Active is set while a drag it accepts is in progress, and Payload
	// is that drag's payload.
	Active  bool
	Payload any
	// Over is set while that drag is over it, and Index is then the
	// list slot it would drop at (-1 when it isn't a list).
	Over  bool
	Index int
}

// DropEvent is a drop on a DropTarget.
type DropEvent struct {
	// Payload is the dropped Draggable's payload, and Source the
	// Draggable.
	Payload any
	Source  *Draggable
	// Index is the list slot it was dropped at: 0 before the first item,
	// one more than an item's index after it. -1 when the target isn't
	// a list.
	Index int
	// Position is the pointer's position in the target, in pixels.
	Position image.Point
	// Keyboard is set for a drop made with the keyboard.
	Keyboard bool
}

// DropTarget is an area drags can be dropped on. Accept chooses the
// payloads it takes; OnDrop receives them. As a list (List), the
// ListItem elements inside it give drops an insertion slot, shown with
// a line between items.
type DropTarget struct {
	content func(DropState) base.Element
	accept  func(any) bool
	onDrop  func(DropEvent)
	onEnter func(payload any)
	onLeave func(payload any)

	list      bool
	axis      Axis
	indicator bool
	disabled  bool

	origin image.Point
	size   image.Point
	clip   *base.VisibleClip
	items  []*listItem
	frame  uint64
}

// NewDropTarget returns a drop target drawing content.
func NewDropTarget(content func(s DropState) base.Element) *DropTarget {
	return &DropTarget{content: content, indicator: true}
}

// Content replaces the content builder. It returns t for chaining.
func (t *DropTarget) Content(fn func(s DropState) base.Element) *DropTarget {
	t.content = fn
	return t
}

// Accept sets which payloads it takes; without it, it takes any. It
// returns t for chaining.
func (t *DropTarget) Accept(fn func(payload any) bool) *DropTarget {
	t.accept = fn
	return t
}

// OnDrop sets the drop handler. It returns t for chaining.
func (t *DropTarget) OnDrop(fn func(e DropEvent)) *DropTarget {
	t.onDrop = fn
	return t
}

// OnEnter and OnLeave set handlers for a drag it accepts moving over it
// and away (or ending). They return t for chaining.
func (t *DropTarget) OnEnter(fn func(payload any)) *DropTarget {
	t.onEnter = fn
	return t
}

// OnLeave: see OnEnter.
func (t *DropTarget) OnLeave(fn func(payload any)) *DropTarget {
	t.onLeave = fn
	return t
}

// List makes it a list along axis (AxisVertical unless AxisHorizontal):
// the ListItem elements laid out inside it are its items, and a drop
// goes into the slot between the two items nearest the pointer. It
// returns t for chaining.
func (t *DropTarget) List(axis Axis) *DropTarget {
	t.list = true
	t.axis = axis
	return t
}

// Indicator turns the insertion line of a list on or off (on by
// default). It returns t for chaining.
func (t *DropTarget) Indicator(v bool) *DropTarget {
	t.indicator = v
	return t
}

// Disabled stops it taking drops. It returns t for chaining.
func (t *DropTarget) Disabled(v bool) *DropTarget {
	t.disabled = v
	return t
}

// accepts reports whether it takes p.
func (t *DropTarget) accepts(p any) bool {
	return !t.disabled && (t.accept == nil || t.accept(p))
}

// State returns its state.
func (t *DropTarget) State() DropState {
	s := active
	if s == nil || !t.accepts(s.payload) {
		return DropState{Index: -1}
	}
	st := DropState{Active: true, Payload: s.payload, Index: -1}
	if s.over == t {
		st.Over, st.Index = true, s.index
	}
	return st
}

// visible is the part of it shown in the window.
func (t *DropTarget) visible() image.Rectangle {
	r := image.Rectangle{Min: t.origin, Max: t.origin.Add(t.size)}
	if t.clip != nil {
		r = r.Intersect(t.clip.Rect())
	}
	return r
}

// Layout lays it out.
func (t *DropTarget) Layout(gtx layout.Context) layout.Dimensions {
	base.NoteDragAware()
	measuring := base.Measuring()
	if !measuring {
		if t.frame != frame+1 {
			t.frame = frame + 1
			targets = append(targets, t)
		}
		clear(t.items)
		t.items = t.items[:0]
		t.clip = base.CurrentVisibleClip()
		base.TrackPosition(&t.origin)
	}
	var el base.Element
	if t.content != nil {
		el = t.content(t.State())
	}
	if t.list {
		listStack = append(listStack, t)
		n := len(listStack)
		defer func() { listStack = listStack[:n-1] }()
	}
	var dims layout.Dimensions
	if el != nil {
		dims = el.Layout(gtx)
	}
	if !measuring {
		t.size = dims.Size
	}
	if active != nil && t.accepts(active.payload) {
		kit.MarkLive()
	}
	return dims
}

// listItem is an item of a list target.
type listItem struct {
	index  int
	el     base.Element
	origin image.Point
	size   image.Point
}

// ListItem makes el the item at index of the innermost list DropTarget
// (see DropTarget.List), for its insertion slots. Indexes are the
// items' indexes in the list's data, so a virtualized list (LazyList,
// VirtualList) that lays out only the visible rows works the same: wrap
// each row its builder returns.
func ListItem(index int, el base.Element) base.Element {
	return &listItem{index: index, el: el}
}

// Layout lays it out.
func (l *listItem) Layout(gtx layout.Context) layout.Dimensions {
	base.NoteDragAware()
	var t *DropTarget
	if n := len(listStack); n > 0 && !base.Measuring() {
		t = listStack[n-1]
		base.TrackPosition(&l.origin)
	}
	var dims layout.Dimensions
	if l.el != nil {
		dims = l.el.Layout(gtx)
	}
	if t != nil {
		l.size = dims.Size
		t.items = append(t.items, l)
	}
	return dims
}

// sorted returns its items in index order.
func (t *DropTarget) sorted() []*listItem {
	items := t.items
	if !slices.IsSortedFunc(items, cmpItems) {
		slices.SortFunc(items, cmpItems)
	}
	return items
}

func cmpItems(a, b *listItem) int { return a.index - b.index }

// along returns the coordinate of p along the list's axis.
func (t *DropTarget) along(p image.Point) int {
	if t.axis == AxisHorizontal {
		return p.X
	}
	return p.Y
}

// span returns where an item starts and ends along the axis.
func (t *DropTarget) span(it *listItem) (int, int) {
	r := image.Rectangle{Min: it.origin, Max: it.origin.Add(it.size)}
	if t.axis == AxisHorizontal {
		return r.Min.X, r.Max.X
	}
	return r.Min.Y, r.Max.Y
}

// reversed reports whether its items run against the axis (a
// right-to-left row, a list laid out bottom up).
func (t *DropTarget) reversed(items []*listItem) bool {
	if len(items) < 2 {
		return false
	}
	a, _ := t.span(items[0])
	b, _ := t.span(items[len(items)-1])
	return b < a
}

// slotAt returns the insertion slot nearest p.
func (t *DropTarget) slotAt(p image.Point) int {
	items := t.sorted()
	if len(items) == 0 {
		return 0
	}
	c := t.along(p)
	rev := t.reversed(items)
	for _, it := range items {
		a, b := t.span(it)
		mid := (a + b) / 2
		if !rev && c < mid || rev && c > mid {
			return it.index
		}
	}
	return items[len(items)-1].index + 1
}

// slotRange returns the slots of the items laid out: from before the
// first to after the last.
func (t *DropTarget) slotRange() (lo, hi int) {
	items := t.sorted()
	if len(items) == 0 {
		return 0, 0
	}
	return items[0].index, items[len(items)-1].index + 1
}

// slotLine returns where slot i's line goes: its position along the
// axis, and the cross-axis span it covers. ok is false when the slot
// isn't among the items laid out.
func (t *DropTarget) slotLine(i int) (at, from, to int, ok bool) {
	items := t.sorted()
	cross := func(it *listItem) (int, int) {
		if t.axis == AxisHorizontal {
			return it.origin.Y, it.origin.Y + it.size.Y
		}
		return it.origin.X, it.origin.X + it.size.X
	}
	vis := t.visible()
	if len(items) == 0 {
		if t.axis == AxisHorizontal {
			return vis.Min.X, vis.Min.Y, vis.Max.Y, true
		}
		return vis.Min.Y, vis.Min.X, vis.Max.X, true
	}
	rev := t.reversed(items)
	for j, it := range items {
		if it.index < i {
			continue
		}
		if it.index > i && j == 0 {
			return 0, 0, 0, false
		}
		a, b := t.span(it)
		edge := a
		if rev {
			edge = b
		}
		from, to = cross(it)
		if j > 0 {
			pa, pb := t.span(items[j-1])
			prev := pb
			if rev {
				prev = pa
			}
			return (prev + edge) / 2, from, to, true
		}
		return edge, from, to, true
	}
	last := items[len(items)-1]
	if last.index+1 != i {
		return 0, 0, 0, false
	}
	a, b := t.span(last)
	from, to = cross(last)
	if rev {
		return a, from, to, true
	}
	return b, from, to, true
}
