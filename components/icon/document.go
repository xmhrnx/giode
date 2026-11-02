package icon

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"io/fs"
	"math"
	"os"
	"strconv"
	"strings"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op/clip"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"

	"github.com/xmhrnx/giode/internal/raster"
)

// maxSVGSize bounds the size of an SVG document LoadSVG accepts, so a
// malformed or hostile file can't exhaust memory.
const maxSVGSize = 4 << 20

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

func (d *Document) readViewBox(attrs map[string]string) error {
	if vb, ok := parseViewBox(attrs["viewBox"]); ok && vb[2] > 0 && vb[3] > 0 {
		d.viewX, d.viewY, d.viewW, d.viewH = vb[0], vb[1], vb[2], vb[3]
		return nil
	}
	if w, h, ok := parseSize(attrs["width"], attrs["height"]); ok && w > 0 && h > 0 {
		d.viewW, d.viewH = w, h
		return nil
	}
	return fmt.Errorf("giode: parse svg: no viewBox or width/height")
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

// paintAttrs are the inheritable presentation attributes.
type paintAttrs struct {
	fill, stroke string
	strokeWidth  float32
	roundCaps    bool
}

func defaultPaint() paintAttrs {
	return paintAttrs{fill: "currentColor", stroke: "none", strokeWidth: 1}
}

// inherit applies an element's own attributes (and style declarations,
// which win) over the inherited ones.
func (p paintAttrs) inherit(attrs map[string]string) paintAttrs {
	apply := func(k, v string) {
		v = strings.TrimSpace(v)
		switch k {
		case "fill":
			p.fill = normalizePaint(v)
		case "stroke":
			p.stroke = normalizePaint(v)
		case "stroke-width":
			if w, ok := parseLength(v); ok && w >= 0 {
				p.strokeWidth = w
			}
		case "stroke-linecap", "stroke-linejoin":
			p.roundCaps = v == "round"
		}
	}
	for _, k := range []string{"fill", "stroke", "stroke-width", "stroke-linecap", "stroke-linejoin"} {
		if v, ok := attrs[k]; ok {
			apply(k, v)
		}
	}
	for _, decl := range strings.Split(attrs["style"], ";") {
		if k, v, ok := strings.Cut(decl, ":"); ok {
			apply(strings.TrimSpace(k), v)
		}
	}
	return p
}

// normalizePaint maps paint values to "none", "currentColor" or a
// color. Paint servers (url(#gradient)) aren't supported and fall back
// to currentColor.
func normalizePaint(v string) string {
	switch {
	case v == "none" || v == "transparent":
		return "none"
	case v == "" || strings.EqualFold(v, "currentColor") || strings.HasPrefix(v, "url("):
		return "currentColor"
	}
	if _, err := properties.ParseColor(v); err != nil {
		return "currentColor"
	}
	return v
}

func attrMap(attrs []xml.Attr) map[string]string {
	m := make(map[string]string, len(attrs))
	for _, a := range attrs {
		m[a.Name.Local] = a.Value
	}
	return m
}

// shapePath compiles a basic shape element.
func shapePath(name string, a map[string]string) compiled {
	num := func(k string) float32 {
		v, _ := parseLength(a[k])
		return v
	}
	switch name {
	case "path":
		if a["d"] == "" {
			return nil
		}
		return compilePath(a["d"])
	case "rect":
		rx, okX := parseLength(a["rx"])
		ry, okY := parseLength(a["ry"])
		if !okX {
			rx = 0
		}
		if !okY {
			ry = 0
		}
		return rectPath(num("x"), num("y"), num("width"), num("height"), rx, ry)
	case "circle":
		r := num("r")
		return ellipsePath(num("cx"), num("cy"), r, r)
	case "ellipse":
		return ellipsePath(num("cx"), num("cy"), num("rx"), num("ry"))
	case "line":
		return polyPath([]float32{num("x1"), num("y1"), num("x2"), num("y2")}, false)
	case "polyline", "polygon":
		return polyPath(parseNumbers(a["points"]), name == "polygon")
	}
	return nil
}

// parseNumbers parses a whitespace/comma separated number list.
func parseNumbers(s string) []float32 {
	var out []float32
	for i := 0; i < len(s); {
		c := s[i]
		if c == ' ' || c == ',' || c == '\t' || c == '\n' || c == '\r' {
			i++
			continue
		}
		j := numberEnd(s, i)
		if j == i {
			i++
			continue
		}
		if v, err := strconv.ParseFloat(s[i:j], 32); err == nil {
			out = append(out, float32(v))
		}
		i = j
	}
	return out
}

// parseTransform parses an SVG transform list into an affine
// transformation. Unknown functions are ignored.
func parseTransform(s string) f32.Affine2D {
	t := f32.AffineId()
	for s = strings.TrimSpace(s); s != ""; s = strings.TrimSpace(s) {
		open := strings.IndexByte(s, '(')
		end := strings.IndexByte(s, ')')
		if open < 0 || end < open {
			break
		}
		name := strings.TrimSpace(strings.Trim(s[:open], ", "))
		args := parseNumbers(s[open+1 : end])
		s = s[end+1:]
		arg := func(i int, def float32) float32 {
			if i < len(args) {
				return args[i]
			}
			return def
		}
		var m f32.Affine2D
		switch name {
		case "matrix":
			if len(args) != 6 {
				continue
			}
			m = f32.NewAffine2D(args[0], args[2], args[4], args[1], args[3], args[5])
		case "translate":
			m = f32.AffineId().Offset(f32.Pt(arg(0, 0), arg(1, 0)))
		case "scale":
			sx := arg(0, 1)
			m = f32.AffineId().Scale(f32.Point{}, f32.Pt(sx, arg(1, sx)))
		case "rotate":
			c := f32.Pt(arg(1, 0), arg(2, 0))
			m = f32.AffineId().Rotate(c, arg(0, 0)*math.Pi/180)
		case "skewX":
			m = f32.AffineId().Shear(f32.Point{}, arg(0, 0)*math.Pi/180, 0)
		case "skewY":
			m = f32.AffineId().Shear(f32.Point{}, 0, arg(0, 0)*math.Pi/180)
		default:
			continue
		}
		// Transforms in a list apply right to left: the rightmost to
		// the coordinates first.
		t = t.Mul(m)
	}
	return t
}

// parseViewBox parses "minX minY width height", numbers separated by
// whitespace and/or commas.
func parseViewBox(s string) (v [4]float32, ok bool) {
	n := parseNumbers(s)
	if len(n) != 4 {
		return v, false
	}
	copy(v[:], n)
	return v, true
}

// parseSize parses SVG width/height attributes, tolerating a trailing
// unit like "24px".
func parseSize(w, h string) (width, height float32, ok bool) {
	pw, okW := parseLength(w)
	ph, okH := parseLength(h)
	return pw, ph, okW && okH
}

func parseLength(s string) (float32, bool) {
	s = strings.TrimSpace(s)
	for _, suffix := range []string{"px", "pt"} {
		s = strings.TrimSuffix(s, suffix)
	}
	if s == "" {
		return 0, false
	}
	n, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return 0, false
	}
	return float32(n), true
}

// docEl renders a Document.
type docEl struct {
	doc     *Document
	st      styles.Styles
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

// docKey identifies a document's rasterized image.
type docKey struct {
	doc     *Document
	size    image.Point
	col     color.NRGBA
	colored bool
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
