package text

import (
	"image"

	"gioui.org/font"
	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"golang.org/x/image/math/fixed"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements/internal/base"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/internal/textrender"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// resolveStyles returns the styles to lay out and paint with: the
// base styles, with the hover styles merged on top while hovered.
func (t *TextElement) resolveStyles() styles.Styles {
	if !t.Enabled() || !t.Hovered() || t.hoverSt == nil {
		return t.st
	}
	return styles.Merge(t.st, *t.hoverSt)
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
	gtx = withWidths(gtx, st)
	if st.Visibility == properties.VisibilityHidden {
		// Measure without painting: keep the space, drop the ops.
		gtx.Ops = new(op.Ops)
	}

	col := base.ResolveColor(st.Color)
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
		base.RecordSemanticLabeled(gtx, base.SemanticText, str, "", dims.Size)
	}
	t.RegisterArea(gtx, image.Pt(0, 0), dims.Size, st.Cursor)
	return dims
}
