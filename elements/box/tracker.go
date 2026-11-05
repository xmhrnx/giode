package box

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/elements/internal/base"
)

// flexSpan is what one flex child registered, and its measured size.
type flexSpan struct {
	track base.ChildTracker
	dims  layout.Dimensions
	// call and off are the recorded drawing and offset of a child
	// painted in z order.
	call op.CallOp
	off  image.Point
}

// flexTracker shifts what flex children registered by their final
// offsets. A child lays out before the flex container knows where it
// goes, so its portals, semantic nodes and tracked positions are
// recorded at the container's origin; once the container is done, each
// child's are shifted by the offset Gio's flex gave it.
type flexTracker struct {
	spans []flexSpan
	// record keeps each child's drawing out of the flex, in its span.
	record bool
}

func (t *flexTracker) wrap(i int, w layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		t.spans[i].track = base.TrackChild()
		var macro op.MacroOp
		if t.record {
			macro = op.Record(gtx.Ops)
		}
		dims := w(gtx)
		if t.record {
			t.spans[i].call = macro.Stop()
		}
		t.spans[i].track.End()
		t.spans[i].dims = dims
		return dims
	}
}

// place replicates the placement step of layout.Flex.Layout.
func (t *flexTracker) place(cs layout.Constraints, f layout.Flex) {
	n := len(t.spans)
	mainMin := axisMain(cs.Min, f.Axis)
	crossMin := axisCross(cs.Min, f.Axis)
	size, maxCross, maxBaseline := 0, crossMin, 0
	for _, s := range t.spans {
		size += axisMain(s.dims.Size, f.Axis)
		maxCross = max(maxCross, axisCross(s.dims.Size, f.Axis))
		maxBaseline = max(maxBaseline, s.dims.Size.Y-s.dims.Baseline)
	}
	if n > 1 && f.Gap > 0 {
		size += f.Gap * (n - 1)
	}
	space := max(0, mainMin-size)
	main := 0
	switch f.Spacing {
	case layout.SpaceSides:
		main += space / 2
	case layout.SpaceStart:
		main += space
	case layout.SpaceEvenly:
		main += space / (1 + n)
	case layout.SpaceAround:
		if n > 0 {
			main += space / (n * 2)
		}
	}
	for i, s := range t.spans {
		cross := 0
		switch f.Alignment {
		case layout.End:
			cross = maxCross - axisCross(s.dims.Size, f.Axis)
		case layout.Middle:
			cross = (maxCross - axisCross(s.dims.Size, f.Axis)) / 2
		case layout.Baseline:
			if f.Axis == layout.Horizontal {
				cross = maxBaseline - (s.dims.Size.Y - s.dims.Baseline)
			}
		}
		t.spans[i].off = axisPoint(f.Axis, main, cross)
		s.track.Place(t.spans[i].off)
		main += axisMain(s.dims.Size, f.Axis)
		if i < n-1 {
			main += f.Gap
			switch f.Spacing {
			case layout.SpaceEvenly:
				main += space / (1 + n)
			case layout.SpaceAround:
				main += space / n
			case layout.SpaceBetween:
				if n > 1 {
					main += space / (n - 1)
				}
			}
		}
	}
}
