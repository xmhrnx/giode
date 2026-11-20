package chart

import (
	"image"
	"image/color"
	"math"
	"sort"
	"time"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/components/kit"
)

// hitRect is a drawn mark: a bar, a point.
type hitRect struct {
	r             image.Rectangle
	series, index int
}

// frame is a cartesian chart's geometry for a frame.
type frame struct {
	plot        image.Rectangle
	x           scale
	y, y2       scale
	categorical bool
	n           int // categories
	rtl         bool
	horizontal  bool
	hits        []hitRect
	// xs are the data x values of hoverable positions, sorted, for
	// numeric charts.
	xs []float64
}

// xPx maps an x value (a category index's center on a categorical
// chart) to pixels.
func (f *frame) xPx(v float64) float64 {
	if f.categorical {
		start, w := f.x.band(0)
		_ = start
		return f.x.from + (v+0.5-f.x.min)*w*float64(f.x.categories)/(f.x.max-f.x.min)
	}
	return f.x.px(v)
}

// bandW is a category's width in pixels.
func (f *frame) bandW() float64 {
	return math.Abs(f.x.to-f.x.from) / math.Max(f.x.max-f.x.min, 1e-9)
}

// visible returns the series to draw.
func (c *Chart) visible() []int {
	var out []int
	for i := range c.series {
		if !c.series[i].Hidden {
			out = append(out, i)
		}
	}
	return out
}

// categoryCount is the number of categories.
func (c *Chart) categoryCount() int {
	n := len(c.categories)
	for _, s := range c.series {
		n = max(n, len(s.Values))
	}
	return n
}

func (c *Chart) isCategorical() bool {
	if len(c.categories) > 0 {
		return true
	}
	for _, s := range c.series {
		if len(s.Values) > 0 {
			return true
		}
	}
	return false
}

// stackKey is the stack a series is drawn in: its Stack, or its own.
func stackKey(s *Series, i int) string {
	if s.Stack != "" {
		return "s:" + s.Stack
	}
	return "i:" + itoa(i)
}

// yRange returns an axis' data range: stacked series added up.
func (c *Chart) yRange(axis int, vals [][]float64, pts [][]Point) (lo, hi float64, any bool) {
	lo, hi = math.Inf(1), math.Inf(-1)
	pos := map[string][]float64{}
	neg := map[string][]float64{}
	hasBar := false
	for _, i := range c.visible() {
		s := &c.series[i]
		if s.Axis != axis {
			continue
		}
		k := c.kindOf(s)
		if k == Bar || k == HorizontalBar || k == Area {
			hasBar = hasBar || k != Area
		}
		any = true
		if len(vals[i]) > 0 {
			key := stackKey(s, i)
			if pos[key] == nil {
				pos[key] = make([]float64, c.categoryCount())
				neg[key] = make([]float64, c.categoryCount())
			}
			for j, v := range vals[i] {
				if math.IsNaN(v) {
					continue
				}
				if v >= 0 {
					pos[key][j] += v
					hi = math.Max(hi, pos[key][j])
					lo = math.Min(lo, pos[key][j])
				} else {
					neg[key][j] += v
					lo = math.Min(lo, neg[key][j])
					hi = math.Max(hi, neg[key][j])
				}
			}
			if s.Stack == "" {
				for _, v := range vals[i] {
					if !math.IsNaN(v) {
						lo, hi = math.Min(lo, v), math.Max(hi, v)
					}
				}
			}
		}
		for _, p := range pts[i] {
			if c.view.set && (p.X < c.view.lo || p.X > c.view.hi) {
				continue
			}
			lo, hi = math.Min(lo, p.Y), math.Max(hi, p.Y)
		}
	}
	if !any || math.IsInf(lo, 0) {
		return 0, 1, any
	}
	if hasBar || c.kind == Area {
		lo, hi = math.Min(lo, 0), math.Max(hi, 0)
	}
	return lo, hi, true
}

func (a *Axis) format(v float64) string {
	if a.Format != nil {
		return a.Format(v)
	}
	return FormatNumber(v)
}

func (c *Chart) formatX(v float64, u timeUnit) string {
	if c.x.Format != nil {
		return c.x.Format(v)
	}
	if c.x.Time {
		loc := c.x.Location
		if loc == nil {
			loc = time.Local
		}
		f := u.format
		if f == "" {
			f = "Jan 2 15:04"
		}
		return time.Unix(int64(v), 0).In(loc).Format(f)
	}
	return FormatNumber(v)
}

// layoutCartesian draws an x-y or category chart in r.
func (c *Chart) layoutCartesian(gtx layout.Context, r image.Rectangle, pal palette, t float64) *frame {
	f := &frame{rtl: kit.RTL(gtx), horizontal: c.kind == HorizontalBar}
	small := unit.Sp(11)
	gap := gtx.Dp(6)
	vals := make([][]float64, len(c.series))
	pts := make([][]Point, len(c.series))
	for i := range c.series {
		vals[i] = c.animValues(i, t)
		pts[i] = c.animPoints(i, t)
	}

	// The x domain.
	f.categorical = c.isCategorical()
	var xlo, xhi float64
	if f.categorical {
		f.n = c.categoryCount()
		xlo, xhi = 0, float64(max(f.n, 1))
	} else {
		xlo, xhi = math.Inf(1), math.Inf(-1)
		for _, i := range c.visible() {
			for _, p := range pts[i] {
				xlo, xhi = math.Min(xlo, p.X), math.Max(xhi, p.X)
			}
		}
		if math.IsInf(xlo, 0) {
			xlo, xhi = 0, 1
		}
		if xlo == xhi {
			xlo, xhi = xlo-1, xhi+1
		}
	}
	if c.x.Fixed {
		xlo, xhi = c.x.Min, c.x.Max
	}
	dlo, dhi := xlo, xhi
	if c.view.set {
		xlo, xhi = math.Max(c.view.lo, dlo), math.Min(c.view.hi, dhi)
		if xhi <= xlo {
			c.view = viewRange{}
			xlo, xhi = dlo, dhi
		}
	}
	c.view.dlo, c.view.dhi = dlo, dhi

	// The value axes.
	ylo, yhi, _ := c.yRange(0, vals, pts)
	y2lo, y2hi, hasY2 := c.yRange(1, vals, pts)
	if c.y.Fixed {
		ylo, yhi = c.y.Min, c.y.Max
	}
	if c.y2.Fixed {
		y2lo, y2hi = c.y2.Min, c.y2.Max
	}
	valueLen := r.Dy()
	if f.horizontal {
		valueLen = r.Dx()
	}
	nt := max(valueLen/gtx.Dp(40), 2)
	yt, ymin, ymax := niceTicks(ylo, yhi, nt)
	if c.y.Fixed {
		ymin, ymax = ylo, yhi
		yt = filterTicks(yt, ymin, ymax)
	}
	var y2t []float64
	var y2min, y2max float64
	if hasY2 {
		y2t, y2min, y2max = niceTicks(y2lo, y2hi, nt)
		if c.y2.Fixed {
			y2min, y2max = y2lo, y2hi
			y2t = filterTicks(y2t, y2min, y2max)
		}
	}

	// Margins from the labels' sizes.
	lineH := text(gtx, "0", small, font.Normal).size().Y
	yLabels := make([]label, len(yt))
	yw := 0
	for i, v := range yt {
		yLabels[i] = text(gtx, c.y.format(v), small, font.Normal)
		yw = max(yw, yLabels[i].size().X)
	}
	y2Labels := make([]label, len(y2t))
	y2w := 0
	for i, v := range y2t {
		y2Labels[i] = text(gtx, c.y2.format(v), small, font.Normal)
		y2w = max(y2w, y2Labels[i].size().X)
	}
	left, right := yw+gap, gap
	if hasY2 {
		right = y2w + gap
	}
	if c.y.Hide {
		left = gap
	}
	bottom := lineH + gap
	if c.x.Title != "" {
		bottom += lineH + gap
	}
	if c.y.Title != "" {
		left += lineH + gap
	}
	if f.horizontal {
		// Category labels down the side, values along the bottom.
		cw := 0
		for _, s := range c.categories {
			cw = max(cw, text(gtx, s, small, font.Normal).size().X)
		}
		left = min(cw+gap, r.Dx()/3)
		right = gap + yw/2
	}
	if f.rtl {
		left, right = right, left
	}
	plot := image.Rect(r.Min.X+left, r.Min.Y+lineH/2, r.Max.X-right, r.Max.Y-bottom)
	if plot.Dx() < 10 || plot.Dy() < 10 {
		return f
	}
	f.plot = plot
	c.plot = plot

	if f.horizontal {
		c.drawHorizontal(gtx, f, pal, vals, yt, yLabels, ymin, ymax, small)
		return f
	}

	// Scales.
	f.x = scale{min: xlo, max: xhi, from: float64(plot.Min.X), to: float64(plot.Max.X)}
	if f.rtl {
		f.x.from, f.x.to = f.x.to, f.x.from
	}
	if f.categorical {
		f.x.categories = f.n
	}
	f.y = scale{min: ymin, max: ymax, from: float64(plot.Max.Y), to: float64(plot.Min.Y), ticks: yt}
	f.y2 = scale{min: y2min, max: y2max, from: float64(plot.Max.Y), to: float64(plot.Min.Y), ticks: y2t}

	// Grid and value labels.
	thin := float32(max(gtx.Dp(1), 1))
	for i, v := range yt {
		y := int(math.Round(f.y.px(v)))
		if !c.y.NoGrid {
			fill(gtx, image.Rect(plot.Min.X, y, plot.Max.X, y+int(thin)), pal.grid)
		}
		if !c.y.Hide {
			if f.rtl {
				yLabels[i].drawAt(gtx, image.Pt(plot.Max.X+gap, y), 0, 0.5, pal.muted)
			} else {
				yLabels[i].drawAt(gtx, image.Pt(plot.Min.X-gap, y), 1, 0.5, pal.muted)
			}
		}
	}
	for i, v := range y2t {
		y := int(math.Round(f.y2.px(v)))
		if f.rtl {
			y2Labels[i].drawAt(gtx, image.Pt(plot.Min.X-gap, y), 1, 0.5, pal.muted)
		} else {
			y2Labels[i].drawAt(gtx, image.Pt(plot.Max.X+gap, y), 0, 0.5, pal.muted)
		}
	}
	// The baseline.
	if ymin <= 0 && ymax >= 0 {
		y := int(math.Round(f.y.px(0)))
		fill(gtx, image.Rect(plot.Min.X, y, plot.Max.X, y+int(thin)), pal.axis)
	}

	// X labels.
	var unitT timeUnit
	labelY := plot.Max.Y + gap/2
	if !c.x.Hide {
		if f.categorical {
			lo, hi := int(math.Floor(xlo)), int(math.Ceil(xhi))
			maxW := 0
			labels := make([]label, 0, hi-lo)
			for i := lo; i < hi && i < f.n; i++ {
				name := ""
				if i < len(c.categories) {
					name = c.categories[i]
				}
				l := text(gtx, name, small, font.Normal)
				labels = append(labels, l)
				maxW = max(maxW, l.size().X)
			}
			step := max(int(math.Ceil(float64(maxW+gap)/math.Max(f.bandW(), 1))), 1)
			for k, l := range labels {
				i := lo + k
				if i%step != 0 {
					continue
				}
				l.drawAt(gtx, image.Pt(int(f.xPx(float64(i))), labelY), 0.5, 0, pal.muted)
			}
		} else {
			var xt []float64
			n := max(plot.Dx()/gtx.Dp(90), 2)
			if c.x.Time {
				loc := c.x.Location
				if loc == nil {
					loc = time.Local
				}
				xt, unitT = timeTicks(xlo, xhi, n, loc)
			} else {
				xt, _, _ = niceTicks(xlo, xhi, n)
			}
			last := image.Rectangle{}
			for _, v := range xt {
				if v < xlo || v > xhi {
					continue
				}
				x := int(math.Round(f.x.px(v)))
				if !c.x.NoGrid {
					fill(gtx, image.Rect(x, plot.Min.Y, x+int(thin), plot.Max.Y), alpha(pal.grid, 0x80))
				}
				l := text(gtx, c.formatX(v, unitT), small, font.Normal)
				rr := image.Rect(x-l.size().X/2, labelY, x+l.size().X/2, labelY+l.size().Y)
				if !last.Empty() && rr.Inset(-gap/2).Overlaps(last) {
					continue
				}
				last = l.drawAt(gtx, image.Pt(x, labelY), 0.5, 0, pal.muted)
			}
		}
	}
	c.xUnit = unitT
	if c.x.Title != "" {
		text(gtx, c.x.Title, small, font.SemiBold).drawAt(gtx, image.Pt((plot.Min.X+plot.Max.X)/2, r.Max.Y), 0.5, 1, pal.muted)
	}
	if c.y.Title != "" {
		l := text(gtx, c.y.Title, small, font.SemiBold)
		x := r.Min.X
		if f.rtl {
			x = r.Max.X - l.size().X
		}
		l.draw(gtx, image.Pt(x, r.Min.Y-lineH/2), pal.muted)
	}

	// The marks, clipped to the plot.
	cl := clipRect(gtx, plot.Inset(-gtx.Dp(4)))
	c.drawAreas(gtx, f, vals, pts)
	c.drawBars(gtx, f, vals)
	c.drawLines(gtx, f, vals, pts)
	c.drawScatter(gtx, f, pts)
	cl.Pop()
	return f
}

func filterTicks(ts []float64, lo, hi float64) []float64 {
	var out []float64
	for _, v := range ts {
		if v >= lo-1e-9 && v <= hi+1e-9 {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		out = []float64{lo, hi}
	}
	return out
}

// yScale is the scale of series s's axis.
func (f *frame) yScale(s *Series) scale {
	if s.Axis == 1 {
		return f.y2
	}
	return f.y
}

// stackBases returns, for series i, where each of its values starts:
// the sum of the series stacked under it.
func (c *Chart) stackBases(i int, vals [][]float64) []float64 {
	s := &c.series[i]
	base := make([]float64, len(vals[i]))
	if s.Stack == "" {
		return base
	}
	for _, j := range c.visible() {
		if j >= i {
			break
		}
		o := &c.series[j]
		if o.Stack != s.Stack || o.Axis != s.Axis || c.kindOf(o) != c.kindOf(s) {
			continue
		}
		for k := range base {
			if k < len(vals[j]) && !math.IsNaN(vals[j][k]) && (vals[j][k] >= 0) == (k >= len(vals[i]) || vals[i][k] >= 0) {
				base[k] += vals[j][k]
			}
		}
	}
	return base
}

func (c *Chart) drawBars(gtx layout.Context, f *frame, vals [][]float64) {
	var bars []int
	for _, i := range c.visible() {
		if c.kindOf(&c.series[i]) == Bar && len(vals[i]) > 0 {
			bars = append(bars, i)
		}
	}
	if len(bars) == 0 || !f.categorical {
		return
	}
	var slots []string
	slotOf := map[string]int{}
	for _, i := range bars {
		k := stackKey(&c.series[i], i)
		if _, ok := slotOf[k]; !ok {
			slotOf[k] = len(slots)
			slots = append(slots, k)
		}
	}
	bw := f.bandW()
	group := bw * 0.72
	slotW := group / float64(len(slots))
	pad := math.Min(slotW*0.12, float64(gtx.Dp(2)))
	rad := min(gtx.Dp(3), int(slotW/3))
	for _, i := range bars {
		s := &c.series[i]
		ys := f.yScale(s)
		col := c.seriesColor(i)
		base := c.stackBases(i, vals)
		slot := slotOf[stackKey(s, i)]
		for j, v := range vals[i] {
			if math.IsNaN(v) {
				continue
			}
			center := f.xPx(float64(j))
			x0 := center - group/2 + float64(slot)*slotW + pad
			x1 := x0 + slotW - 2*pad
			if f.rtl {
				x0 = center + group/2 - float64(slot+1)*slotW + pad
				x1 = x0 + slotW - 2*pad
			}
			y0, y1 := ys.px(base[j]), ys.px(base[j]+v)
			rr := image.Rect(int(math.Round(x0)), int(math.Round(math.Min(y0, y1))), int(math.Round(x1)), int(math.Round(math.Max(y0, y1))))
			if rr.Dy() == 0 && v != 0 {
				rr.Max.Y++
			}
			cc := col
			if c.hover.index >= 0 && c.hover.index != j && c.hover.active {
				cc = alpha(cc, 0xb0)
			}
			fillRound(gtx, rr, rad, cc)
			f.hits = append(f.hits, hitRect{rr, i, j})
		}
	}
}

// linePoints returns a line or area series' points in pixels, and their
// data indexes.
func (c *Chart) linePoints(gtx layout.Context, f *frame, i int, vals [][]float64, pts [][]Point) ([]f32.Point, []int) {
	s := &c.series[i]
	ys := f.yScale(s)
	var out []f32.Point
	var idx []int
	if f.categorical && len(vals[i]) > 0 {
		base := c.stackBases(i, vals)
		for j, v := range vals[i] {
			if math.IsNaN(v) {
				continue
			}
			out = append(out, pt(f.xPx(float64(j)), ys.px(base[j]+v)))
			idx = append(idx, j)
		}
		return out, idx
	}
	p := pts[i]
	// Only what's in view, with one point past each edge.
	lo := sort.Search(len(p), func(k int) bool { return p[k].X >= f.x.min })
	hi := sort.Search(len(p), func(k int) bool { return p[k].X > f.x.max })
	lo, hi = max(lo-1, 0), min(hi+1, len(p))
	view := p[lo:hi]
	if w := f.plot.Dx(); len(view) > 2*w {
		view = downsample(view, w)
	}
	for k, q := range view {
		out = append(out, pt(f.x.px(q.X), ys.px(q.Y)))
		idx = append(idx, lo+k)
	}
	return out, idx
}

func (c *Chart) drawAreas(gtx layout.Context, f *frame, vals [][]float64, pts [][]Point) {
	for _, i := range c.visible() {
		s := &c.series[i]
		if c.kindOf(s) != Area {
			continue
		}
		top, idx := c.linePoints(gtx, f, i, vals, pts)
		if len(top) < 2 {
			continue
		}
		ys := f.yScale(s)
		bottom := make([]f32.Point, len(top))
		base := c.stackBases(i, vals)
		for k, p := range top {
			b := 0.0
			if f.categorical && len(base) > idx[k] {
				b = base[idx[k]]
			}
			bottom[k] = f32.Pt(p.X, float32(ys.px(math.Max(b, ys.min))))
		}
		col := c.seriesColor(i)
		area(gtx, top, bottom, alpha(col, 0x40), c.smooth)
		polyline(gtx, top, float32(gtx.Dp(2)), col, c.smooth, s.Dashed)
		if at := c.hoverIndex(f, i); c.hover.active && at >= 0 {
			for k, p := range top {
				if idx[k] == at {
					circle(gtx, p, float32(gtx.Dp(7)), themePalette().bg)
					circle(gtx, p, float32(gtx.Dp(5)), col)
				}
			}
		}
	}
}

// hoverIndex is series i's hovered index: the hovered category, or its
// point nearest the hovered x.
func (c *Chart) hoverIndex(f *frame, i int) int {
	if !c.hover.active {
		return -1
	}
	if f.categorical {
		return c.hover.index
	}
	pts := c.series[i].Points
	j := sortSearch(pts, c.hover.x)
	best, bd := -1, math.Inf(1)
	for _, k := range []int{j - 1, j} {
		if k >= 0 && k < len(pts) {
			if d := math.Abs(pts[k].X - c.hover.x); d < bd {
				best, bd = k, d
			}
		}
	}
	return best
}

func (c *Chart) drawLines(gtx layout.Context, f *frame, vals [][]float64, pts [][]Point) {
	for _, i := range c.visible() {
		s := &c.series[i]
		if c.kindOf(s) != Line {
			continue
		}
		line, idx := c.linePoints(gtx, f, i, vals, pts)
		col := c.seriesColor(i)
		polyline(gtx, line, float32(gtx.Dp(2)), col, c.smooth, s.Dashed)
		// Points, when few enough to tell apart, or the hovered one.
		few := len(line) > 0 && float64(f.plot.Dx())/float64(len(line)) > float64(gtx.Dp(16))
		hoverAt := c.hoverIndex(f, i)
		for k, p := range line {
			hovered := c.hover.active && idx[k] == hoverAt
			if few || hovered {
				r := float32(gtx.Dp(3))
				if hovered {
					r = float32(gtx.Dp(5))
					circle(gtx, p, r+float32(gtx.Dp(2)), themePalette().bg)
				}
				circle(gtx, p, r, col)
			}
			hr := gtx.Dp(8)
			f.hits = append(f.hits, hitRect{image.Rect(int(p.X)-hr, int(p.Y)-hr, int(p.X)+hr, int(p.Y)+hr), i, idx[k]})
		}
	}
}

func (c *Chart) drawScatter(gtx layout.Context, f *frame, pts [][]Point) {
	maxSize := 0.0
	for _, i := range c.visible() {
		if c.kindOf(&c.series[i]) == Scatter {
			for _, p := range pts[i] {
				maxSize = math.Max(maxSize, p.Size)
			}
		}
	}
	for _, i := range c.visible() {
		s := &c.series[i]
		if c.kindOf(s) != Scatter {
			continue
		}
		ys := f.yScale(s)
		col := c.seriesColor(i)
		for k, p := range pts[i] {
			if p.X < f.x.min || p.X > f.x.max {
				continue
			}
			r := float64(gtx.Dp(4))
			if maxSize > 0 && p.Size > 0 {
				r = float64(gtx.Dp(3)) + math.Sqrt(p.Size/maxSize)*float64(gtx.Dp(11))
			}
			center := pt(f.x.px(p.X), ys.px(p.Y))
			a := uint8(0xc0)
			if maxSize > 0 {
				a = 0x90
			}
			if c.hover.active && c.hover.series == i && c.hover.index == k {
				a = 0xff
				r += float64(gtx.Dp(2))
			}
			circle(gtx, center, float32(r), alpha(col, a))
			ir := int(math.Max(r, float64(gtx.Dp(6))))
			f.hits = append(f.hits, hitRect{image.Rect(int(center.X)-ir, int(center.Y)-ir, int(center.X)+ir, int(center.Y)+ir), i, k})
		}
	}
}

// drawHorizontal draws a horizontal bar chart: categories down the
// side, values across.
func (c *Chart) drawHorizontal(gtx layout.Context, f *frame, pal palette, vals [][]float64, ticks []float64, tickLabels []label, vmin, vmax float64, small unit.Sp) {
	plot := f.plot
	gap := gtx.Dp(6)
	vs := scale{min: vmin, max: vmax, from: float64(plot.Min.X), to: float64(plot.Max.X)}
	if f.rtl {
		vs.from, vs.to = vs.to, vs.from
	}
	n := c.categoryCount()
	f.n = n
	f.categorical = true
	f.x = scale{min: 0, max: float64(max(n, 1)), from: float64(plot.Min.Y), to: float64(plot.Max.Y), categories: n}
	f.y = vs
	thin := max(gtx.Dp(1), 1)
	for i, v := range ticks {
		x := int(math.Round(vs.px(v)))
		if !c.y.NoGrid {
			fill(gtx, image.Rect(x, plot.Min.Y, x+thin, plot.Max.Y), pal.grid)
		}
		tickLabels[i].drawAt(gtx, image.Pt(x, plot.Max.Y+gap/2), 0.5, 0, pal.muted)
	}
	bandH := float64(plot.Dy()) / float64(max(n, 1))
	for j := 0; j < n; j++ {
		name := ""
		if j < len(c.categories) {
			name = c.categories[j]
		}
		y := plot.Min.Y + int(bandH*(float64(j)+0.5))
		l := text(gtx, name, small, font.Normal)
		if f.rtl {
			l.drawAt(gtx, image.Pt(plot.Max.X+gap, y), 0, 0.5, pal.muted)
		} else {
			l.drawAt(gtx, image.Pt(plot.Min.X-gap, y), 1, 0.5, pal.muted)
		}
	}
	var bars []int
	for _, i := range c.visible() {
		if len(vals[i]) > 0 {
			bars = append(bars, i)
		}
	}
	var slots []string
	slotOf := map[string]int{}
	for _, i := range bars {
		k := stackKey(&c.series[i], i)
		if _, ok := slotOf[k]; !ok {
			slotOf[k] = len(slots)
			slots = append(slots, k)
		}
	}
	group := bandH * 0.72
	slotH := group / float64(max(len(slots), 1))
	rad := min(gtx.Dp(3), int(slotH/3))
	for _, i := range bars {
		s := &c.series[i]
		col := c.seriesColor(i)
		base := c.stackBases(i, vals)
		slot := slotOf[stackKey(s, i)]
		for j, v := range vals[i] {
			if math.IsNaN(v) {
				continue
			}
			y0 := float64(plot.Min.Y) + bandH*float64(j) + (bandH-group)/2 + float64(slot)*slotH + 1
			y1 := y0 + slotH - 2
			x0, x1 := vs.px(base[j]), vs.px(base[j]+v)
			rr := image.Rect(int(math.Round(math.Min(x0, x1))), int(math.Round(y0)), int(math.Round(math.Max(x0, x1))), int(math.Round(y1)))
			cc := col
			if c.hover.active && c.hover.index >= 0 && c.hover.index != j {
				cc = alpha(cc, 0xb0)
			}
			fillRound(gtx, rr, rad, cc)
			f.hits = append(f.hits, hitRect{rr, i, j})
		}
	}
}

var _ = color.NRGBA{}
