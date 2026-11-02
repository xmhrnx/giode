package elements

import (
	"image"
	"reflect"
	"sync/atomic"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/internal/textrender"
	"github.com/xmhrnx/giode/state"
	"github.com/xmhrnx/giode/styles/properties"
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
//   - the layout constraints, the display density or the active
//     breakpoint (see ActiveBreakpoint);
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
	deps        []any
	reads       *state.Tracker
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

// Layout lays out build's element, or replays the cached result when
// nothing it depends on changed.
func (c *MemoCache) Layout(gtx layout.Context, build func() Element, deps ...any) layout.Dimensions {
	if c.frame == frameNumber && frameNumber != 0 {
		// Already used this frame: its ops are referenced by the
		// earlier use and must not be re-recorded.
		return build().Layout(gtx)
	}
	c.frame = frameNumber

	// A subtree with input handling but no passes is laid out live
	// while the pointer is active over it, so those widgets see their
	// events.
	active := false
	if c.input.Opaque {
		for {
			ev, ok := gtx.Event(pointer.Filter{
				Target: c,
				Kinds:  pointer.Enter | pointer.Leave | pointer.Move | pointer.Press | pointer.Release | pointer.Drag | pointer.Cancel,
			})
			if !ok {
				break
			}
			pe, ok := ev.(pointer.Event)
			if !ok {
				continue
			}
			active = true
			switch pe.Kind {
			case pointer.Leave, pointer.Cancel:
				c.inside = false
			default:
				c.inside = true
			}
		}
	}

	reuse := c.fresh(gtx, deps) && !(c.input.Opaque && (active || c.inside))
	// Deliver input to the widgets inside without laying them out;
	// rebuild only if one changed.
	if reuse && len(c.input.Passes) > 0 && kit.RunPasses(gtx, c.input.Passes) {
		reuse = false
	}

	if reuse {
		c.hits++
		c.keepAlive()
		origin := AbsolutePosition()
		for i, p := range c.probes {
			*p = origin.Add(c.probeRel[i])
			probes = append(probes, p)
		}
		if semantics != nil {
			for _, n := range c.nodes {
				n.Bounds = n.Bounds.Add(origin)
				*semantics = append(*semantics, n)
			}
		}
		c.call.Add(gtx.Ops)
	} else {
		c.misses++
		c.record(gtx, build, deps)
	}
	c.dirty = false

	// Enclosing caches replaying this one run its passes and keep its
	// state alive through it.
	if c.input.Opaque {
		kit.OpaqueInput()
	} else {
		// Registered even without passes of its own: the pass also
		// notices state the subtree reads changing.
		if c.passFn == nil {
			c.passFn = c.pass
		}
		kit.HandleInput(c.passFn)
	}
	retainKeyed(c.keepAlive)

	if c.input.Opaque {
		// Observe the pointer over the subtree without blocking
		// anything beneath it.
		ps := pointer.PassOp{}.Push(gtx.Ops)
		cl := clip.Rect{Max: c.dims.Size}.Push(gtx.Ops)
		event.Op(gtx.Ops, c)
		cl.Pop()
		ps.Pop()
	}
	return c.dims
}

// fresh reports whether the recorded output is still valid for gtx and
// deps, input aside.
func (c *MemoCache) fresh(gtx layout.Context, deps []any) bool {
	return c.valid && !c.live && !c.portal && !c.dirty &&
		c.constraints == gtx.Constraints &&
		c.metric == gtx.Metric &&
		c.theme == properties.ThemeVersion() &&
		c.epoch == cacheEpoch.Load() &&
		c.color == CurrentColor() &&
		c.breakpoint == ActiveBreakpoint() &&
		c.selection == selectionVersion() &&
		depsEqual(c.deps, deps) &&
		!c.reads.Changed()
}

// pass is the cache's input pass, run by an enclosing cache replaying
// it: it delivers input to the widgets inside, and reports a change
// when they changed or the cache must rebuild anyway.
func (c *MemoCache) pass(gtx layout.Context) bool {
	if !c.valid || c.live || c.reads.Changed() || c.epoch != cacheEpoch.Load() || c.theme != properties.ThemeVersion() {
		c.dirty = true
		return true
	}
	if kit.RunPasses(gtx, c.input.Passes) {
		c.dirty = true
		return true
	}
	return false
}

// keepAlive refreshes the keyed state the subtree looked up when it was
// recorded.
func (c *MemoCache) keepAlive() {
	c.alive = frameNumber
	for _, f := range c.retain {
		f()
	}
}

// record rebuilds and lays out the subtree into the cache's own ops.
func (c *MemoCache) record(gtx layout.Context, build func() Element, deps []any) {
	c.ops.Reset()
	cgtx := gtx
	cgtx.Ops = &c.ops

	c.live = false
	popLive := kit.PushLiveScope(&c.live)
	c.input = kit.InputScope{Passes: c.input.Passes[:0]}
	popInput := kit.PushInputScope(&c.input)
	c.retain = c.retain[:0]
	popRetain := pushRetainScope(&c.retain)
	reads := state.Track()
	portals, backs, probeStart := len(portalQueue), len(backStack), len(probes)
	semStart := 0
	if semantics != nil {
		semStart = len(*semantics)
	}

	macro := op.Record(&c.ops)
	c.dims = build().Layout(cgtx)
	c.call = macro.Stop()

	reads.Stop()
	popRetain()
	popInput()
	popLive()
	c.alive = frameNumber

	c.reads = reads
	c.portal = len(portalQueue) > portals || len(backStack) > backs
	origin := AbsolutePosition()
	c.probes, c.probeRel = c.probes[:0], c.probeRel[:0]
	for _, p := range probes[probeStart:] {
		c.probes = append(c.probes, p)
		c.probeRel = append(c.probeRel, p.Sub(origin))
	}
	c.nodes = c.nodes[:0]
	if semantics != nil {
		for _, n := range (*semantics)[semStart:] {
			n.Bounds = n.Bounds.Sub(origin)
			c.nodes = append(c.nodes, n)
		}
	}
	c.constraints = gtx.Constraints
	c.metric = gtx.Metric
	c.theme = properties.ThemeVersion()
	c.epoch = cacheEpoch.Load()
	c.color = CurrentColor()
	c.breakpoint = ActiveBreakpoint()
	c.selection = selectionVersion()
	c.deps = append(c.deps[:0], deps...)
	c.valid = true

	c.call.Add(gtx.Ops)
}

func depsEqual(a, b []any) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		x, y := a[i], b[i]
		if x == nil || y == nil {
			if x != y {
				return false
			}
			continue
		}
		tx, ty := reflect.TypeOf(x), reflect.TypeOf(y)
		if tx != ty {
			return false
		}
		if tx.Comparable() {
			if x != y {
				return false
			}
			continue
		}
		if !reflect.DeepEqual(x, y) {
			return false
		}
	}
	return true
}

// cacheEpoch invalidates every cache at once when advanced.
var cacheEpoch atomic.Uint64

// InvalidateCaches forces every MemoCache and Memo to rebuild on the
// next frame. Call it after changing something caches can't observe -
// a plain variable not passed as a dep, a component driven
// programmatically (Button.Click, Input.SetText) - when the change
// must show up inside a cached subtree. It is safe to call from any
// goroutine; it doesn't request a frame by itself.
func InvalidateCaches() {
	cacheEpoch.Add(1)
}

// --- Keyed memos -------------------------------------------------------------

// frameNumber counts frames; BeginFrame advances it. Keyed memos not
// used during a frame are dropped at the next EndFrame.
var (
	frameNumber uint64
	memos       = map[any]*MemoCache{}
)

// BeginFrame starts a new frame. The app calls it before laying out
// the view; component authors don't need to.
func BeginFrame() {
	frameNumber++
	kit.NextFrame()
	textrender.Tick()
	clear(probes)
	probes = probes[:0]
}

// EndFrame finishes a frame, releasing keyed memos (see Memo) that
// weren't laid out during it. The app calls it after every frame.
func EndFrame() {
	for k, c := range memos {
		if c.alive != frameNumber {
			delete(memos, k)
		}
	}
	sweepKeyed()
}

// Memo returns an element that caches build's output under key and
// rebuilds it only when deps, the States it reads, its constraints or
// the theme change, or when a widget inside asks for it - see
// MemoCache for the full rules. Use it inside the view function to
// skip rebuilding parts of the UI that didn't change:
//
//	giode.Memo("sidebar", func() giode.Element {
//		return sidebar(user.Name)
//	}, user.Name)
//
// key identifies the cache across frames and must be unique among the
// memos laid out in a frame (any comparable value: a string, an ID, a
// pointer). A key not laid out for a frame is released.
func Memo(key any, build func() Element, deps ...any) Element {
	return &memoEl{key: key, build: build, deps: deps}
}

type memoEl struct {
	key   any
	build func() Element
	deps  []any
}

func (m *memoEl) Layout(gtx layout.Context) layout.Dimensions {
	c, ok := memos[m.key]
	if !ok {
		c = new(MemoCache)
		memos[m.key] = c
	}
	return c.Layout(gtx, m.build, m.deps...)
}
