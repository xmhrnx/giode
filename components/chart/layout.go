package chart

import (
	"image"
	"image/color"
	"math"
	"strings"
	"time"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
)

type hoverState struct {
	active        bool
	pos           image.Point
	series, index int
	// x is the hovered x value on a numeric chart.
	x float64
}

type brushState struct {
	active   bool
	from, to int // pixels
	panning  bool
	panFrom  float64
	view0    viewRange
	lastTap  time.Time
	moved    bool
	pressPos image.Point
}

// viewRange is the zoomed x range; dlo and dhi are the data's.
type viewRange struct {
	set      bool
	lo, hi   float64
	dlo, dhi float64
}

type legendHit struct {
	r     image.Rectangle
	index int
}

func clipRect(gtx layout.Context, r image.Rectangle) clip.Stack {
	return clip.Rect(r).Push(gtx.Ops)
}

// Layout draws the chart in the width it's given.
func (c *Chart) Layout(gtx layout.Context) layout.Dimensions {
	w := gtx.Constraints.Max.X
	if w >= 1<<20 {
		w = gtx.Dp(480)
	}
	h := gtx.Dp(unit.Dp(c.height))
	if gtx.Constraints.Max.Y < 1<<20 {
		h = min(h, gtx.Constraints.Max.Y)
	}
	h = max(h, gtx.Constraints.Min.Y)
	c.size = image.Pt(w, h)
	elements.TrackPosition(&c.origin)
	if !kit.HandleInput(func(gtx layout.Context) bool { return c.update(gtx) }) {
		kit.OpaqueInput()
	}
	c.update(gtx)
	pal := themePalette()
	t := c.progress(gtx)

	r := image.Rectangle{Max: c.size}
	defer clip.Rect(r).Push(gtx.Ops).Pop()
	gap := gtx.Dp(8)
	if c.title != "" {
		l := text(gtx, c.title, 15, font.SemiBold)
		x := 0
		if kit.RTL(gtx) {
			x = w - l.size().X
		}
		l.draw(gtx, image.Pt(x, 0), pal.text)
		r.Min.Y += l.size().Y + gap
	}
	if c.showLegend() {
		lh := c.layoutLegend(gtx, r, pal, c.legend == LegendBottom)
		if c.legend == LegendBottom {
			r.Max.Y -= lh + gap
		} else {
			r.Min.Y += lh + gap
		}
	}
	// Input under the marks.
	area := clip.Rect(image.Rectangle{Max: c.size}).Push(gtx.Ops)
	event.Op(gtx.Ops, &c.tag)
	if c.zoomable {
		c.scroll.Add(gtx.Ops)
	}
	area.Pop()
	kit.BlockDrag(gtx, c.size)

	var f *frame
	switch c.kind {
	case Pie, Donut:
		c.layoutPie(gtx, r, pal, t)
	case Funnel:
		c.layoutFunnel(gtx, r, pal, t)
	case Gauge:
		c.layoutGauge(gtx, r, pal, t)
	case Heatmap:
		c.layoutHeatmap(gtx, r, pal)
	default:
		f = c.layoutCartesian(gtx, r, pal, t)
		c.frame = f
		if f != nil && !f.plot.Empty() {
			c.drawHover(gtx, f, pal)
			c.drawBrush(gtx, f, pal)
		}
	}
	c.drawTooltip(gtx, pal)
	elements.RecordSemantic(gtx, elements.SemanticNode{Kind: elements.SemanticImage, Label: c.Summary(), Target: c}, c.size)
	c.AnchorContext(gtx, c.size)
	return layout.Dimensions{Size: c.size}
}

func (c *Chart) showLegend() bool {
	switch c.legend {
	case LegendNone:
		return false
	case LegendTop, LegendBottom:
		return true
	}
	switch c.kind {
	case Pie, Donut:
		return true
	case Gauge, Heatmap, Funnel:
		return false
	}
	return len(c.series) > 1
}

// legendEntries are the legend's names: categories for pies, series
// otherwise.
func (c *Chart) legendEntries() []string {
	if c.kind == Pie || c.kind == Donut {
		return c.categories
	}
	names := make([]string, len(c.series))
	for i, s := range c.series {
		names[i] = s.Name
	}
	return names
}

func (c *Chart) entryHidden(i int) bool {
	if c.kind == Pie || c.kind == Donut {
		return c.hiddenCats[i]
	}
	return i < len(c.series) && c.series[i].Hidden
}

func (c *Chart) entryColor(i int) color.NRGBA {
	if c.kind == Pie || c.kind == Donut {
		return categoryColor(i)
	}
	return c.seriesColor(i)
}

// layoutLegend draws the legend, wrapping, and returns its height.
func (c *Chart) layoutLegend(gtx layout.Context, r image.Rectangle, pal palette, bottom bool) int {
	names := c.legendEntries()
	dot := gtx.Dp(10)
	gap := gtx.Dp(14)
	rtl := kit.RTL(gtx)
	type item struct {
		l label
		w int
	}
	items := make([]item, len(names))
	lineH := 0
	for i, n := range names {
		l := text(gtx, n, 12, font.Normal)
		items[i] = item{l, dot + gtx.Dp(6) + l.size().X}
		lineH = max(lineH, l.size().Y)
	}
	// Rows.
	var rows [][]int
	x := 0
	for i, it := range items {
		if len(rows) == 0 || x > 0 && x+it.w > r.Dx() {
			rows = append(rows, nil)
			x = 0
		}
		rows[len(rows)-1] = append(rows[len(rows)-1], i)
		x += it.w + gap
	}
	height := len(rows) * (lineH + gtx.Dp(4))
	y := r.Min.Y
	if bottom {
		y = r.Max.Y - height
	}
	c.legendAt = c.legendAt[:0]
	for _, row := range rows {
		x := r.Min.X
		if rtl {
			x = r.Max.X
		}
		for _, i := range row {
			it := items[i]
			x0 := x
			if rtl {
				x0 = x - it.w
			}
			col := c.entryColor(i)
			tc := pal.text
			if c.entryHidden(i) {
				col, tc = alpha(col, 0x50), alpha(pal.muted, 0x90)
			}
			mid := y + lineH/2
			fillRound(gtx, image.Rect(x0, mid-dot/2, x0+dot, mid+dot/2), dot/4, col)
			it.l.draw(gtx, image.Pt(x0+dot+gtx.Dp(6), y), tc)
			c.legendAt = append(c.legendAt, legendHit{image.Rect(x0, y, x0+it.w, y+lineH), i})
			if rtl {
				x -= it.w + gap
			} else {
				x += it.w + gap
			}
		}
		y += lineH + gtx.Dp(4)
	}
	return height
}

// progress is how far the data change animation is, eased.
func (c *Chart) progress(gtx layout.Context) float64 {
	if !c.animating || !c.animate || kit.ReducedMotion() {
		c.animating, c.prev = false, nil
		return 1
	}
	if c.animStart.IsZero() {
		c.animStart = gtx.Now
	}
	t := gtx.Now.Sub(c.animStart).Seconds() / 0.35
	if t >= 1 {
		c.animating, c.prev = false, nil
		return 1
	}
	kit.RequestFrame(gtx)
	return 1 - math.Pow(1-t, 3)
}

// prevSeries is series i's data before the change: by name, or index.
func (c *Chart) prevSeries(i int) *Series {
	s := &c.series[i]
	for k := range c.prev {
		if s.Name != "" && c.prev[k].Name == s.Name {
			return &c.prev[k]
		}
	}
	if s.Name == "" && i < len(c.prev) {
		return &c.prev[i]
	}
	return nil
}

// animValues are series i's values on the way from the previous ones.
func (c *Chart) animValues(i int, t float64) []float64 {
	v := c.series[i].Values
	if t >= 1 || len(v) == 0 {
		return v
	}
	out := make([]float64, len(v))
	p := c.prevSeries(i)
	for k, x := range v {
		from := 0.0
		if p != nil && k < len(p.Values) && !math.IsNaN(p.Values[k]) {
			from = p.Values[k]
		}
		out[k] = from + (x-from)*t
	}
	return out
}

// animPoints are series i's points on the way: their y from the
// previous ones' where the xs match.
func (c *Chart) animPoints(i int, t float64) []Point {
	pts := c.series[i].Points
	if t >= 1 || len(pts) == 0 {
		return pts
	}
	p := c.prevSeries(i)
	if p == nil || len(p.Points) != len(pts) {
		return pts
	}
	out := make([]Point, len(pts))
	for k, q := range pts {
		out[k] = q
		if p.Points[k].X == q.X {
			out[k].Y = p.Points[k].Y + (q.Y-p.Points[k].Y)*t
		}
	}
	return out
}

// update handles pointer input: hover, clicks, the legend, brushing,
// zooming and panning.
func (c *Chart) update(gtx layout.Context) bool {
	changed := false
	if c.zoomable && c.frame != nil && !c.frame.categorical && c.view.dhi > c.view.dlo {
		if dy := c.scroll.Update(gtx.Metric, gtx.Source, gtx.Now, gesture.Vertical,
			pointer.ScrollRange{}, pointer.ScrollRange{Min: -1 << 20, Max: 1 << 20}); dy != 0 {
			c.zoom(float64(dy), c.hover.pos)
			changed = true
		}
	}
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &c.tag, Kinds: pointer.Move | pointer.Press | pointer.Drag | pointer.Release | pointer.Leave | pointer.Cancel})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		changed = true
		p := e.Position.Round()
		switch e.Kind {
		case pointer.Move:
			c.setHover(p)
		case pointer.Leave, pointer.Cancel:
			c.hover.active = false
			c.brush.active, c.brush.panning = false, false
		case pointer.Press:
			c.setHover(p)
			c.brush.pressPos, c.brush.moved = p, false
			for _, lh := range c.legendAt {
				if p.In(lh.r) {
					c.toggleEntry(lh.index)
					return true
				}
			}
			if gtx.Now.Sub(c.brush.lastTap) < 350*time.Millisecond && c.view.set {
				c.view = viewRange{}
			}
			c.brush.lastTap = gtx.Now
			if c.frame != nil && p.In(c.frame.plot) {
				switch {
				case c.onBrush != nil && e.Source == pointer.Mouse:
					c.brush.active, c.brush.from, c.brush.to = true, p.X, p.X
				case c.zoomable && c.view.set:
					c.brush.panning = true
					c.brush.panFrom = float64(p.X)
					c.brush.view0 = c.view
				}
			}
		case pointer.Drag:
			if abs(p.X-c.brush.pressPos.X) > gtx.Dp(4) {
				c.brush.moved = true
			}
			switch {
			case c.brush.active:
				c.brush.to = p.X
			case c.brush.panning && c.frame != nil:
				f := c.frame
				v0 := c.brush.view0
				dx := (float64(p.X) - c.brush.panFrom) / (f.x.to - f.x.from) * (v0.hi - v0.lo)
				lo, hi := v0.lo-dx, v0.hi-dx
				if lo < v0.dlo {
					lo, hi = v0.dlo, v0.dlo+(v0.hi-v0.lo)
				}
				if hi > v0.dhi {
					lo, hi = v0.dhi-(v0.hi-v0.lo), v0.dhi
				}
				c.view.lo, c.view.hi = lo, hi
			}
			c.setHover(p)
		case pointer.Release:
			if c.brush.active {
				c.brush.active = false
				if abs(c.brush.to-c.brush.from) > gtx.Dp(6) && c.frame != nil {
					a, b := c.dataX(c.brush.from), c.dataX(c.brush.to)
					c.onBrush(math.Min(a, b), math.Max(a, b))
					break
				}
			}
			c.brush.panning = false
			if !c.brush.moved {
				c.click(p)
			}
		}
	}
	return changed
}

// dataX is the x value at pixel x.
func (c *Chart) dataX(x int) float64 {
	f := c.frame
	v := f.x.value(float64(x))
	if f.categorical {
		v = math.Max(v-0.5, 0)
	}
	return v
}

// zoom zooms the x range around pixel p by the wheel's dy.
func (c *Chart) zoom(dy float64, p image.Point) {
	f := c.frame
	lo, hi := f.x.min, f.x.max
	factor := math.Exp(dy * 0.003)
	at := f.x.value(float64(p.X))
	if !p.In(f.plot) {
		at = (lo + hi) / 2
	}
	nlo, nhi := at-(at-lo)*factor, at+(hi-at)*factor
	span := c.view.dhi - c.view.dlo
	if nhi-nlo >= span {
		c.view = viewRange{}
		return
	}
	if nhi-nlo < span/1e4 {
		return
	}
	nlo, nhi = math.Max(nlo, c.view.dlo), math.Min(nhi, c.view.dhi)
	c.view.set, c.view.lo, c.view.hi = true, nlo, nhi
}

func (c *Chart) toggleEntry(i int) {
	if c.kind == Pie || c.kind == Donut {
		if c.hiddenCats == nil {
			c.hiddenCats = map[int]bool{}
		}
		c.hiddenCats[i] = !c.hiddenCats[i]
		return
	}
	if i < len(c.series) {
		c.series[i].Hidden = !c.series[i].Hidden
	}
}

// setHover finds what's under p.
func (c *Chart) setHover(p image.Point) {
	c.hover.pos = p
	c.hover.active = false
	c.hover.series, c.hover.index = -1, -1
	switch c.kind {
	case Pie, Donut:
		c.hoverPie(p)
		return
	case Funnel, Heatmap, Gauge:
		c.hoverShapes(p)
		return
	}
	f := c.frame
	if f == nil || !p.In(f.plot.Inset(-4)) {
		return
	}
	// A mark under the pointer.
	for k := len(f.hits) - 1; k >= 0; k-- {
		h := f.hits[k]
		if p.In(h.r) && (c.kindOf(&c.series[h.series]) == Scatter || c.kindOf(&c.series[h.series]) == Bar || c.kind == HorizontalBar) {
			c.hover = hoverState{active: true, pos: p, series: h.series, index: h.index}
			return
		}
	}
	if f.horizontal {
		bandH := float64(f.plot.Dy()) / float64(max(f.n, 1))
		i := int(float64(p.Y-f.plot.Min.Y) / bandH)
		if i >= 0 && i < f.n {
			c.hover = hoverState{active: true, pos: p, series: -1, index: i}
		}
		return
	}
	if f.categorical {
		v := f.x.value(float64(p.X))
		i := int(math.Floor(v))
		if i >= 0 && i < f.n {
			c.hover = hoverState{active: true, pos: p, series: -1, index: i}
		}
		return
	}
	// The nearest point's index on the first line or area series.
	x := f.x.value(float64(p.X))
	best, bestD := -1, math.Inf(1)
	bestS := -1
	for _, i := range c.visible() {
		s := &c.series[i]
		if k := c.kindOf(s); k != Line && k != Area {
			continue
		}
		pts := s.Points
		j := sortSearch(pts, x)
		for _, k := range []int{j - 1, j} {
			if k >= 0 && k < len(pts) {
				if d := math.Abs(pts[k].X - x); d < bestD {
					best, bestD, bestS = k, d, i
				}
			}
		}
		break
	}
	if best >= 0 {
		c.hover = hoverState{active: true, pos: p, series: bestS, index: best, x: c.series[bestS].Points[best].X}
	}
}

func sortSearch(pts []Point, x float64) int {
	lo, hi := 0, len(pts)
	for lo < hi {
		m := (lo + hi) / 2
		if pts[m].X < x {
			lo = m + 1
		} else {
			hi = m
		}
	}
	return lo
}

// click selects what's under p.
func (c *Chart) click(p image.Point) {
	c.setHover(p)
	if !c.hover.active {
		c.selected = [2]int{-1, -1}
		return
	}
	s, i := c.hover.series, c.hover.index
	if s < 0 {
		// A category: the series nearest the pointer there.
		s = c.nearestSeries(p, i)
	}
	c.selected = [2]int{s, i}
	if c.onSelect != nil && s >= 0 {
		c.onSelect(s, i)
	}
}

func (c *Chart) nearestSeries(p image.Point, i int) int {
	f := c.frame
	if f == nil {
		return -1
	}
	best, bd := -1, math.Inf(1)
	for _, h := range f.hits {
		if h.index != i {
			continue
		}
		cy := float64(h.r.Min.Y+h.r.Max.Y) / 2
		if p.In(h.r) {
			return h.series
		}
		if d := math.Abs(cy - float64(p.Y)); d < bd {
			best, bd = h.series, d
		}
	}
	return best
}

// drawHover draws the crosshair at the hovered category or point.
func (c *Chart) drawHover(gtx layout.Context, f *frame, pal palette) {
	if !c.hover.active || c.hover.index < 0 || f.horizontal {
		return
	}
	var x float64
	if f.categorical {
		x = f.xPx(float64(c.hover.index))
		if c.hasKind(Bar) {
			return // bars dim instead
		}
	} else if c.hover.series >= 0 && c.hover.index < len(c.series[c.hover.series].Points) {
		if c.kindOf(&c.series[c.hover.series]) == Scatter {
			return
		}
		x = f.x.px(c.series[c.hover.series].Points[c.hover.index].X)
	} else {
		return
	}
	thin := max(gtx.Dp(1), 1)
	fill(gtx, image.Rect(int(x), f.plot.Min.Y, int(x)+thin, f.plot.Max.Y), alpha(pal.muted, 0x90))
}

func (c *Chart) hasKind(k Kind) bool {
	for _, i := range c.visible() {
		if c.kindOf(&c.series[i]) == k {
			return true
		}
	}
	return false
}

func (c *Chart) drawBrush(gtx layout.Context, f *frame, pal palette) {
	if !c.brush.active {
		return
	}
	a, b := min(c.brush.from, c.brush.to), max(c.brush.from, c.brush.to)
	a, b = max(a, f.plot.Min.X), min(b, f.plot.Max.X)
	col := c.seriesColor(0)
	fill(gtx, image.Rect(a, f.plot.Min.Y, b, f.plot.Max.Y), alpha(col, 0x30))
}

// tooltipLines are the hovered mark's tooltip: a heading and a row per
// series.
type tipRow struct {
	color color.NRGBA
	name  string
	value string
}

func (c *Chart) tooltip() (string, []tipRow) {
	h := c.hover
	if !h.active {
		return "", nil
	}
	switch c.kind {
	case Pie, Donut, Funnel:
		if len(c.series) == 0 || h.index < 0 || h.index >= len(c.series[0].Values) {
			return "", nil
		}
		v := c.series[0].Values[h.index]
		name := ""
		if h.index < len(c.categories) {
			name = c.categories[h.index]
		}
		total := 0.0
		for k, x := range c.series[0].Values {
			if !c.hiddenCats[k] && x > 0 {
				total += x
			}
		}
		val := c.y.format(v)
		if total > 0 && c.kind != Funnel {
			val += " · " + trimFloat(v/total*100, 1) + "%"
		}
		if c.kind == Funnel && len(c.series[0].Values) > 0 && c.series[0].Values[0] != 0 {
			val += " · " + trimFloat(v/c.series[0].Values[0]*100, 1) + "%"
		}
		return name, []tipRow{{categoryColor(h.index), c.series[0].Name, val}}
	case Heatmap:
		if h.series < 0 || h.series >= len(c.heat) || h.index >= len(c.heat[h.series]) {
			return "", nil
		}
		row, col := "", ""
		if h.series < len(c.heatRows) {
			row = c.heatRows[h.series]
		}
		if h.index < len(c.heatCols) {
			col = c.heatCols[h.index]
		}
		return strings.TrimSpace(row + " · " + col), []tipRow{{categoryColor(0), "", c.y.format(c.heat[h.series][h.index])}}
	case Gauge:
		return "", nil
	}
	f := c.frame
	if f == nil {
		return "", nil
	}
	if f.categorical {
		name := ""
		if h.index < len(c.categories) {
			name = c.categories[h.index]
		}
		var rows []tipRow
		for _, i := range c.visible() {
			s := &c.series[i]
			if h.index >= len(s.Values) || math.IsNaN(s.Values[h.index]) {
				continue
			}
			a := &c.y
			if s.Axis == 1 {
				a = &c.y2
			}
			rows = append(rows, tipRow{c.seriesColor(i), s.Name, a.format(s.Values[h.index])})
		}
		return name, rows
	}
	if h.series < 0 || h.index >= len(c.series[h.series].Points) {
		return "", nil
	}
	p := c.series[h.series].Points[h.index]
	head := c.formatX(p.X, timeUnit{format: "Jan 2, 2006 15:04"})
	if p.Label != "" {
		head = p.Label
	}
	var rows []tipRow
	if c.kindOf(&c.series[h.series]) == Scatter {
		s := &c.series[h.series]
		return head, []tipRow{{c.seriesColor(h.series), s.Name, c.y.format(p.Y)}}
	}
	for _, i := range c.visible() {
		s := &c.series[i]
		if k := c.kindOf(s); k == Scatter {
			continue
		}
		j := sortSearch(s.Points, p.X)
		if j >= len(s.Points) || s.Points[j].X != p.X {
			continue
		}
		a := &c.y
		if s.Axis == 1 {
			a = &c.y2
		}
		rows = append(rows, tipRow{c.seriesColor(i), s.Name, a.format(s.Points[j].Y)})
	}
	return head, rows
}

// drawTooltip draws the tooltip beside the pointer, inside the chart.
func (c *Chart) drawTooltip(gtx layout.Context, pal palette) {
	head, rows := c.tooltip()
	if head == "" && len(rows) == 0 {
		return
	}
	pad := gtx.Dp(8)
	dot := gtx.Dp(8)
	var hl label
	w, h := 0, pad
	if head != "" {
		hl = text(gtx, head, 12, font.SemiBold)
		w = hl.size().X
		h += hl.size().Y + gtx.Dp(4)
	}
	type rl struct{ n, v label }
	lines := make([]rl, len(rows))
	for i, r := range rows {
		lines[i] = rl{text(gtx, r.name, 12, font.Normal), text(gtx, r.value, 12, font.SemiBold)}
		w = max(w, dot+gtx.Dp(6)+lines[i].n.size().X+gtx.Dp(12)+lines[i].v.size().X)
		h += lines[i].n.size().Y + gtx.Dp(2)
	}
	w += 2 * pad
	h += pad - gtx.Dp(2)
	p := c.hover.pos.Add(image.Pt(gtx.Dp(14), gtx.Dp(14)))
	if p.X+w > c.size.X {
		p.X = c.hover.pos.X - w - gtx.Dp(14)
	}
	if p.Y+h > c.size.Y {
		p.Y = max(c.size.Y-h, 0)
	}
	p.X = max(p.X, 0)
	box := image.Rectangle{Min: p, Max: p.Add(image.Pt(w, h))}
	fillRound(gtx, box.Inset(-1), gtx.Dp(7), pal.border)
	fillRound(gtx, box, gtx.Dp(6), pal.surface)
	y := p.Y + pad
	if head != "" {
		hl.draw(gtx, image.Pt(p.X+pad, y), pal.text)
		y += hl.size().Y + gtx.Dp(4)
	}
	for i, r := range rows {
		l := lines[i]
		mid := y + l.n.size().Y/2
		fillRound(gtx, image.Rect(p.X+pad, mid-dot/2, p.X+pad+dot, mid+dot/2), dot/4, r.color)
		l.n.draw(gtx, image.Pt(p.X+pad+dot+gtx.Dp(6), y), pal.muted)
		l.v.drawAt(gtx, image.Pt(box.Max.X-pad, y), 1, 0, pal.text)
		y += l.n.size().Y + gtx.Dp(2)
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

var _ = f32.Point{}
