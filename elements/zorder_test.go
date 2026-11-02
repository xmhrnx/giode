package elements

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// layoutAndTap lays root out over a router and taps at p, returning
// which of the pressables was clicked: the one painted on top there.
func layoutAndTap(t *testing.T, root func() Element, p f32.Point) {
	t.Helper()
	var r input.Router
	frame := func() {
		BeginFrame()
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Source: r.Source(), Constraints: layout.Exact(image.Pt(200, 200)), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
		root().Layout(gtx)
		r.Frame(&ops)
		EndFrame()
	}
	frame()
	frame()
	r.Queue(
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: p, Buttons: pointer.ButtonPrimary},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: p},
	)
	frame()
	frame()
}

func TestZIndex_FlowAbovePositioned(t *testing.T) {
	var hit string
	flow := NewPressable(func(PressState) Element { return Box(styles.Styles{Width: 200, Height: 200}) }).
		OnClick(func(ClickEvent) { hit = "flow" })
	abs := NewPressable(func(PressState) Element { return Box(styles.Styles{Width: 200, Height: 200}) }).
		OnClick(func(ClickEvent) { hit = "positioned" })
	root := func(zFlow, zAbs int) func() Element {
		return func() Element {
			return Box(styles.Styles{Width: 200, Height: 200},
				Box(styles.Styles{ZIndex: properties.ZIndex(zFlow)}, flow),
				Box(styles.Styles{Position: properties.PositionAbsolute, Top: properties.Flush, Left: properties.Flush, ZIndex: properties.ZIndex(zAbs)}, abs),
			)
		}
	}
	layoutAndTap(t, root(0, 0), f32.Pt(50, 50))
	if hit != "positioned" {
		t.Errorf("equal z-index: %q got the tap, want the positioned child", hit)
	}
	hit = ""
	layoutAndTap(t, root(2, 1), f32.Pt(50, 50))
	if hit != "flow" {
		t.Errorf("flow z 2 over positioned z 1: %q got the tap, want flow", hit)
	}
}

func TestZIndex_DoesNotReorderLayout(t *testing.T) {
	var pos [2]image.Point
	probe := func(i int, z int) Element {
		return Box(styles.Styles{Height: 20, ZIndex: properties.ZIndex(z)}, Raw(func(gtx layout.Context) layout.Dimensions {
			TrackPosition(&pos[i])
			return layout.Dimensions{Size: image.Pt(10, 20)}
		}))
	}
	BeginFrame()
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Exact(image.Pt(200, 200)), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
	Box(styles.Styles{FlexDirection: properties.FlexDirectionColumn}, probe(0, 5), probe(1, 0)).Layout(gtx)
	EndFrame()
	if pos[0].Y != 0 || pos[1].Y != 20 {
		t.Errorf("rows at %v and %v: z-index moved them", pos[0], pos[1])
	}
}

func TestAbsolute_FlushEdges(t *testing.T) {
	var at image.Point
	var size image.Point
	child := func(st styles.Styles) Element {
		st.Position = properties.PositionAbsolute
		// A column passes its width on to the probe.
		st.FlexDirection = properties.FlexDirectionColumn
		return Box(st, Raw(func(gtx layout.Context) layout.Dimensions {
			TrackPosition(&at)
			size = image.Pt(max(gtx.Constraints.Min.X, 30), max(gtx.Constraints.Min.Y, 10))
			return layout.Dimensions{Size: size}
		}))
	}
	run := func(st styles.Styles) {
		BeginFrame()
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Constraints: layout.Exact(image.Pt(200, 100)), Metric: unit.Metric{PxPerDp: 2, PxPerSp: 2}}
		Box(styles.Styles{Width: -1, Height: -1}, child(st)).Layout(gtx)
		EndFrame()
	}
	cases := []struct {
		name     string
		st       styles.Styles
		at, size image.Point
	}{
		{"bottom right flush", styles.Styles{Bottom: properties.Flush, Right: properties.Flush}, image.Pt(170, 90), image.Pt(30, 10)},
		{"bottom 0 is unset", styles.Styles{Bottom: 0, Right: properties.Flush}, image.Pt(170, 0), image.Pt(30, 10)},
		{"offsets in dp", styles.Styles{Bottom: 5, Right: 5}, image.Pt(160, 80), image.Pt(30, 10)},
		{"stretch between left and right", styles.Styles{Left: properties.Flush, Right: properties.Flush, Bottom: properties.Flush}, image.Pt(0, 90), image.Pt(200, 10)},
		{"stretch with margins", styles.Styles{Left: 10, Right: 10, Top: properties.Flush}, image.Pt(20, 0), image.Pt(160, 10)},
	}
	for _, c := range cases {
		run(c.st)
		if at != c.at || size != c.size {
			t.Errorf("%s: at %v size %v, want at %v size %v", c.name, at, size, c.at, c.size)
		}
	}
}
