// Package chart draws charts: line, area, bar (grouped, stacked,
// horizontal), pie and donut, funnel, scatter and bubble, gauge, heatmap
// and sparklines, with axes, legends, tooltips, selection, brushing,
// zoom and pan, animated data changes, theme colors, right-to-left
// layout and a data table for screen readers.
package chart

import (
	"image"
	"time"

	"gioui.org/gesture"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
)

// Kind is a chart's or a series' type.
type Kind int

const (
	// Inherit is a series' zero kind: the chart's.
	Inherit Kind = iota
	Line
	Area
	Bar
	// HorizontalBar draws bars across, categories down the side.
	HorizontalBar
	Scatter
	Pie
	Donut
	Funnel
	Gauge
	Heatmap
)

// Point is a point of an x-y series: X a number or, on a time axis, Unix
// seconds (TimeX converts).
type Point struct {
	X, Y float64
	// Size is a bubble's size in a scatter chart, relative to the
	// largest; 0 draws plain points.
	Size float64
	// Label names the point in tooltips.
	Label string
}

// TimeX returns t as a Point X for a time axis.
func TimeX(t time.Time) float64 { return float64(t.UnixNano()) / 1e9 }

// Series is a set of values a chart draws.
type Series struct {
	Name string
	// Values are one per category: bars, lines over categories, pie
	// slices and funnel stages (named by the chart's Categories).
	Values []float64
	// Points are an x-y series: lines and areas over numbers or time,
	// scatter plots.
	Points []Point
	// Color overrides the theme's series color ("theme.chart1" and on).
	Color string
	// Kind draws this series differently from the chart: a Line over
	// Bar chart's bars, for example.
	Kind Kind
	// Axis 1 measures the series on the right-hand axis (Y2).
	Axis int
	// Stack stacks the series on the others with the same Stack:
	// stacked bars and areas.
	Stack string
	// Dashed draws a line dashed, as forecasts are.
	Dashed bool
	// Hidden leaves the series out; clicking its legend entry toggles
	// it.
	Hidden bool
}

// Axis configures an axis.
type Axis struct {
	Title string
	// Format writes tick labels and tooltip values; FormatNumber, or
	// dates on a time axis, by default.
	Format func(v float64) string
	// Min and Max fix the range when Fixed is set; otherwise it's the
	// data's, rounded out to ticks (and including zero for bars).
	Min, Max float64
	Fixed    bool
	// Time makes X a time axis of Unix seconds, in Location (local
	// time when nil).
	Time     bool
	Location *time.Location
	// Hide hides the axis line and labels; NoGrid its grid lines.
	Hide, NoGrid bool
}

// Band is a colored range of a gauge: green to 60, amber to 85, red.
type Band struct {
	To    float64
	Color string
}

// LegendPos places the legend.
type LegendPos int

const (
	// LegendAuto shows one above charts of several series and pies.
	LegendAuto LegendPos = iota
	LegendNone
	LegendTop
	LegendBottom
)

// Chart is a stateful chart. Create it once with New and lay it out
// every frame; set data whenever it changes.
type Chart struct {
	elements.Contextual

	kind       Kind
	categories []string
	series     []Series
	x, y, y2   Axis
	legend     LegendPos
	title      string
	desc       string
	height     int // dp
	animate    bool
	smooth     bool
	st         styles.Styles

	onSelect func(series, index int)
	onBrush  func(from, to float64)
	zoomable bool

	gaugeValue, gaugeMin, gaugeMax float64
	bands                          []Band
	heatRows, heatCols             []string
	heat                           [][]float64

	// Animation from the previous data.
	prev      []Series
	animStart time.Time
	animating bool

	// Interaction.
	size     image.Point
	plot     image.Rectangle
	hover    hoverState
	brush    brushState
	view     viewRange
	scroll   gesture.Scroll
	legendAt []legendHit
	tag      int
	origin   image.Point
	selected [2]int // series, index; -1 for none
	frame    *frame
	xUnit    timeUnit
	// hiddenCats are pie slices hidden from the legend.
	hiddenCats map[int]bool
	// shapes are hoverable areas of pies, funnels and heatmaps.
	shapes []shapeHit
	pie    pieGeom
}

// New returns a chart of kind.
func New(kind Kind) *Chart {
	return &Chart{kind: kind, animate: true, height: 280, selected: [2]int{-1, -1}}
}

// Categories names the categories: the x axis of a category chart, pie
// slices, funnel stages.
func (c *Chart) Categories(names ...string) *Chart { c.categories = names; return c }

// SetSeries replaces the data, animating from what was shown.
func (c *Chart) SetSeries(ss ...Series) *Chart {
	if c.animate && len(c.series) > 0 {
		c.prev = c.series
		c.animStart = time.Time{}
		c.animating = true
	}
	c.series = ss
	c.hover.index = -1
	return c
}

// Series returns the chart's data.
func (c *Chart) Series() []Series { return c.series }

// X, Y and Y2 configure the axes; Y2 is the right-hand one, for series
// with Axis 1.
func (c *Chart) X(a Axis) *Chart  { c.x = a; return c }
func (c *Chart) Y(a Axis) *Chart  { c.y = a; return c }
func (c *Chart) Y2(a Axis) *Chart { c.y2 = a; return c }

// Legend places the legend.
func (c *Chart) Legend(p LegendPos) *Chart { c.legend = p; return c }

// Title sets the title drawn above the chart, which also names it to
// screen readers.
func (c *Chart) Title(s string) *Chart { c.title = s; return c }

// Description sets what screen readers hear about the chart, in place
// of the summary of its data.
func (c *Chart) Description(s string) *Chart { c.desc = s; return c }

// Height sets the chart's height in dp, 280 by default; it shrinks to
// the height it's given.
func (c *Chart) Height(dp int) *Chart { c.height = dp; return c }

// Animate turns animating data changes on (the default) or off. With
// reduced motion on, they don't animate either way.
func (c *Chart) Animate(on bool) *Chart { c.animate = on; return c }

// Smooth draws lines and areas as curves.
func (c *Chart) Smooth(on bool) *Chart { c.smooth = on; return c }

// Style sets the chart's box styles.
func (c *Chart) Style(st styles.Styles) *Chart { c.st = st; return c }

// OnSelect sets the handler of a click on a bar, point, slice, stage or
// cell: its series and index (a heatmap's row and column).
func (c *Chart) OnSelect(fn func(series, index int)) *Chart { c.onSelect = fn; return c }

// OnBrush lets dragging across an x-y or category chart select a range,
// passed as x values (category indexes on a category chart).
func (c *Chart) OnBrush(fn func(from, to float64)) *Chart { c.onBrush = fn; return c }

// Zoomable lets the wheel, or Ctrl and the wheel inside a scroll, zoom
// the x axis around the pointer, and dragging pan it; a double click
// resets.
func (c *Chart) Zoomable(on bool) *Chart { c.zoomable = on; return c }

// ResetZoom shows all the data again.
func (c *Chart) ResetZoom() { c.view = viewRange{} }

// SetGauge sets a gauge's value and range, and its colored bands.
func (c *Chart) SetGauge(value, min, max float64, bands ...Band) *Chart {
	if c.animate && c.kind == Gauge && c.gaugeMax > c.gaugeMin {
		c.prev = []Series{{Values: []float64{c.gaugeValue}}}
		c.animStart, c.animating = time.Time{}, true
	}
	c.gaugeValue, c.gaugeMin, c.gaugeMax, c.bands = value, min, max, bands
	return c
}

// SetHeatmap sets a heatmap's rows, columns and values by row.
func (c *Chart) SetHeatmap(rows, cols []string, values [][]float64) *Chart {
	c.heatRows, c.heatCols, c.heat = rows, cols, values
	return c
}

// Selected returns the selected series and index, -1 for none.
func (c *Chart) Selected() (series, index int) { return c.selected[0], c.selected[1] }

// effectiveKind is a series' kind.
func (c *Chart) kindOf(s *Series) Kind {
	if s.Kind != Inherit {
		return s.Kind
	}
	return c.kind
}
