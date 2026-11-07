package text

import (
	"strings"

	"github.com/xmhrnx/giode/elements/internal/base"
	"github.com/xmhrnx/giode/styles"
)

// RichTextElement is a wrapping paragraph of differently styled spans.
type RichTextElement struct {
	// Contextual gives it a context menu (Context).
	base.Contextual

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

// CacheSafe reports true: the element depends only on its fields, which
// don't change once it's built, and on what caches already watch (the
// theme, the selection, its input passes).
func (*RichTextElement) CacheSafe() bool { return true }
