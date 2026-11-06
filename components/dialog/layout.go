package dialog

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles/properties"
)

// Layout updates the dialog and, while it's open, registers the
// overlay that paints it over the whole window: a dimmed scrim with the
// centered panel. It takes no space, wherever it's placed in the view.
// A click on the scrim, Back or Escape closes it unless
// Dismissible(false) was set.
func (d *Dialog) Layout(gtx layout.Context) layout.Dimensions {
	// Handles its input without an input pass; see kit.OpaqueInput.
	kit.OpaqueInput()
	if !d.Opened() {
		return layout.Dimensions{}
	}
	if !d.fixed {
		elements.HandleDismiss(func() { *d.open = false })
	}
	elements.RegisterOverlay(func() elements.Element { return elements.Raw(d.layoutOverlay) })
	return layout.Dimensions{}
}

// layoutOverlay paints the scrim and the panel over the window.
func (d *Dialog) layoutOverlay(gtx layout.Context) layout.Dimensions {
	if !d.Opened() {
		return layout.Dimensions{}
	}
	win := gtx.Constraints.Max
	if d.scrim.Clicked(gtx) && !d.fixed {
		*d.open = false
		kit.RequestFrame(gtx)
		return layout.Dimensions{}
	}
	d.scrim.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		defer clip.Rect{Max: win}.Push(gtx.Ops).Pop()
		paint.Fill(gtx.Ops, color.NRGBA{A: 0x66})
		return layout.Dimensions{Size: win}
	})

	st := d.st
	if st.Background == "" {
		st.Background = defaultBackground
	}
	if st.BorderRadius == 0 {
		st.BorderRadius = 12
	}
	if st.Padding == (properties.Inset{}) && !d.padSet {
		st.Padding = properties.UniformInset(20)
	}

	panelGtx := gtx
	panelGtx.Constraints.Min = image.Point{}
	panelGtx.Constraints.Max.X = win.X - 2*kit.Margin
	panelGtx.Constraints.Max.Y = win.Y - 2*kit.Margin

	// The panel must hug its content, so its minimum is cleared above:
	// Gio's Flex zeros a rigid child's minimum on the main axis but
	// never on the cross axis, so Dialog (however it's placed in the
	// tree) can inherit a cross-axis minimum equal to the full
	// available width, forcing the panel to fill it instead of hugging.
	//
	// The panel is centered using last frame's size and laid out in
	// place, with elements.AbsolutePosition tracking the offset, so
	// content that opens a portal (a Select's option list) anchors it
	// at its true position. Laying it out first and moving it after
	// would leave those portals at the uncentered position. On the
	// first frame, and whenever the size changes, the panel is
	// measured first and another frame is requested.
	if d.size == (image.Point{}) {
		d.size = elements.Measure(panelGtx, elements.Box(st, d.content())).Size
	}
	pos := image.Pt((win.X-d.size.X)/2, (win.Y-d.size.Y)/2)
	off := op.Offset(pos).Push(gtx.Ops)
	pop := elements.PushOffset(pos)
	dims := kit.LayoutClickable(panelGtx, &d.panel, elements.Box(st, d.content()).Layout)
	pop()
	off.Pop()
	if dims.Size != d.size {
		d.size = dims.Size
		kit.RequestFrame(gtx)
	}

	return layout.Dimensions{Size: win}
}

// OutOfFlow marks the dialog as taking no space in a Box, so the box's
// Gap leaves it out; see elements.OutOfFlow.
func (d *Dialog) OutOfFlow() {}
