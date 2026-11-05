package base

import "image"

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
	// registered is where anchor was when the entry was queued: an
	// around entry that containers moved since is measured again, as
	// its content may size itself from the trigger's final position
	// (a dropdown's list fills the room beside its field).
	registered image.Point
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

// WindowSize returns the window's size for the current frame, in
// pixels; zero outside a frame.
func WindowSize() image.Point {
	return windowSize
}
