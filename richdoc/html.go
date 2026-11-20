package richdoc

import (
	"strings"
	"unicode"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// ParseHTML reads pasted HTML (from a browser, Google Docs, Word, a mail
// client) into a document. Only content the model holds survives:
// scripts, styles, event handlers, unsafe links and unknown markup are
// dropped, so the result is safe to show.
func ParseHTML(src string) *Doc {
	root, err := html.Parse(strings.NewReader(src))
	if err != nil {
		return nil
	}
	h := &htmlReader{}
	h.walk(root, Marks{})
	h.flush()
	if len(h.d.Blocks) == 0 {
		h.d.Blocks = []Block{{}}
	}
	return &h.d
}

type htmlList struct{ ordered bool }

type htmlReader struct {
	d      Doc
	cur    *Block // the block text goes into, nil until there's text
	kind   Block  // the kind the next block takes
	lists  []htmlList
	quote  int
	pre    int
	inCell bool
	cell   []Span
}

// flush ends the current block.
func (h *htmlReader) flush() {
	if h.cur == nil {
		return
	}
	b := h.cur
	h.cur = nil
	if b.Kind != Code {
		// Trim the collapsed whitespace at the ends.
		if n := len(b.Spans); n > 0 {
			if b.Spans[0].Atom == nil {
				b.Spans[0].Text = strings.TrimLeftFunc(b.Spans[0].Text, unicode.IsSpace)
			}
			if b.Spans[n-1].Atom == nil {
				b.Spans[n-1].Text = strings.TrimRightFunc(b.Spans[n-1].Text, func(r rune) bool { return r == ' ' || r == '\t' })
			}
		}
	}
	b.Spans = normalize(b.Spans)
	if b.Len() == 0 && b.Kind == Paragraph {
		return
	}
	h.d.Blocks = append(h.d.Blocks, *b)
}

// start begins a block of the given kind for the text that follows.
func (h *htmlReader) start(b Block) {
	h.flush()
	h.kind = b
}

func (h *htmlReader) text(s string, m Marks) {
	if h.pre == 0 {
		s = collapseSpace(s)
	}
	if h.inCell {
		if s == " " && (len(h.cell) == 0 || strings.HasSuffix(spansText(h.cell), " ")) {
			return
		}
		h.cell = append(h.cell, Span{Text: s, Marks: m})
		return
	}
	if h.cur == nil {
		if h.pre == 0 && strings.TrimSpace(s) == "" {
			return
		}
		b := h.kind
		if b.Kind == Paragraph && h.quote > 0 {
			b = Block{Kind: Quote, Level: h.quote - 1}
		}
		h.cur = &b
	}
	if h.pre == 0 && strings.HasPrefix(s, " ") {
		if t := spansText(h.cur.Spans); t == "" || strings.HasSuffix(t, " ") || strings.HasSuffix(t, "\n") {
			s = s[1:]
		}
	}
	if h.cur.Kind == Code {
		m = Marks{}
	}
	h.cur.Spans = append(h.cur.Spans, Span{Text: s, Marks: m})
}

func collapseSpace(s string) string {
	var b strings.Builder
	space := false
	for _, r := range s {
		if unicode.IsSpace(r) && r != ' ' {
			if !space {
				b.WriteByte(' ')
			}
			space = true
			continue
		}
		space = false
		b.WriteRune(r)
	}
	return b.String()
}

func attr(n *html.Node, name string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, name) {
			return a.Val
		}
	}
	return ""
}

// styleMarks applies an element's inline style.
func styleMarks(n *html.Node, m Marks) Marks {
	style := strings.ToLower(attr(n, "style"))
	if style == "" {
		return m
	}
	for _, decl := range strings.Split(style, ";") {
		k, v, ok := strings.Cut(decl, ":")
		if !ok {
			continue
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		switch k {
		case "font-weight":
			switch v {
			case "bold", "bolder", "600", "700", "800", "900":
				m.Bold = true
			case "normal", "400", "lighter", "300":
				m.Bold = false
			}
		case "font-style":
			m.Italic = v == "italic" || v == "oblique"
		case "text-decoration", "text-decoration-line":
			if strings.Contains(v, "underline") {
				m.Underline = true
			}
			if strings.Contains(v, "line-through") {
				m.Strike = true
			}
		case "color":
			if c := safeColor(v); c != "" && !isDefaultColor(c) {
				m.Color = c
			}
		case "background-color", "background":
			if c := safeColor(v); c != "" && !isDefaultBackground(c) {
				m.Highlight = c
			}
		}
	}
	return m
}

// Pasted pages color all their text; plain black and white aren't kept.
func isDefaultColor(c string) bool {
	switch strings.ToLower(strings.ReplaceAll(c, " ", "")) {
	case "black", "#000", "#000000", "rgb(0,0,0)", "inherit", "initial", "windowtext":
		return true
	}
	return false
}

func isDefaultBackground(c string) bool {
	switch strings.ToLower(strings.ReplaceAll(c, " ", "")) {
	case "transparent", "white", "#fff", "#ffffff", "rgb(255,255,255)", "rgba(0,0,0,0)", "inherit", "initial", "none":
		return true
	}
	return false
}

func (h *htmlReader) children(n *html.Node, m Marks) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		h.walk(c, m)
	}
}

func (h *htmlReader) walk(n *html.Node, m Marks) {
	switch n.Type {
	case html.TextNode:
		h.text(n.Data, m)
		return
	case html.DocumentNode:
		h.children(n, m)
		return
	case html.ElementNode:
	default:
		return
	}
	m = styleMarks(n, m)
	switch n.DataAtom {
	case atom.Script, atom.Style, atom.Head, atom.Title, atom.Meta, atom.Link, atom.Noscript, atom.Iframe,
		atom.Object, atom.Embed, atom.Svg, atom.Template, atom.Select, atom.Button, atom.Textarea, atom.Canvas:
		return
	case atom.B, atom.Strong:
		if !strings.Contains(strings.ToLower(attr(n, "style")), "font-weight") {
			m.Bold = true
		}
	case atom.I, atom.Em, atom.Cite, atom.Var:
		m.Italic = true
	case atom.U, atom.Ins:
		m.Underline = true
	case atom.S, atom.Strike, atom.Del:
		m.Strike = true
	case atom.Code, atom.Kbd, atom.Samp, atom.Tt:
		if h.pre == 0 {
			m.Code = true
		}
	case atom.Mark:
		if m.Highlight == "" {
			m.Highlight = "#fff59d"
		}
	case atom.Font:
		if c := safeColor(attr(n, "color")); c != "" && !isDefaultColor(c) {
			m.Color = c
		}
	case atom.A:
		if u := SafeURL(attr(n, "href")); u != "" && !strings.HasPrefix(u, "#") {
			m.Link = u
		}
	case atom.Br:
		h.text("\n", m)
		return
	case atom.Img:
		if src := SafeURL(attr(n, "src")); src != "" && !h.inCell {
			h.flush()
			h.d.Blocks = append(h.d.Blocks, Block{Kind: Image, Src: src, Alt: attr(n, "alt")})
			h.kind = h.blockKind()
		} else if alt := attr(n, "alt"); alt != "" {
			h.text(alt, m)
		}
		return
	case atom.Hr:
		h.flush()
		h.d.Blocks = append(h.d.Blocks, Block{Kind: Rule})
		return
	case atom.Input:
		if strings.EqualFold(attr(n, "type"), "checkbox") && h.cur == nil && len(h.lists) > 0 {
			h.kind.Kind = Task
			h.kind.Checked = hasAttr(n, "checked")
		}
		return
	case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
		if h.inCell {
			break
		}
		h.start(Block{Kind: Heading, Level: int(n.Data[1] - '0')})
		h.children(n, m)
		h.start(h.blockKind())
		return
	case atom.P, atom.Div, atom.Section, atom.Article, atom.Header, atom.Footer, atom.Main, atom.Aside,
		atom.Nav, atom.Figure, atom.Figcaption, atom.Dd, atom.Dt, atom.Address, atom.Details, atom.Summary:
		if h.inCell {
			if len(h.cell) > 0 {
				h.cell = append(h.cell, Span{Text: " "})
			}
			h.children(n, m)
			return
		}
		// A paragraph starting a list item is the item's text.
		if h.cur != nil {
			h.start(h.blockKind())
		}
		h.children(n, m)
		h.flush()
		h.kind = h.blockKind()
		return
	case atom.Blockquote:
		h.flush()
		h.quote++
		h.kind = h.blockKind()
		h.children(n, m)
		h.flush()
		h.quote--
		h.kind = h.blockKind()
		return
	case atom.Pre:
		h.start(Block{Kind: Code, Lang: codeLang(n)})
		h.pre++
		h.children(n, Marks{})
		h.pre--
		if h.cur != nil {
			h.cur.Spans = normalize(h.cur.Spans)
			if n := len(h.cur.Spans); n > 0 {
				h.cur.Spans[n-1].Text = strings.TrimRight(h.cur.Spans[n-1].Text, "\n")
			}
		} else {
			h.d.Blocks = append(h.d.Blocks, Block{Kind: Code, Lang: h.kind.Lang})
		}
		h.flush()
		h.kind = h.blockKind()
		return
	case atom.Ul, atom.Ol:
		if h.inCell {
			break
		}
		h.flush()
		h.lists = append(h.lists, htmlList{ordered: n.DataAtom == atom.Ol})
		h.children(n, m)
		h.flush()
		h.lists = h.lists[:len(h.lists)-1]
		h.kind = h.blockKind()
		return
	case atom.Li:
		if h.inCell {
			break
		}
		h.start(h.listItem())
		h.children(n, m)
		h.flush()
		h.kind = h.blockKind()
		return
	case atom.Table:
		if h.inCell {
			break
		}
		h.flush()
		t := Block{Kind: Table}
		h.rows(n, &t)
		if len(t.Cells) > 0 {
			h.d.Blocks = append(h.d.Blocks, t)
		}
		h.kind = h.blockKind()
		return
	}
	h.children(n, m)
}

func hasAttr(n *html.Node, name string) bool {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, name) {
			return true
		}
	}
	return false
}

func codeLang(n *html.Node) string {
	for _, nn := range []*html.Node{n, n.FirstChild} {
		if nn == nil || nn.Type != html.ElementNode {
			continue
		}
		for _, c := range strings.Fields(attr(nn, "class")) {
			if l, ok := strings.CutPrefix(c, "language-"); ok {
				return l
			}
			if l, ok := strings.CutPrefix(c, "lang-"); ok {
				return l
			}
		}
	}
	return ""
}

// blockKind is the kind of a block outside any element: a list item
// continuing, a quote, or a paragraph.
func (h *htmlReader) blockKind() Block {
	if h.quote > 0 {
		return Block{Kind: Quote, Level: h.quote - 1}
	}
	return Block{}
}

func (h *htmlReader) listItem() Block {
	l := h.lists[len(h.lists)-1]
	b := Block{Kind: Bullet, Level: len(h.lists) - 1}
	if l.ordered {
		b.Kind = Numbered
	}
	return b
}

// rows reads a table's rows of cells.
func (h *htmlReader) rows(n *html.Node, t *Block) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode {
			continue
		}
		switch c.DataAtom {
		case atom.Thead, atom.Tbody, atom.Tfoot:
			h.rows(c, t)
		case atom.Tr:
			var row [][]Span
			for cell := c.FirstChild; cell != nil; cell = cell.NextSibling {
				if cell.Type != html.ElementNode || cell.DataAtom != atom.Td && cell.DataAtom != atom.Th {
					continue
				}
				h.inCell, h.cell = true, nil
				m := Marks{}
				if cell.DataAtom == atom.Th && len(t.Cells) > 0 {
					m.Bold = true
				}
				h.children(cell, styleMarks(cell, m))
				h.inCell = false
				row = append(row, trimSpans(normalize(h.cell)))
			}
			if len(row) > 0 {
				t.Cells = append(t.Cells, row)
			}
		}
	}
	// Rows of a table all have as many cells.
	w := 0
	for _, r := range t.Cells {
		w = max(w, len(r))
	}
	for i := range t.Cells {
		for len(t.Cells[i]) < w {
			t.Cells[i] = append(t.Cells[i], nil)
		}
	}
}

func trimSpans(ss []Span) []Span {
	if n := len(ss); n > 0 {
		if ss[0].Atom == nil {
			ss[0].Text = strings.TrimLeftFunc(ss[0].Text, unicode.IsSpace)
		}
		if ss[n-1].Atom == nil {
			ss[n-1].Text = strings.TrimRightFunc(ss[n-1].Text, unicode.IsSpace)
		}
	}
	return normalize(ss)
}

// HTML writes the document as HTML, for copying to other apps and
// e-mail. Text is escaped and links are only safe ones.
func (d *Doc) HTML() string {
	var b strings.Builder
	var lists []Kind // open lists by level
	closeLists := func(level int) {
		for len(lists) > level {
			if lists[len(lists)-1] == Numbered {
				b.WriteString("</ol>")
			} else {
				b.WriteString("</ul>")
			}
			lists = lists[:len(lists)-1]
		}
	}
	for i := range d.Blocks {
		blk := &d.Blocks[i]
		if !blk.Kind.IsList() {
			closeLists(0)
		} else {
			closeLists(blk.Level + 1)
			kind := blk.Kind
			if kind == Task {
				kind = Bullet
			}
			if len(lists) == blk.Level+1 && lists[blk.Level] != kind {
				closeLists(blk.Level)
			}
			for len(lists) < blk.Level+1 {
				lists = append(lists, kind)
				if kind == Numbered {
					b.WriteString("<ol>")
				} else {
					b.WriteString("<ul>")
				}
			}
		}
		switch blk.Kind {
		case Heading:
			l := string(rune('0' + max(1, min(6, blk.Level))))
			b.WriteString("<h" + l + ">" + inlineHTML(blk.Spans) + "</h" + l + ">")
		case Bullet, Numbered:
			b.WriteString("<li>" + inlineHTML(blk.Spans) + "</li>")
		case Task:
			check := ""
			if blk.Checked {
				check = " checked"
			}
			b.WriteString(`<li><input type="checkbox" disabled` + check + "> " + inlineHTML(blk.Spans) + "</li>")
		case Quote:
			b.WriteString(strings.Repeat("<blockquote>", blk.Level+1) + "<p>" + inlineHTML(blk.Spans) + "</p>" + strings.Repeat("</blockquote>", blk.Level+1))
		case Code:
			cls := ""
			if blk.Lang != "" {
				cls = ` class="language-` + html.EscapeString(blk.Lang) + `"`
			}
			b.WriteString("<pre><code" + cls + ">" + html.EscapeString(blk.Text()) + "</code></pre>")
		case Table:
			b.WriteString("<table>")
			for r, row := range blk.Cells {
				tag := "td"
				if r == 0 {
					tag = "th"
				}
				b.WriteString("<tr>")
				for _, cell := range row {
					b.WriteString("<" + tag + ">" + inlineHTML(cell) + "</" + tag + ">")
				}
				b.WriteString("</tr>")
			}
			b.WriteString("</table>")
		case Image:
			if src := SafeURL(blk.Src); src != "" {
				b.WriteString(`<img src="` + html.EscapeString(src) + `" alt="` + html.EscapeString(blk.Alt) + `">`)
			}
		case Rule:
			b.WriteString("<hr>")
		default:
			b.WriteString("<p>" + inlineHTML(blk.Spans) + "</p>")
		}
	}
	closeLists(0)
	return b.String()
}

func inlineHTML(spans []Span) string {
	var b strings.Builder
	for _, sp := range spans {
		m := sp.Marks
		t := ""
		if sp.Atom != nil {
			t = html.EscapeString(sp.Atom.Label)
			if t == "" {
				t = html.EscapeString(sp.Atom.Plain())
			}
		} else {
			t = strings.ReplaceAll(html.EscapeString(sp.Text), "\n", "<br>")
		}
		wrap := func(open, close string) { t = open + t + close }
		if m.Code {
			wrap("<code>", "</code>")
		}
		if m.Italic {
			wrap("<em>", "</em>")
		}
		if m.Bold {
			wrap("<strong>", "</strong>")
		}
		if m.Strike {
			wrap("<s>", "</s>")
		}
		if m.Underline {
			wrap("<u>", "</u>")
		}
		if c := safeColor(m.Highlight); c != "" {
			wrap(`<mark style="background-color: `+c+`">`, "</mark>")
		}
		if c := safeColor(m.Color); c != "" {
			wrap(`<span style="color: `+c+`">`, "</span>")
		}
		if u := SafeURL(m.Link); u != "" {
			wrap(`<a href="`+html.EscapeString(u)+`">`, "</a>")
		}
		b.WriteString(t)
	}
	return b.String()
}
