package icon

import (
	"encoding/xml"
	"strings"

	"github.com/xmhrnx/giode/styles/properties"
)

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
