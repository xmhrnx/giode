// Package lazylist provides a vertical list of rows of any height that
// only builds and lays out the rows in or near its viewport - giode's
// FlatList, ListView.builder or LazyColumn.

package lazylist

import (
	"gioui.org/gesture"
	"gioui.org/layout"
	"gioui.org/unit"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
)

// List is a vertical list of rows of any height:
//
//	list := giode.LazyList(
//		func() int { return len(msgs) },
//		func(i int) any { return msgs[i].ID },
//		func(i int) giode.Element { return messageRow(msgs[i]) },
//	).Anchor(giode.LazyAnchorBottom).Version(func(i int) any { return msgs[i].Revision })
//
// Rows are identified by keys, so the list keeps its place when rows
// are added above (older messages loaded), removed or change height:
// it remembers the first row in view and where its top is, not a
// pixel offset. Each frame it builds and lays out only the rows in the
// viewport and a margin around it; the heights of the others come from
// when they were last laid out, and an estimate for rows never laid
// out, which is good enough for the scrollbar.
//
// Version, when set, gives each row a version - a revision number, a
// content hash, the value itself if comparable: a row whose version is
// unchanged replays its drawing from cache (see elements.Memo), and a
// row offscreen keeps its measured height until its version changes.
// Without it, visible rows are rebuilt every frame and an offscreen
// row's height is trusted until it's laid out again.
//
// count, key, item and version run on the UI goroutine. Per-row state
// (a Pressable, a field) should be looked up by the row's key with
// Keyed, since rows are built only while near the viewport.
type List struct {
	count   func() int
	key     func(i int) any
	item    func(i int) elements.Element
	version func(i int) any
	st      styles.Styles
	gap     unit.Dp
	anchor  Anchor

	// The first row in view and the position of its top relative to
	// the viewport's top (zero or negative).
	anchorKey any
	anchorIdx int
	anchorOff int
	following bool
	started   bool

	pendingKey   any
	pendingAlign Align
	hasPending   bool

	rows    map[any]*row
	width   int
	measure int // average measured height, for rows never laid out
	frame   uint64

	scroll gesture.Scroll
	gest   kit.ScrollGesture
	// atBottom is set when the last layout ended at the last row.
	atBottom  bool
	bar       kit.Scrollbar
	barOffset float32
	contentH  int
	viewportH int
	n         int
	first     int
	last      int

	onReachTop, onReachEnd func()
	wasAtTop, wasAtEnd     bool
	pass                   func(gtx layout.Context) bool
	movedBy                int

	// est caches estimate's sums, which call key and version for every
	// row: they're redone when a row's height or the rows change, the
	// anchor moves, or every estimateFrames frames for edits offscreen.
	est struct {
		valid                      bool
		n, anchorIdx, gap, measure int
		frame                      uint64
		above, below               int
	}
}

// New returns a list of count() rows, row i identified by key(i) and
// built by item(i). The styles argument is optional: Background fills
// the list, ScrollBar customizes the scrollbar, and Gap spaces rows.
func New(count func() int, key func(i int) any, item func(i int) elements.Element, st ...styles.Styles) *List {
	l := &List{count: count, key: key, item: item, rows: map[any]*row{}, first: 0, last: -1}
	if len(st) > 0 {
		l.st = st[0]
		l.gap = unit.Dp(st[0].Gap)
	}
	return l
}

// Version sets the function giving each row's version; see List. It
// returns l for chaining.
func (l *List) Version(fn func(i int) any) *List {
	l.version = fn
	return l
}

// Anchor sets where the list holds on; see AnchorBottom. It returns l
// for chaining.
func (l *List) Anchor(a Anchor) *List {
	l.anchor = a
	return l
}

// OnReachTop sets a handler called when the first row comes into view,
// to load older items: rows inserted above keep the view where it is.
// It returns l for chaining.
func (l *List) OnReachTop(fn func()) *List {
	l.onReachTop = fn
	return l
}

// OnReachEnd sets a handler called when the last row comes into view.
// It returns l for chaining.
func (l *List) OnReachEnd(fn func()) *List {
	l.onReachEnd = fn
	return l
}

// ScrollToKey scrolls the row with key into view, aligned by align, on
// the next frame. It stops following the end.
func (l *List) ScrollToKey(key any, align Align) {
	l.pendingKey, l.pendingAlign, l.hasPending = key, align, true
	l.following = false
}

// ScrollToBottom scrolls to the end and, with AnchorBottom, follows it
// again.
func (l *List) ScrollToBottom() {
	l.following = true
	l.hasPending = false
}

// Following reports whether the list is keeping to its end.
func (l *List) Following() bool {
	return l.following
}

// VisibleRange returns the first and last row at least partly in the
// viewport as of the last frame; last < first when none is.
func (l *List) VisibleRange() (first, last int) {
	return l.first, l.last
}

// Invalidate forgets the measured height of the row with key, for a row
// that changed without a Version.
func (l *List) Invalidate(key any) {
	delete(l.rows, key)
}

// ScrollBy moves the rows by dy pixels, as a wheel or drag would:
// positive dy scrolls down. dx is ignored. It's how automation (the
// MCP server's ui_scroll) scrolls.
func (l *List) ScrollBy(dx, dy float32) {
	l.movedBy += int(dy)
	if dy < 0 {
		l.following = false
	}
}

// ScrollOffset returns the estimated vertical offset in pixels (x is
// always 0).
func (l *List) ScrollOffset() (x, y float32) {
	return 0, l.barOffset
}

// indexOf finds key's row, starting from the last known index.
func (l *List) indexOf(key any, hint int) int {
	if key == nil {
		return -1
	}
	if hint >= 0 && hint < l.n && l.key(hint) == key {
		return hint
	}
	// Rows near the last index first: items inserted or removed nearby.
	for d := 1; d < l.n; d++ {
		if i := hint + d; i >= 0 && i < l.n && l.key(i) == key {
			return i
		}
		if i := hint - d; i >= 0 && i < l.n && l.key(i) == key {
			return i
		}
		if hint-d < 0 && hint+d >= l.n {
			break
		}
	}
	return -1
}

// sweep forgets rows not laid out for a while.
func (l *List) sweep() {
	if l.frame%60 != 0 {
		return
	}
	for k, r := range l.rows {
		switch idle := l.frame - r.used; {
		case idle > 600:
			delete(l.rows, k)
			l.est.valid = false
		case idle > 60 && r.cache != nil:
			// Out of the viewport for a second: drop its drawing - its
			// ops and the line images they hold - and keep its height
			// for the scrollbar and the anchor.
			r.cache = nil
		}
	}
}
