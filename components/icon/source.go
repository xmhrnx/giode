package icon

import (
	"image"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
)

// Source is anything that renders as an icon: a *Document, a Path, an
// *IconVGIcon, or a custom type.
type Source interface {
	Icon(st ...styles.Styles) elements.Element
}

// Path is a Source from raw SVG path data, for icon sets distributed
// as path strings rather than files.
type Path struct {
	// Data is the SVG path data.
	Data string
	// ViewBox is the coordinate space of Data; zero means 24x24.
	ViewBox image.Point
	// StrokeWidth, when positive, strokes the path instead of filling
	// it, in viewBox units.
	StrokeWidth float32
}

// Icon renders the path.
func (p Path) Icon(st ...styles.Styles) elements.Element {
	vb := p.ViewBox
	if vb == (image.Point{}) {
		vb = image.Pt(24, 24)
	}
	return &svgEl{path: compileCached(p.Data), viewW: float32(vb.X), viewH: float32(vb.Y), st: first(st), stroke: p.StrokeWidth}
}
