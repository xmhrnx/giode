package text

import (
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
