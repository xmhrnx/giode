package elements

import (
	"image"
	"image/color"

	"gioui.org/font"
	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"golang.org/x/image/math/fixed"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/internal/textrender"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// TextElement renders a single string with its styles. Chain OnHover
// to add hover styles.
type TextElement struct {
	str string
	st  styles.Styles

	kit.HoverTracker
	hoverSt     styles.Styles
	lastHovered bool

	// selKey identifies a selectable text's state; see Selectable.
	selKey any
}

// Text renders a string with the given styles. The styles argument is
// optional.
func Text(str string, st ...styles.Styles) *TextElement {
	var s styles.Styles
	if len(st) > 0 {
		s = st[0]
	}
	return &TextElement{str: str, st: s}
}

// OnHover sets styles applied while the pointer is over the text,
// merged on top of its base styles - color, decoration, weight,
// anything Styles' text properties support:
//
//	label := giode.Text("Docs", giode.Styles{Color: "#94a3b8"}).OnHover(giode.Styles{
//		Color:          "#f8fafc",
//		TextDecoration: giode.TextDecorationUnderline,
//		Cursor:         giode.CursorPointer,
//	})
//
// Hover state lives in the element, so create it once and reuse it
// every frame; rebuilding it in the view function resets the hover on
// each frame.
func (t *TextElement) OnHover(st styles.Styles) *TextElement {
	t.Enable()
	t.hoverSt = st
	return t
}

func (t *TextElement) flexWeight(axis layout.Axis) int {
	return weightOf(t.st, axis)
}

func (t *TextElement) percentOf(axis layout.Axis) int {
	return percentOf(t.st, axis)
}

func (t *TextElement) zIndex() int {
	return int(t.st.ZIndex)
}

// resolveStyles returns the styles to lay out and paint with: the
// base styles, with the hover styles merged on top while hovered.
func (t *TextElement) resolveStyles() styles.Styles {
	if !t.Enabled() || !t.Hovered() {
		return t.st
	}
	return styles.Merge(t.st, t.hoverSt)
}

// Layout shapes and paints the text with its decoration.
func (t *TextElement) Layout(gtx layout.Context) layout.Dimensions {
	if t.Update(gtx) != t.lastHovered {
		t.lastHovered = !t.lastHovered
		kit.MarkLive()
	}
	st := t.resolveStyles()
	if st.Display == properties.DisplayNone {
		return layout.Dimensions{}
	}

	shaper := fonts.Shaper()
	if shaper == nil {
		return layout.Dimensions{}
	}
	if st.Visibility == properties.VisibilityHidden {
		// Measure without painting: keep the space, drop the ops.
		gtx.Ops = new(op.Ops)
	}

	col := ResolveColor(st.Color)
	colMacro := op.Record(gtx.Ops)
	paint.ColorOp{Color: col}.Add(gtx.Ops)
	textColor := colMacro.Stop()

	face := font.Font{
		Weight:   st.FontWeight.Weight(),
		Style:    st.FontStyle.Style(),
		Typeface: font.Typeface(st.FontFamily),
	}
	label := widget.Label{
		Alignment:  st.TextAlign.LayoutAlignment(),
		MaxLines:   int(st.MaxLines),
		LineHeight: unit.Sp(st.LineHeight),
	}
	if st.LineHeight != 0 {
		// LineHeight is the distance between baselines, not a size Gio
		// scales by its default 1.2.
		label.LineHeightScale = 1
	}
	if st.TextOverflow == properties.TextOverflowEllipsis {
		label.Truncator = "…"
		if label.MaxLines == 0 {
			label.MaxLines = 1
		}
	}
	str := st.TabWidth.Expand(st.TextTransform.Apply(t.str))
	var dims layout.Dimensions
	if t.selKey != nil && st.Visibility != properties.VisibilityHidden {
		dims = t.layoutSelectable(gtx, st, str, face, properties.ResolveFontSize(st.FontSize), textColor)
	} else {
		// Laid out from cache and drawn as cached line images; see
		// internal/textrender.
		l := textrender.Layout(shaper, text.Parameters{
			Font:            face,
			PxPerEm:         fixed.I(gtx.Sp(properties.ResolveFontSize(st.FontSize))),
			MaxLines:        label.MaxLines,
			Truncator:       label.Truncator,
			Alignment:       label.Alignment,
			Locale:          gtx.Locale,
			LineHeight:      fixed.I(gtx.Sp(label.LineHeight)),
			LineHeightScale: label.LineHeightScale,
		}, gtx.Constraints, str)
		if st.Visibility != properties.VisibilityHidden {
			registerSelectable(gtx, str, l.Dims.Size, l.StopLines, st.Color)
			semantic.LabelOp(str).Add(gtx.Ops)
			l.Draw(gtx.Ops, shaper, col)
		}
		dims = l.Dims
	}

	paintDecoration(gtx, dims.Size, st.TextDecoration, col)
	if st.Visibility != properties.VisibilityHidden {
		recordSemantic(gtx, SemanticText, str, "", dims.Size)
	}
	t.RegisterArea(gtx, image.Pt(0, 0), dims.Size, st.Cursor)
	return dims
}

// paintDecoration draws underline or line-through decorations.
func paintDecoration(gtx layout.Context, size image.Point, deco properties.TextDecoration, col color.NRGBA) {
	if deco == properties.TextDecorationNone || deco == "" {
		return
	}
	th := max(1, gtx.Dp(1))
	var line image.Rectangle
	switch deco {
	case properties.TextDecorationLineThrough:
		line = image.Rect(0, size.Y/2, size.X, size.Y/2+th)
	default:
		line = image.Rect(0, size.Y-th, size.X, size.Y)
	}
	if line.Dx() <= 0 {
		return
	}
	defer clip.Rect(line).Push(gtx.Ops).Pop()
	paint.ColorOp{Color: col}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
}
