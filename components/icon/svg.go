package icon

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"

	"github.com/xmhrnx/giode/internal/raster"
)

// SVG renders SVG path data (commands M, L, H, V, C, S, Q, T, A, Z in
// absolute or relative form) filled with styles.Color, scaled from a
// 24x24 viewBox. The path is parsed once and cached, so passing the
// same string every frame is cheap.
func SVG(pathData string, st ...styles.Styles) elements.Element {
	return &svgEl{path: compileCached(pathData), viewW: 24, viewH: 24, st: first(st)}
}

// SVGCustom renders SVG path data scaled from a custom viewBox.
func SVGCustom(pathData string, viewBox image.Point, st ...styles.Styles) elements.Element {
	return &svgEl{path: compileCached(pathData), viewW: float32(viewBox.X), viewH: float32(viewBox.Y), st: first(st)}
}

// SVGStroke renders SVG path data as a stroke of the given width (in
// viewBox units) instead of a fill, scaled from a 24x24 viewBox - the
// format of outline icon sets like Lucide or Feather.
func SVGStroke(pathData string, width float32, st ...styles.Styles) elements.Element {
	return &svgEl{path: compileCached(pathData), viewW: 24, viewH: 24, st: first(st), stroke: width}
}

// svgEl renders a compiled path scaled from its viewBox.
type svgEl struct {
	path         compiled
	viewW, viewH float32
	st           styles.Styles
	stroke       float32
}

// Layout fills (or strokes) the path into the element's size, as an
// image rasterized once per path, size and color.
func (s *svgEl) Layout(gtx layout.Context) layout.Dimensions {
	size, ok := iconSize(gtx, s.st)
	if !ok {
		return layout.Dimensions{}
	}
	col := elements.ResolveColor(s.st.Color)
	img := raster.Image(svgKey{&s.path[0], len(s.path), s.viewW, s.viewH, s.stroke, size, col}, func() *image.RGBA {
		shape := raster.Shape{Path: s.path.segments()}
		if s.stroke > 0 {
			shape.Stroke, shape.StrokeWidth, shape.Round = col, s.stroke, true
		} else {
			shape.Fill = col
		}
		return raster.Render(size, viewBoxTransform(0, 0, s.viewW, s.viewH, size), []raster.Shape{shape})
	})
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	raster.Draw(gtx.Ops, img)
	return layout.Dimensions{Size: size}
}

// svgKey identifies a path's rasterized image: compiled paths are
// cached by their data, so the first segment's address identifies one.
type svgKey struct {
	path         *segment
	n            int
	viewW, viewH float32
	stroke       float32
	size         image.Point
	col          color.NRGBA
}
