package imageutil

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// ImageElement renders an image scaled into its box. Chain OnHover to
// add hover styles.
type ImageElement struct {
	// Contextual gives it a context menu (Context).
	elements.Contextual

	img image.Image
	st  styles.Styles

	kit.HoverTracker
	hoverSt *styles.Styles // set with OnHover
}

// OnHover sets styles applied while the pointer is over the image,
// merged on top of its base styles - Opacity and BorderRadius are the
// properties Image paints. Hover state lives in the element, so
// create it once and reuse it every frame; rebuilding it in the view
// function resets the hover on each frame.
func (i *ImageElement) OnHover(st styles.Styles) *ImageElement {
	i.Enable()
	i.hoverSt = &st
	return i
}

// Layout renders the image.
func (i *ImageElement) Layout(gtx layout.Context) layout.Dimensions {
	dims := i.layoutSelf(gtx)
	// A right-click or long-press anywhere on it opens its context menu.
	i.AnchorContext(gtx, dims.Size)
	return dims
}

// layoutSelf lays it out without its context menu area.
func (i *ImageElement) layoutSelf(gtx layout.Context) layout.Dimensions {
	i.Update(gtx)
	st := i.st
	if i.Enabled() && i.Hovered() && i.hoverSt != nil {
		st = styles.Merge(i.st, *i.hoverSt)
	}
	st = st.Px(gtx.Metric)

	if st.Width > 0 {
		gtx.Constraints.Min.X, gtx.Constraints.Max.X = int(st.Width), int(st.Width)
	}
	if st.Height > 0 {
		gtx.Constraints.Min.Y, gtx.Constraints.Max.Y = int(st.Height), int(st.Height)
	}
	var fit widget.Fit
	switch st.Fit.Mode() {
	case properties.FitNone:
		fit = widget.Unscaled
	case properties.FitCover:
		fit = widget.Cover
	case properties.FitStretch:
		fit = widget.Fill
	default:
		fit = widget.Contain
	}
	macro := op.Record(gtx.Ops)
	dims := widget.Image{
		Src: properties.ImageOp(i.img),
		Fit: fit,
	}.Layout(gtx)
	imgOps := macro.Stop()

	opacity := float32(st.Opacity)
	if opacity == 0 {
		opacity = 1
	}
	if opacity < 1 {
		opStack := paint.PushOpacity(gtx.Ops, opacity)
		defer opStack.Pop()
	}
	if st.BorderRadius > 0 {
		defer clip.UniformRRect(image.Rectangle{Max: dims.Size}, int(st.BorderRadius)).Push(gtx.Ops).Pop()
	}
	imgOps.Add(gtx.Ops)
	i.RegisterArea(gtx, image.Pt(0, 0), dims.Size, st.Cursor)
	return dims
}
