// Package markdown renders Markdown (CommonMark with the GitHub
// extensions: tables, task lists, strikethrough, autolinks) as giode
// elements: paragraphs and headings as RichText, code blocks with a
// copy button, lists, task lists, block quotes, tables and rules.

package markdown

import (
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
	gast "github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
)

type renderer struct {
	src  []byte
	o    Options
	base styles.Styles
	code styles.Styles
}

func (r *renderer) defaults() {
	r.base = styles.Merge(styles.Styles{Color: "theme.text", FontSize: 14}, r.o.Styles)
	if r.o.CodeFont == "" {
		r.o.CodeFont = "Go Mono"
	}
	if r.o.CodeBackground == "" {
		r.o.CodeBackground = "theme.text/0.08"
	}
	if r.o.LinkColor == "" {
		r.o.LinkColor = "theme.primary"
	}
	if r.o.Gap == 0 {
		r.o.Gap = 10
	}
	r.code = styles.Styles{FontFamily: r.o.CodeFont, Background: r.o.CodeBackground}
}

// blocks renders n's block children as a column.
func (r *renderer) blocks(n gast.Node) elements.Element {
	var kids []elements.Element
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if el := r.block(c); el != nil {
			kids = append(kids, el)
		}
	}
	return elements.Stack(styles.Styles{Gap: properties.Gap(r.o.Gap), Width: -1}, kids...)
}

func (r *renderer) block(n gast.Node) elements.Element {
	switch n := n.(type) {
	case *gast.Paragraph, *gast.TextBlock:
		return r.paragraph(r.base, n)
	case *gast.Heading:
		scale := []float32{1.6, 1.4, 1.25, 1.1, 1, 0.95}[min(max(n.Level, 1), 6)-1]
		st := r.base
		st.FontSize = properties.FontSize(float32(properties.ResolveFontSize(r.base.FontSize)) * scale)
		st.FontWeight = properties.FontWeightSemiBold
		if st.LineHeight != 0 {
			st.LineHeight = properties.LineHeight(float32(st.LineHeight) * scale)
		}
		return r.paragraph(st, n)
	case *gast.FencedCodeBlock:
		return r.codeBlock(n, string(n.Language(r.src)))
	case *gast.CodeBlock:
		return r.codeBlock(n, "")
	case *gast.Blockquote:
		return quote(elements.Box(styles.Styles{Width: -1, Color: "theme.muted"}, r.blocks(n)))
	case *gast.List:
		return r.list(n)
	case *gast.ThematicBreak:
		return elements.Divider(styles.Styles{Color: "theme.border"})
	case *east.Table:
		return r.table(n)
	}
	return nil
}

// paragraph renders n's inline content as one RichText.
func (r *renderer) paragraph(st styles.Styles, n gast.Node) elements.Element {
	var spans []elements.Span
	r.inlines(n, styles.Styles{}, &spans)
	rt := elements.RichText(st, spans...)
	if r.o.OnLink != nil {
		rt.OnLink(r.o.OnLink)
	}
	return rt
}
