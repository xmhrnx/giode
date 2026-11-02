package kit

import (
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/internal/raster"
)

// Glyph is one of the few small indicator shapes components draw
// themselves (chevrons, a check mark) so they don't depend on any
// icon set. Apps bring their own icons through the icon package.
type Glyph int

const (
	GlyphChevronDown Glyph = iota
	GlyphChevronUp
	GlyphChevronLeft
	GlyphChevronRight
	GlyphCheck
	GlyphClose
	GlyphMinus
	GlyphPlus
	// GlyphMaximize is a window's maximize button: a square.
	GlyphMaximize
	// GlyphRestore is a maximized window's restore button: two
	// overlapping squares.
	GlyphRestore
	// GlyphCopy is a copy-to-clipboard indicator: two stacked pages.
	GlyphCopy
	// GlyphChevronForward points in the reading direction: right, or
	// left in a right-to-left locale. Submenus, collapsed rows and
	// "next" use it.
	GlyphChevronForward
	// GlyphChevronBack points against the reading direction.
	GlyphChevronBack
)

// glyphLines holds each glyph as polylines on a 24x24 grid.
var glyphLines = map[Glyph][][]f32.Point{
	GlyphChevronDown:  {{{X: 6, Y: 9}, {X: 12, Y: 15}, {X: 18, Y: 9}}},
	GlyphChevronUp:    {{{X: 6, Y: 15}, {X: 12, Y: 9}, {X: 18, Y: 15}}},
	GlyphChevronLeft:  {{{X: 15, Y: 6}, {X: 9, Y: 12}, {X: 15, Y: 18}}},
	GlyphChevronRight: {{{X: 9, Y: 6}, {X: 15, Y: 12}, {X: 9, Y: 18}}},
	GlyphCheck:        {{{X: 5, Y: 12.5}, {X: 10, Y: 17.5}, {X: 19, Y: 7}}},
	GlyphClose:        {{{X: 6, Y: 6}, {X: 18, Y: 18}}, {{X: 18, Y: 6}, {X: 6, Y: 18}}},
	GlyphMinus:        {{{X: 5, Y: 12}, {X: 19, Y: 12}}},
	GlyphPlus:         {{{X: 5, Y: 12}, {X: 19, Y: 12}}, {{X: 12, Y: 5}, {X: 12, Y: 19}}},
	GlyphMaximize:     {{{X: 5, Y: 5}, {X: 19, Y: 5}, {X: 19, Y: 19}, {X: 5, Y: 19}, {X: 5, Y: 5}}},
	GlyphCopy: {
		{{X: 9, Y: 9}, {X: 19, Y: 9}, {X: 19, Y: 20}, {X: 9, Y: 20}, {X: 9, Y: 9}},
		{{X: 5, Y: 15}, {X: 5, Y: 4}, {X: 15, Y: 4}, {X: 15, Y: 6}},
	},
	GlyphRestore: {
		{{X: 4, Y: 8}, {X: 16, Y: 8}, {X: 16, Y: 20}, {X: 4, Y: 20}, {X: 4, Y: 8}},
		{{X: 8, Y: 8}, {X: 8, Y: 4}, {X: 20, Y: 4}, {X: 20, Y: 16}, {X: 16, Y: 16}},
	},
}

type glyphKey struct {
	g    Glyph
	size image.Point
	col  color.NRGBA
}

// DrawGlyph paints g in col at size dp, with round caps and joins, and
// returns its dimensions.
func DrawGlyph(gtx layout.Context, g Glyph, size unit.Dp, col color.NRGBA) layout.Dimensions {
	switch g {
	case GlyphChevronForward:
		g = GlyphForward(gtx)
	case GlyphChevronBack:
		g = GlyphBack(gtx)
	}
	px := gtx.Dp(size)
	sz := gtx.Constraints.Constrain(image.Pt(px, px))
	if sz.X <= 0 || sz.Y <= 0 {
		return layout.Dimensions{Size: sz}
	}
	// Rasterized once per glyph, size and color, and drawn as an
	// image; see internal/raster.
	img := raster.Image(glyphKey{g, sz, col}, func() *image.RGBA {
		scale := float32(min(sz.X, sz.Y)) / 24
		var shapes []raster.Shape
		for _, line := range glyphLines[g] {
			path := []raster.Segment{{Op: 'M', Pts: [3]f32.Point{line[0]}}}
			for _, pt := range line[1:] {
				path = append(path, raster.Segment{Op: 'L', Pts: [3]f32.Point{pt}})
			}
			shapes = append(shapes, raster.Shape{Path: path, Stroke: col, StrokeWidth: 2.2, Round: true})
		}
		return raster.Render(sz, f32.AffineId().Scale(f32.Point{}, f32.Pt(scale, scale)), shapes)
	})
	raster.Draw(gtx.Ops, img)
	return layout.Dimensions{Size: sz}
}
