package base

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
)

// A measure after the real layout leaves a tracked position alone.
func TestTrackPositionIgnoresMeasure(t *testing.T) {
	var p image.Point
	el := elementFunc(func(gtx layout.Context) layout.Dimensions {
		TrackPosition(&p)
		return layout.Dimensions{Size: image.Pt(10, 10)}
	})
	gtx := layout.Context{Ops: new(op.Ops), Constraints: layout.Exact(image.Pt(100, 100))}
	pop := PushOffset(image.Pt(40, 7))
	el.Layout(gtx)
	pop()
	Measure(gtx, el)
	if p != image.Pt(40, 7) {
		t.Errorf("tracked %v after a measure, want the layout's (40,7)", p)
	}
	probes = probes[:0]
}

type elementFunc func(gtx layout.Context) layout.Dimensions

func (f elementFunc) Layout(gtx layout.Context) layout.Dimensions { return f(gtx) }
