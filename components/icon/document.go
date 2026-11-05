package icon

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"

	"gioui.org/f32"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
)

// Document is a parsed SVG icon: its viewBox and every painted shape,
// compiled once so rendering it costs no parsing. Load one with
// LoadSVG, LoadSVGFile or LoadSVGFS, or register it by name (see
// Register, RegisterFS) and render it with Named.
//
// Supported: <path>, <rect>, <circle>, <ellipse>, <line>, <polyline>
// and <polygon>, nested in any number of <g> groups; fill, stroke,
// stroke-width and stroke-linecap/linejoin (as attributes or in a
// style attribute), inherited from the root and groups as in SVG; and
// transform (matrix, translate, scale, rotate, skewX, skewY) on any
// element. That covers fill-based sets (Material, Font Awesome,
// Bootstrap Icons) and stroke-based ones (Lucide, Feather, Tabler,
// Heroicons outline). Gradients, masks, text and filters are ignored.
type Document struct {
	viewX, viewY float32
	viewW, viewH float32
	shapes       []shape
}

// shape is one painted element of a document.
type shape struct {
	path        compiled
	fill        string // "none", "currentColor" or a color
	stroke      string
	strokeWidth float32
	roundCaps   bool
}

// LoadSVG parses an SVG document. It returns an error for malformed
// XML, a document over 4MB, or one with neither a viewBox nor
// width/height attributes.
func LoadSVG(data []byte) (*Document, error) {
	if len(data) > maxSVGSize {
		return nil, fmt.Errorf("giode: parse svg: document too large (%d bytes)", len(data))
	}
	d := &Document{}
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false

	type frame struct {
		attrs paintAttrs
		xf    f32.Affine2D
		skip  bool
	}
	var stack []frame
	sawRoot := false
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("giode: parse svg: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			parent := frame{attrs: defaultPaint(), xf: f32.AffineId()}
			if len(stack) > 0 {
				parent = stack[len(stack)-1]
			}
			attrs := attrMap(t.Attr)
			cur := frame{
				attrs: parent.attrs.inherit(attrs),
				xf:    parent.xf.Mul(parseTransform(attrs["transform"])),
				skip:  parent.skip || attrs["display"] == "none",
			}
			name := t.Name.Local
			switch {
			case name == "svg" && !sawRoot:
				sawRoot = true
				if err := d.readViewBox(attrs); err != nil {
					return nil, err
				}
				// The root's own transform doesn't apply to its
				// viewBox mapping, but does to its content.
			case name == "defs" || name == "clipPath" || name == "mask" || name == "symbol" ||
				name == "linearGradient" || name == "radialGradient" || name == "pattern" || name == "title" ||
				name == "desc" || name == "metadata" || name == "style":
				// Not painted directly.
				cur.skip = true
			case !cur.skip:
				if p := shapePath(name, attrs); p != nil {
					d.addShape(p, cur.attrs, cur.xf)
				}
			}
			stack = append(stack, cur)
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	if !sawRoot {
		return nil, fmt.Errorf("giode: parse svg: no <svg> element")
	}
	return d, nil
}

// MustLoadSVG is LoadSVG, panicking on error - convenient for an
// icon's bytes known at compile time, e.g. via go:embed.
func MustLoadSVG(data []byte) *Document {
	d, err := LoadSVG(data)
	if err != nil {
		panic(err)
	}
	return d
}

// LoadSVGFile reads and parses an SVG file.
func LoadSVGFile(path string) (*Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("giode: load svg: %w", err)
	}
	return LoadSVG(data)
}

// LoadSVGFS reads and parses an SVG file from fsys, e.g. an embed.FS.
func LoadSVGFS(fsys fs.FS, path string) (*Document, error) {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, fmt.Errorf("giode: load svg: %w", err)
	}
	return LoadSVG(data)
}

// ViewBox returns the document's viewBox.
func (d *Document) ViewBox() (x, y, w, h float32) {
	return d.viewX, d.viewY, d.viewW, d.viewH
}

// Icon renders the document in a single color: every fill and stroke
// paints with styles.Color (default black), so icons follow the theme
// like text does. Size comes from styles.Width/Height (default 24dp);
// the drawing is scaled uniformly to fit and centered.
func (d *Document) Icon(st ...styles.Styles) elements.Element {
	return &docEl{doc: d, st: first(st)}
}

// IconColored renders the document with the colors from the file;
// fills and strokes set to currentColor (or left unset) paint with
// styles.Color. Use it for multi-color icons and illustrations.
func (d *Document) IconColored(st ...styles.Styles) elements.Element {
	return &docEl{doc: d, st: first(st), colored: true}
}

func (d *Document) addShape(p compiled, a paintAttrs, xf f32.Affine2D) {
	if a.fill == "none" && a.stroke == "none" {
		return
	}
	sw := a.strokeWidth
	if xf != f32.AffineId() {
		p = p.transform(xf)
		// Scale the stroke with the transform's average scale.
		sx, hx, _, hy, sy, _ := xf.Elems()
		sw *= float32(math.Sqrt(math.Abs(float64(sx*sy - hx*hy))))
	}
	d.shapes = append(d.shapes, shape{
		path:        p,
		fill:        a.fill,
		stroke:      a.stroke,
		strokeWidth: sw,
		roundCaps:   a.roundCaps,
	})
}

// maxSVGSize bounds the size of an SVG document LoadSVG accepts, so a
// malformed or hostile file can't exhaust memory.
const maxSVGSize = 4 << 20
