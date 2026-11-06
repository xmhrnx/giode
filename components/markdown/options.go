// Package markdown renders Markdown (CommonMark with the GitHub
// extensions: tables, task lists, strikethrough, autolinks) as giode
// elements: paragraphs and headings as RichText, code blocks with a
// copy button, lists, task lists, block quotes, tables and rules.

package markdown

import (
	"sync"

	"gioui.org/layout"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
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
	// Selectable, when set, makes the document selectable as a whole -
	// drag across paragraphs, headings, lists, tables and code, then
	// copy - by putting it in a SelectionArea; its value identifies the
	// document, as a message ID would. A document already inside a
	// SelectionArea is part of that area's selection, with or without
	// it.
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
	doc := r.blocks(parse(src))
	if o.Selectable == nil {
		return doc
	}
	return selectableDoc{key: o.Selectable, doc: doc}
}

// selectableDoc puts a document in its own SelectionArea, unless it's
// already inside one.
type selectableDoc struct {
	key any
	doc elements.Element
}

func (s selectableDoc) Layout(gtx layout.Context) layout.Dimensions {
	if elements.InSelectionArea() {
		return s.doc.Layout(gtx)
	}
	return elements.SelectionArea([2]any{"giode.markdown", s.key}, s.doc).Layout(gtx)
}
