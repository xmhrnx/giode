package chart

import (
	"math"
	"sort"
	"strings"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

var kindNames = map[Kind]string{
	Line: "Line chart", Area: "Area chart", Bar: "Bar chart", HorizontalBar: "Bar chart", Scatter: "Scatter plot",
	Pie: "Pie chart", Donut: "Donut chart", Funnel: "Funnel", Gauge: "Gauge", Heatmap: "Heatmap",
}

// Summary describes the chart in words, for screen readers: its title,
// kind, and each series' range; or the Description set.
func (c *Chart) Summary() string {
	if c.desc != "" {
		return c.desc
	}
	var b strings.Builder
	if c.title != "" {
		b.WriteString(c.title + ". ")
	}
	b.WriteString(kindNames[c.kind])
	switch c.kind {
	case Gauge:
		b.WriteString(": " + c.y.format(c.gaugeValue) + " of " + c.y.format(c.gaugeMin) + " to " + c.y.format(c.gaugeMax))
		return b.String()
	case Heatmap:
		b.WriteString(" of " + itoa(len(c.heatRows)) + " rows by " + itoa(len(c.heatCols)) + " columns")
		return b.String()
	case Pie, Donut, Funnel:
		if len(c.series) > 0 {
			var parts []string
			for i, v := range c.series[0].Values {
				name := ""
				if i < len(c.categories) {
					name = c.categories[i]
				}
				parts = append(parts, name+" "+c.y.format(v))
			}
			b.WriteString(": " + strings.Join(parts, ", "))
		}
		return b.String()
	}
	vis := c.visible()
	if len(vis) > 1 {
		b.WriteString(" of " + itoa(len(vis)) + " series")
	}
	for _, i := range vis {
		s := &c.series[i]
		lo, hi := math.Inf(1), math.Inf(-1)
		first, last := math.NaN(), math.NaN()
		add := func(v float64) {
			if math.IsNaN(v) {
				return
			}
			if math.IsNaN(first) {
				first = v
			}
			last = v
			lo, hi = math.Min(lo, v), math.Max(hi, v)
		}
		for _, v := range s.Values {
			add(v)
		}
		for _, p := range s.Points {
			add(p.Y)
		}
		if math.IsInf(lo, 0) {
			continue
		}
		b.WriteString(". ")
		if s.Name != "" {
			b.WriteString(s.Name + ": ")
		}
		b.WriteString("from " + c.y.format(first) + " to " + c.y.format(last) + ", low " + c.y.format(lo) + ", high " + c.y.format(hi))
	}
	return b.String()
}

// Table returns the chart's data as a table, a text alternative to show
// beside it or instead of it: a row per category or x, a column per
// series.
func (c *Chart) Table() elements.Element {
	cell := func(s string, head bool) elements.Element {
		st := styles.Styles{Padding: properties.SymmetricInset(8, 4), Color: "theme.text", FontSize: 13}
		if head {
			st.FontWeight = properties.FontWeightBold
		}
		return elements.Text(s, st)
	}
	var rows []elements.Element
	row := func(cells ...elements.Element) {
		rows = append(rows, elements.Box(styles.Styles{FlexDirection: properties.FlexDirectionRow,
			BorderWidth: 0}, cells...))
	}
	switch c.kind {
	case Heatmap:
		head := []elements.Element{cell("", true)}
		for _, n := range c.heatCols {
			head = append(head, cell(n, true))
		}
		row(head...)
		for i, r := range c.heat {
			name := ""
			if i < len(c.heatRows) {
				name = c.heatRows[i]
			}
			cells := []elements.Element{cell(name, true)}
			for _, v := range r {
				cells = append(cells, cell(c.y.format(v), false))
			}
			row(cells...)
		}
	case Gauge:
		row(cell("Value", true), cell(c.y.format(c.gaugeValue), false))
	default:
		head := []elements.Element{cell("", true)}
		for _, s := range c.series {
			head = append(head, cell(s.Name, true))
		}
		row(head...)
		if c.isCategorical() {
			for j := 0; j < c.categoryCount(); j++ {
				name := ""
				if j < len(c.categories) {
					name = c.categories[j]
				}
				cells := []elements.Element{cell(name, true)}
				for _, s := range c.series {
					v := ""
					if j < len(s.Values) && !math.IsNaN(s.Values[j]) {
						v = c.y.format(s.Values[j])
					}
					cells = append(cells, cell(v, false))
				}
				row(cells...)
			}
		} else {
			// The union of the series' xs.
			var xs []float64
			seen := map[float64]bool{}
			for _, s := range c.series {
				for _, p := range s.Points {
					if !seen[p.X] {
						seen[p.X] = true
						xs = append(xs, p.X)
					}
				}
			}
			sortFloats(xs)
			if len(xs) > 500 {
				xs = xs[:500]
			}
			for _, x := range xs {
				cells := []elements.Element{cell(c.formatX(x, timeUnit{format: "2006-01-02 15:04"}), true)}
				for _, s := range c.series {
					v := ""
					if j := sortSearch(s.Points, x); j < len(s.Points) && s.Points[j].X == x {
						v = c.y.format(s.Points[j].Y)
					}
					cells = append(cells, cell(v, false))
				}
				row(cells...)
			}
		}
	}
	return elements.Box(styles.Styles{FlexDirection: properties.FlexDirectionColumn}, rows...)
}

func sortFloats(a []float64) { sort.Float64s(a) }
