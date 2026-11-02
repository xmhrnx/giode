// Package lazylist provides a vertical list of rows of any height that
// only builds and lays out the rows in or near its viewport - giode's
// FlatList, ListView.builder or LazyColumn.
package lazylist

import (
	"image"

	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Anchor selects where the list holds on when content changes.
type Anchor int

const (
	// AnchorTop keeps the rows in view where they are as rows are
	// added, removed or resized (the default).
	AnchorTop Anchor = iota
	// AnchorBottom starts at the end and keeps to it as rows are added
	// or the last row grows - a chat - until the user scrolls up; it
	// follows again once they're back at the end.
	AnchorBottom
)

// Align places a row in the viewport for ScrollToKey.
type Align int

const (
	// AlignStart puts the row's top at the top of the viewport.
	AlignStart Align = iota
	// AlignCenter centers the row.
	AlignCenter
	// AlignEnd puts the row's bottom at the bottom of the viewport.
	AlignEnd
)

// overscanDp is how far past the viewport rows are laid out, so a
// scroll doesn't show them appear.
const overscanDp = 300

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
}

// row is what the list remembers of a row.
type row struct {
	height  int
	width   int
	version any
	cache   *elements.MemoCache
	used    uint64
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

// placed is a row laid out this frame.
type placed struct {
	idx   int
	y, h  int
	call  op.CallOp
	track elements.ChildTracker
}

// layoutRow lays out row i at the list's width and returns its drawing
// and height.
func (l *List) layoutRow(gtx layout.Context, i int) placed {
	k := l.key(i)
	r := l.rows[k]
	if r == nil {
		r = &row{}
		l.rows[k] = r
	}
	r.used = l.frame
	rgtx := gtx
	rgtx.Constraints = layout.Constraints{Min: image.Pt(gtx.Constraints.Max.X, 0), Max: image.Pt(gtx.Constraints.Max.X, 1<<24)}
	m := op.Record(gtx.Ops)
	track := elements.TrackChild()
	var dims layout.Dimensions
	if l.version != nil {
		v := l.version(i)
		if r.cache == nil {
			r.cache = elements.NewMemoCache()
		}
		dims = r.cache.Layout(rgtx, func() elements.Element { return l.item(i) }, v)
		r.version = v
	} else if el := l.item(i); el != nil {
		dims = el.Layout(rgtx)
	}
	track.End()
	call := m.Stop()
	r.height, r.width = dims.Size.Y, gtx.Constraints.Max.X
	if l.measure == 0 {
		l.measure = max(1, r.height)
	} else {
		l.measure = (l.measure*7 + r.height) / 8
	}
	return placed{idx: i, h: r.height, call: call, track: track}
}

// heightOf is row i's height as last measured, or the estimate.
func (l *List) heightOf(i int) int {
	if r := l.rows[l.key(i)]; r != nil && r.width == l.width {
		if l.version == nil || r.version == l.version(i) {
			return r.height
		}
	}
	return max(1, l.measure)
}

// Layout lays out the rows in and near the viewport.
func (l *List) Layout(gtx layout.Context) layout.Dimensions {
	viewport := gtx.Constraints.Max
	if viewport.Y >= 1<<24 {
		viewport.Y = max(gtx.Constraints.Min.Y, gtx.Dp(400))
	}
	gtx.Constraints = layout.Exact(viewport)
	l.frame++
	l.viewportH = viewport.Y
	l.n = 0
	if l.count != nil {
		l.n = max(0, l.count())
	}
	if l.width != viewport.X {
		// Heights depend on the width.
		l.width = viewport.X
	}
	if !l.started {
		l.started = true
		l.following = l.anchor == AnchorBottom
	}
	gap := gtx.Dp(l.gap)
	overscan := gtx.Dp(overscanDp)

	// Input: wheel, touch and automation move the anchor.
	l.updateScroll(gtx)
	moved := l.movedBy
	l.movedBy = 0
	l.anchorOff -= moved

	defer clip.Rect{Max: viewport}.Push(gtx.Ops).Pop()
	if bg := properties.CalcColor(l.st.Background); bg.A > 0 {
		paint.Fill(gtx.Ops, bg)
	}
	kit.BlockDragHere(gtx)
	l.scroll.Add(gtx.Ops)

	var rows []placed
	if l.n > 0 {
		rows = l.place(gtx, viewport, gap, overscan)
	}

	// Paint, and record the visible range.
	popSem := elements.PushSemanticScroll(l)
	l.first, l.last = 0, -1
	for _, p := range rows {
		if p.y+p.h <= 0 || p.y >= viewport.Y {
			continue
		}
		if l.last < l.first {
			l.first = p.idx
		}
		l.last = p.idx
	}
	for _, p := range rows {
		// Rows were laid out before their place was known: move what
		// they registered (semantic nodes, anchors) with them.
		p.track.Place(image.Pt(0, p.y))
		st := op.Offset(image.Pt(0, p.y)).Push(gtx.Ops)
		p.call.Add(gtx.Ops)
		st.Pop()
	}
	popSem()

	// Edge callbacks.
	atTop := l.n > 0 && l.first == 0 && l.last >= 0 && l.anchorIdx == 0 && l.anchorOff >= 0
	if atTop && !l.wasAtTop && l.onReachTop != nil {
		l.onReachTop()
	}
	l.wasAtTop = atTop
	atEnd := l.n > 0 && l.last == l.n-1
	if atEnd && !l.wasAtEnd && l.onReachEnd != nil {
		l.onReachEnd()
	}
	l.wasAtEnd = atEnd

	// The scrollbar, from measured and estimated heights.
	contentH, offset := l.estimate(gap)
	l.contentH = contentH
	l.barOffset = float32(offset)
	before := l.barOffset
	l.bar.Layout(gtx, viewport, contentH, &l.barOffset, l.st.ScrollBar)
	if d := l.barOffset - before; d != 0 {
		l.jumpTo(int(l.barOffset), gap)
	}

	if l.pass == nil {
		l.pass = func(gtx layout.Context) bool {
			return l.updateScroll(gtx) != 0
		}
	}
	kit.HandleInput(l.pass)
	if l.scroll.State() != gesture.StateIdle || moved != 0 {
		kit.MarkLive()
	}
	l.sweep()
	elements.RecordSemantic(gtx, elements.SemanticNode{Kind: elements.SemanticScroll, Target: l}, viewport)
	return layout.Dimensions{Size: viewport}
}

// updateScroll returns the wheel and touch scroll distance since the
// last frame. It stops following the end on a scroll up or when a
// touch drag starts, and follows again when the drag (or its fling)
// ends at the end.
func (l *List) updateScroll(gtx layout.Context) int {
	// The range comes from last frame's estimate; layout clamps the
	// distance to the content's actual ends.
	off := int(l.barOffset)
	rng := pointer.ScrollRange{Min: -off, Max: max(0, l.contentH-l.viewportH-off)}
	if l.following {
		rng.Max = 0
	}
	dist := l.scroll.Update(gtx.Metric, gtx.Source, gtx.Now, gesture.Vertical, pointer.ScrollRange{}, rng)
	if dist < 0 {
		l.following = false
	}
	started, ended := l.gest.Track(&l.scroll, dist)
	switch {
	case started:
		// A finger moves the rows: stop following the end, whichever
		// way it goes, or a growing last row undoes its movement.
		l.following = false
	case ended && l.anchor == AnchorBottom && l.atBottom:
		l.following = true
	}
	l.movedBy += dist
	return dist
}

// place lays out rows around the anchor to fill the viewport and the
// overscan, clamps to the content's ends and updates the anchor.
func (l *List) place(gtx layout.Context, viewport image.Point, gap, overscan int) []placed {
	n := l.n
	idx := l.indexOf(l.anchorKey, l.anchorIdx)
	if idx < 0 {
		idx = min(max(l.anchorIdx, 0), n-1)
	}
	off := l.anchorOff
	if l.hasPending {
		l.hasPending = false
		if p := l.indexOf(l.pendingKey, idx); p >= 0 {
			idx = p
			h := l.heightOf(p)
			switch l.pendingAlign {
			case AlignStart:
				off = 0
			case AlignCenter:
				off = (viewport.Y - h) / 2
			case AlignEnd:
				off = viewport.Y - h
			}
		}
	}
	// A long jump moves the anchor past rows without laying them out,
	// by their measured (or estimated) heights.
	for idx < n-1 && off+l.heightOf(idx)+gap <= -overscan {
		off += l.heightOf(idx) + gap
		idx++
	}
	for idx > 0 && off > viewport.Y+overscan {
		idx--
		off -= l.heightOf(idx) + gap
	}
	var rows []placed
	if l.following {
		// From the end up.
		y := viewport.Y
		for i := n - 1; i >= 0 && y > -overscan; i-- {
			p := l.layoutRow(gtx, i)
			y -= p.h
			p.y = y
			rows = append(rows, p)
			y -= gap
		}
		reverse(rows)
		l.atBottom = true
		if top := rows[0]; top.idx == 0 && top.y > 0 {
			// Shorter than the viewport: from the top.
			shift := -top.y
			for i := range rows {
				rows[i].y += shift
			}
		}
	} else {
		// From the anchor down, then up to fill any space above.
		y := off
		for i := idx; i < n && y < viewport.Y+overscan; i++ {
			p := l.layoutRow(gtx, i)
			p.y = y
			rows = append(rows, p)
			y += p.h + gap
		}
		// Past the end: pull the rows down so the last one ends at the
		// bottom.
		if last := rows[len(rows)-1]; last.idx == n-1 && last.y+last.h < viewport.Y {
			shift := viewport.Y - (last.y + last.h)
			for i := range rows {
				rows[i].y += shift
			}
		}
		y = rows[0].y
		var above []placed
		for i := rows[0].idx - 1; i >= 0 && y > -overscan; i-- {
			p := l.layoutRow(gtx, i)
			y -= gap + p.h
			p.y = y
			above = append(above, p)
		}
		reverse(above)
		rows = append(above, rows...)
		// Above the start: the first row starts at the top.
		if top := rows[0]; top.idx == 0 && top.y > 0 {
			shift := -top.y
			for i := range rows {
				rows[i].y += shift
			}
		}
		last := rows[len(rows)-1]
		l.atBottom = last.idx == n-1 && last.y+last.h <= viewport.Y
		if l.anchor == AnchorBottom && !l.gest.Moving() {
			// Back at the end: follow it again - but not while a finger
			// is still moving the rows.
			l.following = l.atBottom
		}
	}
	// The anchor: the first row whose bottom is in view.
	for _, p := range rows {
		if p.y+p.h > 0 || p.idx == rows[len(rows)-1].idx {
			l.anchorIdx, l.anchorKey, l.anchorOff = p.idx, l.key(p.idx), p.y
			break
		}
	}
	return rows
}

// estimate returns the content height and the offset of the viewport
// into it, from measured heights and the average for unmeasured rows.
func (l *List) estimate(gap int) (contentH, offset int) {
	if l.n == 0 {
		return 0, 0
	}
	// Rows are summed only up to a limit: past it, the average stands
	// in, keeping long lists cheap.
	const exact = 2000
	sum := func(from, to int) int {
		if to-from > exact {
			return (to - from) * (max(1, l.measure) + gap)
		}
		s := 0
		for i := from; i < to; i++ {
			s += l.heightOf(i) + gap
		}
		return s
	}
	above := sum(0, l.anchorIdx)
	below := sum(l.anchorIdx, l.n)
	return above + below - gap, above - l.anchorOff
}

// jumpTo moves the anchor to the row at the estimated offset y, for a
// scrollbar drag.
func (l *List) jumpTo(y, gap int) {
	acc := 0
	for i := 0; i < l.n; i++ {
		h := l.heightOf(i) + gap
		if acc+h > y || i == l.n-1 {
			l.anchorIdx, l.anchorKey, l.anchorOff = i, l.key(i), acc-y
			l.following = false
			return
		}
		acc += h
	}
}

// sweep forgets rows not laid out for a while.
func (l *List) sweep() {
	if l.frame%120 != 0 {
		return
	}
	for k, r := range l.rows {
		if l.frame-r.used > 600 {
			delete(l.rows, k)
		}
	}
}

func reverse(p []placed) {
	for i, j := 0, len(p)-1; i < j; i, j = i+1, j-1 {
		p[i], p[j] = p[j], p[i]
	}
}
