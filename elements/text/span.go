package text

import (
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Span is one run of text in a RichText paragraph, with its own
// styles merged over the paragraph's. Create it with SpanText or Link.
type Span struct {
	text string
	st   spanStyle
	url  string
}

// spanStyle is what a span honors of Styles: a tenth of their size, as
// a paragraph holds many spans and a view builds them every frame.
type spanStyle struct {
	Color          properties.Color
	FontSize       properties.FontSize
	FontWeight     properties.FontWeight
	FontStyle      properties.FontStyle
	FontFamily     properties.FontFamily
	TextDecoration properties.TextDecoration
	Background     properties.Background
	TabWidth       properties.TabWidth
}

// spanStyleOf picks out of st what a span honors.
func spanStyleOf(st styles.Styles) spanStyle {
	return spanStyle{
		Color: st.Color, FontSize: st.FontSize, FontWeight: st.FontWeight, FontStyle: st.FontStyle,
		FontFamily: st.FontFamily, TextDecoration: st.TextDecoration, Background: st.Background, TabWidth: st.TabWidth,
	}
}

// over returns s with the fields set in o replacing its own, as
// styles.Merge does.
func (s spanStyle) over(o spanStyle) spanStyle {
	if o.Color != "" {
		s.Color = o.Color
	}
	if o.FontSize != 0 {
		s.FontSize = o.FontSize
	}
	if o.FontWeight != "" {
		s.FontWeight = o.FontWeight
	}
	if o.FontStyle != "" {
		s.FontStyle = o.FontStyle
	}
	if o.FontFamily != "" {
		s.FontFamily = o.FontFamily
	}
	if o.TextDecoration != "" {
		s.TextDecoration = o.TextDecoration
	}
	if o.Background != "" {
		s.Background = o.Background
	}
	if o.TabWidth != 0 {
		s.TabWidth = o.TabWidth
	}
	return s
}

// SpanText returns a run of text styled with st (optional), merged
// over the paragraph's styles. Spans honor Color, FontSize, FontWeight,
// FontStyle, FontFamily, TextDecoration (underline, line-through),
// Background (a highlight behind the run, as for inline code) and
// TabWidth.
func SpanText(text string, st ...styles.Styles) Span {
	s := Span{text: text}
	if len(st) > 0 {
		s.st = spanStyleOf(st[0])
	}
	return s
}

// Link returns a tappable run of text that opens url: through the
// paragraph's OnLink handler when set, otherwise in the browser (see
// SetLinkOpener). It's underlined in the theme's primary color unless
// st says otherwise.
func Link(text, url string, st ...styles.Styles) Span {
	s := Span{text: text, url: url, st: spanStyle{
		Color:          "theme.primary",
		TextDecoration: properties.TextDecorationUnderline,
	}}
	if len(st) > 0 {
		s.st = s.st.over(spanStyleOf(st[0]))
	}
	return s
}
