package icon

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"gioui.org/f32"
)

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
