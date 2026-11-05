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
	// Element is anything that can lay itself out in a frame.
	Element = base.Element
	// SemanticKind classifies a node of the semantic tree.
	SemanticKind = base.SemanticKind
	// SemanticNode describes one meaningful piece of the UI.
	SemanticNode = base.SemanticNode
	// Placement says on which side of its trigger a portal goes.
	Placement = base.Placement
	// ChildTracker records what a child registers while it's laid out.
	ChildTracker = base.ChildTracker
	// Describer names an element for layout diagnostics.
	Describer = base.Describer
	// KeyedCache is a typed, self-contained version of Keyed.
	KeyedCache[K comparable, V any] = base.KeyedCache[K, V]
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
	SemanticText     = base.SemanticText
	SemanticButton   = base.SemanticButton
	SemanticInput    = base.SemanticInput
	SemanticCheckbox = base.SemanticCheckbox
	SemanticSlider   = base.SemanticSlider
	SemanticSelect   = base.SemanticSelect
	SemanticTab      = base.SemanticTab
	SemanticMenuItem = base.SemanticMenuItem
	SemanticLink     = base.SemanticLink
	SemanticImage    = base.SemanticImage
	SemanticScroll   = base.SemanticScroll
	SemanticCustom   = base.SemanticCustom
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
	Measure        = base.Measure
)

// Absolute position and tracking.
var (
	AbsolutePosition = base.AbsolutePosition
	PushOffset       = base.PushOffset
	TrackPosition    = base.TrackPosition
	TrackChild       = base.TrackChild
)

// Semantic tree.
var (
	BeginSemantics     = base.BeginSemantics
	EndSemantics       = base.EndSemantics
	SemanticsEnabled   = base.SemanticsEnabled
	RecordSemantic     = base.RecordSemantic
	PushSemanticScroll = base.PushSemanticScroll
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
	DispatchKeys  = base.DispatchKeys
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
