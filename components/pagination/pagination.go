// Package pagination provides a stateful page-number control built
// on Button.
package pagination

import (
	"github.com/xmhrnx/giode/components/kit"
	"strconv"

	"gioui.org/layout"

	"github.com/xmhrnx/giode/components/button"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

var defaultEllipsisColor = "#94a3b8"

// Pagination is a stateful row of page-number controls: prev/next and
// a windowed set of page buttons (first, last, and a few pages around
// the current one, with an ellipsis for the gap). Create it once with
// New and lay it out every frame.
type Pagination struct {
	// Contextual gives it a context menu (Context).
	elements.Contextual

	current  int
	total    int
	onChange func(page int)
	st       styles.Styles

	prev, next *button.Button
	pages      map[int]*button.Button
}

// New returns a Pagination over total pages (1-indexed), starting at
// page 1. The styles argument is optional and styles every button
// (prev/next and page numbers alike); the current page uses
// button.VariantContained, every other button
// button.VariantOutlined.
func New(total int, st ...styles.Styles) *Pagination {
	p := &Pagination{total: total, current: 1, pages: map[int]*button.Button{}}
	if len(st) > 0 {
		p.st = st[0]
	}
	p.prev = button.New("<", p.st).Variant(button.VariantOutlined)
	p.next = button.New(">", p.st).Variant(button.VariantOutlined)
	p.prev.OnClick(func() { p.goTo(p.current - 1) })
	p.next.OnClick(func() { p.goTo(p.current + 1) })
	return p
}

// Current returns the current page (1-indexed).
func (p *Pagination) Current() int {
	return p.current
}

// SetCurrent changes the current page without invoking OnChange,
// clamped to [1, total].
func (p *Pagination) SetCurrent(page int) {
	p.current = clamp(page, 1, p.total)
}

// SetTotal changes the page count, clamping the current page if it no
// longer fits.
func (p *Pagination) SetTotal(total int) {
	p.total = total
	p.current = clamp(p.current, 1, total)
}

// OnChange sets the handler invoked when the current page changes
// (clicking a page button, or prev/next). It returns p for chaining.
func (p *Pagination) OnChange(fn func(page int)) *Pagination {
	p.onChange = fn
	return p
}

func (p *Pagination) goTo(page int) {
	if page < 1 || page > p.total || page == p.current {
		return
	}
	p.current = page
	if p.onChange != nil {
		p.onChange(page)
	}
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// pageButton returns the Button for page, creating and caching it the
// first time it's needed so its click state persists across frames
// even though which pages are visible changes as the window shifts.
func (p *Pagination) pageButton(page int) *button.Button {
	if b, ok := p.pages[page]; ok {
		return b
	}
	b := button.New(strconv.Itoa(page), p.st)
	b.OnClick(func() { p.goTo(page) })
	p.pages[page] = b
	return b
}

// windowed returns the sequence of page numbers to show - 0 stands
// for an ellipsis - always including the first and last page, plus a
// window of up to 3 pages around the current one.
func (p *Pagination) windowed() []int {
	if p.total <= 7 {
		seq := make([]int, p.total)
		for i := range seq {
			seq[i] = i + 1
		}
		return seq
	}

	var seq []int
	add := func(n int) { seq = append(seq, n) }

	add(1)
	lo, hi := clamp(p.current-1, 2, p.total-1), clamp(p.current+1, 2, p.total-1)
	if lo > 2 {
		add(0)
	}
	for n := lo; n <= hi; n++ {
		add(n)
	}
	if hi < p.total-1 {
		add(0)
	}
	add(p.total)
	return seq
}

// Layout lays out prev, the windowed page buttons, and next.
func (p *Pagination) Layout(gtx layout.Context) layout.Dimensions {
	dims := p.layoutSelf(gtx)
	// A right-click or long-press anywhere on it opens its context menu.
	p.AnchorContext(gtx, dims.Size)
	return dims
}

// layoutSelf lays it out without its context menu area.
func (p *Pagination) layoutSelf(gtx layout.Context) layout.Dimensions {
	// Handles its input without an input pass; see kit.OpaqueInput.
	kit.OpaqueInput()
	if p.total <= 0 {
		return layout.Dimensions{}
	}

	children := make([]elements.Element, 0, len(p.pages)+2)
	children = append(children, p.prev)
	for _, n := range p.windowed() {
		if n == 0 {
			children = append(children, elements.Box(
				styles.Styles{Padding: properties.SymmetricInset(8, 8)},
				elements.Text("...", styles.Styles{Color: defaultEllipsisColor}),
			))
			continue
		}
		b := p.pageButton(n)
		if n == p.current {
			b.Variant(button.VariantContained)
		} else {
			b.Variant(button.VariantOutlined)
		}
		children = append(children, b)
	}
	children = append(children, p.next)

	return elements.Row(styles.Styles{Gap: 4, Align: properties.AlignCenter}, children...).Layout(gtx)
}
