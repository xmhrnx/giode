package input

import (
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"github.com/xmhrnx/giode/components/kit"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// unbounded mirrors the sentinel Gio uses for infinite constraints.
const unbounded = 1e6

// editorEl renders the editor with a placeholder underneath.
type editorEl struct {
	input *Input
	st    styles.Styles
}

// Layout renders the editor.
func (e *editorEl) Layout(gtx layout.Context) layout.Dimensions {
	st := e.st
	size := properties.ResolveFontSize(st.FontSize)
	// The editor fills the field so a tap anywhere inside it places
	// the caret. The Box hands its content the field's width as the
	// maximum (its flex layout clears the minimum); an unbounded width
	// (a field inside a horizontal scroll) falls back to hugging the
	// text, at least 80dp wide while empty. The height minimum, set
	// when the field has a Height, already reaches here.
	if in := e.input; in.minLines > 0 || in.maxLines > 0 {
		// Grow between min and max lines; the editor scrolls past max.
		lineH := gtx.Sp(size * 1.2)
		e.input.editor.LineHeight = size * 1.2
		e.input.editor.LineHeightScale = 1
		if in.minLines > 0 {
			gtx.Constraints.Min.Y = min(lineH*in.minLines, gtx.Constraints.Max.Y)
		}
		if in.maxLines > 0 {
			gtx.Constraints.Max.Y = max(gtx.Constraints.Min.Y, min(lineH*in.maxLines, gtx.Constraints.Max.Y))
		}
	}
	if gtx.Constraints.Max.X < unbounded {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
	} else if e.input.editor.Text() == "" {
		gtx.Constraints.Min.X = min(gtx.Dp(80), gtx.Constraints.Max.X)
	}

	textMacro := op.Record(gtx.Ops)
	paint.ColorOp{Color: properties.ResolveColor(st.Color)}.Add(gtx.Ops)
	textColor := textMacro.Stop()

	selMacro := op.Record(gtx.Ops)
	sel := properties.ResolveColor(st.Color)
	sel.A = uint8(float32(sel.A) * 0.3)
	paint.ColorOp{Color: sel}.Add(gtx.Ops)
	selColor := selMacro.Stop()

	// The placeholder is painted first so the editor (caret and
	// selection) renders on top of it.
	if e.input.editor.Text() == "" && e.input.placeholder != "" {
		hintColor := properties.ResolveColor(st.Color)
		hintColor.A = uint8(float32(hintColor.A) * 0.5)
		hint := styles.Styles{Color: properties.CalcColorReverse(hintColor), FontSize: st.FontSize}
		// The placeholder gets the field's line limit, cut off with an
		// ellipsis, so a long one can't wrap out of a single-line field.
		switch {
		case e.input.editor.SingleLine:
			hint.MaxLines, hint.TextOverflow = 1, properties.TextOverflowEllipsis
		case e.input.maxLines > 0:
			hint.MaxLines, hint.TextOverflow = properties.MaxLines(e.input.maxLines), properties.TextOverflowEllipsis
		}
		cl := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
		kit.Unselectable(elements.Text(e.input.placeholder, hint).Layout)(gtx)
		cl.Pop()
	}

	e.input.editor.Alignment = st.TextAlign.LayoutAlignment()
	dims := e.input.editor.Layout(gtx, fonts.Shaper(), font.Font{}, size, textColor, selColor)
	// Long-press (or right-click) for Cut, Copy, Paste and Select all.
	e.input.menu.layout(gtx, e.input, dims.Size)
	return dims
}
