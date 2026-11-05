package icon

import (
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/internal/raster"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// docEl renders a Document.
type docEl struct {
	doc     *Document
	st      styles.Styles
	colored bool
}

// docKey identifies a document's rasterized image.
type docKey struct {
	doc     *Document
	size    image.Point
	col     color.NRGBA
	colored bool
}

func (e *docEl) Layout(gtx layout.Context) layout.Dimensions {
	size, ok := iconSize(gtx, e.st)
	if !ok {
		return layout.Dimensions{}
	}
	d := e.doc
	current := elements.ResolveColor(e.st.Color)
	// Rasterized once per document, size and color, and drawn as an
	// image; see internal/raster.
	img := raster.Image(docKey{d, size, current, e.colored}, func() *image.RGBA {
		colorFor := func(v string) color.NRGBA {
			switch {
			case v == "none":
				return color.NRGBA{}
			case !e.colored || v == "currentColor":
				return current
			}
			c := properties.CalcColor(v)
			c.A = uint8(uint16(c.A) * uint16(current.A) / 255)
			return c
		}
		shapes := make([]raster.Shape, 0, len(d.shapes))
		for _, s := range d.shapes {
			shapes = append(shapes, raster.Shape{
				Path:        s.path.segments(),
				Fill:        colorFor(s.fill),
				Stroke:      colorFor(s.stroke),
				StrokeWidth: s.strokeWidth,
				Round:       s.roundCaps,
			})
		}
		return raster.Render(size, viewBoxTransform(d.viewX, d.viewY, d.viewW, d.viewH, size), shapes)
	})
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	raster.Draw(gtx.Ops, img)
	elements.RecordSemantic(gtx, elements.SemanticNode{Kind: elements.SemanticImage, Label: "icon"}, size)
	return layout.Dimensions{Size: size}
}

// viewBoxTransform maps a viewBox onto size, scaled uniformly to fit
// and centered (SVG's default preserveAspectRatio, xMidYMid meet).
func viewBoxTransform(x, y, w, h float32, size image.Point) f32.Affine2D {
	if w <= 0 || h <= 0 {
		return f32.AffineId()
	}
	s := min(float32(size.X)/w, float32(size.Y)/h)
	ox := (float32(size.X) - w*s) / 2
	oy := (float32(size.Y) - h*s) / 2
	return f32.AffineId().
		Offset(f32.Pt(-x, -y)).
		Scale(f32.Point{}, f32.Pt(s, s)).
		Offset(f32.Pt(ox, oy))
}
