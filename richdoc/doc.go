// Package richdoc is the document model of Giode's rich text editor:
// blocks (paragraphs, headings, lists, tasks, quotes, code, tables,
// images, rules) holding text spans with inline marks and atoms
// (mentions, references, emoji). Edits are operations that can be
// inverted, for undo, and streamed, for syncing with CRDT or OT.
// Documents convert to and from Markdown, JSON and pasted HTML.
//
// It has no UI and can be used on a server.
package richdoc

import (
	"strings"
	"unicode/utf8"
)

// Marks are the inline formatting of text.
type Marks struct {
	Bold      bool   `json:"bold,omitempty"`
	Italic    bool   `json:"italic,omitempty"`
	Underline bool   `json:"underline,omitempty"`
	Strike    bool   `json:"strike,omitempty"`
	Code      bool   `json:"code,omitempty"`
	Color     string `json:"color,omitempty"`
	Highlight string `json:"highlight,omitempty"`
	Link      string `json:"link,omitempty"`
}

// Atom is an inline token edited as a whole: a mention, a reference to
// a ticket or a deal, an emoji.
type Atom struct {
	// Kind is "mention", "ref", "emoji" or the app's own.
	Kind string `json:"kind"`
	// ID is what it refers to: a user's or a ticket's ID, an emoji's
	// shortcode.
	ID string `json:"id,omitempty"`
	// Label is shown: "@Ada Lovelace", "#4521".
	Label string `json:"label,omitempty"`
	// Text is its plain text, for copying and search; Label when empty.
	Text string `json:"text,omitempty"`
}

// Plain returns the atom's plain text.
func (a *Atom) Plain() string {
	if a.Text != "" {
		return a.Text
	}
	return a.Label
}

// Span is a run of text with the same marks, or one atom.
type Span struct {
	Text  string `json:"text,omitempty"`
	Marks Marks  `json:"marks"`
	Atom  *Atom  `json:"atom,omitempty"`
}

// Len is the span's length in positions: its runes, or 1 for an atom.
func (s Span) Len() int {
	if s.Atom != nil {
		return 1
	}
	return utf8.RuneCountInString(s.Text)
}

// Kind is a block's type.
type Kind uint8

const (
	Paragraph Kind = iota
	Heading
	Bullet   // a bulleted list item
	Numbered // a numbered list item
	Task     // a checklist item
	Quote
	Code // a code block
	Table
	Image
	Rule // a horizontal rule
)

var kindNames = [...]string{"paragraph", "heading", "bullet", "numbered", "task", "quote", "code", "table", "image", "rule"}

func (k Kind) String() string {
	if int(k) < len(kindNames) {
		return kindNames[k]
	}
	return "paragraph"
}

// ParseKind returns the kind named s, Paragraph for unknown names.
func ParseKind(s string) Kind {
	for i, n := range kindNames {
		if n == s {
			return Kind(i)
		}
	}
	return Paragraph
}

// HasText reports whether blocks of the kind hold spans: all but tables,
// images and rules.
func (k Kind) HasText() bool { return k != Table && k != Image && k != Rule }

// IsList reports whether the kind is a list item.
func (k Kind) IsList() bool { return k == Bullet || k == Numbered || k == Task }

// Block is one block of a document.
type Block struct {
	Kind Kind
	// Level is a heading's level, 1 to 6, or a list item's or quote's
	// nesting depth from 0.
	Level int
	// Checked is a task's state.
	Checked bool
	// Lang is a code block's language.
	Lang string
	// Spans are a text block's content.
	Spans []Span
	// Cells are a table's rows of cells; the first row is its header.
	Cells [][][]Span
	// Src and Alt are an image's URL and description.
	Src, Alt string
}

// Len is the block's length in positions: its spans' lengths, 0 for
// tables, images and rules.
func (b *Block) Len() int { return spansLen(b.Spans) }

// Text is the block's plain text, atoms as their plain text.
func (b *Block) Text() string { return spansText(b.Spans) }

// Attrs returns the block without its content.
func (b Block) Attrs() Block {
	b.Spans, b.Cells = nil, nil
	return b
}

// Clone returns a deep copy.
func (b Block) Clone() Block {
	b.Spans = cloneSpans(b.Spans)
	if b.Cells != nil {
		rows := make([][][]Span, len(b.Cells))
		for i, r := range b.Cells {
			rows[i] = make([][]Span, len(r))
			for j, c := range r {
				rows[i][j] = cloneSpans(c)
			}
		}
		b.Cells = rows
	}
	return b
}

// Doc is a document: its blocks, never none.
type Doc struct {
	Blocks []Block
}

// New returns a document of one empty paragraph.
func New() *Doc { return &Doc{Blocks: []Block{{}}} }

// FromText returns a document of plain paragraphs, one per line.
func FromText(s string) *Doc {
	d := &Doc{}
	for _, line := range strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n") {
		b := Block{}
		if line != "" {
			b.Spans = []Span{{Text: line}}
		}
		d.Blocks = append(d.Blocks, b)
	}
	return d
}

// Clone returns a deep copy.
func (d *Doc) Clone() *Doc {
	c := &Doc{Blocks: make([]Block, len(d.Blocks))}
	for i, b := range d.Blocks {
		c.Blocks[i] = b.Clone()
	}
	return c
}

// Text is the document's plain text, blocks on their own lines and
// table cells separated by tabs.
func (d *Doc) Text() string {
	var b strings.Builder
	for i := range d.Blocks {
		if i > 0 {
			b.WriteByte('\n')
		}
		blk := &d.Blocks[i]
		switch blk.Kind {
		case Table:
			for r, row := range blk.Cells {
				if r > 0 {
					b.WriteByte('\n')
				}
				for c, cell := range row {
					if c > 0 {
						b.WriteByte('\t')
					}
					b.WriteString(spansText(cell))
				}
			}
		case Image:
			b.WriteString(blk.Alt)
		default:
			b.WriteString(blk.Text())
		}
	}
	return b.String()
}

// Len is the number of characters in the document's text blocks.
func (d *Doc) Len() int {
	n := 0
	for i := range d.Blocks {
		n += d.Blocks[i].Len()
	}
	return n
}

func spansLen(ss []Span) int {
	n := 0
	for _, s := range ss {
		n += s.Len()
	}
	return n
}

func spansText(ss []Span) string {
	var b strings.Builder
	for _, s := range ss {
		if s.Atom != nil {
			b.WriteString(s.Atom.Plain())
		} else {
			b.WriteString(s.Text)
		}
	}
	return b.String()
}

func cloneSpans(ss []Span) []Span {
	if ss == nil {
		return nil
	}
	out := make([]Span, len(ss))
	for i, s := range ss {
		if s.Atom != nil {
			a := *s.Atom
			s.Atom = &a
		}
		out[i] = s
	}
	return out
}

// runeOffset returns the byte offset of the n-th rune of s.
func runeOffset(s string, n int) int {
	if n <= 0 {
		return 0
	}
	i := 0
	for j := range s {
		if i == n {
			return j
		}
		i++
	}
	return len(s)
}

// sliceSpans returns the spans covering positions [from, to).
func sliceSpans(ss []Span, from, to int) []Span {
	var out []Span
	pos := 0
	for _, s := range ss {
		l := s.Len()
		a, b := max(from, pos), min(to, pos+l)
		if a < b {
			if s.Atom != nil {
				out = append(out, s)
			} else {
				t := s.Text[runeOffset(s.Text, a-pos):runeOffset(s.Text, b-pos)]
				out = append(out, Span{Text: t, Marks: s.Marks})
			}
		}
		pos += l
		if pos >= to {
			break
		}
	}
	return cloneSpans(out)
}

// replaceSpans replaces positions [from, to) of ss with ins.
func replaceSpans(ss []Span, from, to int, ins []Span) []Span {
	n := spansLen(ss)
	out := sliceSpans(ss, 0, from)
	out = append(out, cloneSpans(ins)...)
	out = append(out, sliceSpans(ss, to, n)...)
	return normalize(out)
}

// normalize merges neighbouring spans with the same marks and drops
// empty ones.
func normalize(ss []Span) []Span {
	var out []Span
	for _, s := range ss {
		if s.Atom == nil && s.Text == "" {
			continue
		}
		if k := len(out) - 1; k >= 0 && s.Atom == nil && out[k].Atom == nil && out[k].Marks == s.Marks {
			out[k].Text += s.Text
			continue
		}
		out = append(out, s)
	}
	return out
}

// MarksAt returns the marks of the position before off in ss (the one
// after at the start), which typed text takes on.
func MarksAt(ss []Span, off int) Marks {
	pos := 0
	var last Marks
	for i, s := range ss {
		l := s.Len()
		if off <= pos+l && (off > pos || i == 0) {
			if s.Atom != nil {
				m := s.Marks
				m.Link = ""
				return m
			}
			return s.Marks
		}
		pos += l
		last = s.Marks
	}
	return last
}
