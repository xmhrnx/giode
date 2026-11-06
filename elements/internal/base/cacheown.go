package base

import "gioui.org/layout"

// CacheSafe is implemented by elements whose output depends only on
// their own fields, which don't change after they're built (or whose
// changes they track themselves), on States they read, and on input
// they handle through input passes. A Box made only of such elements can
// cache its output across frames: nothing it draws can change without
// the cache noticing. Raw drawing functions, builder functions and
// unknown elements aren't: they may read any variable.
type CacheSafe interface {
	CacheSafe() bool
}

// IsCacheSafe reports whether el is a CacheSafe element that says it
// is safe to cache.
func IsCacheSafe(el Element) bool {
	c, ok := el.(CacheSafe)
	return ok && c.CacheSafe()
}

// Watcher reports whether a component's own fields changed since it
// last drew them (Changed), and records the fields Changed saw once it
// has drawn them (Commit). Watch implements it.
type Watcher interface {
	Changed() bool
	Commit()
}

// Watch is a Watcher over a snapshot of a component's fields: Snap
// returns the fields it draws from, as a comparable struct, which is
// compared with the one recorded when it last drew - without
// allocating, unlike a list of values. Set Snap once, when the
// component is made or first laid out.
type Watch[T comparable] struct {
	Snap func() T
	// cur is the snapshot Changed took; last the one recorded.
	cur, last T
	set       bool
}

// Changed reports whether Snap differs from the recorded snapshot.
func (w *Watch[T]) Changed() bool {
	w.cur = w.Snap()
	return !w.set || !equalSafe(w.cur, w.last)
}

// Commit records the snapshot the last Changed took: the fields as
// they were when the component started drawing.
func (w *Watch[T]) Commit() {
	w.last, w.set = w.cur, true
}

// equalSafe compares a and b, reporting false where == panics (an
// interface field holding an incomparable value).
func equalSafe[T comparable](a, b T) (eq bool) {
	defer func() {
		if recover() != nil {
			eq = false
		}
	}()
	return a == b
}

// LayoutOwn lays a stateful component out through its own cache: the
// output recorded last time is replayed until something it draws
// changes - its own fields (label, value, styles), which watch tracks,
// deps (the layout context: direction, a parent's stretch), or
// anything a Memo watches (States read, constraints, theme, input
// through its passes, animations). watch is also checked when an
// enclosing cache replays the component without laying it out, so a
// setter changing a field is never missed. watch may be nil for a
// component whose fields don't change once built. Pass the same build
// function every frame, rather than a new method value, so laying the
// component out allocates nothing.
func (c *MemoCache) LayoutOwn(gtx layout.Context, build func() Element, watch Watcher, deps ...any) layout.Dimensions {
	if InSelectionArea() {
		// Text in a selection area registers with it as it's laid out,
		// which a replay would skip.
		return build().Layout(gtx)
	}
	c.watch = watch
	if watch != nil && watch.Changed() {
		c.valid = false
	}
	// Commit (below) records the snapshot Changed just took.
	misses := c.misses
	dims := c.Layout(gtx, build, deps...)
	if c.misses != misses && watch != nil {
		// Rebuilt: record what it drew.
		watch.Commit()
	}
	return dims
}

// InSelectionArea reports whether a SelectionArea is being laid out;
// the text package sets it.
var InSelectionArea = func() bool { return false }
