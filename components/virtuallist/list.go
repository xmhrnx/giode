// Package virtuallist provides a vertical list that only builds the
// rows in or near its viewport, so it handles any number of items at a
// constant cost per frame.

package virtuallist

import (
	"gioui.org/gesture"
	"gioui.org/io/key"
	"gioui.org/unit"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
)

// List is a vertical list of equal-height rows that only builds the
// rows in or near the viewport:
//
//	list := giode.VirtualList(func() int { return len(model.rows) }, 32,
//		func(i int) giode.Element { return row(model.rows[i]) })
//
// count and item run on the UI goroutine every frame; item only for
// visible rows (plus a few above and below, see Overscan). The item
// height is in dp, like every other size. The list fills the space it
// is given and scrolls with the wheel, touch, its scrollbar and the
// keyboard: once focused (click it, or Tab to it) Up/Down,
// PageUp/PageDown and Home/End move the focused index, keeping it in
// view.
//
// Rows get exactly the item height and the list's width. Per-row
// state (a Pressable, a field) should be looked up by the item's key
// with Keyed, since rows are built only while visible.
type List struct {
	count    func() int
	itemH    unit.Dp
	item     func(i int) elements.Element
	st       styles.Styles
	overscan int

	offset    float32 // px
	scroll    gesture.Scroll
	gest      kit.ScrollGesture
	bar       kit.Scrollbar
	viewportH int
	itemPx    int
	n         int

	first, last int // visible range, inclusive; last < first when empty

	focused    bool
	focusIdx   int
	onFocus    func(i int)
	onActivate func(i int)

	wantFocus    bool
	pendingIdx   int
	pendingAlign Align
	hasPending   bool

	onReachEnd func()
	wasAtEnd   bool

	noClickRow bool
	// pointerFocus marks focus requested by a press, so gaining it
	// doesn't also focus the first row the way Tab does.
	pointerFocus bool
	extraKeys    []key.Name
	onKey        func(e key.Event, focused int)
}

// New returns a list of count() rows, each itemHeight dp tall and built
// by item(i). The styles argument is optional: Background fills the
// list, ScrollBar customizes the scrollbar, and BorderColor is the
// focused row's highlight (default a translucent primary color).
func New(count func() int, itemHeight int, item func(i int) elements.Element, st ...styles.Styles) *List {
	l := &List{count: count, itemH: unit.Dp(itemHeight), item: item, overscan: defaultOverscan, focusIdx: -1, last: -1}
	if len(st) > 0 {
		l.st = st[0]
	}
	return l
}

// Overscan sets how many rows beyond the viewport are built on each
// side (default 3), so fast scrolling doesn't reveal empty space. It
// returns l for chaining.
func (l *List) Overscan(n int) *List {
	l.overscan = max(0, n)
	return l
}

// OnReachEnd sets a handler run once when the last row scrolls into
// view - edge-triggered, for loading more rows. It returns l for
// chaining.
func (l *List) OnReachEnd(fn func()) *List {
	l.onReachEnd = fn
	return l
}

// OnFocusChange sets a handler run when keyboard navigation (or
// SetFocusedIndex) moves the focused index. It returns l for chaining.
func (l *List) OnFocusChange(fn func(i int)) *List {
	l.onFocus = fn
	return l
}

// OnActivate sets a handler run when Enter is pressed on the focused
// row. It returns l for chaining.
func (l *List) OnActivate(fn func(i int)) *List {
	l.onActivate = fn
	return l
}

// OnKey handles additional keys while the list has focus: fn runs on
// presses of names with the focused row. Components built on the list
// (a tree's Left/Right) use it. It returns l for chaining.
func (l *List) OnKey(fn func(e key.Event, focused int), names ...key.Name) *List {
	l.onKey, l.extraKeys = fn, names
	return l
}

// ClickFocusesRow controls whether a press on a row moves the focused
// index to it (the default). Components whose rows handle their own
// clicks turn it off; the press still focuses the list. It returns l
// for chaining.
func (l *List) ClickFocusesRow(v bool) *List {
	l.noClickRow = !v
	return l
}

// Focus requests keyboard focus for the list on the next frame.
func (l *List) Focus() {
	l.wantFocus = true
}

// ScrollToIndex scrolls so item i is shown according to align. It
// takes effect on the next frame, when the viewport size is known.
func (l *List) ScrollToIndex(i int, align Align) {
	l.pendingIdx, l.pendingAlign, l.hasPending = i, align, true
}

// VisibleRange returns the first and last row at least partly in the
// viewport as of the last frame; last < first when no row is.
func (l *List) VisibleRange() (first, last int) {
	return l.first, l.last
}

// FocusedIndex returns the focused row, or -1.
func (l *List) FocusedIndex() int {
	return l.focusIdx
}

// SetFocusedIndex focuses row i and scrolls it into view; -1 clears
// the focus. An index past the end is clamped at the next layout,
// once the row count is current.
func (l *List) SetFocusedIndex(i int) {
	if i < 0 {
		l.focusIdx = -1
		return
	}
	if i != l.focusIdx {
		l.focusIdx = i
		if l.onFocus != nil {
			l.onFocus(i)
		}
	}
	l.ScrollToIndex(i, AlignNearest)
}

// HasFocus reports whether the list has keyboard focus.
func (l *List) HasFocus() bool {
	return l.focused
}

// OffsetDp returns the scroll offset in dp.
func (l *List) OffsetDp() float32 {
	return elements.PxToDp(int(l.offset))
}

// ScrollBy moves the rows by dy pixels, as a wheel or drag would:
// positive dy scrolls down. dx is ignored. It's how automation (the
// MCP server's ui_scroll) scrolls.
func (l *List) ScrollBy(dx, dy float32) {
	l.offset = l.clamp(l.offset + dy)
}

// ScrollOffset returns the vertical offset in pixels (x is always 0).
func (l *List) ScrollOffset() (x, y float32) {
	return 0, l.offset
}
