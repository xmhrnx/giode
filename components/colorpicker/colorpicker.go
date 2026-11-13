// Package colorpicker provides a color picker: a saturation and
// brightness square, a hue bar, an optional alpha bar, a hex field and
// preset swatches.
package colorpicker

import (
	"image"
	"image/color"
	"strings"

	"gioui.org/gesture"
	"gioui.org/op/paint"
	"gioui.org/widget"

	"github.com/xmhrnx/giode/components/input"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// ColorPicker is a stateful color picker. Create it once with New and
// lay it out every frame:
//
//	picker := colorpicker.New("#3366FF").
//		Presets("#000000", "#FFFFFF", "#E53935", "#43A047", "#1E88E5").
//		OnChange(func(c string) { setInk(c) })
//
// Colors are strings Styles take: "#RRGGBB", or "#RRGGBBAA" with Alpha
// on and some transparency. Dragging in the square sets saturation
// (across) and brightness (down); the bars set hue and alpha. The hex
// field takes "#RGB", "#RRGGBB" or, with Alpha, "#RRGGBBAA" (and color
// names), applying as soon as it reads as a color.
type ColorPicker struct {
	// Contextual gives it a context menu (Context).
	elements.Contextual

	c        hsv
	alpha    bool
	presets  []preset
	onChange func(c string)
	st       styles.Styles
	hex      *input.Input

	sv, hue, alphaBar gesture.Drag
	// Sizes of the square and the bars in the last layout, for the
	// drags' positions.
	svSize, barSize image.Point

	// svOp is the square's picture for svFor's hue; hueOp the hue
	// bar's, made once.
	svOp    paint.ImageOp
	svFor   float64
	svBuilt bool
	hueOp   paint.ImageOp
}

type preset struct {
	value string
	col   color.NRGBA
	click widget.Clickable
}

// New returns a picker showing c, any color Styles take ("#3366FF",
// "theme.primary"). The styles argument is optional: Width (default
// 240dp) and Height (the square's, default two thirds of the width).
func New(c string, st ...styles.Styles) *ColorPicker {
	p := &ColorPicker{c: hsv{v: 1, a: 1}}
	if len(st) > 0 {
		p.st = st[0]
	}
	p.hex = input.New("#RRGGBB").Label("Hex")
	p.hex.OnChange(p.typed).Submit(func(string) { p.showHex() }).OnBlur(p.showHex)
	p.SetValue(c)
	return p
}

// Alpha shows the alpha bar and lets colors be transparent. It returns
// p for chaining.
func (p *ColorPicker) Alpha(on bool) *ColorPicker {
	p.alpha = on
	if !on {
		p.c.a = 1
	}
	p.showHex()
	return p
}

// Presets sets the swatches shown under the field; a click picks one.
// It returns p for chaining.
func (p *ColorPicker) Presets(colors ...string) *ColorPicker {
	p.presets = make([]preset, len(colors))
	for i, c := range colors {
		p.presets[i] = preset{value: c, col: properties.CalcColor(c)}
	}
	return p
}

// OnChange sets the handler called as the user changes the color, with
// the color as Value returns it. It returns p for chaining.
func (p *ColorPicker) OnChange(fn func(c string)) *ColorPicker {
	p.onChange = fn
	return p
}

// Styles replaces the styles (see New). It returns p for chaining.
func (p *ColorPicker) Styles(st styles.Styles) *ColorPicker {
	p.st = st
	return p
}

// Value returns the color: "#RRGGBB", or "#RRGGBBAA" when it's
// transparent.
func (p *ColorPicker) Value() string {
	return hexOf(p.c.rgb())
}

// NRGBA returns the color.
func (p *ColorPicker) NRGBA() color.NRGBA {
	return p.c.rgb()
}

// SetValue shows c, any color Styles take, without calling OnChange.
func (p *ColorPicker) SetValue(c string) {
	p.SetNRGBA(properties.CalcColor(c))
}

// SetNRGBA shows c without calling OnChange.
func (p *ColorPicker) SetNRGBA(c color.NRGBA) {
	if !p.alpha {
		c.A = 255
	}
	p.c = fromRGB(c, p.c.h)
	p.showHex()
}

// Field returns the hex field.
func (p *ColorPicker) Field() *input.Input {
	return p.hex
}

// hexOf writes c as "#RRGGBB", or "#RRGGBBAA" when it's transparent.
func hexOf(c color.NRGBA) string {
	s := properties.CalcColorReverse(c)
	if c.A == 255 {
		s = s[:7]
	}
	return s
}

// showHex puts the color in the hex field.
func (p *ColorPicker) showHex() {
	p.hex.SetText(p.Value())
}

// changed reports a user change.
func (p *ColorPicker) changed() {
	if p.onChange != nil {
		p.onChange(p.Value())
	}
}

// set applies a color the user picked, outside the field.
func (p *ColorPicker) set(c hsv) {
	if !p.alpha {
		c.a = 1
	}
	if c == p.c {
		return
	}
	p.c = c
	p.showHex()
	p.changed()
}

// typed applies the field's text once it reads as a color.
func (p *ColorPicker) typed(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	if !strings.HasPrefix(text, "#") && isHex(text) {
		text = "#" + text
	}
	col, err := properties.ParseColor(text)
	if err != nil {
		return
	}
	if !p.alpha {
		col.A = 255
	}
	if col == p.c.rgb() {
		return
	}
	p.c = fromRGB(col, p.c.h)
	p.changed()
}

func isHex(s string) bool {
	for _, r := range s {
		if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
			return false
		}
	}
	return len(s) == 3 || len(s) == 4 || len(s) == 6 || len(s) == 8
}
