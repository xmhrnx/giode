// Package textrender lays out and draws text for giode's text elements.
//
// Gio draws text as vector paths, which its GPU renderer rasterizes
// again on every frame - for a screen of text, most of the frame. This
// package draws each line of text as an image instead: glyphs are
// rasterized once on the CPU (golang.org/x/image/vector, from the same
// font outlines Gio uses) into a glyph cache, composed into line images
// cached by their glyphs and color, and drawn as textured rectangles,
// the way Skia and the platform text stacks do. Lines whose glyphs it
// can't rasterize (color emoji, fallback system fonts) are drawn as
// Gio's paths, so output never goes missing.
//
// Layouts are cached too: a text laid out with the same string and
// parameters as before isn't shaped or iterated again, and long text is
// laid out per paragraph, so appending to it (a streamed reply)
// re-lays out only its last paragraph.
package textrender

import (
	"image"
	"math"
	"sync"
	"sync/atomic"

	"gioui.org/text"
	gotext "github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"

	"github.com/xmhrnx/giode/fonts"
)

// vectorOnly makes every run fall back to Gio's paths, for comparing
// and debugging; see SetVectorText.
var vectorOnly atomic.Bool

// SetVectorText draws all text as Gio's vector paths instead of cached
// images, when on.
func SetVectorText(on bool) { vectorOnly.Store(on) }

// Gio packs a glyph's face index, size and glyph ID into a GlyphID; see
// gioui.org/text's newGlyphID. TestGlyphIDLayout checks this matches.
const (
	faceBits = 16
	sizeBits = 16
	gidBits  = 64 - faceBits - sizeBits
)

func splitGlyphID(g text.GlyphID) (ppem fixed.Int26_6, faceIdx int, gid gotext.GID) {
	faceIdx = int(uint64(g) >> (gidBits + sizeBits))
	ppem = fixed.Int26_6((g & ((1<<sizeBits - 1) << gidBits)) >> gidBits)
	gid = gotext.GID(g) & (1<<gidBits - 1)
	return ppem, faceIdx, gid
}

// faceTable maps the shaper's face indices to faces: the shaper loads
// the registered collection in order, skipping repeated fonts. Faces
// it loads later (system fallbacks) aren't known here, so their glyphs
// fall back to paths.
var faceTable struct {
	sync.Mutex
	shaper *text.Shaper
	faces  []*gotext.Face
}

func faces(sh *text.Shaper) []*gotext.Face {
	faceTable.Lock()
	defer faceTable.Unlock()
	if faceTable.shaper == sh {
		return faceTable.faces
	}
	faceTable.shaper = sh
	faceTable.faces = faceTable.faces[:0]
	if sh != fonts.Shaper() {
		return nil
	}
	seen := make(map[*gotext.Font]bool)
	for _, ff := range fonts.Collection() {
		f, ok := ff.Face.(interface{ Face() *gotext.Face })
		if !ok {
			faceTable.faces = nil
			return nil
		}
		face := f.Face()
		if seen[face.Font] {
			continue
		}
		seen[face.Font] = true
		faceTable.faces = append(faceTable.faces, face)
	}
	return faceTable.faces
}

// subpixel positions per pixel at which glyphs are rasterized.
const subpixels = 4

type glyphKey struct {
	id  text.GlyphID
	sub uint8
}

// glyphBitmap is a rasterized glyph: its coverage and the offset of its
// top left corner from the glyph's origin (the dot moved to a whole
// pixel). img is nil for a glyph with no outline, like a space.
type glyphBitmap struct {
	img *image.Alpha
	off image.Point
}

var glyphCache struct {
	sync.Mutex
	m map[glyphKey]*glyphBitmap
}

// maxGlyphs bounds the glyph cache; past it, the cache starts over.
const maxGlyphs = 16384

// rasterGlyph returns the glyph rasterized at a subpixel offset of
// sub/subpixels pixels, or false when it can't be: an unknown face, or
// a bitmap or color glyph.
func rasterGlyph(fs []*gotext.Face, id text.GlyphID, sub uint8) (*glyphBitmap, bool) {
	key := glyphKey{id, sub}
	glyphCache.Lock()
	if b, ok := glyphCache.m[key]; ok {
		glyphCache.Unlock()
		return b, b != nil
	}
	glyphCache.Unlock()

	b := rasterize(fs, id, sub)

	glyphCache.Lock()
	if glyphCache.m == nil || len(glyphCache.m) >= maxGlyphs {
		glyphCache.m = make(map[glyphKey]*glyphBitmap)
	}
	glyphCache.m[key] = b
	glyphCache.Unlock()
	return b, b != nil
}

func rasterize(fs []*gotext.Face, id text.GlyphID, sub uint8) *glyphBitmap {
	ppem, faceIdx, gid := splitGlyphID(id)
	if faceIdx >= len(fs) || fs[faceIdx] == nil {
		return nil
	}
	face := fs[faceIdx]
	var outline gotext.GlyphOutline
	switch d := face.GlyphData(gid).(type) {
	case gotext.GlyphOutline:
		outline = d
	case gotext.GlyphSVG:
		// As Gio does: the SVG glyph's outline, in the text color.
		outline = d.Outline
	default:
		return nil
	}
	if len(outline.Segments) == 0 {
		return &glyphBitmap{}
	}
	scale := float32(ppem) / 64 / float32(face.Upem())
	// Heights are scaled to put the x-height on whole pixels at small
	// sizes; see gridFit.
	scaleY := scale * gridFit(face, faceIdx, ppem, scale)
	dx := float32(sub) / subpixels
	minX, minY := float32(math.Inf(1)), float32(math.Inf(1))
	maxX, maxY := float32(math.Inf(-1)), float32(math.Inf(-1))
	for _, s := range outline.Segments {
		for i := 0; i < segArgs(s.Op); i++ {
			x, y := s.Args[i].X*scale+dx, -s.Args[i].Y*scaleY
			minX, maxX = min(minX, x), max(maxX, x)
			minY, maxY = min(minY, y), max(maxY, y)
		}
	}
	r := image.Rect(int(math.Floor(float64(minX)))-1, int(math.Floor(float64(minY)))-1,
		int(math.Ceil(float64(maxX)))+1, int(math.Ceil(float64(maxY)))+1)
	if r.Empty() {
		return &glyphBitmap{}
	}
	z := vector.NewRasterizer(r.Dx(), r.Dy())
	ox, oy := float32(r.Min.X), float32(r.Min.Y)
	pt := func(p opentype.SegmentPoint) (float32, float32) {
		return p.X*scale + dx - ox, -p.Y*scaleY - oy
	}
	open := false
	for _, s := range outline.Segments {
		switch s.Op {
		case opentype.SegmentOpMoveTo:
			if open {
				z.ClosePath()
			}
			z.MoveTo(pt(s.Args[0]))
			open = true
		case opentype.SegmentOpLineTo:
			z.LineTo(pt(s.Args[0]))
		case opentype.SegmentOpQuadTo:
			x1, y1 := pt(s.Args[0])
			x2, y2 := pt(s.Args[1])
			z.QuadTo(x1, y1, x2, y2)
		case opentype.SegmentOpCubeTo:
			x1, y1 := pt(s.Args[0])
			x2, y2 := pt(s.Args[1])
			x3, y3 := pt(s.Args[2])
			z.CubeTo(x1, y1, x2, y2, x3, y3)
		}
	}
	if open {
		z.ClosePath()
	}
	img := image.NewAlpha(image.Rect(0, 0, r.Dx(), r.Dy()))
	z.Draw(img, img.Bounds(), image.Opaque, image.Point{})
	return &glyphBitmap{img: img, off: r.Min}
}

// gridFitMaxPPEM is the largest size, in pixels per em, whose glyphs
// are fitted to the pixel grid: past it, stems are wide enough to look
// sharp without.
const gridFitMaxPPEM = 28

var xHeights struct {
	sync.Mutex
	m map[[2]int]float32
}

// gridFit returns the vertical scale that puts the face's x-height,
// at ppem, on a whole number of pixels - as FreeType's light hinting
// does. With the baseline on a pixel boundary too, the tops and
// bottoms of lowercase letters and the horizontal strokes resting on
// them land on pixel edges instead of blurring across two rows. Widths
// and advances are left alone, so text keeps its length.
func gridFit(face *gotext.Face, faceIdx int, ppem fixed.Int26_6, scale float32) float32 {
	if ppem > fixed.I(gridFitMaxPPEM) {
		return 1
	}
	key := [2]int{faceIdx, int(ppem)}
	xHeights.Lock()
	xh, ok := xHeights.m[key]
	xHeights.Unlock()
	if !ok {
		xh = xHeight(face) * scale
		xHeights.Lock()
		if xHeights.m == nil {
			xHeights.m = make(map[[2]int]float32)
		}
		xHeights.m[key] = xh
		xHeights.Unlock()
	}
	if xh < 2 {
		return 1
	}
	return float32(math.Round(float64(xh))) / xh
}

// xHeight is the top of the face's "x", in font units, or 0.
func xHeight(face *gotext.Face) float32 {
	gid, ok := face.NominalGlyph('x')
	if !ok {
		return 0
	}
	outline, ok := face.GlyphData(gid).(gotext.GlyphOutline)
	if !ok {
		return 0
	}
	top := float32(0)
	for _, s := range outline.Segments {
		for i := 0; i < segArgs(s.Op); i++ {
			top = max(top, s.Args[i].Y)
		}
	}
	return top
}

func segArgs(op opentype.SegmentOp) int {
	switch op {
	case opentype.SegmentOpQuadTo:
		return 2
	case opentype.SegmentOpCubeTo:
		return 3
	}
	return 1
}
