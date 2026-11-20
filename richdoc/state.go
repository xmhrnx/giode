package richdoc

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Selection is a range of a document from Anchor, where it started, to
// Head, where the caret is. It's a caret when they're equal.
type Selection struct {
	Anchor Pos `json:"anchor"`
	Head   Pos `json:"head"`
}

// Caret returns a selection of just p.
func Caret(p Pos) Selection { return Selection{p, p} }

// Empty reports whether the selection is a caret.
func (s Selection) Empty() bool { return s.Anchor == s.Head }

// Start and End are the selection's first and last positions.
func (s Selection) Start() Pos {
	if s.Head.Less(s.Anchor) {
		return s.Head
	}
	return s.Anchor
}

func (s Selection) End() Pos {
	if s.Head.Less(s.Anchor) {
		return s.Anchor
	}
	return s.Head
}

// Tx is a transaction: the operations one command applied, and the
// selection before and after.
type Tx struct {
	Ops    []Op      `json:"ops"`
	Before Selection `json:"before"`
	After  Selection `json:"after"`
	// Label groups transactions in history: "type" and "delete" ones in
	// a row undo together.
	Label string `json:"label,omitempty"`

	// rule is the index of the first operation an input rule added, and
	// mid the selection before it: history undoes the rule first.
	rule int
	mid  Selection
}

// State is a document being edited: the document, the selection, and the
// marks the next typed text takes.
type State struct {
	Doc *Doc
	Sel Selection
	// Stored are marks set at a caret (Bold with nothing selected) for
	// the text typed next; nil takes the marks around the caret.
	Stored *Marks
	// Rules turns Markdown typed at the start of a paragraph and around
	// text into formatting: "- ", "1. ", "> ", "# ", "```", **bold**.
	Rules bool

	tx *Tx
}

// NewState returns a state editing d (a new document when nil), with the
// caret at its start and input rules on.
func NewState(d *Doc) *State {
	if d == nil || len(d.Blocks) == 0 {
		d = New()
	}
	return &State{Doc: d, Rules: true}
}

// BlockLen is a block's length in positions: its text's, or 1 for a
// table, image or rule, which a caret is before (0) or after (1).
func (d *Doc) BlockLen(i int) int {
	if i < 0 || i >= len(d.Blocks) {
		return 0
	}
	if b := &d.Blocks[i]; b.Kind.HasText() {
		return b.Len()
	}
	return 1
}

// Clamp returns p moved into the document.
func (d *Doc) Clamp(p Pos) Pos {
	p.Block = max(0, min(p.Block, len(d.Blocks)-1))
	p.Offset = max(0, min(p.Offset, d.BlockLen(p.Block)))
	return p
}

// End is the position at the end of the document.
func (d *Doc) End() Pos {
	n := len(d.Blocks) - 1
	return Pos{n, d.BlockLen(n)}
}

func (s *State) begin(label string) {
	s.tx = &Tx{Before: s.Sel, Label: label, rule: -1}
}

func (s *State) apply(op Op) {
	if err := s.Doc.Apply(op); err != nil {
		panic(err) // commands only build operations that fit
	}
	s.tx.Ops = append(s.tx.Ops, op)
}

func (s *State) commit() *Tx {
	tx := s.tx
	s.tx = nil
	s.Sel = Selection{s.Doc.Clamp(s.Sel.Anchor), s.Doc.Clamp(s.Sel.Head)}
	tx.After = s.Sel
	if len(tx.Ops) == 0 && tx.Before == tx.After {
		return nil
	}
	return tx
}

// Apply applies a transaction made elsewhere (by a collaborator, or
// History), mapping the selection through it.
func (s *State) Apply(ops []Op) error {
	for _, op := range ops {
		if err := s.Doc.Apply(op); err != nil {
			return err
		}
		s.Sel = Selection{op.Map(s.Sel.Anchor, false), op.Map(s.Sel.Head, true)}
	}
	s.Sel = Selection{s.Doc.Clamp(s.Sel.Anchor), s.Doc.Clamp(s.Sel.Head)}
	return nil
}

// Select sets the selection, returning a transaction for history only
// when it changed.
func (s *State) Select(sel Selection) {
	s.Sel = Selection{s.Doc.Clamp(sel.Anchor), s.Doc.Clamp(sel.Head)}
	s.Stored = nil
}

// replace replaces [from, to) of block i with ins.
func (s *State) replace(i, from, to int, ins []Span) {
	b := &s.Doc.Blocks[i]
	if from == to && len(ins) == 0 {
		return
	}
	s.apply(Op{Kind: Replace, Block: i, From: from, To: to, Old: sliceSpans(b.Spans, from, to), New: cloneSpans(ins)})
}

func (s *State) setBlock(i int, nb Block) {
	old := s.Doc.Blocks[i].Clone()
	nb = nb.Clone()
	s.apply(Op{Kind: SetBlock, Block: i, OldBlock: &old, NewBlock: &nb})
}

// deleteRange deletes [from, to) and returns where the caret goes.
func (s *State) deleteRange(from, to Pos) Pos {
	d := s.Doc
	if !from.Less(to) {
		return from
	}
	if from.Block == to.Block {
		if d.Blocks[from.Block].Kind.HasText() {
			s.replace(from.Block, from.Offset, to.Offset, nil)
			return from
		}
		return s.removeBlocks(from.Block, 1)
	}
	F, L := from.Block, to.Block
	keepLast, keepFirst := true, true
	// The last block: its start, or all of a table or image.
	if d.Blocks[L].Kind.HasText() {
		s.replace(L, 0, to.Offset, nil)
	} else if to.Offset > 0 {
		keepLast = false
	}
	if d.Blocks[F].Kind.HasText() {
		s.replace(F, from.Offset, d.Blocks[F].Len(), nil)
	} else if from.Offset == 0 {
		keepFirst = false
	}
	// Remove the middle, and the ends that went entirely.
	start, end := F+1, L // [start, end) removed
	if !keepFirst {
		start = F
	}
	if !keepLast {
		end = L + 1
	}
	if end > start {
		if end-start >= len(d.Blocks) {
			s.apply(Op{Kind: InsertBlocks, Block: end, Blocks: []Block{{}}})
		}
		blocks := make([]Block, end-start)
		for i := range blocks {
			blocks[i] = d.Blocks[start+i].Clone()
		}
		s.apply(Op{Kind: RemoveBlocks, Block: start, Blocks: blocks})
	}
	if !keepFirst {
		return Pos{Block: start, Offset: 0}
	}
	// Join the last block's rest onto the first.
	if keepLast && F+1 < len(d.Blocks) && d.Blocks[F].Kind.HasText() && d.Blocks[F+1].Kind.HasText() {
		attrs := d.Blocks[F+1].Attrs()
		s.apply(Op{Kind: Join, Block: F, From: d.Blocks[F].Len(), Attrs: &attrs})
	}
	return from
}

// removeBlocks removes n blocks at i, leaving an empty paragraph when
// they were all, and returns where the caret goes.
func (s *State) removeBlocks(i, n int) Pos {
	d := s.Doc
	if n >= len(d.Blocks) {
		s.apply(Op{Kind: InsertBlocks, Block: i + n, Blocks: []Block{{}}})
	}
	blocks := make([]Block, n)
	for k := range blocks {
		blocks[k] = d.Blocks[i+k].Clone()
	}
	s.apply(Op{Kind: RemoveBlocks, Block: i, Blocks: blocks})
	if i >= len(d.Blocks) {
		return d.End()
	}
	return Pos{Block: i, Offset: 0}
}

// textCaret makes sure the caret is in a text block, adding a paragraph
// next to a table, image or rule it's on.
func (s *State) textCaret(p Pos) Pos {
	if s.Doc.Blocks[p.Block].Kind.HasText() {
		return p
	}
	at := p.Block + p.Offset // before (0) or after (1) it
	s.apply(Op{Kind: InsertBlocks, Block: at, Blocks: []Block{{}}})
	return Pos{Block: at, Offset: 0}
}

// typingMarks are the marks text typed at p takes.
func (s *State) typingMarks(p Pos) Marks {
	if s.Stored != nil {
		return *s.Stored
	}
	b := &s.Doc.Blocks[p.Block]
	if b.Kind == Code {
		return Marks{}
	}
	m := MarksAt(b.Spans, p.Offset)
	// Links and code don't extend past their end.
	if m.Link != "" || m.Code {
		after := MarksAt(b.Spans, p.Offset+1)
		if p.Offset >= b.Len() || after.Link != m.Link {
			m.Link = ""
		}
		if p.Offset >= b.Len() || !after.Code {
			m.Code = false
		}
	}
	return m
}

// InsertText replaces the selection with text, typed or pasted; new
// lines split blocks, except in code blocks.
func (s *State) InsertText(text string) *Tx {
	s.begin("type")
	if strings.ContainsAny(text, "\n\r") {
		s.tx.Label = "paste"
	}
	p := s.deleteRange(s.Sel.Start(), s.Sel.End())
	p = s.textCaret(p)
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	lines := []string{text}
	if s.Doc.Blocks[p.Block].Kind != Code {
		lines = strings.Split(text, "\n")
	}
	for i, line := range lines {
		if i > 0 {
			p = s.split(p)
		}
		if line != "" {
			m := s.typingMarks(p)
			s.replace(p.Block, p.Offset, p.Offset, []Span{{Text: line, Marks: m}})
			p.Offset += utf8.RuneCountInString(line)
		}
	}
	s.Sel = Caret(p)
	if s.Rules && len(lines) == 1 && utf8.RuneCountInString(text) == 1 {
		s.inputRules()
	}
	return s.commit()
}

// split splits the block at p, the new block continuing lists and
// quotes, and returns the start of the new block.
func (s *State) split(p Pos) Pos {
	b := &s.Doc.Blocks[p.Block]
	attrs := b.Attrs()
	switch b.Kind {
	case Heading:
		if p.Offset >= b.Len() {
			attrs = Block{}
		}
	case Task:
		attrs.Checked = false
	}
	s.apply(Op{Kind: Split, Block: p.Block, From: p.Offset, Attrs: &attrs})
	return Pos{Block: p.Block + 1}
}

// Enter splits the block at the caret: Enter in a list continues it, and
// on an empty item ends it; in a code block it adds a line, and on a
// second empty line at the end leaves it.
func (s *State) Enter() *Tx {
	s.begin("enter")
	p := s.deleteRange(s.Sel.Start(), s.Sel.End())
	p = s.textCaret(p)
	b := &s.Doc.Blocks[p.Block]
	if t := b.Text(); s.Rules && b.Kind == Paragraph && strings.HasPrefix(t, "```") && p.Offset == b.Len() &&
		!strings.ContainsAny(t[3:], " `") {
		s.startRule()
		s.setBlock(p.Block, Block{Kind: Code, Lang: t[3:]})
		s.Sel = Caret(Pos{Block: p.Block})
		return s.commit()
	}
	switch {
	case b.Kind == Code:
		t := b.Text()
		if p.Offset == b.Len() && strings.HasSuffix(t, "\n") {
			// Leave the block: drop the empty line, and a paragraph follows.
			s.replace(p.Block, p.Offset-1, p.Offset, nil)
			s.apply(Op{Kind: Split, Block: p.Block, From: p.Offset - 1, Attrs: &Block{}})
			p = Pos{Block: p.Block + 1}
		} else {
			s.replace(p.Block, p.Offset, p.Offset, []Span{{Text: "\n"}})
			p.Offset++
		}
	case (b.Kind.IsList() || b.Kind == Quote) && b.Len() == 0:
		nb := b.Attrs()
		if b.Level > 0 {
			nb.Level--
		} else {
			nb = Block{}
		}
		s.setBlock(p.Block, nb)
	default:
		p = s.split(p)
	}
	s.Sel = Caret(p)
	return s.commit()
}

// prevStop returns the start of the character before off in text: a
// grapheme cluster, approximately (combining marks, emoji modifiers and
// joined emoji stay together).
func prevStop(text []rune, off int) int {
	if off <= 0 {
		return 0
	}
	i := off - 1
	for i > 0 {
		r := text[i]
		prev := text[i-1]
		switch {
		case isExtend(r), prev == '‍':
			i--
			continue
		case isRegional(r) && isRegional(prev):
			// Flags are pairs; count the run.
			n := 0
			for j := i; j >= 0 && isRegional(text[j]); j-- {
				n++
			}
			if n%2 == 0 {
				i--
			}
		}
		break
	}
	return i
}

func nextStop(text []rune, off int) int {
	if off >= len(text) {
		return len(text)
	}
	i := off + 1
	if isRegional(text[off]) && i < len(text) && isRegional(text[i]) {
		i++
	}
	for i < len(text) && (isExtend(text[i]) || text[i-1] == '‍') {
		i++
	}
	return i
}

func isExtend(r rune) bool {
	return unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) || r == '‍' ||
		r >= 0xfe00 && r <= 0xfe0f || r >= 0x1f3fb && r <= 0x1f3ff || r >= 0xe0020 && r <= 0xe007f
}

func isRegional(r rune) bool { return r >= 0x1f1e6 && r <= 0x1f1ff }

// blockRunes returns a text block's content as runes, atoms as U+FFFC.
func blockRunes(b *Block) []rune {
	var out []rune
	for _, sp := range b.Spans {
		if sp.Atom != nil {
			out = append(out, '￼')
		} else {
			out = append(out, []rune(sp.Text)...)
		}
	}
	return out
}

// wordStart returns the start of the word before off; wordEnd the end of
// the one after.
func wordStart(text []rune, off int) int {
	i := off
	for i > 0 && unicode.IsSpace(text[i-1]) {
		i--
	}
	if i > 0 && !isWord(text[i-1]) {
		for i > 0 && !isWord(text[i-1]) && !unicode.IsSpace(text[i-1]) {
			i--
		}
		return i
	}
	for i > 0 && isWord(text[i-1]) {
		i--
	}
	return i
}

func wordEnd(text []rune, off int) int {
	i := off
	for i < len(text) && unicode.IsSpace(text[i]) {
		i++
	}
	if i < len(text) && !isWord(text[i]) {
		for i < len(text) && !isWord(text[i]) && !unicode.IsSpace(text[i]) {
			i++
		}
		return i
	}
	for i < len(text) && isWord(text[i]) {
		i++
	}
	return i
}

func isWord(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.Is(unicode.Mn, r) || r == '_' || r == '\''
}

// Unit is how far a delete or move reaches.
type Unit uint8

const (
	Char Unit = iota
	Word
	LineEdge // to the start or end of the block
)

// Delete deletes the selection, or at a caret what's before it (back) or
// after it, by unit. At the start of a block, a list item, quote or
// heading becomes a paragraph first; then the block joins the one
// before.
func (s *State) Delete(back bool, unit Unit) *Tx {
	s.begin("delete")
	if !s.Sel.Empty() {
		s.Sel = Caret(s.deleteRange(s.Sel.Start(), s.Sel.End()))
		s.tx.Label = "cut"
		return s.commit()
	}
	d := s.Doc
	p := s.Sel.Head
	b := &d.Blocks[p.Block]
	if !b.Kind.HasText() {
		s.Sel = Caret(s.removeBlocks(p.Block, 1))
		return s.commit()
	}
	text := blockRunes(b)
	if back {
		if p.Offset == 0 {
			switch {
			case b.Kind.IsList() && b.Level > 0:
				nb := b.Attrs()
				nb.Level--
				nb.Spans = b.Spans
				s.setBlock(p.Block, nb)
			case b.Kind != Paragraph && b.Kind != Code || b.Kind == Code && b.Len() == 0:
				nb := Block{Spans: b.Spans}
				s.setBlock(p.Block, nb)
			case p.Block > 0 && !d.Blocks[p.Block-1].Kind.HasText():
				s.removeBlocks(p.Block-1, 1)
				p.Block--
			case p.Block > 0:
				prev := p.Block - 1
				if d.Blocks[prev].Len() == 0 && d.Blocks[prev].Kind == Paragraph {
					// An empty paragraph before goes, keeping this block's kind.
					s.removeBlocks(prev, 1)
					p = Pos{Block: prev}
				} else {
					n := d.Blocks[prev].Len()
					attrs := b.Attrs()
					s.apply(Op{Kind: Join, Block: prev, From: n, Attrs: &attrs})
					p = Pos{Block: prev, Offset: n}
				}
			}
			s.Sel = Caret(p)
			return s.commit()
		}
		from := prevStop(text, p.Offset)
		switch unit {
		case Word:
			from = wordStart(text, p.Offset)
		case LineEdge:
			from = 0
		}
		s.replace(p.Block, from, p.Offset, nil)
		s.Sel = Caret(Pos{p.Block, from})
		return s.commit()
	}
	if p.Offset >= len(text) {
		if p.Block+1 < len(d.Blocks) {
			if !d.Blocks[p.Block+1].Kind.HasText() {
				s.removeBlocks(p.Block+1, 1)
			} else {
				attrs := d.Blocks[p.Block+1].Attrs()
				s.apply(Op{Kind: Join, Block: p.Block, From: p.Offset, Attrs: &attrs})
			}
		}
		s.Sel = Caret(p)
		return s.commit()
	}
	to := nextStop(text, p.Offset)
	switch unit {
	case Word:
		to = wordEnd(text, p.Offset)
	case LineEdge:
		to = len(text)
	}
	s.replace(p.Block, p.Offset, to, nil)
	return s.commit()
}

// MarkKind names an on-off mark.
type MarkKind uint8

const (
	Bold MarkKind = iota
	Italic
	Underline
	Strike
	InlineCode
)

func (k MarkKind) get(m *Marks) *bool {
	switch k {
	case Italic:
		return &m.Italic
	case Underline:
		return &m.Underline
	case Strike:
		return &m.Strike
	case InlineCode:
		return &m.Code
	}
	return &m.Bold
}

// eachRange calls fn with each text block's part of the selection.
func (s *State) eachRange(fn func(i, from, to int)) {
	start, end := s.Sel.Start(), s.Sel.End()
	for i := start.Block; i <= end.Block; i++ {
		b := &s.Doc.Blocks[i]
		if !b.Kind.HasText() {
			continue
		}
		from, to := 0, b.Len()
		if i == start.Block {
			from = start.Offset
		}
		if i == end.Block {
			to = end.Offset
		}
		if from < to {
			fn(i, from, to)
		}
	}
}

// HasMark reports whether all the selected text (or the text typed at the
// caret) has the mark.
func (s *State) HasMark(k MarkKind) bool {
	if s.Sel.Empty() {
		m := s.typingMarks(s.Sel.Head)
		return *k.get(&m)
	}
	all, any := true, false
	s.eachRange(func(i, from, to int) {
		for _, sp := range sliceSpans(s.Doc.Blocks[i].Spans, from, to) {
			any = true
			if !*k.get(&sp.Marks) {
				all = false
			}
		}
	})
	return any && all
}

// Marks returns the marks at the caret or the selection's start.
func (s *State) Marks() Marks {
	p := s.Sel.Start()
	if !s.Doc.Blocks[p.Block].Kind.HasText() {
		return Marks{}
	}
	if s.Sel.Empty() {
		return s.typingMarks(p)
	}
	return MarksAt(s.Doc.Blocks[p.Block].Spans, p.Offset+1)
}

// ToggleMark turns a mark on for the selection, or off when all of it
// has it; at a caret, for the text typed next.
func (s *State) ToggleMark(k MarkKind) *Tx {
	on := !s.HasMark(k)
	return s.formatMarks(func(m *Marks) { *k.get(m) = on })
}

// SetLink links the selection to url, or unlinks it when url is empty.
func (s *State) SetLink(url string) *Tx {
	return s.formatMarks(func(m *Marks) { m.Link = url })
}

// SetColor sets the selection's text color, or highlight; "" clears it.
func (s *State) SetColor(color string, highlight bool) *Tx {
	return s.formatMarks(func(m *Marks) {
		if highlight {
			m.Highlight = color
		} else {
			m.Color = color
		}
	})
}

// ClearMarks removes all marks from the selection.
func (s *State) ClearMarks() *Tx {
	return s.formatMarks(func(m *Marks) { *m = Marks{} })
}

func (s *State) formatMarks(change func(*Marks)) *Tx {
	s.begin("format")
	if s.Sel.Empty() {
		m := s.typingMarks(s.Sel.Head)
		change(&m)
		s.Stored = &m
		return s.commit()
	}
	s.eachRange(func(i, from, to int) {
		spans := sliceSpans(s.Doc.Blocks[i].Spans, from, to)
		for k := range spans {
			change(&spans[k].Marks)
		}
		s.replace(i, from, to, normalize(spans))
	})
	return s.commit()
}

// selectedBlocks returns the indexes of the blocks the selection touches.
func (s *State) selectedBlocks() (int, int) {
	return s.Sel.Start().Block, s.Sel.End().Block
}

// SetKind makes the selected text blocks kind (with level for headings),
// or paragraphs when they all already are.
func (s *State) SetKind(kind Kind, level int) *Tx {
	s.begin("block")
	a, z := s.selectedBlocks()
	all := true
	for i := a; i <= z; i++ {
		b := &s.Doc.Blocks[i]
		if b.Kind.HasText() && (b.Kind != kind || kind == Heading && b.Level != level) {
			all = false
		}
	}
	for i := a; i <= z; i++ {
		b := &s.Doc.Blocks[i]
		if !b.Kind.HasText() {
			continue
		}
		nb := Block{Kind: kind, Spans: b.Spans}
		if kind == Heading {
			nb.Level = max(1, min(6, level))
		}
		switch {
		case all:
			nb = Block{Spans: b.Spans}
		case kind.IsList() && b.Kind.IsList(), kind == Quote && b.Kind == Quote:
			nb.Level = b.Level
		case kind == Code:
			nb.Lang = b.Lang
		}
		if kind == Code && !all {
			// Code keeps no marks.
			nb.Spans = []Span{{Text: b.Text()}}
			if b.Len() == 0 {
				nb.Spans = nil
			}
		}
		s.setBlock(i, nb)
	}
	return s.commit()
}

// Indent nests the selected list items and quotes one level deeper, or
// shallower (out).
func (s *State) Indent(out bool) *Tx {
	s.begin("block")
	a, z := s.selectedBlocks()
	for i := a; i <= z; i++ {
		b := &s.Doc.Blocks[i]
		if !b.Kind.IsList() && b.Kind != Quote {
			continue
		}
		nb := b.Clone()
		if out {
			nb.Level = max(0, nb.Level-1)
		} else {
			nb.Level = min(8, nb.Level+1)
		}
		if nb.Level != b.Level {
			s.setBlock(i, nb)
		}
	}
	return s.commit()
}

// ToggleTask checks or unchecks the task at the caret, or block i when
// i >= 0.
func (s *State) ToggleTask(i int) *Tx {
	s.begin("block")
	if i < 0 {
		i = s.Sel.Head.Block
	}
	if i < len(s.Doc.Blocks) && s.Doc.Blocks[i].Kind == Task {
		nb := s.Doc.Blocks[i].Clone()
		nb.Checked = !nb.Checked
		s.setBlock(i, nb)
	}
	return s.commit()
}

// SetLang sets the language of the code block at the caret.
func (s *State) SetLang(lang string) *Tx {
	s.begin("block")
	i := s.Sel.Head.Block
	if b := &s.Doc.Blocks[i]; b.Kind == Code && b.Lang != lang {
		nb := b.Clone()
		nb.Lang = lang
		s.setBlock(i, nb)
	}
	return s.commit()
}

// InsertAtom replaces the selection, or the range [from, caret) when from
// is before the caret in its block (the "@ada" typed to find a mention),
// with an atom and then suffix, usually a space.
func (s *State) InsertAtom(a Atom, from *Pos, suffix string) *Tx {
	s.begin("atom")
	start, end := s.Sel.Start(), s.Sel.End()
	if from != nil && from.Block == end.Block && from.Offset <= end.Offset {
		start = *from
	}
	p := s.textCaret(s.deleteRange(start, end))
	m := s.typingMarks(p)
	m.Link = ""
	ins := []Span{{Atom: &a, Marks: m}}
	if suffix != "" {
		ins = append(ins, Span{Text: suffix, Marks: m})
	}
	s.replace(p.Block, p.Offset, p.Offset, ins)
	p.Offset += spansLen(ins)
	s.Sel = Caret(p)
	return s.commit()
}

// InsertBlock puts a table, image, rule or other block after the caret's
// block (in place of it when that's an empty paragraph), with a
// paragraph after it for the caret when the document ends there.
func (s *State) InsertBlock(b Block) *Tx {
	s.begin("insert")
	p := s.deleteRange(s.Sel.Start(), s.Sel.End())
	cur := &s.Doc.Blocks[p.Block]
	at := p.Block + 1
	switch {
	case cur.Kind == Paragraph && cur.Len() == 0:
		at = p.Block
	case cur.Kind.HasText() && p.Offset < cur.Len():
		s.split(p)
	}
	s.apply(Op{Kind: InsertBlocks, Block: at, Blocks: []Block{b}})
	if at == p.Block {
		// Replace the empty paragraph.
		s.apply(Op{Kind: RemoveBlocks, Block: at + 1, Blocks: []Block{s.Doc.Blocks[at+1].Clone()}})
	}
	next := at + 1
	if next >= len(s.Doc.Blocks) || !s.Doc.Blocks[next].Kind.HasText() {
		s.apply(Op{Kind: InsertBlocks, Block: next, Blocks: []Block{{}}})
	}
	if b.Kind.HasText() {
		s.Sel = Caret(Pos{Block: at, Offset: b.Len()})
	} else {
		s.Sel = Caret(Pos{Block: next})
	}
	return s.commit()
}

// InsertDoc replaces the selection with a document fragment, pasted or
// dropped: one text block's spans go inline; more blocks split the
// caret's block around them.
func (s *State) InsertDoc(frag *Doc) *Tx {
	if frag == nil || len(frag.Blocks) == 0 {
		return nil
	}
	s.begin("paste")
	p := s.textCaret(s.deleteRange(s.Sel.Start(), s.Sel.End()))
	blocks := frag.Clone().Blocks
	first, last := blocks[0], blocks[len(blocks)-1]
	if len(blocks) == 1 && first.Kind.HasText() {
		if s.Doc.Blocks[p.Block].Kind == Code {
			first.Spans = []Span{{Text: first.Text()}}
		}
		s.replace(p.Block, p.Offset, p.Offset, first.Spans)
		p.Offset += first.Len()
		s.Sel = Caret(p)
		return s.commit()
	}
	cur := &s.Doc.Blocks[p.Block]
	attrs := cur.Attrs()
	// Split around the insertion, then fill the halves.
	s.apply(Op{Kind: Split, Block: p.Block, From: p.Offset, Attrs: &attrs})
	middle := blocks
	if first.Kind.HasText() {
		if cur := &s.Doc.Blocks[p.Block]; cur.Len() == 0 && cur.Kind == Paragraph {
			s.setBlock(p.Block, first)
		} else {
			s.replace(p.Block, p.Offset, p.Offset, first.Spans)
		}
		middle = middle[1:]
	}
	tail := Pos{Block: p.Block + 1}
	var lastSpans []Span
	if len(middle) > 0 && last.Kind.HasText() {
		lastSpans = last.Spans
		middle = middle[:len(middle)-1]
	}
	if len(middle) > 0 {
		s.apply(Op{Kind: InsertBlocks, Block: tail.Block, Blocks: middle})
		tail.Block += len(middle)
	}
	if lastSpans != nil || last.Kind.HasText() && len(blocks) > 1 {
		if t := &s.Doc.Blocks[tail.Block]; t.Len() == 0 && t.Kind == Paragraph {
			s.setBlock(tail.Block, last)
		} else {
			s.replace(tail.Block, 0, 0, lastSpans)
		}
		tail.Offset = spansLen(lastSpans)
	}
	s.Sel = Caret(tail)
	return s.commit()
}

// Fragment returns the selected part of the document, for copying.
func (s *State) Fragment() *Doc {
	start, end := s.Sel.Start(), s.Sel.End()
	d := &Doc{}
	for i := start.Block; i <= end.Block; i++ {
		b := s.Doc.Blocks[i]
		if !b.Kind.HasText() {
			if (i > start.Block || start.Offset == 0) && (i < end.Block || end.Offset > 0) {
				d.Blocks = append(d.Blocks, b.Clone())
			}
			continue
		}
		from, to := 0, b.Len()
		if i == start.Block {
			from = start.Offset
		}
		if i == end.Block {
			to = end.Offset
		}
		nb := b.Attrs()
		nb.Spans = sliceSpans(b.Spans, from, to)
		if start.Block == end.Block {
			nb = Block{Spans: nb.Spans} // inline text keeps no block kind
			if b.Kind == Code {
				nb = b.Attrs()
				nb.Spans = sliceSpans(b.Spans, from, to)
				if !(from == 0 && to == b.Len()) {
					nb = Block{Spans: []Span{{Text: spansText(nb.Spans)}}}
				}
			}
		}
		d.Blocks = append(d.Blocks, nb)
	}
	if len(d.Blocks) == 0 {
		d.Blocks = []Block{{}}
	}
	return d
}

// SelectAll selects the whole document.
func (s *State) SelectAll() {
	s.Select(Selection{Anchor: Pos{}, Head: s.Doc.End()})
}

// SetDoc replaces the whole document, as one undoable transaction.
func (s *State) SetDoc(d *Doc) *Tx {
	s.begin("set")
	if d == nil || len(d.Blocks) == 0 {
		d = New()
	}
	old := make([]Block, len(s.Doc.Blocks))
	for i, b := range s.Doc.Blocks {
		old[i] = b.Clone()
	}
	s.apply(Op{Kind: InsertBlocks, Block: len(old), Blocks: d.Clone().Blocks})
	s.apply(Op{Kind: RemoveBlocks, Block: 0, Blocks: old})
	s.Sel = Caret(s.Doc.End())
	return s.commit()
}

// Query reports the text typed since one of the trigger characters
// ("@", "#", "/", ":") on the caret's block, for suggestion lists: the
// trigger, the text after it and where the trigger is. The trigger must
// start the block or follow a space, and the text can't hold a space
// unless spaces is set.
func (s *State) Query(triggers string, spaces bool) (trigger rune, query string, at Pos, ok bool) {
	if !s.Sel.Empty() {
		return 0, "", Pos{}, false
	}
	p := s.Sel.Head
	b := &s.Doc.Blocks[p.Block]
	if !b.Kind.HasText() || b.Kind == Code {
		return 0, "", Pos{}, false
	}
	text := blockRunes(b)
	for i := p.Offset - 1; i >= 0 && p.Offset-i <= 64; i-- {
		r := text[i]
		if r == '￼' || r == '\n' || unicode.IsSpace(r) && !spaces {
			break
		}
		if strings.ContainsRune(triggers, r) && (i == 0 || unicode.IsSpace(text[i-1])) {
			if MarksAt(b.Spans, i+1).Code {
				break
			}
			return r, string(text[i+1 : p.Offset]), Pos{p.Block, i}, true
		}
	}
	return 0, "", Pos{}, false
}

// ReplaceRange replaces [from, to) of one block with text in the
// surrounding marks, as completing a word or slash command does.
func (s *State) ReplaceRange(from, to Pos, text string) *Tx {
	if from.Block != to.Block {
		return nil
	}
	s.begin("type")
	s.Sel = Selection{from, to}
	p := s.deleteRange(from, to)
	if text != "" {
		m := s.typingMarks(p)
		s.replace(p.Block, p.Offset, p.Offset, []Span{{Text: text, Marks: m}})
		p.Offset += utf8.RuneCountInString(text)
	}
	s.Sel = Caret(p)
	return s.commit()
}

// InsertLineBreak breaks the line within the block (Shift+Enter): the
// text holds a newline.
func (s *State) InsertLineBreak() *Tx {
	s.begin("type")
	p := s.textCaret(s.deleteRange(s.Sel.Start(), s.Sel.End()))
	m := s.typingMarks(p)
	s.replace(p.Block, p.Offset, p.Offset, []Span{{Text: "\n", Marks: m}})
	p.Offset++
	s.Sel = Caret(p)
	return s.commit()
}

// Text navigation over a block's runes (atoms as one), for editors.

// PrevStop is the start of the character before off: a grapheme
// cluster, approximately.
func PrevStop(text []rune, off int) int { return prevStop(text, off) }

// NextStop is the end of the character after off.
func NextStop(text []rune, off int) int { return nextStop(text, off) }

// WordStart is the start of the word before off; WordEnd the end of the
// one after.
func WordStart(text []rune, off int) int { return wordStart(text, off) }
func WordEnd(text []rune, off int) int   { return wordEnd(text, off) }

// IsWordRune reports whether r is part of a word.
func IsWordRune(r rune) bool { return isWord(r) }
