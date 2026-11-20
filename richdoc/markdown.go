package richdoc

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// mdUnescape resolves backslash escapes and entities in Markdown text.
func mdUnescape(b []byte) string {
	return string(util.ResolveEntityNames(util.ResolveNumericReferences(util.UnescapePunctuations(b))))
}

// Markdown writes the document as GitHub-flavored Markdown. Underline,
// color and highlight, which Markdown lacks, are written as <u>,
// <span style="color: ..."> and <mark>, which ParseMarkdown reads back;
// atoms are links to kind:id ([@Ada](mention:42)), emoji as their text.
func (d *Doc) Markdown() string {
	var b strings.Builder
	num := map[int]int{} // numbered list counters by level
	for i := range d.Blocks {
		blk := &d.Blocks[i]
		if i > 0 {
			prev := &d.Blocks[i-1]
			switch {
			case blk.Kind.IsList() && prev.Kind.IsList():
				b.WriteString("\n")
			case blk.Kind == Quote && prev.Kind == Quote:
				b.WriteString("\n" + strings.Repeat(">", min(blk.Level, prev.Level)+1) + "\n")
			default:
				b.WriteString("\n\n")
			}
		}
		if blk.Kind != Numbered {
			if !blk.Kind.IsList() {
				clear(num)
			}
		}
		indent := strings.Repeat("   ", blk.Level)
		switch blk.Kind {
		case Heading:
			b.WriteString(strings.Repeat("#", max(1, min(6, blk.Level))) + " " + trimEnd(inlineMarkdown(blk.Spans, false)))
		case Bullet:
			b.WriteString(indent + "- " + trimEnd(inlineMarkdown(blk.Spans, false)))
		case Numbered:
			for l := range num {
				if l > blk.Level {
					delete(num, l)
				}
			}
			num[blk.Level]++
			b.WriteString(indent + strconv.Itoa(num[blk.Level]) + ". " + trimEnd(inlineMarkdown(blk.Spans, false)))
		case Task:
			box := "[ ] "
			if blk.Checked {
				box = "[x] "
			}
			b.WriteString(indent + "- " + box + trimEnd(inlineMarkdown(blk.Spans, false)))
		case Quote:
			b.WriteString(strings.Repeat(">", blk.Level+1) + " " + trimEnd(inlineMarkdown(blk.Spans, false)))
		case Code:
			t := blk.Text()
			fence := "```"
			for strings.Contains(t, fence) {
				fence += "`"
			}
			b.WriteString(fence + blk.Lang + "\n" + t + "\n" + fence)
		case Table:
			for r, row := range blk.Cells {
				b.WriteString("|")
				for _, cell := range row {
					b.WriteString(" " + strings.ReplaceAll(inlineMarkdown(cell, false), "|", `\|`) + " |")
				}
				b.WriteString("\n")
				if r == 0 {
					b.WriteString("|" + strings.Repeat(" --- |", len(row)) + "\n")
				}
			}
			s := b.String()
			b.Reset()
			b.WriteString(strings.TrimSuffix(s, "\n"))
		case Image:
			fmt.Fprintf(&b, "![%s](%s)", escapeMarkdown(blk.Alt), mdURL(blk.Src))
		case Rule:
			b.WriteString("---")
		default:
			t := trimEnd(inlineMarkdown(blk.Spans, true))
			if t == "" && i > 0 && i < len(d.Blocks)-1 {
				t = "&nbsp;" // an empty paragraph survives the round trip
			}
			b.WriteString(t)
		}
	}
	return b.String()
}

var mdSpecial = strings.NewReplacer(`\`, `\\`, "*", `\*`, "_", `\_`, "`", "\\`", "[", `\[`, "]", `\]`, "<", `\<`, "~", `\~`, "&", `\&`)

// trimEnd drops trailing spaces, which Markdown drops too.
func trimEnd(s string) string { return strings.TrimRight(s, " \t") }

func escapeMarkdown(s string) string { return mdSpecial.Replace(s) }

var blockStart = regexp.MustCompile(`^(#{1,6}\s|[-+*]\s|\d+[.)]\s|>|---|===|\x60\x60\x60)`)

func mdURL(u string) string {
	if strings.ContainsAny(u, " ()<>") {
		return "<" + strings.NewReplacer("<", "%3C", ">", "%3E").Replace(u) + ">"
	}
	return u
}

// The order marks nest in, outermost first.
type mdMark struct {
	on    func(Marks) bool
	open  func(Marks) string
	close string
}

var mdMarks = []mdMark{
	{func(m Marks) bool { return m.Link != "" }, func(Marks) string { return "[" }, ""},
	{func(m Marks) bool { return m.Color != "" }, func(m Marks) string { return `<span style="color: ` + htmlAttr(m.Color) + `">` }, "</span>"},
	{func(m Marks) bool { return m.Highlight != "" }, func(m Marks) string { return `<mark style="background: ` + htmlAttr(m.Highlight) + `">` }, "</mark>"},
	{func(m Marks) bool { return m.Underline }, func(Marks) string { return "<u>" }, "</u>"},
	{func(m Marks) bool { return m.Strike }, func(Marks) string { return "~~" }, "~~"},
	{func(m Marks) bool { return m.Bold }, func(Marks) string { return "**" }, "**"},
	{func(m Marks) bool { return m.Italic }, func(Marks) string { return "*" }, "*"},
}

func htmlAttr(s string) string {
	return strings.NewReplacer(`"`, "", "<", "", ">", "", ";", "").Replace(s)
}

// inlineMarkdown writes spans, opening and closing marks as few times as
// it can and keeping spaces outside them, as emphasis needs.
func inlineMarkdown(spans []Span, paragraph bool) string {
	var b strings.Builder
	var stack []int // open marks, indexes into mdMarks
	var link string // the open link's URL
	closeTo := func(n int) {
		for len(stack) > n {
			k := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if k == 0 {
				b.WriteString("](" + mdURL(link) + ")")
			} else {
				b.WriteString(mdMarks[k].close)
			}
		}
	}
	// keep returns how many of the open marks m continues.
	keep := func(m Marks) int {
		for i, k := range stack {
			if !mdMarks[k].on(m) || k == 0 && m.Link != link {
				return i
			}
		}
		return len(stack)
	}
	for i, sp := range spans {
		m := sp.Marks
		if sp.Atom != nil {
			m.Link = ""
		}
		var core, lead, trail string
		switch {
		case sp.Atom != nil && sp.Atom.Kind == "emoji":
			core = sp.Atom.Plain()
			if core == "" {
				core = ":" + sp.Atom.ID + ":"
			}
		case sp.Atom != nil:
			label := sp.Atom.Label
			if label == "" {
				label = sp.Atom.Plain()
			}
			core = "[" + escapeMarkdown(label) + "](" + mdURL(sp.Atom.Kind+":"+sp.Atom.ID) + ")"
		case m.Code:
			lead, core, trail = splitSpace(sp.Text)
			fence := "`"
			for strings.Contains(core, fence) {
				fence += "`"
			}
			pad := ""
			if strings.HasPrefix(core, "`") || strings.HasSuffix(core, "`") {
				pad = " "
			}
			if core != "" {
				core = fence + pad + core + pad + fence
			}
		default:
			lead, core, trail = splitSpace(sp.Text)
			core = escapeMarkdown(core)
			core = strings.ReplaceAll(core, "\n", "\\\n")
		}
		if i == 0 && paragraph && blockStart.MatchString(lead+core) {
			core = `\` + core
		}
		closeTo(keep(m))
		b.WriteString(lead)
		if core != "" {
			for k := range mdMarks {
				if !mdMarks[k].on(m) || containsInt(stack, k) {
					continue
				}
				stack = append(stack, k)
				if k == 0 {
					link = m.Link
				}
				b.WriteString(mdMarks[k].open(m))
			}
			b.WriteString(core)
		}
		if trail != "" {
			next := Marks{}
			if i+1 < len(spans) {
				next = spans[i+1].Marks
			}
			closeTo(keep(next))
			b.WriteString(trail)
		}
	}
	closeTo(0)
	return b.String()
}

func containsInt(s []int, v int) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func splitSpace(s string) (lead, core, trail string) {
	t := strings.TrimLeft(s, " \t")
	lead = s[:len(s)-len(t)]
	core = strings.TrimRight(t, " \t")
	trail = t[len(core):]
	return
}

// MarkdownOptions configure ParseMarkdown.
type MarkdownOptions struct {
	// AtomKinds are the link schemes read as atoms: [@Ada](mention:42)
	// becomes a mention of 42. Nil reads "mention" and "ref".
	AtomKinds []string
	// Highlight is the color of a <mark> without one; "#fff59d" when
	// empty.
	Highlight string
}

var mdParser = goldmark.New(goldmark.WithExtensions(extension.GFM))

// ParseMarkdown reads GitHub-flavored Markdown, and the inline HTML
// Markdown writes, into a document.
func ParseMarkdown(src string, opts *MarkdownOptions) *Doc {
	if opts == nil {
		opts = &MarkdownOptions{}
	}
	if opts.AtomKinds == nil {
		opts.AtomKinds = []string{"mention", "ref"}
	}
	if opts.Highlight == "" {
		opts.Highlight = "#fff59d"
	}
	b := []byte(src)
	p := &mdReader{src: b, o: opts}
	root := mdParser.Parser().Parse(text.NewReader(b))
	p.blocks(root, 0, 0)
	if len(p.d.Blocks) == 0 {
		p.d.Blocks = []Block{{}}
	}
	return &p.d
}

type mdReader struct {
	src []byte
	o   *MarkdownOptions
	d   Doc
}

func (r *mdReader) blocks(n gast.Node, listLevel, quoteLevel int) {
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		r.block(c, listLevel, quoteLevel)
	}
}

func (r *mdReader) add(b Block) { r.d.Blocks = append(r.d.Blocks, b) }

func (r *mdReader) block(n gast.Node, listLevel, quoteLevel int) {
	switch n := n.(type) {
	case *gast.Heading:
		r.add(Block{Kind: Heading, Level: n.Level, Spans: r.inline(n)})
	case *gast.Paragraph, *gast.TextBlock:
		// A paragraph of just a picture is an image block.
		if im, ok := n.FirstChild().(*gast.Image); ok && n.FirstChild() == n.LastChild() {
			r.add(Block{Kind: Image, Src: string(im.Destination), Alt: r.plain(im)})
			return
		}
		spans := r.inline(n)
		if quoteLevel > 0 {
			r.add(Block{Kind: Quote, Level: quoteLevel - 1, Spans: spans})
			return
		}
		if len(spans) == 1 && spans[0].Atom == nil && spans[0].Text == " " {
			spans = nil // the &nbsp; of an empty paragraph
		}
		r.add(Block{Spans: spans})
	case *gast.Blockquote:
		r.blocks(n, listLevel, quoteLevel+1)
	case *gast.List:
		for item := n.FirstChild(); item != nil; item = item.NextSibling() {
			kind := Bullet
			if n.IsOrdered() {
				kind = Numbered
			}
			first := true
			for c := item.FirstChild(); c != nil; c = c.NextSibling() {
				switch c.(type) {
				case *gast.Paragraph, *gast.TextBlock:
					if !first {
						r.block(c, listLevel, quoteLevel)
						continue
					}
					first = false
					b := Block{Kind: kind, Level: listLevel}
					if box, ok := c.FirstChild().(*east.TaskCheckBox); ok {
						b.Kind, b.Checked = Task, box.IsChecked
					}
					b.Spans = trimLeadingSpace(r.inline(c))
					r.add(b)
				case *gast.List:
					if first {
						r.add(Block{Kind: kind, Level: listLevel})
						first = false
					}
					r.block(c, listLevel+1, quoteLevel)
				default:
					r.block(c, listLevel, quoteLevel)
				}
			}
			if first {
				r.add(Block{Kind: kind, Level: listLevel})
			}
		}
	case *gast.FencedCodeBlock, *gast.CodeBlock:
		var t strings.Builder
		lines := n.Lines()
		for i := 0; i < lines.Len(); i++ {
			seg := lines.At(i)
			t.Write(seg.Value(r.src))
		}
		b := Block{Kind: Code}
		if f, ok := n.(*gast.FencedCodeBlock); ok {
			b.Lang = string(f.Language(r.src))
		}
		if s := strings.TrimSuffix(t.String(), "\n"); s != "" {
			b.Spans = []Span{{Text: s}}
		}
		r.add(b)
	case *gast.ThematicBreak:
		r.add(Block{Kind: Rule})
	case *east.Table:
		b := Block{Kind: Table}
		for row := n.FirstChild(); row != nil; row = row.NextSibling() {
			var cells [][]Span
			for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
				cells = append(cells, r.inline(cell))
			}
			b.Cells = append(b.Cells, cells)
		}
		r.add(b)
	case *gast.HTMLBlock:
		var t strings.Builder
		lines := n.Lines()
		for i := 0; i < lines.Len(); i++ {
			seg := lines.At(i)
			t.Write(seg.Value(r.src))
		}
		if d := ParseHTML(t.String()); d != nil {
			for _, b := range d.Blocks {
				if b.Kind != Paragraph || b.Len() > 0 {
					r.add(b)
				}
			}
		}
	default:
		r.blocks(n, listLevel, quoteLevel)
	}
}

func trimLeadingSpace(ss []Span) []Span {
	if len(ss) > 0 && ss[0].Atom == nil {
		ss[0].Text = strings.TrimLeft(ss[0].Text, " ")
	}
	return normalize(ss)
}

func (r *mdReader) plain(n gast.Node) string {
	var b strings.Builder
	_ = gast.Walk(n, func(c gast.Node, entering bool) (gast.WalkStatus, error) {
		if !entering {
			return gast.WalkContinue, nil
		}
		switch c := c.(type) {
		case *gast.Text:
			if _, code := c.Parent().(*gast.CodeSpan); code {
				b.Write(c.Segment.Value(r.src))
			} else {
				b.WriteString(mdUnescape(c.Segment.Value(r.src)))
			}
		case *gast.String:
			b.Write(c.Value)
		}
		return gast.WalkContinue, nil
	})
	return b.String()
}

func (r *mdReader) inline(n gast.Node) []Span {
	var out []Span
	r.inlines(n, Marks{}, &out)
	return normalize(out)
}

var styleColor = regexp.MustCompile(`(?i)(?:^|;)\s*(color|background(?:-color)?)\s*:\s*([^;"]+)`)

// inlines appends n's children as spans. Inline HTML tags open and close
// marks across the siblings between them.
func (r *mdReader) inlines(n gast.Node, m Marks, out *[]Span) {
	var opened []func(*Marks) // undo for each open HTML tag
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch c := c.(type) {
		case *gast.Text:
			s := mdUnescape(c.Segment.Value(r.src))
			switch {
			case c.HardLineBreak():
				s += "\n"
			case c.SoftLineBreak():
				s += " "
			}
			*out = append(*out, Span{Text: s, Marks: m})
		case *gast.String:
			*out = append(*out, Span{Text: string(c.Value), Marks: m})
		case *gast.Emphasis:
			sub := m
			if c.Level >= 2 {
				sub.Bold = true
			} else {
				sub.Italic = true
			}
			r.inlines(c, sub, out)
		case *east.Strikethrough:
			sub := m
			sub.Strike = true
			r.inlines(c, sub, out)
		case *gast.CodeSpan:
			sub := m
			sub.Code = true
			*out = append(*out, Span{Text: r.plain(c), Marks: sub})
		case *gast.Link:
			dest := string(c.Destination)
			if kind, id, ok := strings.Cut(dest, ":"); ok && containsStr(r.o.AtomKinds, kind) {
				label := r.plain(c)
				*out = append(*out, Span{Atom: &Atom{Kind: kind, ID: id, Label: label}, Marks: m})
				continue
			}
			sub := m
			sub.Link = SafeURL(dest)
			r.inlines(c, sub, out)
		case *gast.AutoLink:
			url := string(c.URL(r.src))
			if c.AutoLinkType == gast.AutoLinkEmail && !strings.HasPrefix(url, "mailto:") {
				url = "mailto:" + url
			}
			sub := m
			sub.Link = SafeURL(url)
			*out = append(*out, Span{Text: string(c.Label(r.src)), Marks: sub})
		case *gast.Image:
			*out = append(*out, Span{Text: r.plain(c), Marks: m})
		case *east.TaskCheckBox:
		case *gast.RawHTML:
			var raw strings.Builder
			for i := 0; i < c.Segments.Len(); i++ {
				seg := c.Segments.At(i)
				raw.Write(seg.Value(r.src))
			}
			tag := strings.ToLower(raw.String())
			switch {
			case strings.HasPrefix(tag, "<br"):
				*out = append(*out, Span{Text: "\n", Marks: m})
			case strings.HasPrefix(tag, "</"):
				if len(opened) > 0 {
					opened[len(opened)-1](&m)
					opened = opened[:len(opened)-1]
				}
			case strings.HasPrefix(tag, "<u") || strings.HasPrefix(tag, "<ins"):
				prev := m.Underline
				m.Underline = true
				opened = append(opened, func(m *Marks) { m.Underline = prev })
			case strings.HasPrefix(tag, "<mark"):
				prev := m.Highlight
				m.Highlight = r.o.Highlight
				if c := cssColor(tag, "background"); c != "" {
					m.Highlight = c
				}
				opened = append(opened, func(m *Marks) { m.Highlight = prev })
			case strings.HasPrefix(tag, "<span"):
				prevC, prevH := m.Color, m.Highlight
				if c := cssColor(tag, "color"); c != "" {
					m.Color = c
				}
				if c := cssColor(tag, "background"); c != "" {
					m.Highlight = c
				}
				opened = append(opened, func(m *Marks) { m.Color, m.Highlight = prevC, prevH })
			case strings.HasPrefix(tag, "<s>") || strings.HasPrefix(tag, "<del") || strings.HasPrefix(tag, "<strike"):
				prev := m.Strike
				m.Strike = true
				opened = append(opened, func(m *Marks) { m.Strike = prev })
			case strings.HasPrefix(tag, "<b>") || strings.HasPrefix(tag, "<strong"):
				prev := m.Bold
				m.Bold = true
				opened = append(opened, func(m *Marks) { m.Bold = prev })
			case strings.HasPrefix(tag, "<i>") || strings.HasPrefix(tag, "<em"):
				prev := m.Italic
				m.Italic = true
				opened = append(opened, func(m *Marks) { m.Italic = prev })
			default:
				opened = append(opened, func(*Marks) {})
			}
		default:
			r.inlines(c, m, out)
		}
	}
}

// cssColor returns a style attribute's color ("color") or background
// ("background" or "background-color") in an HTML tag.
func cssColor(tag, prop string) string {
	i := strings.Index(tag, "style=")
	if i < 0 {
		return ""
	}
	style := tag[i+6:]
	style = strings.Trim(style, `"'>/ `)
	for _, m := range styleColor.FindAllStringSubmatch(style, -1) {
		name := strings.ToLower(m[1])
		if prop == "color" && name == "color" || prop == "background" && name != "color" {
			return safeColor(strings.TrimSpace(m[2]))
		}
	}
	return ""
}

var colorValue = regexp.MustCompile(`^(#[0-9a-fA-F]{3,8}|[a-zA-Z]{3,20}|rgba?\([0-9., %]+\))$`)

// safeColor returns c when it's a plain CSS color: a hex value, a name or
// rgb().
func safeColor(c string) string {
	if colorValue.MatchString(c) {
		return c
	}
	return ""
}

// SafeURL returns u when it's a link that's safe to open (http, https,
// mailto, tel, or relative), and "" otherwise: no javascript: or data:.
func SafeURL(u string) string {
	t := strings.TrimSpace(u)
	scheme, _, ok := strings.Cut(t, ":")
	if !ok || strings.ContainsAny(scheme, "/?#") {
		return t // relative
	}
	switch strings.ToLower(scheme) {
	case "http", "https", "mailto", "tel":
		return t
	}
	return ""
}

func containsStr(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
