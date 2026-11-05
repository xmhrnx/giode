package canvas

import (
	"gioui.org/layout"
	"gioui.org/op/clip"

	"github.com/xmhrnx/giode/elements/box"
	"github.com/xmhrnx/giode/elements/internal/base"
	"github.com/xmhrnx/giode/elements/raw"
	"github.com/xmhrnx/giode/styles"
)

// Canvas is an element drawn by draw, for charts, gauges, progress
// rings and other custom graphics. It's sized like a Box by st (Width,
// Height, or -1 to fill; without a size it fills what its parent
// gives it) and draw runs every frame with a *Painter for that area:
//
//	giode.Canvas(giode.Styles{Width: 96, Height: 96}, func(p *giode.Painter) {
//		c := p.Width() / 2
//		p.Arc(c, c, c-6, 8, 0, 360, "theme.border")
//		p.Arc(c, c, c-6, 8, 0, 360*progress, "theme.primary")
//	})
//
// Coordinates and lengths are in dp from the canvas's top-left corner;
// colors are any Color, theme tokens included, and "" or
// "currentColor" is the inherited text color.
func Canvas(st styles.Styles, draw func(p *Painter)) base.Element {
	return box.Box(st, raw.Raw(func(gtx layout.Context) layout.Dimensions {
		size := gtx.Constraints.Max
		if size.X >= unboundedPx {
			size.X = gtx.Constraints.Min.X
		}
		if size.Y >= unboundedPx {
			size.Y = gtx.Constraints.Min.Y
		}
		defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
		draw(&Painter{gtx: gtx, size: size, scale: gtx.Metric.PxPerDp})
		return layout.Dimensions{Size: size}
	}))
}

// unboundedPx is where a constraint stops meaning a real size.
const unboundedPx = 1e5
