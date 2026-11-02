// Package virtuallist provides a vertical list that only builds the
// rows in or near its viewport, so it handles any number of items at a
// constant cost per frame.
package virtuallist

import (
	"image"
	"math"

	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/io/key"
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

// Align places an item in the viewport for ScrollToIndex.
type Align int

const (
	// AlignNearest scrolls as little as possible to show the item
	// fully: not at all if it already is.
	AlignNearest Align = iota
	// AlignStart puts the item at the top.
	AlignStart
	// AlignCenter centers the item.
	AlignCenter
	// AlignEnd puts the item at the bottom.
	AlignEnd
)

const defaultOverscan = 3

var defaultFocus = "theme.primary/0.18"

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

func (l *List) maxOffset() float32 {
	return float32(max(0, l.n*l.itemPx-l.viewportH))
}

func (l *List) clamp(v float32) float32 {
	return max(0, min(l.maxOffset(), v))
}

// resolveScroll applies a pending ScrollToIndex.
func (l *List) resolveScroll() {
	if !l.hasPending || l.n == 0 || l.itemPx == 0 {
		return
	}
	l.hasPending = false
	i := max(0, min(l.n-1, l.pendingIdx))
	top := float32(i * l.itemPx)
	bottom := top + float32(l.itemPx)
	vh := float32(l.viewportH)
	switch l.pendingAlign {
	case AlignStart:
		l.offset = top
	case AlignCenter:
		l.offset = top - (vh-float32(l.itemPx))/2
	case AlignEnd:
		l.offset = bottom - vh
	default:
		if top < l.offset {
			l.offset = top
		} else if bottom > l.offset+vh {
			l.offset = bottom - vh
		}
	}
	l.offset = l.clamp(l.offset)
}

// handleKeys processes focus and navigation keys.
func (l *List) handleKeys(gtx layout.Context) {
	page := max(1, l.viewportH/max(1, l.itemPx)-1)
	if l.wantFocus {
		l.wantFocus = false
		gtx.Execute(key.FocusCmd{Tag: l})
	}
	filters := []event.Filter{
		key.FocusFilter{Target: l},
		key.Filter{Focus: l, Name: key.NameUpArrow},
		key.Filter{Focus: l, Name: key.NameDownArrow},
		key.Filter{Focus: l, Name: key.NamePageUp},
		key.Filter{Focus: l, Name: key.NamePageDown},
		key.Filter{Focus: l, Name: key.NameHome},
		key.Filter{Focus: l, Name: key.NameEnd},
		key.Filter{Focus: l, Name: key.NameReturn},
		key.Filter{Focus: l, Name: key.NameEnter},
	}
	for _, n := range l.extraKeys {
		filters = append(filters, key.Filter{Focus: l, Name: n})
	}
	for {
		ev, ok := gtx.Event(filters...)
		if !ok {
			break
		}
		switch e := ev.(type) {
		case key.FocusEvent:
			l.focused = e.Focus
			if e.Focus && l.focusIdx < 0 && l.n > 0 && !l.pointerFocus {
				l.SetFocusedIndex(max(0, l.first))
			}
			l.pointerFocus = false
		case key.Event:
			if e.State != key.Press || l.n == 0 {
				continue
			}
			cur := max(0, l.focusIdx)
			// Keyboard moves stay within the current rows.
			clampN := func(i int) int { return max(0, min(l.n-1, i)) }
			switch e.Name {
			case key.NameUpArrow:
				l.SetFocusedIndex(clampN(cur - 1))
			case key.NameDownArrow:
				if l.focusIdx < 0 {
					l.SetFocusedIndex(0)
				} else {
					l.SetFocusedIndex(clampN(cur + 1))
				}
			case key.NamePageUp:
				l.SetFocusedIndex(clampN(cur - page))
			case key.NamePageDown:
				l.SetFocusedIndex(clampN(cur + page))
			case key.NameHome:
				l.SetFocusedIndex(0)
			case key.NameEnd:
				l.SetFocusedIndex(l.n - 1)
			case key.NameReturn, key.NameEnter:
				if l.focusIdx >= 0 && l.onActivate != nil {
					l.onActivate(l.focusIdx)
				}
			default:
				if l.onKey != nil {
					l.onKey(e, l.focusIdx)
				}
			}
			kit.RequestFrame(gtx)
		}
	}
	// A press anywhere in the list focuses it.
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &l.focusIdx, Kinds: pointer.Press})
		if !ok {
			break
		}
		if pe, ok := ev.(pointer.Event); ok && pe.Kind == pointer.Press {
			if !l.focused {
				l.pointerFocus = true
			}
			gtx.Execute(key.FocusCmd{Tag: l})
			if l.itemPx > 0 && !l.noClickRow {
				if i := int((pe.Position.Y + l.offset) / float32(l.itemPx)); i >= 0 && i < l.n {
					l.SetFocusedIndex(i)
				}
			}
		}
	}
}

// Layout lays out the visible rows.
func (l *List) Layout(gtx layout.Context) layout.Dimensions {
	// Handles its input without an input pass; see kit.OpaqueInput.
	kit.OpaqueInput()
	viewport := gtx.Constraints.Max
	if viewport.X >= 1e6 {
		viewport.X = gtx.Constraints.Min.X
	}
	if viewport.Y >= 1e6 {
		// Unbounded height (e.g. inside a Scroll): show a screenful.
		viewport.Y = max(gtx.Constraints.Min.Y, gtx.Dp(400))
	}
	gtx.Constraints = layout.Exact(viewport)
	l.viewportH = viewport.Y
	l.itemPx = max(1, gtx.Dp(l.itemH))
	l.n = 0
	if l.count != nil {
		l.n = max(0, l.count())
	}
	if l.focusIdx >= l.n {
		l.focusIdx = l.n - 1
	}

	l.handleKeys(gtx)
	l.resolveScroll()

	defer clip.Rect{Max: viewport}.Push(gtx.Ops).Pop()
	if bg := properties.CalcColor(l.st.Background); bg.A > 0 {
		paint.Fill(gtx.Ops, bg)
	}

	// Wheel and touch scrolling, plus press-to-focus, beneath the rows
	// so interactive rows keep priority.
	kit.BlockDragHere(gtx)
	l.scroll.Add(gtx.Ops)
	ps := pointer.PassOp{}.Push(gtx.Ops)
	event.Op(gtx.Ops, &l.focusIdx)
	event.Op(gtx.Ops, l)
	ps.Pop()
	l.offset = l.clamp(l.offset)
	dist := l.scroll.Update(gtx.Metric, gtx.Source, gtx.Now, gesture.Vertical,
		pointer.ScrollRange{},
		pointer.ScrollRange{Min: -int(l.offset), Max: int(l.maxOffset() - l.offset)})
	if dist != 0 {
		l.offset = l.clamp(l.offset + float32(dist))
	}
	l.gest.Track(&l.scroll, dist)
	if dist != 0 || l.scroll.State() != gesture.StateIdle {
		kit.MarkLive()
	}

	// Visible rows.
	l.first, l.last = 0, -1
	if l.n > 0 {
		l.first = int(l.offset) / l.itemPx
		l.last = min(l.n-1, int(math.Ceil(float64(l.offset+float32(viewport.Y))/float64(l.itemPx)))-1)
	}
	from := max(0, l.first-l.overscan)
	to := min(l.n-1, l.last+l.overscan)
	focusColor := l.st.BorderColor
	if focusColor == "" {
		focusColor = defaultFocus
	}
	rowSize := image.Pt(viewport.X, l.itemPx)
	popSem := elements.PushSemanticScroll(l)
	for i := from; i <= to; i++ {
		y := i*l.itemPx - int(l.offset)
		off := op.Offset(image.Pt(0, y)).Push(gtx.Ops)
		pop := elements.PushOffset(image.Pt(0, y))
		if l.focused && i == l.focusIdx {
			paint.FillShape(gtx.Ops, properties.CalcColor(focusColor), clip.Rect{Max: rowSize}.Op())
		}
		rgtx := gtx
		rgtx.Constraints = layout.Exact(rowSize)
		if l.item != nil {
			if el := l.item(i); el != nil {
				el.Layout(rgtx)
			}
		}
		pop()
		off.Pop()
	}
	popSem()
	if l.focused {
		kit.MarkLive()
	}

	atEnd := l.n > 0 && l.last >= l.n-1
	if atEnd && !l.wasAtEnd && l.onReachEnd != nil {
		l.onReachEnd()
	}
	l.wasAtEnd = atEnd

	l.bar.Layout(gtx, viewport, l.n*l.itemPx, &l.offset, l.st.ScrollBar)
	elements.RecordSemantic(gtx, elements.SemanticNode{Kind: elements.SemanticScroll, Target: l}, viewport)
	return layout.Dimensions{Size: viewport}
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
