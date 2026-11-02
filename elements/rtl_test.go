package elements

import (
	"image"
	"testing"

	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// positions lays root out in a 200x100 window in the given direction
// and returns the probes' positions.
func positions(t *testing.T, dir system.TextDirection, root func(probe func(i int) Element) Element, n int) []image.Point {
	t.Helper()
	pos := make([]image.Point, n)
	probe := func(i int) Element {
		return Box(styles.Styles{Width: 50, Height: 20}, Raw(func(gtx layout.Context) layout.Dimensions {
			TrackPosition(&pos[i])
			return layout.Dimensions{Size: image.Pt(50, 20)}
		}))
	}
	BeginFrame()
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Exact(image.Pt(200, 100)), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Locale: system.Locale{Language: "fa", Direction: dir}}
	root(probe).Layout(gtx)
	EndFrame()
	return pos
}

func TestRTL_RowRunsFromTheRight(t *testing.T) {
	row := func(probe func(int) Element) Element {
		return Box(styles.Styles{Width: -1}, probe(0), probe(1))
	}
	if p := positions(t, system.LTR, row, 2); p[0].X != 0 || p[1].X != 50 {
		t.Errorf("LTR row at %v", p)
	}
	if p := positions(t, system.RTL, row, 2); p[0].X != 150 || p[1].X != 100 {
		t.Errorf("RTL row at %v, want the first child at 150 and the second at 100", p)
	}
}

func TestRTL_ColumnAlignsRight(t *testing.T) {
	col := func(probe func(int) Element) Element {
		return Box(styles.Styles{Width: -1, FlexDirection: properties.FlexDirectionColumn}, probe(0), probe(1))
	}
	if p := positions(t, system.RTL, col, 2); p[0] != image.Pt(150, 0) || p[1] != image.Pt(150, 20) {
		t.Errorf("RTL column at %v", p)
	}
}

func TestRTL_WrapMirrorsLines(t *testing.T) {
	wrap := func(probe func(int) Element) Element {
		return Box(styles.Styles{Width: 120, Wrap: properties.WrapWrap}, probe(0), probe(1), probe(2))
	}
	// Two per 120-wide line: 0 and 1 on the first, 2 below.
	if p := positions(t, system.RTL, wrap, 3); p[0].X != 70 || p[1].X != 20 || p[2].X != 70 {
		t.Errorf("RTL wrap at %v", p)
	}
}

func TestRTL_PositionedKeepPhysicalSides(t *testing.T) {
	var at image.Point
	root := func(probe func(int) Element) Element {
		return Box(styles.Styles{Width: -1, Height: -1},
			Box(styles.Styles{Position: properties.PositionAbsolute, Left: 10, Top: properties.Flush},
				Raw(func(gtx layout.Context) layout.Dimensions {
					TrackPosition(&at)
					return layout.Dimensions{Size: image.Pt(30, 10)}
				})))
	}
	positions(t, system.RTL, root, 0)
	if at.X != 10 {
		t.Errorf("Left: 10 in RTL at x %d, want 10 (physical, like CSS left)", at.X)
	}
}
