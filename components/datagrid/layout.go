package datagrid

import (
	"image"
	"image/color"
	"strconv"

	"gioui.org/f32"
	"gioui.org/io/event"
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

// frameLayout is where the last frame put things, in px.
type frameLayout struct {
	size            image.Point
	headerH, rowH   int
	checkW, frozenW int
	// colX are the display columns' left edges in the content, before
	// scrolling (frozen ones never scroll); colX[len] is the content's
	// width.
	colX        []int
	frozenCols  int
	bodyH       int
	first, last int // rows on screen, [first, last)
	maxX, maxY  int
	pxPerDp     float32
}

// colors the grid paints with.
type palette struct {
	bg, header, text, muted, border, selected, hover, focus, check color.NRGBA
}

func (g *DataGrid) palette() palette {
	c := func(v properties.Color, def string) color.NRGBA {
		if v == "" {
			v = properties.Color(def)
		}
		return elements.ResolveColor(v)
	}
	return palette{
		bg:       c(properties.Color(g.st.Background), "theme.background"),
		header:   c("", "theme.surface"),
		text:     c(g.st.Color, "theme.text"),
		muted:    c("", "theme.muted"),
		border:   c(g.st.BorderColor, "theme.border"),
		selected: c("", "theme.primary/0.14"),
		hover:    c("", "theme.text/0.04"),
		focus:    c("", "theme.primary"),
		check:    c("", "theme.primary"),
	}
}

func (g *DataGrid) fontSize() properties.FontSize {
	if g.st.FontSize != 0 {
		return g.st.FontSize
	}
	return 13
}

// Layout lays the grid out, filling the space it's given.
func (g *DataGrid) Layout(gtx layout.Context) layout.Dimensions {
	dims := g.layoutSelf(gtx)
	g.AnchorContext(gtx, dims.Size)
	return dims
}

func (g *DataGrid) layoutSelf(gtx layout.Context) layout.Dimensions {
	if g.pass == nil {
		g.pass = func(gtx layout.Context) bool { return g.update(gtx) }
	}
	g.update(gtx)
	if !kit.HandleInput(g.pass) {
		kit.OpaqueInput()
	}
	size := gtx.Constraints.Max
	if size.Y >= 1<<20 {
		size.Y = gtx.Dp(400)
	}
	size = gtx.Constraints.Constrain(size)
	g.computeLayout(gtx, size)
	l := &g.lay
	n := g.src.Len()
	pal := g.palette()

	// Scrolling: wheel and touch, then keeping the focus in view.
	g.applyScroll(gtx)
	if g.reveal {
		g.reveal = false
		g.revealFocus()
	}
	g.offY = min(max(g.offY, 0), float32(l.maxY))
	g.offX = min(max(g.offX, 0), float32(l.maxX))
	l.first = min(int(g.offY)/l.rowH, n)
	l.last = min((int(g.offY)+l.bodyH)/l.rowH+1, n)
	g.load(l.first, l.last)

	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	paint.FillShape(gtx.Ops, pal.bg, clip.Rect{Max: size}.Op())

	// Input areas: the grid's own, under everything; the scroll
	// gestures over the body.
	event.Op(gtx.Ops, g)
	body := image.Rect(0, l.headerH, size.X, size.Y)
	bodyArea := clip.Rect(body).Push(gtx.Ops)
	g.scrollY.Add(gtx.Ops)
	g.scrollX.Add(gtx.Ops)
	bodyArea.Pop()
	kit.BlockDrag(gtx, size)

	// The body: the scrolling columns, then the frozen ones over them.
	bc := clip.Rect(body).Push(gtx.Ops)
	g.paintRows(gtx, pal, false)
	g.paintRows(gtx, pal, true)
	if l.frozenW > 0 && g.offX > 0 {
		// A shadow where the scrolling columns slide under.
		paintShadow(gtx.Ops, image.Rect(l.frozenW, l.headerH, l.frozenW+gtx.Dp(6), size.Y), pal.border)
	}
	bc.Pop()

	g.paintHeader(gtx, pal)
	g.paintScrollbars(gtx)
	g.paintStatus(gtx, pal)

	// The row menu opens over the body.
	if g.rowMenu != nil {
		off := op.Offset(body.Min).Push(gtx.Ops)
		pop := elements.PushOffset(body.Min)
		g.rowCtx.AnchorContext(gtx, body.Size())
		pop()
		off.Pop()
	}
	elements.RecordSemantic(gtx, elements.SemanticNode{Kind: elements.SemanticScroll, Target: g}, size)
	return layout.Dimensions{Size: size}
}

// computeLayout sizes the columns and rows for size.
func (g *DataGrid) computeLayout(gtx layout.Context, size image.Point) {
	l := &g.lay
	l.size = size
	l.pxPerDp = gtx.Metric.PxPerDp
	if l.pxPerDp == 0 {
		l.pxPerDp = 1
	}
	l.headerH, l.rowH = gtx.Dp(unit.Dp(g.headerH)), max(gtx.Dp(unit.Dp(g.rowH)), 1)
	l.checkW = 0
	if g.checks {
		l.checkW = gtx.Dp(40)
	}
	l.colX = l.colX[:0]
	x := l.checkW
	for _, ci := range g.order {
		l.colX = append(l.colX, x)
		x += gtx.Dp(unit.Dp(g.widths[ci]))
	}
	l.colX = append(l.colX, x)
	l.frozenCols = min(g.frozen, len(g.order))
	l.frozenW = l.colX[l.frozenCols]
	l.bodyH = max(size.Y-l.headerH, 0)
	l.maxY = max(g.src.Len()*l.rowH-l.bodyH, 0)
	l.maxX = max(x-size.X, 0)
}

// cellX returns display column c's left edge on screen.
func (l *frameLayout) cellX(c int, offX float32) int {
	if c < l.frozenCols {
		return l.colX[c]
	}
	return l.colX[c] - int(offX)
}

// rowY returns row's top on screen.
func (l *frameLayout) rowY(row int, offY float32) int {
	return l.headerH + row*l.rowH - int(offY)
}

// paintRows paints the visible rows' frozen part or scrolling part.
func (g *DataGrid) paintRows(gtx layout.Context, pal palette, frozen bool) {
	l := &g.lay
	region := image.Rect(l.frozenW, l.headerH, l.size.X, l.size.Y)
	c0, c1 := l.frozenCols, len(g.order)
	if frozen {
		region = image.Rect(0, l.headerH, l.frozenW, l.size.Y)
		c0, c1 = 0, l.frozenCols
	}
	if region.Empty() {
		return
	}
	defer clip.Rect(region).Push(gtx.Ops).Pop()
	line := max(gtx.Dp(1), 1)
	loader, _ := g.src.(Loader)
	for row := l.first; row < l.last; row++ {
		y := l.rowY(row, g.offY)
		rr := image.Rect(region.Min.X, y, region.Max.X, y+l.rowH)
		paint.FillShape(gtx.Ops, pal.bg, clip.Rect(rr).Op())
		sel := g.Selected(row)
		switch {
		case sel:
			paint.FillShape(gtx.Ops, pal.selected, clip.Rect(rr).Op())
		case row == g.hoverRow:
			paint.FillShape(gtx.Ops, pal.hover, clip.Rect(rr).Op())
		}
		paint.FillShape(gtx.Ops, pal.border, clip.Rect(image.Rect(rr.Min.X, rr.Max.Y-line, rr.Max.X, rr.Max.Y)).Op())
		if frozen && l.checkW > 0 {
			g.paintCheck(gtx, pal, image.Rect(0, y, l.checkW, y+l.rowH), sel, false)
		}
		loaded := loader == nil || loader.Loaded(row)
		for c := c0; c < c1; c++ {
			x := l.cellX(c, g.offX)
			w := l.colX[c+1] - l.colX[c]
			if x+w <= region.Min.X || x >= region.Max.X {
				continue
			}
			r := image.Rect(x, y, x+w, y+l.rowH-line)
			paint.FillShape(gtx.Ops, pal.border, clip.Rect(image.Rect(r.Max.X-line, y, r.Max.X, y+l.rowH)).Op())
			if !loaded {
				g.paintPlaceholder(gtx, pal, r)
				continue
			}
			g.paintCell(gtx, pal, r, g.cols[g.order[c]], row)
		}
		if g.focused && row == g.focus.row && g.focus.col >= c0 && g.focus.col < c1 {
			c := g.focus.col
			x := l.cellX(c, g.offX)
			r := image.Rect(x, y, l.cellX(c, g.offX)+l.colX[c+1]-l.colX[c], y+l.rowH)
			strokeRect(gtx.Ops, r, max(gtx.Dp(2), 1), pal.focus)
		}
	}
}

// paintCell draws a cell's content in r.
func (g *DataGrid) paintCell(gtx layout.Context, pal palette, r image.Rectangle, col Column, row int) {
	pad := gtx.Dp(8)
	inner := image.Rect(r.Min.X+pad, r.Min.Y, r.Max.X-pad, r.Max.Y)
	if inner.Dx() <= 0 {
		return
	}
	defer clip.Rect(r).Push(gtx.Ops).Pop()
	v := g.src.Value(row, col.Key)
	cgtx := gtx
	cgtx.Constraints = layout.Constraints{Max: inner.Size()}
	if col.Cell != nil {
		cgtx.Constraints.Min = inner.Size()
		off := op.Offset(inner.Min).Push(gtx.Ops)
		pop := elements.PushOffset(inner.Min)
		col.Cell(row, v).Layout(cgtx)
		pop()
		off.Pop()
		return
	}
	s := formatValue(col, v)
	if s == "" {
		return
	}
	align := col.Align
	el := elements.Text(s, styles.Styles{FontSize: g.fontSize(), Color: properties.CalcColorReverse(pal.text), MaxLines: 1, TextOverflow: properties.TextOverflowEllipsis, TextAlign: align, Width: -1})
	rec := op.Record(gtx.Ops)
	pop := elements.PushOffset(inner.Min)
	dims := kit.Unselectable(el.Layout)(cgtx)
	pop()
	call := rec.Stop()
	off := op.Offset(image.Pt(inner.Min.X, inner.Min.Y+(inner.Dy()-dims.Size.Y)/2)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	off.Pop()
}

func formatValue(col Column, v any) string {
	if col.Format != nil {
		return col.Format(v)
	}
	return text(v)
}

// paintPlaceholder draws a loading bar in a cell not loaded yet.
func (g *DataGrid) paintPlaceholder(gtx layout.Context, pal palette, r image.Rectangle) {
	pad := gtx.Dp(8)
	h := gtx.Dp(8)
	w := max(r.Dx()*3/5-pad, 0)
	y := r.Min.Y + (r.Dy()-h)/2
	b := image.Rect(r.Min.X+pad, y, r.Min.X+pad+w, y+h)
	c := pal.muted
	c.A /= 4
	paint.FillShape(gtx.Ops, c, clip.UniformRRect(b, h/2).Op(gtx.Ops))
}

// paintCheck draws a checkbox centered in r: checked, or with mixed a
// dash for some rows selected.
func (g *DataGrid) paintCheck(gtx layout.Context, pal palette, r image.Rectangle, checked, mixed bool) {
	s := gtx.Dp(16)
	b := image.Rect(r.Min.X+(r.Dx()-s)/2, r.Min.Y+(r.Dy()-s)/2, r.Min.X+(r.Dx()-s)/2+s, r.Min.Y+(r.Dy()-s)/2+s)
	rad := gtx.Dp(4)
	if checked || mixed {
		paint.FillShape(gtx.Ops, pal.check, clip.UniformRRect(b, rad).Op(gtx.Ops))
		gl := kit.GlyphCheck
		if mixed {
			gl = kit.GlyphMinus
		}
		gs := unit.Dp(12)
		off := op.Offset(b.Min.Add(image.Pt((s-gtx.Dp(gs))/2, (s-gtx.Dp(gs))/2))).Push(gtx.Ops)
		kit.DrawGlyph(gtx, gl, gs, elements.ResolveColor("theme.onprimary"))
		off.Pop()
		return
	}
	strokeRRect(gtx.Ops, b, rad, max(gtx.Dp(1), 1), pal.muted)
}

// paintHeader draws the header row over the body.
func (g *DataGrid) paintHeader(gtx layout.Context, pal palette) {
	l := &g.lay
	hr := image.Rect(0, 0, l.size.X, l.headerH)
	paint.FillShape(gtx.Ops, pal.header, clip.Rect(hr).Op())
	line := max(gtx.Dp(1), 1)
	for pass := 0; pass < 2; pass++ {
		frozen := pass == 1
		region := image.Rect(l.frozenW, 0, l.size.X, l.headerH)
		c0, c1 := l.frozenCols, len(g.order)
		if frozen {
			region = image.Rect(0, 0, l.frozenW, l.headerH)
			c0, c1 = 0, l.frozenCols
			paint.FillShape(gtx.Ops, pal.header, clip.Rect(region).Op())
		}
		if region.Empty() {
			continue
		}
		cl := clip.Rect(region).Push(gtx.Ops)
		for c := c0; c < c1; c++ {
			x := l.cellX(c, g.offX)
			w := l.colX[c+1] - l.colX[c]
			if x+w <= region.Min.X || x >= region.Max.X {
				continue
			}
			col := g.cols[g.order[c]]
			r := image.Rect(x, 0, x+w, l.headerH)
			g.paintHeaderCell(gtx, pal, r, col)
			paint.FillShape(gtx.Ops, pal.border, clip.Rect(image.Rect(r.Max.X-line, r.Min.Y+l.headerH/4, r.Max.X, r.Max.Y-l.headerH/4)).Op())
			// The resize handle: a strip at the column's right edge.
			hw := gtx.Dp(6)
			ha := clip.Rect(image.Rect(r.Max.X-hw/2-line, 0, r.Max.X+hw/2, l.headerH)).Push(gtx.Ops)
			pointer.CursorColResize.Add(gtx.Ops)
			ha.Pop()
		}
		if frozen && l.checkW > 0 {
			all := g.sel.count(g.src.Len())
			g.paintCheck(gtx, pal, image.Rect(0, 0, l.checkW, l.headerH), all > 0 && all == g.src.Len(), all > 0 && all < g.src.Len())
		}
		cl.Pop()
	}
	paint.FillShape(gtx.Ops, pal.border, clip.Rect(image.Rect(0, l.headerH-line, l.size.X, l.headerH)).Op())
}

// paintHeaderCell draws a column's title and sort indicator.
func (g *DataGrid) paintHeaderCell(gtx layout.Context, pal palette, r image.Rectangle, col Column) {
	pad := gtx.Dp(8)
	defer clip.Rect(r).Push(gtx.Ops).Pop()
	if col.Sortable {
		area := clip.Rect(r).Push(gtx.Ops)
		pointer.CursorPointer.Add(gtx.Ops)
		area.Pop()
	}
	sortIdx, desc := -1, false
	for i, k := range g.sort {
		if k.Key == col.Key {
			sortIdx, desc = i, k.Descending
		}
	}
	ind := 0
	if sortIdx >= 0 {
		ind = gtx.Dp(14)
		if len(g.sort) > 1 {
			ind += gtx.Dp(8)
		}
	}
	inner := image.Rect(r.Min.X+pad, r.Min.Y, r.Max.X-pad-ind, r.Max.Y)
	if inner.Dx() > 0 {
		cgtx := gtx
		cgtx.Constraints = layout.Constraints{Max: inner.Size()}
		el := elements.Text(col.Title, styles.Styles{FontSize: g.fontSize(), FontWeight: properties.FontWeightSemiBold, Color: properties.CalcColorReverse(pal.text), MaxLines: 1, TextOverflow: properties.TextOverflowEllipsis, TextAlign: col.Align, Width: -1})
		rec := op.Record(gtx.Ops)
		pop := elements.PushOffset(inner.Min)
		dims := kit.Unselectable(el.Layout)(cgtx)
		pop()
		call := rec.Stop()
		off := op.Offset(image.Pt(inner.Min.X, (r.Dy()-dims.Size.Y)/2)).Push(gtx.Ops)
		call.Add(gtx.Ops)
		off.Pop()
	}
	if sortIdx >= 0 {
		gl := kit.GlyphChevronUp
		if desc {
			gl = kit.GlyphChevronDown
		}
		gs := unit.Dp(10)
		x := r.Max.X - pad - ind
		off := op.Offset(image.Pt(x, (r.Dy()-gtx.Dp(gs))/2)).Push(gtx.Ops)
		kit.DrawGlyph(gtx, gl, gs, pal.text)
		off.Pop()
		if len(g.sort) > 1 {
			cgtx := gtx
			cgtx.Constraints = layout.Constraints{Max: image.Pt(gtx.Dp(10), r.Dy())}
			o := op.Offset(image.Pt(x+gtx.Dp(12), (r.Dy()-gtx.Dp(14))/2)).Push(gtx.Ops)
			elements.Text(strconv.Itoa(sortIdx+1), styles.Styles{FontSize: 10, Color: properties.CalcColorReverse(pal.muted)}).Layout(cgtx)
			o.Pop()
		}
	}
}

// paintScrollbars draws the scrollbars of the body.
func (g *DataGrid) paintScrollbars(gtx layout.Context) {
	l := &g.lay
	sb := g.st.ScrollBar
	if l.maxY > 0 {
		off := op.Offset(image.Pt(0, l.headerH)).Push(gtx.Ops)
		y := g.offY
		g.barY.Layout(gtx, image.Pt(l.size.X, l.bodyH), g.src.Len()*l.rowH, &y, sb)
		g.offY = y
		off.Pop()
	}
	if l.maxX > 0 {
		// The vertical bar drawn across: x and y swapped.
		w := l.size.X - l.frozenW
		t := f32.NewAffine2D(0, 1, float32(l.frozenW), 1, 0, float32(l.headerH))
		st := op.Affine(t).Push(gtx.Ops)
		x := g.offX
		g.barX.Layout(gtx, image.Pt(l.bodyH, w), l.colX[len(l.colX)-1]-l.frozenW, &x, sb)
		g.offX = x
		st.Pop()
	}
}

// paintStatus shows a loading source's progress and errors.
func (g *DataGrid) paintStatus(gtx layout.Context, pal palette) {
	ld, ok := g.src.(Loader)
	if !ok {
		return
	}
	loading, err := ld.Status()
	l := &g.lay
	if loading {
		h := max(gtx.Dp(2), 1)
		paint.FillShape(gtx.Ops, pal.focus, clip.Rect(image.Rect(0, l.headerH, l.size.X, l.headerH+h)).Op())
	}
	if err != nil {
		msg := elements.Box(styles.Styles{Background: "theme.danger", Padding: properties.SymmetricInset(10, 6), BorderRadius: 6},
			elements.Text(err.Error(), styles.Styles{Color: "#ffffff", FontSize: 12, MaxLines: 2}))
		cgtx := gtx
		cgtx.Constraints = layout.Constraints{Max: image.Pt(l.size.X-gtx.Dp(24), l.size.Y)}
		rec := op.Record(gtx.Ops)
		dims := msg.Layout(cgtx)
		call := rec.Stop()
		at := image.Pt((l.size.X-dims.Size.X)/2, l.headerH+gtx.Dp(12))
		off := op.Offset(at).Push(gtx.Ops)
		call.Add(gtx.Ops)
		off.Pop()
	}
}

// load asks a Loader for rows on screen it doesn't have, and a page
// around them.
func (g *DataGrid) load(first, last int) {
	ld, ok := g.src.(Loader)
	if !ok || first >= last {
		return
	}
	page := last - first
	from, to := max(first-page, 0), min(last+page, g.src.Len())
	missing := false
	for r := first; r < last; r++ {
		if !ld.Loaded(r) {
			missing = true
			break
		}
	}
	if missing {
		ld.Load(from, to)
	}
}

func strokeRect(ops *op.Ops, r image.Rectangle, w int, c color.NRGBA) {
	paint.FillShape(ops, c, clip.Stroke{Path: clip.Rect(r.Inset(w / 2)).Path(), Width: float32(w)}.Op())
}

func strokeRRect(ops *op.Ops, r image.Rectangle, rad, w int, c color.NRGBA) {
	paint.FillShape(ops, c, clip.Stroke{Path: clip.UniformRRect(r, rad).Path(ops), Width: float32(w)}.Op())
}

// paintShadow fades c from opaque-ish at r's left to clear at its right.
func paintShadow(ops *op.Ops, r image.Rectangle, c color.NRGBA) {
	n := max(r.Dx(), 1)
	for i := range n {
		cc := c
		cc.A = uint8(int(c.A) * (n - i) / (n * 2))
		paint.FillShape(ops, cc, clip.Rect(image.Rect(r.Min.X+i, r.Min.Y, r.Min.X+i+1, r.Max.Y)).Op())
	}
}
