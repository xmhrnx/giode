// Package markdown renders Markdown (CommonMark with the GitHub
// extensions: tables, task lists, strikethrough, autolinks) as giode
// elements: paragraphs and headings as RichText, code blocks with a
// copy button, lists, task lists, block quotes, tables and rules.

package markdown

import (
	"bytes"
	"strings"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
	gast "github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
)

// inlines appends n's inline children as spans styled with st.
func (r *renderer) inlines(n gast.Node, st styles.Styles, spans *[]elements.Span) {
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch c := c.(type) {
		case *gast.Text:
			s := string(c.Segment.Value(r.src))
			switch {
			case c.HardLineBreak():
				s += "\n"
			case c.SoftLineBreak():
				s += " "
			}
			*spans = append(*spans, elements.SpanText(s, st))
		case *gast.String:
			*spans = append(*spans, elements.SpanText(string(c.Value), st))
		case *gast.Emphasis:
			sub := st
			if c.Level >= 2 {
				sub.FontWeight = properties.FontWeightBold
			} else {
				sub.FontStyle = properties.FontStyleItalic
			}
			r.inlines(c, sub, spans)
		case *gast.CodeSpan:
			*spans = append(*spans, elements.SpanText(r.plain(c), styles.Merge(st, r.code)))
		case *gast.Link:
			*spans = append(*spans, elements.Link(r.plain(c), string(c.Destination), styles.Merge(st, styles.Styles{Color: r.o.LinkColor})))
		case *gast.AutoLink:
			url := string(c.URL(r.src))
			label := string(c.Label(r.src))
			if c.AutoLinkType == gast.AutoLinkEmail && !strings.HasPrefix(url, "mailto:") {
				url = "mailto:" + url
			}
			*spans = append(*spans, elements.Link(label, url, styles.Merge(st, styles.Styles{Color: r.o.LinkColor})))
		case *east.Strikethrough:
			sub := st
			sub.TextDecoration = properties.TextDecorationLineThrough
			r.inlines(c, sub, spans)
		case *gast.Image:
			*spans = append(*spans, elements.SpanText(r.plain(c), styles.Merge(st, styles.Styles{Color: "theme.muted"})))
		case *east.TaskCheckBox, *gast.RawHTML:
			// Task boxes are drawn by the list item; raw HTML is dropped.
		default:
			r.inlines(c, st, spans)
		}
	}
}

// plain returns the text inside an inline node.
func (r *renderer) plain(n gast.Node) string {
	var b bytes.Buffer
	var walk func(gast.Node)
	walk = func(n gast.Node) {
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			switch c := c.(type) {
			case *gast.Text:
				b.Write(c.Segment.Value(r.src))
			case *gast.String:
				b.Write(c.Value)
			default:
				walk(c)
			}
		}
	}
	walk(n)
	return b.String()
}
