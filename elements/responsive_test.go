package elements

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/components/kit"
)

func TestResponsive_MaxWidthInDp(t *testing.T) {
	var got []string
	probe := func() Element {
		return Raw(func(gtx layout.Context) layout.Dimensions {
			got = append(got, ActiveBreakpoint())
			return layout.Dimensions{}
		})
	}
	r := Responsive(
		Breakpoint{MaxWidth: 400, Name: "compact", View: probe},
		Breakpoint{View: probe},
	)
	layoutAt := func(width int, scale float32) string {
		got = nil
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: scale}, Constraints: layout.Constraints{Max: image.Pt(width, 800)}}
		r.Layout(gtx)
		return got[0]
	}

	// 1080px on a 3x phone is 360dp: compact.
	if name := layoutAt(1080, 3); name != "compact" {
		t.Errorf("1080px at 3x = %q, want compact", name)
	}
	// 1080px at 1x is 1080dp: the fallback.
	if name := layoutAt(1080, 1); name != "" {
		t.Errorf("1080px at 1x = %q, want the fallback", name)
	}
	if name := ActiveBreakpoint(); name != "" {
		t.Errorf("ActiveBreakpoint after layout = %q, want empty", name)
	}
}

func TestResponsive_NestedAndCompact(t *testing.T) {
	var outer, inner string
	var compact bool
	r := Responsive(Breakpoint{MaxWidth: 600, Name: "compact", View: func() Element {
		return Raw(func(gtx layout.Context) layout.Dimensions {
			compact = Compact()
			Responsive(Breakpoint{Name: "inner", View: func() Element {
				return Raw(func(gtx layout.Context) layout.Dimensions {
					inner = ActiveBreakpoint()
					return layout.Dimensions{}
				})
			}}).Layout(gtx)
			outer = ActiveBreakpoint()
			return layout.Dimensions{}
		})
	}})
	var ops op.Ops
	r.Layout(layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: image.Pt(300, 300)}})
	if !compact || inner != "inner" || outer != "compact" {
		t.Errorf("compact=%v inner=%q outer=%q, want true inner compact", compact, inner, outer)
	}
}

func TestResponsive_DeprecatedWidthInPx(t *testing.T) {
	var picked string
	view := func(name string) func() Element {
		return func() Element {
			picked = name
			return Raw(func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} })
		}
	}
	r := Responsive(Breakpoint{Width: 500, View: view("narrow")}, Breakpoint{View: view("wide")})
	var ops op.Ops
	r.Layout(layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 3}, Constraints: layout.Constraints{Max: image.Pt(450, 300)}})
	if picked != "narrow" {
		t.Errorf("picked %q, want narrow (Width stays in pixels)", picked)
	}
}

func TestResponsive_ReportsScreenToScrollbars(t *testing.T) {
	SetWindowSize(image.Pt(500, 800))
	defer SetWindowSize(image.Point{})
	var bp string
	var w int
	r := Responsive(Breakpoint{MaxWidth: 600, Name: "compact", View: func() Element {
		return Raw(func(gtx layout.Context) layout.Dimensions {
			w, bp = kit.Screen()
			return layout.Dimensions{}
		})
	}})
	gtx := layout.Context{Ops: new(op.Ops), Constraints: layout.Exact(image.Pt(500, 800)), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
	r.Layout(gtx)
	if w != 500 || bp != "compact" {
		t.Errorf("kit.Screen() = %d, %q; want 500, \"compact\"", w, bp)
	}
}
