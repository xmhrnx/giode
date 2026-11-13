package colorpicker

import (
	"image"
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles/properties"
)

// Sizes, in dp.
const (
	defaultWidth = 240
	barHeight    = 14
	gap          = 10
	swatchSize   = 22
	swatchGap    = 6
	thumbRadius  = 8
)

// svRes is the resolution of the square's picture, which is drawn
// scaled and smoothed.
const svRes = 64

// update applies the drags in the square and the bars, with the sizes
// of the last layout, and the preset clicks.
func (p *ColorPicker) update(gtx layout.Context) {
	c := p.c
	drag := func(d *gesture.Drag, axis gesture.Axis, apply func(pos f32.Point)) {
		for {
			ev, ok := d.Update(gtx.Metric, gtx.Source, axis)
			if !ok {
				break
			}
			if ev.Kind == pointer.Press || ev.Kind == pointer.Drag {
				apply(ev.Position)
			}
		}
	}
	frac := func(v float32, n int) float64 {
		if n <= 0 {
			return 0
		}
		return math.Min(math.Max(float64(v)/float64(n), 0), 1)
	}
	drag(&p.sv, gesture.Both, func(pos f32.Point) {
		c.s = frac(pos.X, p.svSize.X)
		c.v = 1 - frac(pos.Y, p.svSize.Y)
	})
	drag(&p.hue, gesture.Horizontal, func(pos f32.Point) {
		c.h = 360 * frac(pos.X, p.barSize.X)
		if c.h >= 360 {
			c.h = 359.999
		}
	})
	drag(&p.alphaBar, gesture.Horizontal, func(pos f32.Point) {
		c.a = frac(pos.X, p.barSize.X)
	})
	for i := range p.presets {
		if p.presets[i].click.Clicked(gtx) {
			col := p.presets[i].col
			if !p.alpha {
				col.A = 255
			}
			c = fromRGB(col, c.h)
		}
	}
	if c != p.c {
		p.set(c)
		kit.RequestFrame(gtx)
	}
	if p.sv.Dragging() || p.hue.Dragging() || p.alphaBar.Dragging() {
		kit.MarkLive()
	}
}

// Layout lays out the picker: the square, the bars, the field and the
// swatches, one under another.
func (p *ColorPicker) Layout(gtx layout.Context) layout.Dimensions {
	kit.OpaqueInput()
	p.update(gtx)

	w := gtx.Dp(defaultWidth)
	switch {
	case p.st.Width < 0:
		w = gtx.Constraints.Max.X
	case p.st.Width > 0:
		w = gtx.Dp(unit.Dp(p.st.Width))
	}
	w = min(max(w, gtx.Constraints.Min.X), gtx.Constraints.Max.X)
	svH := w * 2 / 3
	if p.st.Height > 0 {
		svH = gtx.Dp(unit.Dp(p.st.Height))
	}
	g := gtx.Dp(gap)
	y := 0

	at := func(dy int, f func()) {
		off := op.Offset(image.Pt(0, dy)).Push(gtx.Ops)
		pop := elements.PushOffset(image.Pt(0, dy))
		f()
		pop()
		off.Pop()
	}

	p.svSize = image.Pt(w, svH)
	at(y, func() { p.layoutSquare(gtx) })
	y += svH + g

	bh := gtx.Dp(barHeight)
	p.barSize = image.Pt(w, bh)
	at(y, func() { p.layoutHue(gtx) })
	y += bh + g
	if p.alpha {
		at(y, func() { p.layoutAlpha(gtx) })
		y += bh + g
	}

	at(y, func() { y += p.layoutField(gtx, w) })
	if len(p.presets) > 0 {
		y += g
		at(y, func() { y += p.layoutPresets(gtx, w) })
	}
	size := image.Pt(w, y)
	p.AnchorContext(gtx, size)
	return layout.Dimensions{Size: size}
}

// dragArea registers d over size, clear of a frameless window's drag
// area.
func dragArea(gtx layout.Context, d *gesture.Drag, size image.Point) {
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	kit.BlockDragHere(gtx)
	pointer.CursorCrosshair.Add(gtx.Ops)
	d.Add(gtx.Ops)
}

// layoutSquare draws the saturation and brightness square for the hue,
// with a ring at the color.
func (p *ColorPicker) layoutSquare(gtx layout.Context) {
	size := p.svSize
	if !p.svBuilt || p.svFor != p.c.h {
		img := image.NewNRGBA(image.Rect(0, 0, svRes, svRes))
		for y := range svRes {
			for x := range svRes {
				img.SetNRGBA(x, y, hsv{h: p.c.h, s: float64(x) / (svRes - 1), v: 1 - float64(y)/(svRes-1), a: 1}.rgb())
			}
		}
		p.svOp = paint.NewImageOp(img)
		p.svOp.Filter = paint.FilterLinear
		p.svFor, p.svBuilt = p.c.h, true
	}
	r := gtx.Dp(6)
	drawImage(gtx, p.svOp, image.Rectangle{Max: size}, r)
	dragArea(gtx, &p.sv, size)
	ring(gtx, image.Pt(int(p.c.s*float64(size.X)), int((1-p.c.v)*float64(size.Y))), withAlpha(p.c.rgb(), 255))
}

// layoutHue draws the hue bar with a ring at the hue.
func (p *ColorPicker) layoutHue(gtx layout.Context) {
	size := p.barSize
	if p.hueOp.Size() == (image.Point{}) {
		img := image.NewNRGBA(image.Rect(0, 0, 361, 1))
		for x := range 361 {
			img.SetNRGBA(x, 0, hsv{h: float64(x), s: 1, v: 1, a: 1}.rgb())
		}
		p.hueOp = paint.NewImageOp(img)
		p.hueOp.Filter = paint.FilterLinear
	}
	drawImage(gtx, p.hueOp, image.Rectangle{Max: size}, size.Y/2)
	dragArea(gtx, &p.hue, size)
	ring(gtx, image.Pt(int(p.c.h/360*float64(size.X)), size.Y/2), hsv{h: p.c.h, s: 1, v: 1, a: 1}.rgb())
}

// layoutAlpha draws the alpha bar, the color fading in over a
// checkerboard, with a ring at the alpha.
func (p *ColorPicker) layoutAlpha(gtx layout.Context) {
	size := p.barSize
	rr := clip.UniformRRect(image.Rectangle{Max: size}, size.Y/2).Push(gtx.Ops)
	checker(gtx, size)
	col := withAlpha(p.c.rgb(), 255)
	paint.LinearGradientOp{
		Stop1: f32.Pt(0, 0), Color1: withAlpha(col, 0),
		Stop2: f32.Pt(float32(size.X), 0), Color2: col,
	}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	rr.Pop()
	dragArea(gtx, &p.alphaBar, size)
	ring(gtx, image.Pt(int(p.c.a*float64(size.X)), size.Y/2), p.c.rgb())
}

// layoutField lays out the preview swatch and the hex field, returning
// their height.
func (p *ColorPicker) layoutField(gtx layout.Context, w int) int {
	fgtx := gtx
	fgtx.Constraints = layout.Constraints{Max: image.Pt(w, gtx.Constraints.Max.Y)}
	h := elements.Measure(fgtx, p.hex).Size.Y
	// The preview: the color over a checkerboard, as tall as the field.
	sq := image.Pt(h, h)
	rr := clip.UniformRRect(image.Rectangle{Max: sq}, gtx.Dp(6)).Push(gtx.Ops)
	checker(gtx, sq)
	paint.Fill(gtx.Ops, p.c.rgb())
	rr.Pop()
	properties.PaintBorder(gtx.Ops, sq, properties.BorderWidth(max(1, gtx.Dp(1))), "theme.border", properties.BorderRadius(gtx.Dp(6)))

	x := h + gtx.Dp(8)
	off := op.Offset(image.Pt(x, 0)).Push(gtx.Ops)
	pop := elements.PushOffset(image.Pt(x, 0))
	fgtx.Constraints = layout.Exact(image.Pt(max(0, w-x), h))
	p.hex.Layout(fgtx)
	pop()
	off.Pop()
	return h
}

// layoutPresets lays out the swatches in rows, returning their height.
func (p *ColorPicker) layoutPresets(gtx layout.Context, w int) int {
	s, sg := gtx.Dp(swatchSize), gtx.Dp(swatchGap)
	perRow := max(1, (w+sg)/(s+sg))
	cur := p.c.rgb()
	for i := range p.presets {
		sw := &p.presets[i]
		pt := image.Pt(i%perRow*(s+sg), i/perRow*(s+sg))
		off := op.Offset(pt).Push(gtx.Ops)
		pop := elements.PushOffset(pt)
		cgtx := gtx
		cgtx.Constraints = layout.Exact(image.Pt(s, s))
		sw.click.Layout(cgtx, func(gtx layout.Context) layout.Dimensions {
			sz := image.Pt(s, s)
			circle := clip.Ellipse{Max: sz}
			cl := circle.Push(gtx.Ops)
			checker(gtx, sz)
			paint.Fill(gtx.Ops, sw.col)
			cl.Pop()
			// A ring for the picked one and the hovered one, and a thin
			// edge so white shows on white.
			edge, width := properties.CalcColor("theme.border"), max(1, gtx.Dp(1))
			if sw.col == cur || (!p.alpha && withAlpha(sw.col, 255) == cur) {
				edge, width = properties.CalcColor("theme.text"), gtx.Dp(2)
			} else if sw.click.Hovered() {
				edge = properties.CalcColor("theme.muted")
			}
			paint.FillShape(gtx.Ops, edge, clip.Stroke{Path: circle.Path(gtx.Ops), Width: float32(width)}.Op())
			pointer.CursorPointer.Add(gtx.Ops)
			return layout.Dimensions{Size: sz}
		})
		kit.BlockDrag(gtx, image.Pt(s, s))
		pop()
		off.Pop()
	}
	rows := (len(p.presets) + perRow - 1) / perRow
	return rows*(s+sg) - sg
}

// drawImage draws img scaled into r, its corners rounded by radius.
func drawImage(gtx layout.Context, img paint.ImageOp, r image.Rectangle, radius int) {
	defer clip.UniformRRect(r, radius).Push(gtx.Ops).Pop()
	is := img.Size()
	if is.X == 0 || is.Y == 0 {
		return
	}
	// Half a source pixel in at each edge, so the smoothing doesn't
	// blend past the picture's border.
	sx := float32(r.Dx()) / float32(is.X-1)
	sy := float32(r.Dy()) / float32(is.Y-1)
	if is.Y == 1 {
		sy = float32(r.Dy())
	}
	if is.X == 1 {
		sx = float32(r.Dx())
	}
	tr := f32.Affine2D{}.Offset(f32.Pt(-0.5, -0.5)).Scale(f32.Point{}, f32.Pt(sx, sy)).Offset(f32.Pt(float32(r.Min.X), float32(r.Min.Y)))
	if is.Y == 1 {
		tr = f32.Affine2D{}.Offset(f32.Pt(-0.5, 0)).Scale(f32.Point{}, f32.Pt(sx, sy)).Offset(f32.Pt(float32(r.Min.X), float32(r.Min.Y)))
	}
	defer op.Affine(tr).Push(gtx.Ops).Pop()
	img.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
}

// checker paints a light checkerboard over size, under transparent
// colors.
func checker(gtx layout.Context, size image.Point) {
	paint.Fill(gtx.Ops, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
	c := max(1, gtx.Dp(5))
	grey := color.NRGBA{R: 0xd0, G: 0xd0, B: 0xd0, A: 0xff}
	for y := 0; y < size.Y; y += c {
		for x := (y / c % 2) * c; x < size.X; x += 2 * c {
			paint.FillShape(gtx.Ops, grey, clip.Rect{Min: image.Pt(x, y), Max: image.Pt(x+c, y+c)}.Op())
		}
	}
}

// ring draws the marker at p: the color in a white ring with a dark
// edge, visible on any background.
func ring(gtx layout.Context, p image.Point, fill color.NRGBA) {
	r := gtx.Dp(thumbRadius)
	disc := func(rad int, c color.NRGBA) {
		paint.FillShape(gtx.Ops, c, clip.Ellipse{Min: p.Sub(image.Pt(rad, rad)), Max: p.Add(image.Pt(rad, rad))}.Op(gtx.Ops))
	}
	disc(r+1, color.NRGBA{A: 0x60})
	disc(r, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
	disc(r-gtx.Dp(2), withAlpha(fill, 255))
}

func withAlpha(c color.NRGBA, a uint8) color.NRGBA {
	c.A = a
	return c
}
