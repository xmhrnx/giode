package elements

import (
	"gioui.org/io/system"
	"image"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/components/kit"
)

// portalEntry is one piece of content queued to render above
// everything else, regardless of how deeply nested its caller was.
type portalEntry struct {
	// anchor is the window-absolute point the content is placed near;
	// containers that place their children after laying them out move
	// it (see ChildTracker). It's clamped to the window when drained.
	anchor image.Point
	// size is the content's measured size, and win the window it's
	// clamped to.
	size, win image.Point
	build     func() Element
	// overlay marks a full-window overlay (RegisterOverlay): laid out
	// at the window's origin, never clamped or moved.
	overlay bool
	// fixed keeps the anchor where it was registered: a window position
	// containers don't move (RegisterPortalAt).
	fixed bool
	// alignEnd lines an around entry's right edge up with its
	// trigger's, for right-to-left layouts.
	alignEnd bool
	// around, when set, is the trigger's rectangle relative to anchor
	// (RegisterPortalAround): the content goes gap pixels below or
	// above it, per place, rather than at the anchor.
	around    image.Rectangle
	hasAround bool
	gap       int
	place     Placement
}

// Placement says on which side of its trigger a portal goes.
type Placement int

const (
	// PlaceAuto puts the portal below its trigger, or above it when
	// it wouldn't fit below.
	PlaceAuto Placement = iota
	// PlaceBelow and PlaceAbove keep it on one side.
	PlaceBelow
	PlaceAbove
)

var portalQueue []portalEntry

// windowSize is the window's root constraints for the current frame,
// recorded by SetWindowSize before the view is laid out.
// RegisterPortal measures and positions against it instead of its
// caller's own ambient constraints, which shrink the deeper a portal
// trigger sits in the tree (Tooltip, Dropdown and Menu's nested
// submenus are typically nested arbitrarily deep, unlike Menu's own
// right-click panel, which happens to be laid out directly off the
// window) - using the wrong, smaller bound wouldn't crash (the portal
// still paints in the right place, at DrainPortals' own root
// position), but would measure/clamp against a box smaller than the
// real window, positioning it more conservatively than necessary.
var windowSize image.Point

// SetWindowSize records the window's size for the current frame. The
// app calls this once, before laying out the view; component authors
// calling RegisterPortal don't need to call it themselves.
func SetWindowSize(size image.Point) {
	windowSize = size
}

// RegisterPortal queues build to be laid out and painted for real
// after the rest of the view - the mechanism behind Tooltip, nested
// Menu submenus and AutoComplete's suggestion panel, and the building
// block for any component that needs to render on top of everything
// else no matter how deeply it's nested in the tree.
//
// anchor is a window-absolute point (typically AbsolutePosition(),
// optionally offset - e.g. by the trigger element's own height) that
// the content is positioned near; it's first laid out once on a
// disabled scratch context, bounded by the window (see
// SetWindowSize) rather than gtx's own ambient constraints, to
// measure its size (the same technique Menu's own panel-width
// measurement uses) so the position can be clamped to stay inside the
// window, flipping above the anchor if it would overflow the bottom
// edge - see kit.ClampPos. build is then laid out for real, exactly
// once, when the queue is drained.
//
// The anchor follows the caller when an enclosing Row, Stack or Scroll
// places it after the fact (see ChildTracker), so a portal opened from
// one of several children of a row lands below that child, and the
// clamping happens once its final position is known.
func RegisterPortal(gtx layout.Context, anchor image.Point, build func() Element) {
	win := windowSize
	if win == (image.Point{}) {
		win = gtx.Constraints.Max
	}

	mgtx := gtx
	mgtx.Constraints = layout.Constraints{Max: win}
	dims := Measure(mgtx, build())

	portalQueue = append(portalQueue, portalEntry{anchor: anchor, size: dims.Size, win: win, build: build})
}

// RegisterPortalAt queues build like RegisterPortal at anchor, a
// window position that stays put: containers that place their children
// after laying them out don't move it, unlike RegisterPortal's anchor.
// For content opened at a known window position - a context menu at the
// pointer. With alignEnd, anchor is the content's top-right corner.
func RegisterPortalAt(gtx layout.Context, anchor image.Point, alignEnd bool, build func() Element) {
	RegisterPortal(gtx, anchor, build)
	e := &portalQueue[len(portalQueue)-1]
	e.fixed = true
	if alignEnd {
		e.anchor.X -= e.size.X
	}
}

// RegisterPortalAround queues build like RegisterPortal, placed next to
// trigger, a window-absolute rectangle (AbsolutePosition() and the
// trigger's size): gap pixels below it or above it, per place. Unlike
// RegisterPortal's flip, which only knows a point, flipping around the
// rectangle never covers the trigger - a tooltip on a control bar at
// the bottom of the window goes above its button.
//
// Its start edges line up with the trigger's: left edges, or right
// edges when gtx reads right to left.
func RegisterPortalAround(gtx layout.Context, trigger image.Rectangle, gap int, place Placement, build func() Element) {
	RegisterPortal(gtx, trigger.Min, build)
	e := &portalQueue[len(portalQueue)-1]
	e.around = trigger.Sub(trigger.Min)
	e.hasAround, e.gap, e.place = true, gap, place
	e.alignEnd = gtx.Locale.Direction == system.RTL
}

// RegisterPortalEnd queues build like RegisterPortal with anchor as the
// content's top-right corner rather than its top-left: a menu opened
// at a point in a right-to-left layout extends to the left of it.
func RegisterPortalEnd(gtx layout.Context, anchor image.Point, build func() Element) {
	RegisterPortal(gtx, anchor, build)
	e := &portalQueue[len(portalQueue)-1]
	e.anchor.X -= e.size.X
}

// RegisterOverlay queues build to be laid out over the whole window,
// above the rest of the view, like a portal anchored at the window's
// origin that's neither measured nor clamped: build gets the window's
// size as its maximum constraints and positions its own content - a
// bottom sheet, a full-screen picker. Overlays and portals paint in
// registration order.
func RegisterOverlay(build func() Element) {
	portalQueue = append(portalQueue, portalEntry{build: build, overlay: true})
}

// WindowSize returns the window's size for the current frame, in
// pixels; zero outside a frame.
func WindowSize() image.Point {
	return windowSize
}

// pos clamps the entry's anchor to the window.
func (e portalEntry) pos() image.Point {
	if e.overlay {
		return image.Point{}
	}
	if e.hasAround {
		return e.aroundPos()
	}
	winF := f32.Pt(float32(e.win.X), float32(e.win.Y))
	size := f32.Pt(float32(e.size.X), float32(e.size.Y))
	return kit.ClampPos(winF, size, f32.Pt(float32(e.anchor.X), float32(e.anchor.Y)), kit.Margin).Round()
}

// DrainPortals lays out and paints every portal registered so far
// this frame, in registration order (later registrations paint on
// top of earlier ones), then clears the queue. The app calls this
// once per frame, after the rest of the view.
//
// AbsolutePosition/PushOffset are kept in sync with the paint offset
// here (not just op.Offset) so that content built inside a portal -
// nested Menu submenus register a further portal of their own,
// anchored off their own row's position - reports its true screen
// position rather than whatever was accumulated before the portal
// that contains it was originally triggered.
//
// Portals registered while draining (a submenu opened by its parent
// menu's row) are drained too, in the same frame.
func DrainPortals(gtx layout.Context) {
	for len(portalQueue) > 0 {
		entries := portalQueue
		portalQueue = nil
		for _, e := range entries {
			pos := e.pos()
			off := op.Offset(pos).Push(gtx.Ops)
			pop := PushOffset(pos)
			e.build().Layout(gtx)
			pop()
			off.Pop()
		}
	}
}

// aroundPos places a RegisterPortalAround entry beside its trigger.
func (e portalEntry) aroundPos() image.Point {
	m := int(kit.Margin)
	r := e.around.Add(e.anchor)
	below := r.Max.Y + e.gap
	above := r.Min.Y - e.gap - e.size.Y
	fitsBelow := below+e.size.Y+m <= e.win.Y
	fitsAbove := above >= m
	y := below
	switch e.place {
	case PlaceAbove:
		y = above
	case PlaceAuto:
		if !fitsBelow && (fitsAbove || r.Min.Y > e.win.Y-r.Max.Y) {
			y = above
		}
	}
	// Keep it in the window: along the side it's on, it may then
	// overlap the trigger, but only when the window is too small for
	// either side.
	y = max(m, min(y, e.win.Y-e.size.Y-m))
	x := r.Min.X
	if e.alignEnd {
		x = r.Max.X - e.size.X
	}
	x = max(m, min(x, e.win.X-e.size.X-m))
	return image.Pt(x, y)
}
