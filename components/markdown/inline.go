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

// inlineOut collects a block's inline content: runs of spans, broken
// by the images Options.LoadImage loads.
type inlineOut struct {
	spans []elements.Span
	parts []inlinePart
}

// inlinePart is a run of spans or, with src set, an image.
type inlinePart struct {
	spans    []elements.Span
	src, alt string
}

// image ends the current run of spans with an image.
func (o *inlineOut) image(src, alt string) {
	o.flush()
	o.parts = append(o.parts, inlinePart{src: src, alt: alt})
}

func (o *inlineOut) flush() {
	if len(o.spans) > 0 {
		o.parts = append(o.parts, inlinePart{spans: o.spans})
		o.spans = nil
	}
}

// inlines appends n's inline children as spans styled with st.
func (r *renderer) inlines(n gast.Node, st styles.Styles, out *inlineOut) {
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
			out.spans = append(out.spans, elements.SpanText(s, st))
		case *gast.String:
			out.spans = append(out.spans, elements.SpanText(string(c.Value), st))
		case *gast.Emphasis:
			sub := st
			if c.Level >= 2 {
				sub.FontWeight = properties.FontWeightBold
			} else {
				sub.FontStyle = properties.FontStyleItalic
			}
			r.inlines(c, sub, out)
		case *gast.CodeSpan:
			out.spans = append(out.spans, elements.SpanText(r.plain(c), styles.Merge(st, r.code)))
		case *gast.Link:
			if r.o.LoadImage != nil && hasImage(c) {
				// A linked picture ([![alt](src)](url)) shows as the
				// picture; text beside it stays a link.
				for k := c.FirstChild(); k != nil; k = k.NextSibling() {
					if im, ok := k.(*gast.Image); ok && len(im.Destination) > 0 {
						out.image(string(im.Destination), r.plain(im))
						continue
					}
					if t := r.plainNode(k); strings.TrimSpace(t) != "" {
						out.spans = append(out.spans, elements.Link(t, string(c.Destination), styles.Merge(st, styles.Styles{Color: r.o.LinkColor})))
					}
				}
				continue
			}
			out.spans = append(out.spans, elements.Link(r.plain(c), string(c.Destination), styles.Merge(st, styles.Styles{Color: r.o.LinkColor})))
		case *gast.AutoLink:
			url := string(c.URL(r.src))
			label := string(c.Label(r.src))
			if c.AutoLinkType == gast.AutoLinkEmail && !strings.HasPrefix(url, "mailto:") {
				url = "mailto:" + url
			}
			out.spans = append(out.spans, elements.Link(label, url, styles.Merge(st, styles.Styles{Color: r.o.LinkColor})))
		case *east.Strikethrough:
			sub := st
			sub.TextDecoration = properties.TextDecorationLineThrough
			r.inlines(c, sub, out)
		case *gast.Image:
			if r.o.LoadImage != nil && len(c.Destination) > 0 {
				out.image(string(c.Destination), r.plain(c))
				continue
			}
			out.spans = append(out.spans, elements.SpanText(r.plain(c), styles.Merge(st, styles.Styles{Color: "theme.muted"})))
		case *east.TaskCheckBox, *gast.RawHTML:
			// Task boxes are drawn by the list item; raw HTML is dropped.
		default:
			r.inlines(c, st, out)
		}
	}
}

// hasImage reports whether n has an image among its children.
func hasImage(n gast.Node) bool {
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if _, ok := c.(*gast.Image); ok {
			return true
		}
	}
	return false
}

// plainNode returns the text of n itself and inside it.
func (r *renderer) plainNode(n gast.Node) string {
	switch n := n.(type) {
	case *gast.Text:
		return string(n.Segment.Value(r.src))
	case *gast.String:
		return string(n.Value)
	}
	return r.plain(n)
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
