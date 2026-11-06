package box

import (
	"image"

	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/xmhrnx/giode/elements/internal/base"
	"github.com/xmhrnx/giode/styles/properties"
)

func (b *BoxElement) layoutBody(gtx layout.Context) layout.Dimensions {
	st := b.resolveStyles(gtx)
	if st.Display == properties.DisplayNone {
		return layout.Dimensions{}
	}
	st = st.Px(gtx.Metric)
	// Start and End sides follow the reading direction.
	rtl := gtx.Locale.Direction == system.RTL
	st.Padding, st.Margin = st.Padding.Physical(rtl), st.Margin.Physical(rtl)
	if b.stretchX && st.Width == 0 && st.WidthPct == 0 {
		st.Width = -1
	}
	if b.stretchY && st.Height == 0 && st.HeightPct == 0 {
		st.Height = -1
	}

	// Margin carves space out of the constraints; the painted content
	// is then offset inside it.
	margin := st.Margin
	gtx.Constraints = insetConstraints(gtx.Constraints, margin)

	// Fixed and expanding sizes.
	if st.Width != 0 {
		gtx.Constraints.Min.X, gtx.Constraints.Max.X = st.Width.Constrain(gtx.Constraints.Min.X, gtx.Constraints.Max.X)
	}
	if st.Height != 0 {
		gtx.Constraints.Min.Y, gtx.Constraints.Max.Y = st.Height.Constrain(gtx.Constraints.Min.Y, gtx.Constraints.Max.Y)
	}

	// Clamps, percentage sizes (which take precedence) and aspect
	// ratio.
	applyMinMax(&gtx.Constraints, st)
	applyPct(&gtx.Constraints, layout.Horizontal, int(st.WidthPct))
	applyPct(&gtx.Constraints, layout.Vertical, int(st.HeightPct))
	applyAspect(&gtx.Constraints, st)

	flow, absolute, outside, ordered := b.split()

	// Lay the content out into a macro so the background can be
	// painted underneath it once its size is known. Children's own
	// Layout calls happen inside this block, so the absolute-position
	// tracker (see absolute.go) must be pushed around it, not around
	// the later replay below - by the time that runs, every child has
	// already finished laying itself out.
	macro := op.Record(gtx.Ops)
	padding := st.Padding
	cgtx := gtx
	cgtx.Constraints = insetConstraints(gtx.Constraints, padding)
	contentOff := image.Pt(margin.Left+padding.Left, margin.Top+padding.Top)
	popAbs := base.PushOffset(contentOff)
	popColor := base.PushColor(st.Color)
	var zs *[]zPaint
	if ordered || rtl {
		// A child sets a z-index, or the locale reads right to left:
		// record every child and paint them once all are placed, in z
		// order, mirrored for right to left.
		zs = new([]zPaint)
	}
	for _, el := range outside {
		el.Layout(cgtx)
	}
	dims := b.layoutFlex(cgtx, st, flow, zs)
	if len(absolute) > 0 {
		b.layoutAbsolute(cgtx, absolute, dims.Size, zs)
	}
	if zs != nil {
		if rtl {
			mirror(*zs, dims.Size.X)
		}
		paintZ(gtx.Ops, *zs)
	}
	popColor()
	popAbs()
	content := macro.Stop()

	size := dims.Size
	size.X += padding.Left + padding.Right
	size.Y += padding.Top + padding.Bottom
	outer := image.Pt(size.X+margin.Left+margin.Right, size.Y+margin.Top+margin.Bottom)

	// A hidden box keeps its space but paints nothing and takes no
	// input: the recorded content is simply dropped.
	if st.Visibility == properties.VisibilityHidden {
		return layout.Dimensions{Size: outer}
	}

	// Opacity 0 means unset (fully opaque); it applies to the
	// background and the content.
	opacity := float32(st.Opacity)
	if opacity == 0 {
		opacity = 1
	}
	if opacity < 1 {
		defer paint.PushOpacity(gtx.Ops, opacity).Pop()
	}

	// Background stack, offset into the margin area. The shadow
	// paints first so it stays behind the box.
	defer op.Offset(image.Pt(margin.Left, margin.Top)).Push(gtx.Ops).Pop()
	if !st.Transform.IsIdentity() {
		defer op.Affine(st.Transform.Affine(size)).Push(gtx.Ops).Pop()
	}
	corners := st.Corners()
	box := image.Rectangle{Max: size}
	// Rounded shapes are drawn as rectangles plus small corner paths
	// (see properties.FillRRect); under a scale or rotation those pieces
	// could show seams, so a transformed box draws each as one path.
	exact := !st.Transform.IsIdentity()
	if exact {
		properties.PaintBoxShadowExact(gtx.Ops, box, corners, st.BoxShadow)
	} else {
		properties.PaintBoxShadowCorners(gtx.Ops, box, corners, st.BoxShadow)
	}
	if col := properties.CalcColor(st.Background); col.A != 0 {
		properties.FillRRect(gtx.Ops, box, corners, exact, func() {
			paint.ColorOp{Color: col}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
		})
	}
	if g := st.BackgroundGradient; g != nil {
		properties.FillRRect(gtx.Ops, box, corners, exact, func() { properties.PaintGradient(gtx.Ops, size, g) })
	}
	if st.BackgroundImage != nil {
		properties.FillRRect(gtx.Ops, box, corners, true, func() {
			properties.PaintBackgroundImage(gtx.Ops, box, st.BackgroundImage, st.BackgroundFit)
		})
	}
	if col := properties.CalcColor(st.BorderColor); col.A != 0 {
		properties.StrokeRRect(gtx.Ops, box, corners, int(st.BorderWidth), exact, func() {
			paint.ColorOp{Color: col}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
		})
	}

	// Replay the content at its padded position. When overflow is
	// hidden it's clipped to the padding box - inside the border, with
	// the corner radii shrunk to match, like CSS - so children can't
	// cover the border or spill past rounded corners. A rounded clip is
	// a path over the whole box, so when the content's own bounds stay
	// clear of the rounding a plain rectangle clips it instead; content
	// painting outside its bounds (a child's shadow, say) then isn't
	// rounded off. Absolutely positioned children can sit anywhere, so
	// they keep the rounded clip.
	var clipStack clip.Stack
	hidden := st.Overflow.Mode() == properties.OverflowHidden
	if hidden {
		bw := int(st.BorderWidth)
		shrink := func(v int) int { return max(0, v-bw) }
		inner := properties.Corners{TopLeft: shrink(corners.TopLeft), TopRight: shrink(corners.TopRight), BottomRight: shrink(corners.BottomRight), BottomLeft: shrink(corners.BottomLeft)}
		pbox := box.Inset(bw)
		contentRect := image.Rectangle{Max: dims.Size}.Add(image.Pt(padding.Left, padding.Top))
		if !exact && len(absolute) == 0 && properties.RoundedCornersClear(pbox, contentRect, inner) {
			clipStack = clip.Rect(pbox).Push(gtx.Ops)
		} else {
			clipStack = properties.RRect(pbox, inner).Push(gtx.Ops)
		}
	}
	off := op.Offset(image.Pt(padding.Left, padding.Top)).Push(gtx.Ops)
	content.Add(gtx.Ops)
	off.Pop()
	if hidden {
		clipStack.Pop()
	}

	// Register the hover area over the painted box. A pass-through
	// region sees pointer events without blocking children above it.
	if b.hover != nil {
		ps := pointer.PassOp{}.Push(gtx.Ops)
		cl := clip.Rect(box).Push(gtx.Ops)
		b.hover.Add(gtx.Ops)
		if b.hover.hovered && st.Cursor != "" {
			st.Cursor.CursorFor().Add(gtx.Ops)
		}
		cl.Pop()
		ps.Pop()
	}

	return layout.Dimensions{Size: outer}
}
