package licenses

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/fonts"
)

const data = `[
  {"module": "Go standard library", "type": "BSD-3-Clause", "text": "Copyright (c) 2009 The Go Authors."},
  {"module": "gioui.org", "version": "v0.10.2", "type": "Unlicense OR MIT", "text": "This is free and unencumbered software."}
]`

func TestView(t *testing.T) {
	fonts.UseGofont()
	v := New([]byte(data))
	if len(v.Licenses()) != 2 {
		t.Fatalf("%d licenses", len(v.Licenses()))
	}
	var r input.Router
	labels := func() map[string]elements.SemanticNode {
		elements.BeginFrame()
		elements.BeginSemantics()
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Source: r.Source(), Constraints: layout.Exact(image.Pt(400, 600)), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
		v.Layout(gtx)
		nodes := elements.EndSemantics()
		r.Frame(&ops)
		elements.EndFrame()
		out := map[string]elements.SemanticNode{}
		for _, n := range nodes {
			out[n.Label] = n
		}
		return out
	}
	labels()
	got := labels()
	head, ok := got["gioui.org"]
	if !ok {
		t.Fatalf("no row for gioui.org: %v", got)
	}
	if _, ok := got["This is free and unencumbered software."]; ok {
		t.Fatal("the text shows before the row is opened")
	}
	c := head.Bounds.Min.Add(head.Bounds.Max).Div(2)
	r.Queue(
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: f32.Pt(float32(c.X), float32(c.Y)), Buttons: pointer.ButtonPrimary},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: f32.Pt(float32(c.X), float32(c.Y))},
	)
	labels()
	if _, ok := labels()["This is free and unencumbered software."]; !ok {
		t.Error("tapping the row didn't show the license text")
	}
}
