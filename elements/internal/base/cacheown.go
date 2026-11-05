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

// LayoutOwn lays a stateful component out through its own cache: the
// output recorded last time is replayed until something it draws
// changes - watch's values (the component's own fields: label, value,
// styles), deps (the layout context: direction, a parent's stretch),
// or anything a Memo watches (States read, constraints, theme, input
// through its passes, animations). watch is also checked when an
// enclosing cache replays the component without laying it out, so a
// setter changing a field is never missed.
func (c *MemoCache) LayoutOwn(gtx layout.Context, build func() Element, watch func() []any, deps ...any) layout.Dimensions {
	if InSelectionArea() {
		// Text in a selection area registers with it as it's laid out,
		// which a replay would skip.
		return build().Layout(gtx)
	}
	c.watch = watch
	w := watch()
	if !depsEqual(c.watched, w) {
		c.valid = false
	}
	misses := c.misses
	dims := c.Layout(gtx, build, deps...)
	if c.misses != misses {
		// Rebuilt: what it drew is w.
		c.watched = append(c.watched[:0], w...)
	}
	return dims
}

// InSelectionArea reports whether a SelectionArea is being laid out;
// the text package sets it.
var InSelectionArea = func() bool { return false }
