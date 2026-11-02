package slider

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/xmhrnx/giode/styles/properties"
)

// paintTrack draws the background track.
func paintTrack(gtx layout.Context, size image.Point, track properties.Color, m metrics) {
	c := resolve(track, defaultTrack)
	trackRect := image.Rect(0, (m.height-m.trackH)/2, size.X, (m.height-m.trackH)/2+m.trackH)
	defer clip.UniformRRect(trackRect, m.trackH/2).Push(gtx.Ops).Pop()
	paint.Fill(gtx.Ops, properties.CalcColor(c))
}

// paintFill draws the filled portion up to v.
func paintFill(gtx layout.Context, size image.Point, v float32, fill properties.Color, m metrics) {
	c := resolve(fill, defaultFill)
	w := int(v * float32(size.X))
	if w <= m.trackH/2 {
		return
	}
	fillRect := image.Rect(0, (m.height-m.trackH)/2, w, (m.height-m.trackH)/2+m.trackH)
	defer clip.UniformRRect(fillRect, m.trackH/2).Push(gtx.Ops).Pop()
	paint.Fill(gtx.Ops, properties.CalcColor(c))
}

// paintThumb draws the draggable thumb.
func paintThumb(gtx layout.Context, size image.Point, v float32, thumb properties.Color, m metrics) {
	c := resolve(thumb, defaultThumb)
	cx := m.padding + int(v*float32(size.X-2*m.padding))
	r := image.Rect(cx-m.thumbD/2, (m.height-m.thumbD)/2, cx+m.thumbD/2, (m.height-m.thumbD)/2+m.thumbD)
	defer clip.Ellipse(r).Push(gtx.Ops).Pop()
	paint.Fill(gtx.Ops, properties.CalcColor(c))
}
