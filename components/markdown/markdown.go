// Package markdown renders Markdown (CommonMark with the GitHub
// extensions: tables, task lists, strikethrough, autolinks) as giode
// elements: paragraphs and headings as RichText, code blocks with a
// copy button, lists, task lists, block quotes, tables and rules.
package markdown

import (
	"bytes"
	"fmt"
	"strings"
	"sync"

	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"

	"github.com/xmhrnx/giode/components/copybutton"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Options configures how Markdown renders. The zero value works.
type Options struct {
	// Styles is the body text: Color (default theme.text), FontSize
	// (default 14), LineHeight, FontFamily.
	Styles styles.Styles
	// CodeFont is the typeface of code spans and blocks, default
	// "Go Mono" (register it, or your own monospace face, with
	// RegisterFonts).
	CodeFont properties.FontFamily
	// CodeBackground fills code spans and blocks, default a faint tint
	// of the text color.
	CodeBackground properties.Color
	// LinkColor colors links, default theme.primary.
	LinkColor properties.Color
	// OnLink handles taps on links; by default they open in the
	// browser.
	OnLink func(url string)
	// Selectable, when set, makes every paragraph, heading and code
	// block selectable (see elements.Text.Selectable); its value
	// identifies the document, as a message ID would.
	Selectable any
	// Gap is the space between blocks in dp, default 10.
	Gap int
}

var parser = goldmark.New(goldmark.WithExtensions(extension.GFM))

// docCache keeps the parsed tree of recently rendered sources, so a
// document shown every frame is parsed once.
var (
	docMu    sync.Mutex
	docCache = map[string]gast.Node{}
	docOrder []string
)

const maxCachedDocs = 64

func parse(src string) gast.Node {
	docMu.Lock()
	defer docMu.Unlock()
	if doc, ok := docCache[src]; ok {
		return doc
	}
	doc := parser.Parser().Parse(text.NewReader([]byte(src)))
	docCache[src] = doc
	docOrder = append(docOrder, src)
	if len(docOrder) > maxCachedDocs {
		delete(docCache, docOrder[0])
		docOrder = docOrder[1:]
	}
	return doc
}

// New renders src. Like Text it's stateless: build it in the view
// function every frame; parsing is cached per source, so a reply that
// streams in is re-parsed only when it changes.
//
//	giode.Markdown(reply, giode.MarkdownOptions{OnLink: openURL, Selectable: msg.ID})
func New(src string, opts ...Options) elements.Element {
	var o Options
	if len(opts) > 0 {
		o = opts[0]
	}
	r := &renderer{src: []byte(src), o: o}
	r.defaults()
	return r.blocks(parse(src))
}

type renderer struct {
	src  []byte
	o    Options
	base styles.Styles
	code styles.Styles
	// n counts selectable blocks, for their keys.
	n int
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

// selKey returns the next block's selection key, or nil.
func (r *renderer) selKey() any {
	if r.o.Selectable == nil {
		return nil
	}
	r.n++
	return [2]any{r.o.Selectable, r.n}
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
	if k := r.selKey(); k != nil {
		rt.Selectable(k)
	}
	return rt
}

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

// codeBlock renders a code block: a tinted box with the language and a
// copy button above the code.
func (r *renderer) codeBlock(n gast.Node, lang string) elements.Element {
	var b strings.Builder
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		b.Write(seg.Value(r.src))
	}
	code := strings.TrimRight(b.String(), "\n")
	fontSize := properties.FontSize(float32(properties.ResolveFontSize(r.base.FontSize)) * 0.92)
	copyBtn := elements.Keyed("giode.markdown.copy", code, func() *copybutton.CopyButton {
		return copybutton.New(func() string { return code }, styles.Styles{FontSize: 12, Color: "theme.muted"})
	})
	header := elements.Row(styles.Styles{Width: -1, Align: properties.AlignCenter, Wrap: properties.WrapNoWrap},
		elements.Box(styles.Styles{FlexGrow: 1}, elements.Text(lang, styles.Styles{Color: "theme.muted", FontSize: 12})),
		copyBtn,
	)
	body := elements.Text(code, styles.Styles{FontFamily: r.o.CodeFont, FontSize: fontSize, Color: r.base.Color, TabWidth: 4})
	if k := r.selKey(); k != nil {
		body.Selectable(k)
	}
	return elements.Stack(styles.Styles{
		Width: -1, Background: r.o.CodeBackground, BorderRadius: 8,
		Padding: properties.Inset{Top: 6, Right: 10, Bottom: 10, Left: 12}, Gap: 4,
	}, header, sideways("code\x00"+code, body))
}

// list renders a bulleted, numbered or task list.
func (r *renderer) list(n *gast.List) elements.Element {
	var items []elements.Element
	num := n.Start
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		marker := r.marker(n, c, num)
		num++
		items = append(items, elements.Row(styles.Styles{Gap: 8, Width: -1, Wrap: properties.WrapNoWrap},
			marker,
			elements.Box(styles.Styles{FlexGrow: 1}, r.blocks(c)),
		))
	}
	gap := 4
	if !n.IsTight {
		gap = r.o.Gap
	}
	return elements.Stack(styles.Styles{Gap: properties.Gap(gap), Width: -1}, items...)
}

// marker returns a list item's bullet, number or task box.
func (r *renderer) marker(list *gast.List, item gast.Node, num int) elements.Element {
	if box := taskBox(item); box != nil {
		st := styles.Styles{Width: 16, Height: 16, BorderWidth: 1, BorderColor: "theme.muted", BorderRadius: 4, Justify: properties.JustifyCenter, Align: properties.AlignCenter, Margin: properties.Inset{Top: 2}}
		if box.IsChecked {
			st.Background = "theme.primary"
			st.BorderColor = "theme.primary"
			return elements.Box(st, elements.Glyph(kit.GlyphCheck, styles.Styles{Color: "theme.onprimary", Width: 12}))
		}
		return elements.Box(st)
	}
	label := "•"
	if list.IsOrdered() {
		label = fmt.Sprintf("%d.", num)
	}
	return elements.Box(styles.Styles{MinWidth: 16}, elements.Text(label, styles.Styles{Color: "theme.muted", FontSize: r.base.FontSize, LineHeight: r.base.LineHeight}))
}

// taskBox returns the task check box at the start of a list item.
func taskBox(item gast.Node) *east.TaskCheckBox {
	first := item.FirstChild()
	if first == nil {
		return nil
	}
	box, _ := first.FirstChild().(*east.TaskCheckBox)
	return box
}

// table renders a GFM table.
func (r *renderer) table(t *east.Table) elements.Element {
	var rows []elements.Element
	for c := t.FirstChild(); c != nil; c = c.NextSibling() {
		var cells []elements.Element
		for cell := c.FirstChild(); cell != nil; cell = cell.NextSibling() {
			st := r.base
			if tc, ok := cell.(*east.TableCell); ok {
				switch tc.Alignment {
				case east.AlignCenter:
					st.TextAlign = properties.TextAlignCenter
				case east.AlignRight:
					st.TextAlign = properties.TextAlignEnd
				}
			}
			var spans []elements.Span
			r.inlines(cell, styles.Styles{}, &spans)
			rt := elements.RichText(st, spans...)
			if r.o.OnLink != nil {
				rt.OnLink(r.o.OnLink)
			}
			cells = append(cells, elements.TableCell(styles.Styles{}, rt))
		}
		if _, ok := c.(*east.TableHeader); ok {
			rows = append(rows, elements.TableHeader(styles.Styles{Background: r.o.CodeBackground}, cells...))
		} else {
			rows = append(rows, elements.TableRow(styles.Styles{}, cells...))
		}
	}
	return sideways("table\x00"+string(t.Text(r.src)), elements.Table(styles.Styles{BorderWidth: 1, BorderColor: "theme.border", BorderRadius: 6}, rows...))
}
