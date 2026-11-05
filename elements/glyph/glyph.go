package glyph

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements/internal/base"
	"github.com/xmhrnx/giode/styles"
)

// Glyph renders one of the built-in indicator shapes (chevrons, check,
// close, plus, minus) components use, sized by Width (dp, default 16)
// and colored by Color. It is not an icon set - see the icon package
// for loading your own icons.
func Glyph(g kit.Glyph, st ...styles.Styles) base.Element {
	var s styles.Styles
	if len(st) > 0 {
		s = st[0]
	}
	return glyphEl{g: g, st: s}
}

// glyphEl draws a Glyph. It depends only on its fields, so a cache
// may keep its output (base.CacheSafe).
type glyphEl struct {
	g  kit.Glyph
	st styles.Styles
}

func (e glyphEl) Layout(gtx layout.Context) layout.Dimensions {
	size := unit.Dp(e.st.Width)
	if size <= 0 {
		size = 16
	}
	return kit.DrawGlyph(gtx, e.g, size, base.ResolveColor(e.st.Color))
}

func (glyphEl) CacheSafe() bool { return true }

// SizedPx lays child out at an exact size in physical pixels on each
// axis given as positive, leaving the other axis free. Components use
// it to apply sizes they measured themselves (already in pixels), which
// must not be scaled again the way Styles lengths (in dp) are.
func SizedPx(w, h int, child base.Element) base.Element {
	return sizedEl{w: w, h: h, child: child}
}

type sizedEl struct {
	w, h  int
	child base.Element
}

func (e sizedEl) CacheSafe() bool { return base.IsCacheSafe(e.child) }

func (e sizedEl) Layout(gtx layout.Context) layout.Dimensions {
	w, h, child := e.w, e.h, e.child
	{
		if w > 0 {
			gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
		}
		if h > 0 {
			gtx.Constraints.Min.Y, gtx.Constraints.Max.Y = h, h
		}
		dims := child.Layout(gtx)
		if w > 0 {
			dims.Size.X = w
		}
		if h > 0 {
			dims.Size.Y = h
		}
		return dims
	}
}

// OffsetPx lays child out translated by off physical pixels, keeping
// AbsolutePosition in sync.
func OffsetPx(off image.Point, child base.Element) base.Element {
	return offsetEl{off: off, child: child}
}

type offsetEl struct {
	off   image.Point
	child base.Element
}

func (e offsetEl) CacheSafe() bool { return base.IsCacheSafe(e.child) }

func (e offsetEl) Layout(gtx layout.Context) layout.Dimensions {
	pop := base.PushOffset(e.off)
	defer pop()
	defer op.Offset(e.off).Push(gtx.Ops).Pop()
	dims := e.child.Layout(gtx)
	dims.Size = dims.Size.Add(e.off)
	return dims
}
