// Package markdown renders Markdown (CommonMark with the GitHub
// extensions: tables, task lists, strikethrough, autolinks) as giode
// elements: paragraphs and headings as RichText, code blocks with a
// copy button, lists, task lists, block quotes, tables and rules.

package markdown

import (
	"image"
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
	// LoadImage, when set, loads the pictures of ![alt](src), in
	// paragraphs, headings and table cells, given src: FetchImage
	// reads URLs, data: URIs and files, or wrap it to resolve relative
	// paths or allow only some hosts. It's called once per src, off the
	// UI goroutine, and the image is kept for the next documents that
	// show it. An image shows at its pixel size in dp, narrowed to the
	// width there is, on a line of its own (side by side with images
	// next to it); its alt text shows while it loads and if it fails.
	// Without LoadImage, images show as their alt text.
	LoadImage func(src string) (image.Image, error)
	// Math sets formulas, with package formula imported (which adds its
	// math font): $$...$$ and \[...\] starting a line as display
	// formulas, centered; $...$ and \(...\) in a paragraph inline, on
	// the text's baseline and wrapping with the words. A $ opens a
	// formula when no space follows it and closes one when no space
	// comes before it and no digit after it, as on GitHub, so "$5 and
	// $10" stays text; \$ is a dollar sign. A selection copies a
	// formula as its source. One that doesn't read (formula.Check), or
	// without package formula, shows its source as code.
	Math bool
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

func parse(src string, math bool) gast.Node {
	key := src
	p := parser
	if math {
		key, p = "\x00math\x00"+src, mathParser
	}
	docMu.Lock()
	defer docMu.Unlock()
	if doc, ok := docCache[key]; ok {
		return doc
	}
	doc := p.Parser().Parse(text.NewReader([]byte(src)))
	docCache[key] = doc
	docOrder = append(docOrder, key)
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
	doc := r.blocks(parse(src, o.Math))
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
