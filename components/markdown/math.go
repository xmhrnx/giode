package markdown

import (
	"bytes"
	"strings"

	"gioui.org/layout"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/internal/mathhook"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	gparser "github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// mathParser is the parser with Options.Math: GFM and formulas.
var mathParser = goldmark.New(goldmark.WithExtensions(extension.GFM),
	goldmark.WithParserOptions(
		gparser.WithBlockParsers(util.Prioritized(mathBlockParser{}, 650)),
		gparser.WithInlineParsers(util.Prioritized(mathInlineParser{}, 150)),
	))

// mathBlock is a display formula: $$...$$ or \[...\] starting a line.
type mathBlock struct {
	gast.BaseBlock
	open, close string
	src         []byte
	closed      bool
}

var kindMathBlock = gast.NewNodeKind("MathBlock")

func (n *mathBlock) Kind() gast.NodeKind              { return kindMathBlock }
func (n *mathBlock) Dump(source []byte, level int)    { gast.DumpHelper(n, source, level, nil, nil) }
func (n *mathBlock) IsRaw() bool                      { return true }
func (n *mathBlock) appendLine(line []byte, end bool) { n.src = append(n.src, line...) }
func (n *mathBlock) source() string                   { return strings.TrimSpace(string(n.src)) }
func (n *mathBlock) original() string                 { return n.open + string(n.src) + n.close }
func (n *mathBlock) setClosed()                       { n.closed = true }
func (n *mathBlock) isClosed() bool                   { return n.closed }
func (n *mathBlock) delimiters() (open, close string) { return n.open, n.close }

// advanceToEOL moves to the end of the line, before its newline: the
// parser moves to the next line itself.
func advanceToEOL(reader text.Reader) {
	_, seg := reader.PeekLine()
	reader.Advance(max(seg.Len()-1, 0))
}

type mathBlockParser struct{}

func (mathBlockParser) Trigger() []byte { return []byte{'$', '\\'} }

// opener returns the closer of the display formula line opens, or "".
func opener(line []byte) (open, close string) {
	switch {
	case bytes.HasPrefix(line, []byte("$$")):
		return "$$", "$$"
	case bytes.HasPrefix(line, []byte(`\[`)):
		return `\[`, `\]`
	}
	return "", ""
}

func (mathBlockParser) Open(parent gast.Node, reader text.Reader, pc gparser.Context) (gast.Node, gparser.State) {
	line, _ := reader.PeekLine()
	pos := pc.BlockOffset()
	if pos < 0 || pos >= len(line) {
		return nil, gparser.NoChildren
	}
	open, close := opener(line[pos:])
	if open == "" {
		return nil, gparser.NoChildren
	}
	rest := line[pos+len(open):]
	n := &mathBlock{open: open, close: close}
	trimmed := bytes.TrimRight(rest, " \t\r\n")
	if i := bytes.Index(trimmed, []byte(close)); i >= 0 {
		if i+len(close) != len(trimmed) {
			// Text follows the formula: it's inline, in a paragraph.
			return nil, gparser.NoChildren
		}
		n.appendLine(trimmed[:i], true)
		n.setClosed()
	} else {
		n.appendLine(rest, false)
	}
	advanceToEOL(reader)
	return n, gparser.NoChildren
}

func (mathBlockParser) Continue(node gast.Node, reader text.Reader, pc gparser.Context) gparser.State {
	n := node.(*mathBlock)
	if n.isClosed() {
		return gparser.Close
	}
	line, _ := reader.PeekLine()
	if line == nil {
		return gparser.Close
	}
	_, close := n.delimiters()
	trimmed := bytes.TrimRight(line, " \t\r\n")
	if i := bytes.Index(trimmed, []byte(close)); i >= 0 {
		n.appendLine(trimmed[:i], true)
		n.setClosed()
		advanceToEOL(reader)
		return gparser.Close
	}
	n.appendLine(line, false)
	advanceToEOL(reader)
	return gparser.Continue | gparser.NoChildren
}

func (mathBlockParser) Close(node gast.Node, reader text.Reader, pc gparser.Context) {}
func (mathBlockParser) CanInterruptParagraph() bool                                  { return true }
func (mathBlockParser) CanAcceptIndentedLine() bool                                  { return false }

// mathInline is a formula in a line: $...$, \(...\), or $$...$$ with
// text around it.
type mathInline struct {
	gast.BaseInline
	src      string
	original string
	display  bool
}

var kindMathInline = gast.NewNodeKind("MathInline")

func (n *mathInline) Kind() gast.NodeKind           { return kindMathInline }
func (n *mathInline) Dump(source []byte, level int) { gast.DumpHelper(n, source, level, nil, nil) }

type mathInlineParser struct{}

func (mathInlineParser) Trigger() []byte { return []byte{'$', '\\'} }

func isSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }

// Parse reads a formula as GitHub does: $ opens one when no space
// follows it, and closes it when no space comes before it and no digit
// after it, so "$5 and $10" stays text.
func (mathInlineParser) Parse(parent gast.Node, block text.Reader, pc gparser.Context) gast.Node {
	line, _ := block.PeekLine()
	var open, close string
	switch {
	case bytes.HasPrefix(line, []byte(`\(`)):
		open, close = `\(`, `\)`
	case bytes.HasPrefix(line, []byte("$$")):
		open, close = "$$", "$$"
	case len(line) > 1 && line[0] == '$' && !isSpace(line[1]):
		open, close = "$", "$"
	default:
		return nil
	}
	block.Advance(len(open))
	var src []byte
	for {
		bs, _ := block.PeekLine()
		if bs == nil {
			return nil
		}
		found := -1
		for i := 0; i < len(bs); {
			if bytes.HasPrefix(bs[i:], []byte(close)) {
				ok := true
				if close == "$" {
					before := byte(' ')
					if i > 0 {
						before = bs[i-1]
					} else if len(src) > 0 {
						before = src[len(src)-1]
					}
					after := byte(' ')
					if i+1 < len(bs) {
						after = bs[i+1]
					}
					ok = !isSpace(before) && !(after >= '0' && after <= '9') && (i > 0 || len(src) > 0)
				}
				if ok {
					found = i
					break
				}
				if close == "$" {
					// A dollar that can't close the formula can't be in
					// it either: this one isn't a formula.
					return nil
				}
			}
			if bs[i] == '\\' {
				i += 2
				continue
			}
			i++
		}
		if found >= 0 {
			src = append(src, bs[:found]...)
			block.Advance(found + len(close))
			break
		}
		src = append(src, bs...)
		block.AdvanceLine()
	}
	s := strings.TrimSpace(string(src))
	if s == "" {
		return nil
	}
	return &mathInline{src: s, original: open + string(src) + close, display: open == "$$"}
}

// formula returns the formula src styled for st, or nil when it can't
// be set: it doesn't read, or package formula isn't imported.
func (r *renderer) formula(src string, display bool, st styles.Styles) elements.Element {
	render := mathhook.Get()
	if render == nil {
		return nil
	}
	el, err := render(src, display, styles.Styles{FontSize: st.FontSize, Color: st.Color})
	if err != nil || el == nil {
		return nil
	}
	return elements.Raw(el.Layout)
}

// mathSpan returns an inline formula as a span of the paragraph,
// copied as its source; one that can't be set shows its source as code.
func (r *renderer) mathSpan(n *mathInline, st styles.Styles) elements.Span {
	para := styles.Merge(r.base, st)
	if el := r.formula(n.src, n.display, para); el != nil {
		return elements.SpanElement(el, n.original)
	}
	return elements.SpanText(n.original, styles.Merge(st, r.code))
}

// mathBlockElement returns a display formula: centered, scrolling
// sideways when wider than the text, copied as its source. One that
// can't be set shows its source as a code block.
func (r *renderer) mathBlockElement(n *mathBlock) elements.Element {
	src := n.source()
	var el elements.Element
	if n.isClosed() {
		el = r.formula(src, true, r.base)
	}
	if el == nil {
		return r.codeBox(n.original(), "latex")
	}
	copyText := "$$\n" + src + "\n$$"
	return displayMath{key: "math\x00" + src, el: el, copyText: copyText, st: r.base, onLink: r.o.OnLink}
}

// displayMath centers a display formula, or scrolls it sideways when
// it's wider than the text.
type displayMath struct {
	key, copyText string
	el            elements.Element
	st            styles.Styles
	onLink        func(string)
}

func (d displayMath) Layout(gtx layout.Context) layout.Dimensions {
	line := func(align properties.TextAlign) elements.Element {
		st := d.st
		st.TextAlign = align
		return elements.RichText(st, elements.SpanElement(d.el, d.copyText))
	}
	mg := gtx
	mg.Constraints.Min.X = 0
	if w := elements.Measure(mg, d.el).Size.X; w <= gtx.Constraints.Max.X {
		return line(properties.TextAlignCenter).Layout(gtx)
	}
	return sideways(d.key, line(properties.TextAlignStart)).Layout(gtx)
}
