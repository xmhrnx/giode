package base

import (
	"image"

	"gioui.org/io/system"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/state"
)

// MemoCache caches the painted output of a subtree across frames, so
// the subtree is only rebuilt and laid out again when something it
// depends on changed. Everything else replays the ops recorded last
// time, which costs next to nothing regardless of how large the
// subtree is.
//
// A cached subtree is rebuilt when any of these changed since it was
// recorded:
//
//   - the deps passed to Layout (compared with ==, or
//     reflect.DeepEqual for values that aren't comparable);
//   - any State the subtree read while being built (tracked
//     automatically);
//   - the layout constraints, the display density, the active
//     breakpoint (see ActiveBreakpoint), touch input (see TouchInput)
//     or the locale;
//   - the theme (SetMode/SetPalette);
//   - a widget inside asked for it: an animation, or a change that
//     came from outside input (see kit.MarkLive);
//   - input changed a widget inside: on a replayed frame the cache runs
//     its widgets' input passes (see kit.HandleInput) - so hovering,
//     tapping or scrolling over it still reaches them - and lays the
//     subtree out again only when one reports a visible change, like a
//     hover highlight or a press. A subtree with input handling that
//     registers no pass (see kit.OpaqueInput) is instead laid out on
//     every frame the pointer is active over it.
//
// Values read from plain variables (not States) must be passed as
// deps, or the cache won't notice them change.
//
// Create a MemoCache once and lay it out every frame, or use Memo for
// a cache keyed by an identifier inside the view function. A cache
// laid out twice in the same frame lays the second use out uncached.
type MemoCache struct {
	ops   op.Ops
	call  op.CallOp
	dims  layout.Dimensions
	valid bool

	constraints layout.Constraints
	metric      unit.Metric
	theme       uint64
	epoch       uint64
	color       string
	breakpoint  string
	// touch and locale are the touch-input mode (minimum touch targets)
	// and the locale (reading direction) the subtree was laid out with.
	touch  bool
	locale system.Locale
	deps   []any
	// watch, for a component caching its own output (LayoutOwn),
	// returns what it draws from its own fields; watched is the value
	// recorded. It's compared in the cache's pass too, so a setter
	// called on a component replayed by an enclosing cache still
	// rebuilds it.
	watch   func() []any
	watched []any
	reads   *state.Tracker
	// selection is the version of the enclosing SelectionArea's
	// selection, which the subtree's text draws.
	selection uint64

	// live is set (through kit.MarkLive) by widgets that changed
	// while the subtree was laid out.
	live bool
	// input holds the input passes of the subtree's widgets, run on
	// replayed frames.
	input kit.InputScope
	// retain keeps the subtree's keyed state alive on replayed frames,
	// when its Keyed lookups don't run.
	retain []func()
	// dirty is set when an enclosing cache ran this one's passes and
	// one reported a change, so its next layout must rebuild.
	dirty bool
	// passFn is the cache's own pass, registered with enclosing caches.
	passFn func(gtx layout.Context) bool
	// inside tracks whether the pointer is over the subtree.
	inside bool
	// portal is set when the subtree registered portals or Back
	// handlers, which can't be replayed from cache.
	portal bool
	// nodes are the semantic nodes recorded with the subtree, relative
	// to its origin.
	nodes []SemanticNode
	// probes are the positions tracked inside the subtree (see
	// TrackPosition) and their offsets from its origin, restored when
	// it's replayed.
	probes   []*image.Point
	probeRel []image.Point

	frame uint64
	// alive is the last frame the cache was laid out or replayed by an
	// enclosing cache; keyed memos not alive in a frame are dropped.
	alive uint64

	// Stats, for tests and diagnostics.
	hits, misses uint64
}

// NewMemoCache returns an empty cache.
func NewMemoCache() *MemoCache {
	return new(MemoCache)
}

// Invalidate drops the cached output, forcing a rebuild on the next
// layout.
func (c *MemoCache) Invalidate() {
	c.valid = false
}

// Stats returns how many layouts were served from cache (hits) and how
// many rebuilt the subtree (misses).
func (c *MemoCache) Stats() (hits, misses uint64) {
	return c.hits, c.misses
}
