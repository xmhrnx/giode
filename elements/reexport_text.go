package elements

import (
	"github.com/xmhrnx/giode/elements/text"
)

// Text, RichText and selection, re-exported from elements/text.
type (
	// TextElement renders a single string with its styles.
	TextElement = text.TextElement
	// RichTextElement is a wrapping paragraph of differently styled
	// spans.
	RichTextElement = text.RichTextElement
	// Span is one run of text in a RichText paragraph.
	Span = text.Span
)

var (
	Text          = text.Text
	RichText      = text.RichText
	SpanText      = text.SpanText
	Link          = text.Link
	SetLinkOpener = text.SetLinkOpener
	SelectionArea = text.SelectionArea
)
