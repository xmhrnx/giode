package text

import (
	"image"
	"image/color"
	"strings"
	"unicode/utf8"

	"gioui.org/font"
	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements/internal/base"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/internal/textrender"
	"github.com/xmhrnx/giode/styles/properties"
)

// resolvedSpan is a span with its paragraph styles applied.
type resolvedSpan struct {
	st    spanStyle
	text  string
	url   string
	click *gesture.Click
}

// Layout shapes and paints the paragraph.
func (r *RichTextElement) Layout(gtx layout.Context) layout.Dimensions {
	dims := r.layoutSelf(gtx)
	// A right-click or long-press anywhere on it opens its context menu.
	r.AnchorContext(gtx, dims.Size)
	return dims
}

// layoutSelf lays it out without its context menu area.
func (r *RichTextElement) layoutSelf(gtx layout.Context) layout.Dimensions {
	shaper := fonts.Shaper()
	if shaper == nil || r.st.Display == properties.DisplayNone {
		return layout.Dimensions{}
	}
	gtx = withWidths(gtx, r.st)
	pst := r.st.Px(gtx.Metric)

	spans := make([]resolvedSpan, 0, len(r.spans))
	inherited := spanStyle{
		Color: r.st.Color, FontSize: r.st.FontSize, FontWeight: r.st.FontWeight,
		FontStyle: r.st.FontStyle, FontFamily: r.st.FontFamily, TabWidth: r.st.TabWidth,
	}
	if inherited.Color == "" {
		inherited.Color = base.CurrentColor()
	}
	for _, s := range r.spans {
		if s.text == "" {
			continue
		}
		rs := resolvedSpan{st: inherited.over(s.st), url: s.url}
		rs.text = rs.st.TabWidth.Expand(s.text)
		if s.url != "" {
			rs.click = linkClick(s.url, s.text)
		}
		spans = append(spans, rs)
	}
	if len(spans) == 0 {
		return layout.Dimensions{}
	}

	// Taps on links, here and in the input pass of enclosing caches.
	var links []resolvedSpan
	for _, s := range spans {
		if s.click != nil {
			links = append(links, s)
		}
	}
	if len(links) > 0 {
		r.handleLinks(gtx, links)
		kit.HandleInput(func(gtx layout.Context) bool {
			r.handleLinks(gtx, links)
			return false
		})
	}

	// Laid out from cache and drawn as cached line images; see
	// internal/textrender.
	trSpans := make([]textrender.Span, len(spans))
	colors := make([]color.NRGBA, len(spans))
	hasBackground := false
	for i, s := range spans {
		trSpans[i] = textrender.Span{
			Font: fonts.Resolve(font.Font{
				Typeface: font.Typeface(s.st.FontFamily),
				Weight:   s.st.FontWeight.Weight(),
				Style:    s.st.FontStyle.Style(),
			}),
			Size:    properties.ResolveFontSize(s.st.FontSize),
			Content: s.text,
		}
		colors[i] = base.ResolveColor(s.st.Color)
		if s.st.Background != "" {
			hasBackground = true
		}
	}
	rp := textrender.RichParams{
		Alignment:  pst.TextAlign.LayoutAlignment(),
		LineHeight: unit.Sp(pst.LineHeight),
	}
	if pst.LineHeight != 0 {
		// LineHeight is the baseline distance, as for Text.
		rp.LineHeightScale = 1
	}
	laid := textrender.LayoutRich(shaper, gtx, rp, trSpans)

	sel := r.selection(gtx)
	selected := sel != nil && sel.all

	// Backgrounds go under the text.
	if hasBackground || selected {
		rr := gtx.Dp(3)
		for _, seg := range laid.Segments {
			bg := spans[seg.Span].st.Background
			if selected {
				bg = selectionColor(spans[seg.Span].st.Color)
			}
			if bg == "" {
				continue
			}
			st := op.Offset(seg.Offset).Push(gtx.Ops)
			cl := clip.UniformRRect(image.Rectangle{Max: seg.Size}, rr).Push(gtx.Ops)
			paint.Fill(gtx.Ops, properties.CalcColor(bg))
			cl.Pop()
			st.Pop()
		}
	}

	if sel == nil && len(selAreaStack) > 0 && kit.TextSelectable() {
		var text strings.Builder
		starts := make([]int, len(spans))
		n := 0
		for i, s := range spans {
			starts[i] = n
			n += utf8.RuneCountInString(s.text)
			text.WriteString(s.text)
		}
		stops := func() []textrender.StopLine { return laid.StopLines(starts) }
		registerSelectable(gtx, text.String(), laid.Dims.Size, stops, inherited.Color)
	}
	semantic.LabelOp(r.PlainText()).Add(gtx.Ops)
	laid.Draw(gtx.Ops, shaper, func(i int) color.NRGBA { return colors[i] })
	for _, seg := range laid.Segments {
		s := spans[seg.Span]
		st := op.Offset(seg.Offset).Push(gtx.Ops)
		paintDecoration(gtx, seg.Size, s.st.TextDecoration, colors[seg.Span])
		if s.click != nil {
			cl := clip.Rect{Max: seg.Size}.Push(gtx.Ops)
			pointer.CursorPointer.Add(gtx.Ops)
			s.click.Add(gtx.Ops)
			cl.Pop()
		}
		st.Pop()
	}
	dims := laid.Dims

	plain := r.PlainText()
	if sel != nil {
		sel.layoutCopy(gtx, dims.Size, plain)
	}
	base.RecordSemanticLabeled(gtx, base.SemanticText, plain, "", dims.Size)
	return dims
}
