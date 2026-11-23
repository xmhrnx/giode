package terminal

import (
	"image/color"

	"github.com/xmhrnx/giode/styles/properties"
)

// Palette is a terminal's colors. Each is any Color, a "theme.*" token
// included.
type Palette struct {
	Foreground, Background properties.Color
	// Cursor is the cursor's color; empty is the foreground.
	Cursor properties.Color
	// Selection fills selected text; empty is the foreground at 30%.
	Selection properties.Color
	// ANSI are the 16 indexed colors: black, red, green, yellow, blue,
	// magenta, cyan and white, then their bright versions.
	ANSI [16]properties.Color
}

// DefaultDarkPalette and DefaultLightPalette are the palettes for the
// dark and light modes, in effect until Terminal.Palette sets another.
var (
	DefaultDarkPalette = Palette{
		Foreground: "#d4d4d4", Background: "#1e1e1e",
		ANSI: [16]properties.Color{
			"#000000", "#cd3131", "#0dbc79", "#e5e510", "#2472c8", "#bc3fbc", "#11a8cd", "#e5e5e5",
			"#666666", "#f14c4c", "#23d18b", "#f5f543", "#3b8eea", "#d670d6", "#29b8db", "#ffffff",
		},
	}
	DefaultLightPalette = Palette{
		Foreground: "#1f2328", Background: "#ffffff",
		ANSI: [16]properties.Color{
			"#000000", "#cd3131", "#00bc00", "#949800", "#0451a5", "#bc05bc", "#0598bc", "#555555",
			"#666666", "#cd3131", "#14ce14", "#b5ba00", "#0451a5", "#bc05bc", "#0598bc", "#a5a5a5",
		},
	}
)

// resolved is a palette with its colors resolved, and the 256 indexed
// colors.
type resolved struct {
	fg, bg, cursor, selection color.NRGBA
	indexed                   [256]color.NRGBA
}

func (p Palette) resolve() *resolved {
	r := &resolved{
		fg: properties.CalcColor(p.Foreground),
		bg: properties.CalcColor(p.Background),
	}
	r.cursor = r.fg
	if p.Cursor != "" {
		r.cursor = properties.CalcColor(p.Cursor)
	}
	r.selection = r.fg
	r.selection.A = 77
	if p.Selection != "" {
		r.selection = properties.CalcColor(p.Selection)
	}
	for i, c := range p.ANSI {
		r.indexed[i] = properties.CalcColor(c)
	}
	levels := [6]uint8{0, 95, 135, 175, 215, 255}
	for i := range 216 {
		r.indexed[16+i] = color.NRGBA{R: levels[i/36], G: levels[i/6%6], B: levels[i%6], A: 255}
	}
	for i := range 24 {
		v := uint8(8 + 10*i)
		r.indexed[232+i] = color.NRGBA{R: v, G: v, B: v, A: 255}
	}
	return r
}

// color resolves c, def standing for the default color.
func (r *resolved) color(c Color, def color.NRGBA) color.NRGBA {
	if i, ok := c.Index(); ok {
		return r.indexed[i]
	}
	if cr, cg, cb, ok := c.RGB(); ok {
		return color.NRGBA{R: cr, G: cg, B: cb, A: 255}
	}
	return def
}

// colors returns a cell style's text and background colors.
func (r *resolved) colors(st Style, reverse bool) (fg, bg color.NRGBA) {
	dfg, dbg := r.fg, r.bg
	if reverse {
		dfg, dbg = dbg, dfg
	}
	fg, bg = r.color(st.FG, dfg), r.color(st.BG, dbg)
	if st.Attr&AttrInverse != 0 {
		fg, bg = bg, fg
	}
	if st.Attr&AttrDim != 0 {
		fg = blend(fg, bg)
	}
	if st.Attr&AttrHidden != 0 {
		fg = bg
	}
	return fg, bg
}

func blend(a, b color.NRGBA) color.NRGBA {
	return color.NRGBA{R: uint8((int(a.R) + int(b.R)) / 2), G: uint8((int(a.G) + int(b.G)) / 2), B: uint8((int(a.B) + int(b.B)) / 2), A: a.A}
}
