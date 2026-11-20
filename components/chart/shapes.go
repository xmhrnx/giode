package chart

import (
	"image"
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// shapeHit is a hoverable shape: a funnel stage, a heatmap cell.
type shapeHit struct {
	r             image.Rectangle
	series, index int
}

// pieGeom is the last laid out pie's center and radii.
type pieGeom struct {
	center   f32.Point
	r0, r1   float32
	starts   []float64
	ends     []float64
	sliceIdx []int
}

func (c *Chart) layoutPie(gtx layout.Context, r image.Rectangle, pal palette, t float64) {
	c.pie = pieGeom{}
	if len(c.series) == 0 {
		return
	}
	vals := c.animValues(0, t)
	total := 0.0
	for i, v := range vals {
		if v > 0 && !c.hiddenCats[i] {
			total += v
		}
	}
	if total <= 0 {
		return
	}
	d := float32(min(r.Dx(), r.Dy()))
	center := f32.Pt(float32(r.Min.X+r.Dx()/2), float32(r.Min.Y+r.Dy()/2))
	r1 := d/2 - float32(gtx.Dp(6))
	var r0 float32
	if c.kind == Donut {
		r0 = r1 * 0.6
	}
	c.pie.center, c.pie.r0, c.pie.r1 = center, r0, r1
	a := -math.Pi / 2
	gap := 0.0
	if c.kind == Donut {
		gap = 0.012
	}
	rtl := kit.RTL(gtx)
	for i, v := range vals {
		if v <= 0 || c.hiddenCats[i] {
			continue
		}
		sweep := v / total * 2 * math.Pi
		a0, a1 := a, a+sweep
		if rtl {
			// Counterclockwise from the top.
			a0 = -math.Pi/2 - (a + sweep + math.Pi/2)
			a1 = a0 + sweep
		}
		c.pie.starts = append(c.pie.starts, a0)
		c.pie.ends = append(c.pie.ends, a1)
		c.pie.sliceIdx = append(c.pie.sliceIdx, i)
		cc := categoryColor(i)
		rr1 := r1
		if c.hover.active && c.hover.index == i {
			rr1 += float32(gtx.Dp(4))
		}
		fillArc(gtx, center, r0, rr1, a0+gap, a1-gap, cc)
		// A percentage on slices big enough for it.
		if sweep > 0.35 {
			mid := (a0 + a1) / 2
			lr := (r0 + r1) / 2
			if r0 == 0 {
				lr = r1 * 0.62
			}
			p := image.Pt(int(center.X+lr*float32(math.Cos(mid))), int(center.Y+lr*float32(math.Sin(mid))))
			text(gtx, trimFloat(v/total*100, 0)+"%", 12, font.SemiBold).drawAt(gtx, p, 0.5, 0.5, contrastOn(cc))
		}
		a += sweep
	}
	if c.kind == Donut {
		l := text(gtx, c.y.format(total), unit.Sp(float32(r0)/gtx.Metric.PxPerSp/2.2), font.Bold)
		l.drawAt(gtx, image.Pt(int(center.X), int(center.Y)), 0.5, 0.5, pal.text)
	}
}

// contrastOn is black or white, whichever reads on c.
func contrastOn(c color.NRGBA) color.NRGBA {
	lum := 0.2126*float64(c.R) + 0.7152*float64(c.G) + 0.0722*float64(c.B)
	if lum > 150 {
		return color.NRGBA{A: 0xe0}
	}
	return color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
}

func (c *Chart) hoverPie(p image.Point) {
	g := c.pie
	d := f32.Pt(float32(p.X), float32(p.Y)).Sub(g.center)
	r := float32(math.Hypot(float64(d.X), float64(d.Y)))
	if r < g.r0 || r > g.r1+4 {
		return
	}
	a := math.Atan2(float64(d.Y), float64(d.X))
	for k := range g.starts {
		for _, off := range []float64{0, 2 * math.Pi, -2 * math.Pi} {
			if a+off >= g.starts[k] && a+off < g.ends[k] {
				c.hover = hoverState{active: true, pos: p, series: 0, index: g.sliceIdx[k]}
				return
			}
		}
	}
}

func (c *Chart) hoverShapes(p image.Point) {
	for _, s := range c.shapes {
		if p.In(s.r) {
			c.hover = hoverState{active: true, pos: p, series: s.series, index: s.index}
			return
		}
	}
}

// layoutFunnel draws stages as centered bars narrowing with their
// values, with each stage's value and its share of the first.
func (c *Chart) layoutFunnel(gtx layout.Context, r image.Rectangle, pal palette, t float64) {
	c.shapes = c.shapes[:0]
	if len(c.series) == 0 {
		return
	}
	vals := c.animValues(0, t)
	n := len(vals)
	if n == 0 {
		return
	}
	maxV := 0.0
	for _, v := range vals {
		maxV = math.Max(maxV, v)
	}
	if maxV <= 0 {
		return
	}
	gap := gtx.Dp(4)
	labelW := min(r.Dx()/3, gtx.Dp(160))
	rtl := kit.RTL(gtx)
	barArea := image.Rect(r.Min.X+labelW, r.Min.Y, r.Max.X, r.Max.Y)
	if rtl {
		barArea = image.Rect(r.Min.X, r.Min.Y, r.Max.X-labelW, r.Max.Y)
	}
	h := (r.Dy() - gap*(n-1)) / n
	first := vals[0]
	for i, v := range vals {
		y := r.Min.Y + i*(h+gap)
		w := int(float64(barArea.Dx()) * math.Max(v, 0) / maxV)
		cx := barArea.Min.X + barArea.Dx()/2
		rect := image.Rect(cx-w/2, y, cx+w/2, y+h)
		col := categoryColor(0)
		if len(c.series[0].Color) > 0 {
			col = properties.CalcColor(c.series[0].Color)
		}
		col = alpha(col, uint8(255-min(i*22, 140)))
		if c.hover.active && c.hover.index == i {
			col = alpha(col, 0xff)
		}
		fillRound(gtx, rect, gtx.Dp(4), col)
		c.shapes = append(c.shapes, shapeHit{image.Rect(barArea.Min.X, y, barArea.Max.X, y+h), 0, i})
		name := ""
		if i < len(c.categories) {
			name = c.categories[i]
		}
		nl := text(gtx, name, 13, font.Normal)
		share := ""
		if first > 0 {
			share = trimFloat(v/first*100, 1) + "%"
		}
		vl := text(gtx, c.y.format(v)+"  "+share, 12, font.SemiBold)
		if rtl {
			nl.drawAt(gtx, image.Pt(r.Max.X, y+h/2), 1, 1, pal.text)
			vl.drawAt(gtx, image.Pt(r.Max.X, y+h/2), 1, 0, pal.muted)
		} else {
			nl.drawAt(gtx, image.Pt(r.Min.X, y+h/2), 0, 1, pal.text)
			vl.drawAt(gtx, image.Pt(r.Min.X, y+h/2), 0, 0, pal.muted)
		}
	}
}

// layoutGauge draws a 270° gauge: the track, its bands, the value's arc
// and the value.
func (c *Chart) layoutGauge(gtx layout.Context, r image.Rectangle, pal palette, t float64) {
	lo, hi := c.gaugeMin, c.gaugeMax
	if hi <= lo {
		return
	}
	v := c.gaugeValue
	if t < 1 && len(c.prev) > 0 && len(c.prev[0].Values) > 0 {
		from := c.prev[0].Values[0]
		v = from + (v-from)*t
	}
	frac := math.Max(0, math.Min(1, (v-lo)/(hi-lo)))
	// A 270° arc is its radius tall above the center and 0.71 radii
	// below, where the range's ends are labeled.
	labelH := float32(text(gtx, "0", 11, font.Normal).size().Y)
	r1 := min(float32(r.Dx())/2, (float32(r.Dy())-labelH)/1.71) - float32(gtx.Dp(2))
	center := f32.Pt(float32(r.Min.X+r.Dx()/2), float32(r.Min.Y)+r1+float32(gtx.Dp(2)))
	thick := r1 * 0.18
	r0 := r1 - thick
	start, sweep := math.Pi*3/4, math.Pi*3/2
	track := pal.border
	track.A = 0x80
	fillArc(gtx, center, r0, r1, start, start+sweep, track)
	// The bands, thin, inside the track.
	prev := lo
	for _, b := range c.bands {
		f0 := (prev - lo) / (hi - lo)
		f1 := (math.Min(b.To, hi) - lo) / (hi - lo)
		if f1 > f0 {
			fillArc(gtx, center, r0-float32(gtx.Dp(6)), r0-float32(gtx.Dp(3)), start+sweep*f0, start+sweep*f1, properties.CalcColor(b.Color))
		}
		prev = b.To
	}
	col := categoryColor(0)
	for _, b := range c.bands {
		if c.gaugeValue <= b.To {
			col = properties.CalcColor(b.Color)
			break
		}
	}
	fillArc(gtx, center, r0, r1, start, start+sweep*frac, col)
	vl := text(gtx, c.y.format(v), unit.Sp(float32(r0)/gtx.Metric.PxPerSp/2.6), font.Bold)
	vl.drawAt(gtx, image.Pt(int(center.X), int(center.Y)), 0.5, 0.5, pal.text)
	ml := text(gtx, c.y.format(lo), 11, font.Normal)
	xl := text(gtx, c.y.format(hi), 11, font.Normal)
	ay := int(center.Y + r1*float32(math.Sin(start)) + float32(gtx.Dp(2)))
	ml.drawAt(gtx, image.Pt(int(center.X+r0*float32(math.Cos(start))), ay), 0.5, 0, pal.muted)
	xl.drawAt(gtx, image.Pt(int(center.X+r0*float32(math.Cos(start+sweep))), ay), 0.5, 0, pal.muted)
}

// layoutHeatmap draws a grid of cells shaded by value, rows and columns
// labeled.
func (c *Chart) layoutHeatmap(gtx layout.Context, r image.Rectangle, pal palette) {
	c.shapes = c.shapes[:0]
	rows := len(c.heat)
	if rows == 0 {
		return
	}
	cols := 0
	for _, row := range c.heat {
		cols = max(cols, len(row))
	}
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, row := range c.heat {
		for _, v := range row {
			if !math.IsNaN(v) {
				lo, hi = math.Min(lo, v), math.Max(hi, v)
			}
		}
	}
	if hi <= lo {
		hi = lo + 1
	}
	rowW := 0
	for _, n := range c.heatRows {
		rowW = max(rowW, text(gtx, n, 11, font.Normal).size().X)
	}
	colH := text(gtx, "0", 11, font.Normal).size().Y
	gap := gtx.Dp(6)
	rtl := kit.RTL(gtx)
	grid := image.Rect(r.Min.X+rowW+gap, r.Min.Y+colH+gap, r.Max.X, r.Max.Y)
	if rtl {
		grid = image.Rect(r.Min.X, r.Min.Y+colH+gap, r.Max.X-rowW-gap, r.Max.Y)
	}
	cw := float64(grid.Dx()) / float64(cols)
	ch := float64(grid.Dy()) / float64(rows)
	base := categoryColor(0)
	empty := pal.border
	empty.A = 0x40
	cell := func(i, j int) image.Rectangle {
		x0 := grid.Min.X + int(float64(j)*cw)
		if rtl {
			x0 = grid.Max.X - int(float64(j+1)*cw)
		}
		y0 := grid.Min.Y + int(float64(i)*ch)
		return image.Rect(x0+1, y0+1, x0+int(cw)-1, y0+int(ch)-1)
	}
	for i, row := range c.heat {
		for j, v := range row {
			rr := cell(i, j)
			col := empty
			if !math.IsNaN(v) {
				f := (v - lo) / (hi - lo)
				col = alpha(base, uint8(40+f*215))
			}
			if c.hover.active && c.hover.series == i && c.hover.index == j {
				fillRound(gtx, rr.Inset(-1), gtx.Dp(3), pal.text)
			}
			fillRound(gtx, rr, gtx.Dp(2), col)
			c.shapes = append(c.shapes, shapeHit{rr, i, j})
		}
	}
	for i, n := range c.heatRows {
		y := grid.Min.Y + int((float64(i)+0.5)*ch)
		l := text(gtx, n, 11, font.Normal)
		if rtl {
			l.drawAt(gtx, image.Pt(grid.Max.X+gap, y), 0, 0.5, pal.muted)
		} else {
			l.drawAt(gtx, image.Pt(grid.Min.X-gap, y), 1, 0.5, pal.muted)
		}
	}
	step := 1
	maxW := 0
	for _, n := range c.heatCols {
		maxW = max(maxW, text(gtx, n, 11, font.Normal).size().X)
	}
	if cw > 0 {
		step = max(int(math.Ceil(float64(maxW+gap)/cw)), 1)
	}
	for j, n := range c.heatCols {
		if j%step != 0 {
			continue
		}
		rr := cell(0, j)
		text(gtx, n, 11, font.Normal).drawAt(gtx, image.Pt((rr.Min.X+rr.Max.X)/2, grid.Min.Y-gap), 0.5, 1, pal.muted)
	}
}

// Sparkline is a small line of values without axes, for tables and
// stat cards: its area shaded, the last value dotted. Styles set its
// size (Width, Height; 120 by 32 dp by default) and Color.
func Sparkline(values []float64, st ...styles.Styles) elements.Element {
	var s styles.Styles
	if len(st) > 0 {
		s = st[0]
	}
	return elements.Raw(func(gtx layout.Context) layout.Dimensions {
		w, h := gtx.Dp(120), gtx.Dp(32)
		if s.Width > 0 {
			w = gtx.Dp(unit.Dp(s.Width))
		}
		if s.Height > 0 {
			h = gtx.Dp(unit.Dp(s.Height))
		}
		w = min(w, gtx.Constraints.Max.X)
		size := image.Pt(w, h)
		if len(values) < 2 {
			return layout.Dimensions{Size: size}
		}
		col := categoryColor(0)
		if s.Color != "" {
			col = properties.CalcColor(string(s.Color))
		}
		lo, hi := math.Inf(1), math.Inf(-1)
		for _, v := range values {
			lo, hi = math.Min(lo, v), math.Max(hi, v)
		}
		if hi == lo {
			hi, lo = hi+1, lo-1
		}
		pad := float64(gtx.Dp(3))
		pts := make([]f32.Point, len(values))
		rtl := kit.RTL(gtx)
		for i, v := range values {
			x := pad + float64(i)/float64(len(values)-1)*(float64(w)-2*pad)
			if rtl {
				x = float64(w) - x
			}
			pts[i] = pt(x, pad+(hi-v)/(hi-lo)*(float64(h)-2*pad))
		}
		bottom := make([]f32.Point, len(pts))
		for i, p := range pts {
			bottom[i] = f32.Pt(p.X, float32(h))
		}
		area(gtx, pts, bottom, alpha(col, 0x30), false)
		polyline(gtx, pts, float32(max(gtx.Dp(1.5), 1)), col, false, false)
		circle(gtx, pts[len(pts)-1], float32(gtx.Dp(2.5)), col)
		sum := "from " + FormatNumber(values[0]) + " to " + FormatNumber(values[len(values)-1]) +
			", low " + FormatNumber(lo) + ", high " + FormatNumber(hi)
		elements.RecordSemantic(gtx, elements.SemanticNode{Kind: elements.SemanticImage, Label: "Trend " + sum}, size)
		return layout.Dimensions{Size: size}
	})
}

var _ = clip.Rect{}
var _ = paint.ColorOp{}
