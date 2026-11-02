package elements

import (
	"image"
	"image/color"
	"strings"
	"sync/atomic"
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
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/internal/textrender"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Span is one run of text in a RichText paragraph, with its own
// styles merged over the paragraph's. Create it with SpanText or Link.
type Span struct {
	text string
	st   styles.Styles
	url  string
}

// SpanText returns a run of text styled with st (optional), merged
// over the paragraph's styles. Spans honor Color, FontSize, FontWeight,
// FontStyle, FontFamily, TextDecoration (underline, line-through),
// Background (a highlight behind the run, as for inline code) and
// TabWidth.
func SpanText(text string, st ...styles.Styles) Span {
	s := Span{text: text}
	if len(st) > 0 {
		s.st = st[0]
	}
	return s
}

// Link returns a tappable run of text that opens url: through the
// paragraph's OnLink handler when set, otherwise in the browser (see
// SetLinkOpener). It's underlined in the theme's primary color unless
// st says otherwise.
func Link(text, url string, st ...styles.Styles) Span {
	s := Span{text: text, url: url, st: styles.Styles{
		Color:          "theme.primary",
		TextDecoration: properties.TextDecorationUnderline,
	}}
	if len(st) > 0 {
		s.st = styles.Merge(s.st, st[0])
	}
	return s
}

// linkOpener opens links no OnLink handler took; the app sets it to
// App.OpenURL.
var linkOpener atomic.Pointer[func(url string)]

// SetLinkOpener sets what opens a Link whose paragraph has no OnLink
// handler. The app sets it to open the browser; component authors
// don't need to.
func SetLinkOpener(fn func(url string)) {
	if fn == nil {
		linkOpener.Store(nil)
		return
	}
	linkOpener.Store(&fn)
}

// RichTextElement is a wrapping paragraph of differently styled spans.
type RichTextElement struct {
	st     styles.Styles
	spans  []Span
	onLink func(url string)
	selKey any
}

// RichText lays out spans as one wrapping paragraph:
//
//	giode.RichText(giode.Styles{FontSize: 15, LineHeight: 22},
//		giode.SpanText("Run "),
//		giode.SpanText("go test", giode.Styles{FontFamily: "JetBrains Mono", Background: "theme.surface"}),
//		giode.SpanText(" or read the "),
//		giode.Link("docs", "https://gioui.org"),
//	).OnLink(func(url string) { ... })
//
// st styles the paragraph: Color, FontSize, FontWeight, FontStyle and
// FontFamily are the defaults spans inherit; TextAlign, LineHeight and
// TabWidth apply to the whole paragraph. Like Text it's stateless:
// build it in the view function every frame.
func RichText(st styles.Styles, spans ...Span) *RichTextElement {
	return &RichTextElement{st: st, spans: spans}
}

// OnLink sets the handler for taps on links, replacing the default of
// opening them in the browser. It returns r for chaining.
func (r *RichTextElement) OnLink(fn func(url string)) *RichTextElement {
	r.onLink = fn
	return r
}

// Selectable lets the user copy the paragraph: a long-press on a touch
// screen, or a right-click, selects it all and offers Copy. key
// identifies the paragraph across frames (a message ID, say), since
// RichText itself is rebuilt every frame. Selecting part of a rich
// paragraph isn't supported; Text.Selectable is. It returns r for
// chaining.
func (r *RichTextElement) Selectable(key any) *RichTextElement {
	r.selKey = key
	return r
}

// PlainText returns the paragraph's text without styling.
func (r *RichTextElement) PlainText() string {
	var b strings.Builder
	for _, s := range r.spans {
		b.WriteString(s.text)
	}
	return r.st.TabWidth.Expand(b.String())
}

// linkKey identifies a link's click state across frames.
type linkKey struct {
	url, text string
	n         int
}

// linkCount numbers repeated identical links within a frame, so each
// keeps its own click state.
var (
	linkCount      = map[linkKey]int{}
	linkCountFrame uint64
)

func linkClick(url, text string) *gesture.Click {
	if linkCountFrame != frameNumber {
		clear(linkCount)
		linkCountFrame = frameNumber
	}
	k := linkKey{url: url, text: text}
	k.n = linkCount[k]
	linkCount[linkKey{url: url, text: text}]++
	return Keyed("giode.richtext.link", k, func() *gesture.Click { return new(gesture.Click) })
}

// resolvedSpan is a span with its paragraph styles applied.
type resolvedSpan struct {
	st    styles.Styles
	text  string
	url   string
	click *gesture.Click
}

// Layout shapes and paints the paragraph.
func (r *RichTextElement) Layout(gtx layout.Context) layout.Dimensions {
	shaper := fonts.Shaper()
	if shaper == nil || r.st.Display == properties.DisplayNone {
		return layout.Dimensions{}
	}
	pst := r.st.Px(gtx.Metric)

	spans := make([]resolvedSpan, 0, len(r.spans))
	base := styles.Styles{
		Color: r.st.Color, FontSize: r.st.FontSize, FontWeight: r.st.FontWeight,
		FontStyle: r.st.FontStyle, FontFamily: r.st.FontFamily, TabWidth: r.st.TabWidth,
	}
	if base.Color == "" {
		base.Color = CurrentColor()
	}
	for _, s := range r.spans {
		if s.text == "" {
			continue
		}
		rs := resolvedSpan{st: styles.Merge(base, s.st), url: s.url}
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
			Font: font.Font{
				Typeface: font.Typeface(s.st.FontFamily),
				Weight:   s.st.FontWeight.Weight(),
				Style:    s.st.FontStyle.Style(),
			},
			Size:    properties.ResolveFontSize(s.st.FontSize),
			Content: s.text,
		}
		colors[i] = ResolveColor(s.st.Color)
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
		registerSelectable(gtx, text.String(), laid.Dims.Size, stops, base.Color)
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
	recordSemantic(gtx, SemanticText, plain, "", dims.Size)
	return dims
}

// handleLinks opens the links tapped since the last frame.
func (r *RichTextElement) handleLinks(gtx layout.Context, links []resolvedSpan) {
	for _, s := range links {
		for {
			e, ok := s.click.Update(gtx.Source)
			if !ok {
				break
			}
			if e.Kind == gesture.KindClick {
				r.openLink(s.url)
			}
		}
	}
}

func (r *RichTextElement) openLink(url string) {
	if r.onLink != nil {
		r.onLink(url)
		return
	}
	if fn := linkOpener.Load(); fn != nil {
		(*fn)(url)
	}
}

func (r *RichTextElement) selection(gtx layout.Context) *selectState {
	if r.selKey == nil {
		return nil
	}
	s := Keyed("giode.selectable", r.selKey, newSelectState)
	s.update(gtx)
	kit.HandleInput(s.passFunc())
	return s
}

// selectionColor is the highlight behind selected text of color c.
func selectionColor(c properties.Color) properties.Color {
	col := ResolveColor(c)
	col.A = uint8(float32(col.A) * 0.3)
	return properties.CalcColorReverse(col)
}
