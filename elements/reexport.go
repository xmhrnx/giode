// Package elements provides the stateless building blocks of Giode:
// box, flex, text and raw. They are cheap to build and are meant to be
// created inside the view function on every frame. Stateful components
// (see components/) hold the persistent state interactive widgets
// need.
//
// The building blocks live in per-element subpackages (box, text, ...)
// on top of the shared runtime in elements/internal/base; this package
// re-exports them so callers keep using elements.Box, elements.Text
// and so on.
package elements

import (
	base "github.com/xmhrnx/giode/elements/internal/base"
)

// Element, semantics and the shared runtime, re-exported from
// elements/internal/base.
type (
	// NamedKey is a key, with its modifiers, that an OnKey handler asks
	// Gio for by name (Tab).
	NamedKey = base.NamedKey
	// Contextual gives an element a context menu (Context); every
	// element and component embeds it.
	Contextual = base.Contextual
	// ContextMenu is a menu an element opens at the pointer when it's
	// right-clicked or long-pressed.
	ContextMenu = base.ContextMenu
	// StickyScope and StickyItem are a scrolling container's sticky
	// elements; see base.StickyItem.
	StickyScope = base.StickyScope
	StickyItem  = base.StickyItem
	// VisibleClip is the part of the window a scrolling container
	// shows; see base.VisibleClip.
	VisibleClip = base.VisibleClip
	// Element is anything that can lay itself out in a frame.
	Element = base.Element
	// SemanticKind classifies a node of the semantic tree.
	SemanticKind = base.SemanticKind
	// SemanticNode describes one meaningful piece of the UI.
	SemanticNode = base.SemanticNode
	// SemanticContainer is a container node being laid out
	// (BeginSemanticGroup).
	SemanticContainer = base.SemanticContainer
	// CheckState, ExpandState and LiveMode are accessibility states of
	// a SemanticNode.
	CheckState  = base.CheckState
	ExpandState = base.ExpandState
	LiveMode    = base.LiveMode
	// Placement says on which side of its trigger a portal goes.
	Placement = base.Placement
	// ChildTracker records what a child registers while it's laid out.
	ChildTracker = base.ChildTracker
	// Describer names an element for layout diagnostics.
	Describer = base.Describer
	// KeyedCache is a typed, self-contained version of Keyed.
	KeyedCache[K comparable, V any] = base.KeyedCache[K, V]
	// Watch tracks the fields a component caching its own output draws
	// from; see MemoCache.LayoutOwn.
	Watch[T comparable] = base.Watch[T]
	// MemoCache caches the painted output of a subtree across frames.
	MemoCache = base.MemoCache
	// CacheSafe is implemented by elements whose output a cache may
	// keep: it depends only on their own fields and what caches watch.
	CacheSafe = base.CacheSafe
	// OutOfFlow is implemented by elements that paint over the window
	// and take no space where they're placed: Dialog, Sheet and
	// Drawer.
	OutOfFlow = base.OutOfFlow
)

// Semantic kinds.
const (
	SemanticText         = base.SemanticText
	SemanticButton       = base.SemanticButton
	SemanticInput        = base.SemanticInput
	SemanticCheckbox     = base.SemanticCheckbox
	SemanticSlider       = base.SemanticSlider
	SemanticSelect       = base.SemanticSelect
	SemanticTab          = base.SemanticTab
	SemanticMenuItem     = base.SemanticMenuItem
	SemanticLink         = base.SemanticLink
	SemanticImage        = base.SemanticImage
	SemanticScroll       = base.SemanticScroll
	SemanticCustom       = base.SemanticCustom
	SemanticHeading      = base.SemanticHeading
	SemanticSwitch       = base.SemanticSwitch
	SemanticRadio        = base.SemanticRadio
	SemanticProgress     = base.SemanticProgress
	SemanticGroup        = base.SemanticGroup
	SemanticList         = base.SemanticList
	SemanticListItem     = base.SemanticListItem
	SemanticTable        = base.SemanticTable
	SemanticRow          = base.SemanticRow
	SemanticCell         = base.SemanticCell
	SemanticColumnHeader = base.SemanticColumnHeader
	SemanticDialog       = base.SemanticDialog
	SemanticAlert        = base.SemanticAlert
	SemanticStatus       = base.SemanticStatus
	SemanticMenu         = base.SemanticMenu
	SemanticMenuBar      = base.SemanticMenuBar
	SemanticTabList      = base.SemanticTabList
	SemanticTabPanel     = base.SemanticTabPanel
	SemanticTree         = base.SemanticTree
	SemanticTreeItem     = base.SemanticTreeItem
	SemanticComboBox     = base.SemanticComboBox
	SemanticToolbar      = base.SemanticToolbar
	SemanticRegion       = base.SemanticRegion
)

// Accessibility states (SemanticNode.Checked, Expanded, Live).
const (
	NotCheckable  = base.NotCheckable
	Unchecked     = base.Unchecked
	Checked       = base.Checked
	Mixed         = base.Mixed
	NotExpandable = base.NotExpandable
	Collapsed     = base.Collapsed
	Expanded      = base.Expanded
	LiveOff       = base.LiveOff
	LivePolite    = base.LivePolite
	LiveAssertive = base.LiveAssertive
)

// Portal placements.
const (
	PlaceAuto  = base.PlaceAuto
	PlaceBelow = base.PlaceBelow
	PlaceAbove = base.PlaceAbove
)

// Keyed state.
const (
	// DefaultRetainFrames is how many frames an entry may go
	// unrequested before it's dropped.
	DefaultRetainFrames = base.DefaultRetainFrames
	// CompactWidth is the window width, in dp, under which Compact
	// reports true outside any Responsive.
	CompactWidth = base.CompactWidth
)

// Frames and layout helpers.
var (
	AfterLayout    = base.AfterLayout
	RunAfterLayout = base.RunAfterLayout
	SetMetric      = base.SetMetric
	DisplayScale   = base.DisplayScale
	DpToPx         = base.DpToPx
	PxToDp         = base.PxToDp
	PushColor      = base.PushColor
	CurrentColor   = base.CurrentColor
	ResolveColor   = base.ResolveColor
	// PushStickyScope collects the sticky elements a scrolling
	// container's content lays out; see base.StickyItem.
	PushStickyScope = base.PushStickyScope
	// PaintSticky paints a scrolling container's sticky elements over
	// its content, held at its edges.
	PaintSticky = base.PaintSticky
	// PushVisibleClip makes a scrolling container's viewport the
	// visible area of its content (drop targets use it).
	PushVisibleClip = base.PushVisibleClip
	Measure         = base.Measure
)

// Absolute position and tracking.
var (
	AbsolutePosition = base.AbsolutePosition
	PushOffset       = base.PushOffset
	TrackPosition    = base.TrackPosition
	TrackChild       = base.TrackChild
	ReorderTracked   = base.ReorderTracked
)

// Semantic tree.
var (
	BeginSemantics     = base.BeginSemantics
	EndSemantics       = base.EndSemantics
	SemanticsEnabled   = base.SemanticsEnabled
	RecordSemantic     = base.RecordSemantic
	PushSemanticScroll = base.PushSemanticScroll
	// BeginSemanticGroup starts a container node (a list, a table, a
	// dialog) whose children are the nodes laid out until End.
	BeginSemanticGroup = base.BeginSemanticGroup
	// Accessible lays an element out and adjusts the nodes it records
	// (a text as a heading); AccessibleGroup makes it a container node.
	Accessible      = base.Accessible
	AccessibleGroup = base.AccessibleGroup
	// SetSemanticsWanted records the semantic tree every frame, for
	// accessibility bridges; SemanticsWanted reports it.
	SetSemanticsWanted = base.SetSemanticsWanted
	SemanticsWanted    = base.SemanticsWanted
)

// Portals and overlays.
var (
	SetWindowSize        = base.SetWindowSize
	RegisterPortal       = base.RegisterPortal
	RegisterPortalAt     = base.RegisterPortalAt
	RegisterPortalAround = base.RegisterPortalAround
	RegisterPortalEnd    = base.RegisterPortalEnd
	RegisterOverlay      = base.RegisterOverlay
	WindowSize           = base.WindowSize
	DrainPortals         = base.DrainPortals
)

// Back and keyboard input.
var (
	HandleBack    = base.HandleBack
	HandleDismiss = base.HandleDismiss
	DispatchBack  = base.DispatchBack
	OnKey         = base.OnKey
	// DrainContextMenus lays out the context menus elements opened;
	// the app calls it every frame.
	DrainContextMenus = base.DrainContextMenus
	// ShowMenu lays a menu out every frame while it's open, wherever
	// it was opened from.
	ShowMenu = base.ShowMenu
	// FrameNumber is the number of the frame being laid out.
	FrameNumber = base.FrameNumber
	// ContextClaimed reports whether an element opened its context
	// menu for a press.
	ContextClaimed = base.ContextClaimed
	DispatchKeys   = base.DispatchKeys
	// MarkTyping records that a focused text field takes typing this
	// frame, so typed keys don't run OnKey shortcuts; see base.
	MarkTyping = base.MarkTyping
	// WatchKeys registers, for this frame, a watcher of every key the
	// app gets; see base.WatchKeys.
	WatchKeys = base.WatchKeys
	// SeenKey reports a key a component took to the watchers;
	// ForwardKey passes one it didn't use on to the OnKey handlers.
	SeenKey    = base.SeenKey
	ForwardKey = base.ForwardKey
)

// Touch input and diagnostics.
var (
	TouchInput     = base.TouchInput
	SetTouchInput  = base.SetTouchInput
	MinTouchTarget = base.MinTouchTarget
	SetDebugLogger = base.SetDebugLogger
)

// Responsive breakpoints.
var (
	ActiveBreakpoint = base.ActiveBreakpoint
	Compact          = base.Compact
)

// Keyed and memo caches.
var (
	ForgetKeyed       = base.ForgetKeyed
	SetKeyedRetention = base.SetKeyedRetention
	InvalidateCaches  = base.InvalidateCaches
	BeginFrame        = base.BeginFrame
	EndFrame          = base.EndFrame
	Memo              = base.Memo
	NewMemoCache      = base.NewMemoCache
	IsCacheSafe       = base.IsCacheSafe
)

// Keyed returns the state stored under (scope, key), creating it with
// init the first time it's asked for.
func Keyed[T any](scope, key any, init func() T) T {
	return base.Keyed(scope, key, init)
}

// NewKeyedCache returns an empty cache that keeps entries for
// DefaultRetainFrames unrequested frames.
func NewKeyedCache[K comparable, V any]() *KeyedCache[K, V] {
	return base.NewKeyedCache[K, V]()
}

// InSelectionArea reports whether a SelectionArea is being laid out
// around the caller: its text is part of that area's selection.
func InSelectionArea() bool { return base.InSelectionArea() }
