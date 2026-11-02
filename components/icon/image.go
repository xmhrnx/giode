package icon

import (
	"image"

	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Image renders an image scaled according to styles.Fit. Width and
// Height (in dp) fix its size.
func Image(img image.Image, st ...styles.Styles) elements.Element {
	return &imageEl{img: img, st: first(st)}
}

// imageEl adapts widget.Image to an element.
type imageEl struct {
	img image.Image
	st  styles.Styles
}

// Layout renders the image.
func (i *imageEl) Layout(gtx layout.Context) layout.Dimensions {
	st := i.st.Px(gtx.Metric)
	if st.Width > 0 {
		gtx.Constraints.Min.X, gtx.Constraints.Max.X = int(st.Width), int(st.Width)
	}
	if st.Height > 0 {
		gtx.Constraints.Min.Y, gtx.Constraints.Max.Y = int(st.Height), int(st.Height)
	}
	var fit widget.Fit
	switch i.st.Fit.Mode() {
	case properties.FitNone:
		fit = widget.Unscaled
	case properties.FitCover:
		fit = widget.Cover
	case properties.FitStretch:
		fit = widget.Fill
	default:
		fit = widget.Contain
	}
	return widget.Image{Src: properties.ImageOp(i.img), Fit: fit}.Layout(gtx)
}
