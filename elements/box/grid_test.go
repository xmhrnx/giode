package box

import (
	"image"
	"testing"

	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/elements/internal/base"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// probe is a child of a fixed size that records where it lands.
type probe struct {
	st   styles.Styles
	size image.Point
	at   image.Point
	got  image.Point
}

func (p *probe) Layout(gtx layout.Context) layout.Dimensions {
	base.TrackPosition(&p.at)
	p.got = gtx.Constraints.Constrain(p.size)
	return layout.Dimensions{Size: p.got}
}

func (p *probe) GridPlacement() (col, row properties.GridSpan) { return p.st.GridColumn, p.st.GridRow }
func (p *probe) FlexWeight(layout.Axis) int                    { return 0 }
func (p *probe) FlexPercent(layout.Axis) int                   { return 0 }
func (p *probe) ZIndex() int                                   { return 0 }

func layoutGridOf(t *testing.T, st styles.Styles, width int, kids ...*probe) layout.Dimensions {
	t.Helper()
	els := make([]base.Element, len(kids))
	for i, k := range kids {
		els[i] = k
	}
	base.BeginFrame()
	defer base.EndFrame()
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Constraints{Max: image.Pt(width, 100000)}}
	return Grid(st, els...).Layout(gtx)
}

func box(w, h int) *probe { return &probe{size: image.Pt(w, h)} }

func TestGridColumnsAndPlacement(t *testing.T) {
	a, b, c, d := box(50, 20), box(50, 40), box(50, 10), box(50, 10)
	d.st.GridColumn = properties.Span(2)
	dims := layoutGridOf(t, styles.Styles{GridColumns: "100 1fr 2fr", Gap: 10, RowGap: 4}, 420, a, b, c, d)
	// 100, then 420-100-2*10 = 300 shared 1:2: 100 and 200.
	if a.at != image.Pt(0, 0) || b.at != image.Pt(110, 0) || c.at != image.Pt(220, 0) {
		t.Errorf("first row at %v %v %v", a.at, b.at, c.at)
	}
	if a.got.X != 100 || b.got.X != 100 || c.got.X != 200 {
		t.Errorf("widths %d %d %d", a.got.X, b.got.X, c.got.X)
	}
	// Row one is 40 tall (b), cells stretch to it.
	if a.got.Y != 40 || c.got.Y != 40 {
		t.Errorf("stretched heights %d %d", a.got.Y, c.got.Y)
	}
	// d spans two columns on the next row.
	if d.at != image.Pt(0, 44) || d.got.X != 210 {
		t.Errorf("spanning item at %v, %d wide", d.at, d.got.X)
	}
	if dims.Size != image.Pt(420, 54) {
		t.Errorf("grid %v", dims.Size)
	}
}

func TestGridExplicitCellsAndRows(t *testing.T) {
	side, x, y := box(10, 100), box(10, 10), box(10, 10)
	side.st.GridColumn, side.st.GridRow = properties.Cell(3, 1), properties.Cell(1, 2)
	layoutGridOf(t, styles.Styles{GridColumns: "repeat(3, 100)", GridRows: "30 auto"}, 300, side, x, y)
	if side.at != image.Pt(200, 0) || x.at != image.Pt(0, 0) || y.at != image.Pt(100, 0) {
		t.Errorf("placed %v %v %v", side.at, x.at, y.at)
	}
	// side spans 2 rows: 30 fixed, then the auto row grows to fit 100.
	if side.got.Y != 100 || x.got.Y != 30 {
		t.Errorf("heights %d %d", side.got.Y, x.got.Y)
	}
}

func TestGridAutoFillAndMinimums(t *testing.T) {
	var kids []*probe
	for range 7 {
		kids = append(kids, box(10, 10))
	}
	layoutGridOf(t, styles.Styles{GridColumns: "repeat(auto-fill, minmax(200, 1fr))", Gap: 10}, 650, kids...)
	// (650+10)/(200+10) = 3 columns of 210.
	if kids[2].at.X != 440 || kids[3].at != image.Pt(0, 20) || kids[0].got.X != 210 {
		t.Errorf("auto-fill: %v %v %d", kids[2].at, kids[3].at, kids[0].got.X)
	}
	p, q := box(10, 10), box(10, 10)
	layoutGridOf(t, styles.Styles{GridColumns: "minmax(250, 1fr) 1fr"}, 400, p, q)
	if p.got.X != 250 || q.got.X != 150 {
		t.Errorf("minimum: %d %d", p.got.X, q.got.X)
	}
	label, field := box(80, 10), box(10, 10)
	layoutGridOf(t, styles.Styles{GridColumns: "auto 1fr", Gap: 8}, 300, label, field)
	if label.got.X != 80 || field.at.X != 88 || field.got.X != 212 {
		t.Errorf("auto column: %d, field at %d, %d wide", label.got.X, field.at.X, field.got.X)
	}
}

func TestGridAlignCenter(t *testing.T) {
	tall, short := box(10, 50), box(10, 10)
	layoutGridOf(t, styles.Styles{GridColumns: "1fr 1fr", Align: properties.AlignCenter}, 200, tall, short)
	if short.got.Y != 10 || short.at.Y != 20 {
		t.Errorf("centered: %d tall at %v", short.got.Y, short.at)
	}
}

// Right to left, the first column is on the right.
func TestGridRTL(t *testing.T) {
	a, b := box(10, 10), box(10, 10)
	els := []base.Element{a, b}
	base.BeginFrame()
	defer base.EndFrame()
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Constraints{Max: image.Pt(300, 1000)}}
	gtx.Locale.Direction = system.RTL
	Grid(styles.Styles{GridColumns: "100 200"}, els...).Layout(gtx)
	if a.at.X != 200 || b.at.X != 0 {
		t.Errorf("right to left: first at %d, second at %d", a.at.X, b.at.X)
	}
}
