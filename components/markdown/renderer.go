// Package markdown renders Markdown (CommonMark with the GitHub
// extensions: tables, task lists, strikethrough, autolinks) as giode
// elements: paragraphs and headings as RichText, code blocks with a
// copy button, lists, task lists, block quotes, tables and rules.

package markdown

import (
	"strings"

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
		level := n.Level
		return elements.Accessible(r.paragraph(st, n), func(sn *elements.SemanticNode) {
			if sn.Kind == elements.SemanticText {
				sn.Kind, sn.Level = elements.SemanticHeading, level
			}
		})
	case *gast.FencedCodeBlock:
		return r.codeBlock(n, string(n.Language(r.src)))
	case *gast.CodeBlock:
		return r.codeBlock(n, "")
	case *gast.Blockquote:
		return quote(elements.Box(styles.Styles{Width: -1, Color: "theme.muted"}, r.blocks(n)))
	case *gast.List:
		return elements.AccessibleGroup(elements.SemanticNode{Kind: elements.SemanticList}, r.list(n))
	case *gast.ThematicBreak:
		return elements.Divider(styles.Styles{Color: "theme.border"})
	case *east.Table:
		return r.table(n)
	case *mathBlock:
		return r.mathBlockElement(n)
	}
	return nil
}

// paragraph renders n's inline content as one RichText, or with
// loaded images, as its runs of text and its images.
func (r *renderer) paragraph(st styles.Styles, n gast.Node) elements.Element {
	var out inlineOut
	r.inlines(n, styles.Styles{}, &out)
	if len(out.parts) == 0 {
		return r.richText(st, out.spans)
	}
	out.flush()
	var kids []elements.Element
	var pics []elements.Element
	endPics := func() {
		switch len(pics) {
		case 0:
		case 1:
			kids = append(kids, pics[0])
		default:
			kids = append(kids, elements.Row(styles.Styles{Gap: 6, Wrap: properties.WrapWrap}, pics...))
		}
		pics = nil
	}
	for _, p := range out.parts {
		if p.src != "" {
			pics = append(pics, mdImage{src: p.src, alt: p.alt, load: r.o.LoadImage, altStyle: styles.Merge(st, styles.Styles{Color: "theme.muted"})})
			continue
		}
		if blank(p.spans) {
			continue
		}
		endPics()
		kids = append(kids, r.richText(st, trimRun(p.spans)))
	}
	endPics()
	if len(kids) == 1 {
		return kids[0]
	}
	return elements.Stack(styles.Styles{Gap: 6, Width: -1}, kids...)
}

func (r *renderer) richText(st styles.Styles, spans []elements.Span) elements.Element {
	rt := elements.RichText(st, spans...)
	if r.o.OnLink != nil {
		rt.OnLink(r.o.OnLink)
	}
	return rt
}

// trimRun drops the white space at the ends of a run of text, which
// stood between it and an image.
func trimRun(spans []elements.Span) []elements.Span {
	spans = append([]elements.Span(nil), spans...)
	for len(spans) > 0 {
		t := strings.TrimLeft(spans[0].Text(), " \t\n")
		if t != "" {
			spans[0] = spans[0].WithText(t)
			break
		}
		spans = spans[1:]
	}
	for len(spans) > 0 {
		last := len(spans) - 1
		t := strings.TrimRight(spans[last].Text(), " \t\n")
		if t != "" {
			spans[last] = spans[last].WithText(t)
			break
		}
		spans = spans[:last]
	}
	return spans
}

// blank reports whether spans hold only white space.
func blank(spans []elements.Span) bool {
	for _, s := range spans {
		if strings.TrimSpace(s.Text()) != "" {
			return false
		}
	}
	return true
}
