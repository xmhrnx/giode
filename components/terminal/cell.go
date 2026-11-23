package terminal

import (
	"unicode"

	"golang.org/x/text/unicode/norm"
	"golang.org/x/text/width"
)

// Color is a cell's foreground or background color: the default, one
// of the 256 indexed colors (Indexed), or an RGB color (RGB).
type Color uint32

const (
	// DefaultColor is the terminal's default foreground or background.
	DefaultColor Color = 0
	colorIndexed Color = 1 << 24
	colorRGB     Color = 2 << 24
)

// Indexed returns indexed color i: 0-7 the standard colors, 8-15 their
// bright versions, 16-231 a 6x6x6 color cube and 232-255 grays.
func Indexed(i uint8) Color { return colorIndexed | Color(i) }

// RGB returns a 24-bit color.
func RGB(r, g, b uint8) Color { return colorRGB | Color(r)<<16 | Color(g)<<8 | Color(b) }

// Index returns the color's index and whether it's an indexed color.
func (c Color) Index() (uint8, bool) { return uint8(c), c&0xff000000 == colorIndexed }

// RGB returns the color's components and whether it's an RGB color.
func (c Color) RGB() (r, g, b uint8, ok bool) {
	return uint8(c >> 16), uint8(c >> 8), uint8(c), c&0xff000000 == colorRGB
}

// Attr is a set of text attributes.
type Attr uint16

const (
	AttrBold Attr = 1 << iota
	AttrDim
	AttrItalic
	AttrUnderline
	AttrBlink
	AttrInverse
	AttrHidden
	AttrStrike
)

// Style is how a cell's text is drawn.
type Style struct {
	FG, BG Color
	Attr   Attr
}

// Cell is one character cell of the screen.
type Cell struct {
	// Rune is the character; 0 for a blank cell.
	Rune rune
	// Width is 1, or 2 for a wide character (CJK, most emoji), whose
	// right half is the next cell, with Width 0.
	Width uint8
	Style Style
}

// Line is a row of cells. Wrapped is set when the text continues on the
// next line because it ran past the edge, rather than at a new line.
type Line struct {
	Cells   []Cell
	Wrapped bool
}

// Text returns the line's characters, with trailing blanks dropped.
func (l Line) Text() string {
	var rs []rune
	end := len(l.Cells)
	for end > 0 && l.Cells[end-1].Rune == 0 && l.Cells[end-1].Width != 0 {
		end--
	}
	for _, c := range l.Cells[:end] {
		switch {
		case c.Width == 0:
		case c.Rune == 0:
			rs = append(rs, ' ')
		default:
			rs = append(rs, c.Rune)
		}
	}
	return string(rs)
}

func blankLine(cols int, st Style) Line {
	l := Line{Cells: make([]Cell, cols)}
	fillBlank(l.Cells, st)
	return l
}

func fillBlank(cells []Cell, st Style) {
	// Erased cells keep the background color (as xterm does), nothing
	// else of the style.
	b := Cell{Width: 1, Style: Style{BG: st.BG}}
	for i := range cells {
		cells[i] = b
	}
}

// runeWidth is how many cells r takes: 0 for combining marks and other
// zero-width characters, 2 for wide ones.
func runeWidth(r rune) int {
	switch {
	case r < 0x300:
		return 1
	case unicode.Is(unicode.Mn, r), unicode.Is(unicode.Me, r), unicode.Is(unicode.Cf, r):
		return 0
	}
	switch width.LookupRune(r).Kind() {
	case width.EastAsianWide, width.EastAsianFullwidth:
		return 2
	}
	return 1
}

// compose combines base and a combining mark into one character, when
// Unicode has one ("e" and U+0301 make "é").
func compose(base, mark rune) (rune, bool) {
	s := norm.NFC.String(string([]rune{base, mark}))
	rs := []rune(s)
	if len(rs) == 1 {
		return rs[0], true
	}
	return 0, false
}
