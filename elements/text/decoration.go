package text

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/xmhrnx/giode/styles/properties"
)

// paintDecoration draws underline or line-through decorations.
func paintDecoration(gtx layout.Context, size image.Point, deco properties.TextDecoration, col color.NRGBA) {
	if deco == properties.TextDecorationNone || deco == "" {
		return
	}
	th := max(1, gtx.Dp(1))
	var line image.Rectangle
	switch deco {
	case properties.TextDecorationLineThrough:
		line = image.Rect(0, size.Y/2, size.X, size.Y/2+th)
	default:
		line = image.Rect(0, size.Y-th, size.X, size.Y)
	}
	if line.Dx() <= 0 {
		return
	}
	defer clip.Rect(line).Push(gtx.Ops).Pop()
	paint.ColorOp{Color: col}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
}
