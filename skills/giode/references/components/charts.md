# Charts

`giode.Chart(kind)` draws line, area, bar (grouped, stacked and horizontal), scatter and
bubble, pie and donut, funnel, gauge and heatmap charts; `giode.Sparkline` draws a tiny line
for tables and stat cards. Charts take their colors from the theme, follow light and dark
mode and right-to-left layouts, and describe themselves to screen readers.

```go
revenue := giode.Chart(giode.ChartBar).
	Title("Revenue by region").
	Categories("Jan", "Feb", "Mar", "Apr").
	SetSeries(
		giode.ChartSeries{Name: "EU", Values: []float64{120, 180, 150, 210}},
		giode.ChartSeries{Name: "US", Values: []float64{200, 220, 190, 250}},
		giode.ChartSeries{Name: "Target", Values: []float64{180, 200, 200, 230}, Kind: giode.ChartLine, Dashed: true},
	).
	Y(giode.ChartAxis{Format: func(v float64) string { return "$" + giode.FormatNumber(v) + "k" }}).
	OnSelect(func(series, i int) { drillDown(series, i) })
```

A chart is created once and laid out every frame; call `SetSeries` when the data changes and
it animates from the old values (`Animate(false)`, or the system's reduced motion, turns that
off). It's 280 dp tall by default (`Height`) and as wide as it's given.

## Kinds

| Kind | Data |
|---|---|
| `ChartLine`, `ChartArea` | `Values` per category, or `Points` over numbers or time |
| `ChartBar`, `ChartHorizontalBar` | `Values` per category; series sharing a `Stack` stack, negative values below zero |
| `ChartScatter` | `Points`; a point's `Size` makes it a bubble |
| `ChartPie`, `ChartDonut` | the first series' `Values`, one per category; a donut shows the total in its middle |
| `ChartFunnel` | the first series' `Values`, one per stage, with each stage's share of the first |
| `ChartGauge` | `SetGauge(value, min, max, bands...)`, with colored `ChartBand`s |
| `ChartHeatmap` | `SetHeatmap(rows, cols, values)` |

A series' `Kind` draws it differently from the chart, for combinations: a target line over
bars, an area under a line. `Axis: 1` measures a series on the right-hand axis (`Y2`), and
`Dashed` draws a forecast. `Smooth(true)` draws lines as curves.

## Axes

`X`, `Y` and `Y2` take a `ChartAxis`: a `Title`, a `Format` for tick labels and tooltips
(`FormatNumber` by default: 950, 12.3k, 3.4M), a fixed range (`Min`, `Max`, `Fixed`), and
`Hide` and `NoGrid`. Ticks fall on round numbers. A time axis (`Time: true`, X values from
`giode.ChartTimeX(t)`) ticks on round times, from seconds to decades, in its `Location`.

```go
load := giode.Chart(giode.ChartArea).
	X(giode.ChartAxis{Time: true}).
	SetSeries(giode.ChartSeries{Name: "Requests", Points: points}).
	Zoomable(true)
```

Long series are downsampled to the plot's width with largest-triangle-three-buckets, which
keeps peaks and dips, so hundreds of thousands of points draw quickly.

## Interaction

- **Tooltips** follow the pointer (or finger): every series' value at the hovered category or
  time, a slice's share, a cell's value.
- **Legend**: clicking an entry hides or shows its series (or slice).
- **Selection**: `OnSelect(func(series, index int))` on a click on a bar, point, slice, stage
  or cell; `Selected()` returns it.
- **Brushing**: `OnBrush(func(from, to float64))` lets dragging across the plot select an x
  range (category indexes on a category chart).
- **Zoom and pan**: `Zoomable(true)` zooms an x-y chart with the wheel around the pointer;
  dragging pans and a double click shows everything again (`ResetZoom`).

## Colors

Series take the theme's chart colors in order, `theme.chart1` to `theme.chart8`, chosen to
tell apart side by side, with lighter ones in dark mode and stronger ones in high contrast.
`Palette.Chart` replaces them; a series' `Color` overrides its own.

## Accessibility

Screen readers hear the chart's `Summary()`: its title, kind and each series' range ("Revenue
by region. Bar chart of 3 series. EU: from 120 to 210, low 120, high 210…"), or the
`Description` set. `Table()` returns the data as a table to show beside the chart or instead
of it.

## Sparklines

```go
giode.Sparkline(lastWeek, giode.Styles{Width: 120, Height: 32, Color: "theme.success"})
```
