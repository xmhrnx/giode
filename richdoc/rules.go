package richdoc

import (
	"strings"
	"unicode"
)

// startRule marks the rest of the transaction as an input rule's.
func (s *State) startRule() {
	if s.tx.rule < 0 {
		s.tx.rule = len(s.tx.Ops)
		s.tx.mid = s.Sel
	}
}

// inputRules applies Markdown typed just now: block prefixes at the
// start of a paragraph and inline delimiters closing around text.
func (s *State) inputRules() {
	p := s.Sel.Head
	b := &s.Doc.Blocks[p.Block]
	if b.Kind == Code || !b.Kind.HasText() {
		return
	}
	text := blockRunes(b)
	before := string(text[:p.Offset])
	if s.blockRule(p, b, before) {
		return
	}
	if MarksAt(b.Spans, p.Offset-1).Code && text[p.Offset-1] != '`' {
		return
	}
	s.inlineRule(p, text)
}

// blockRule turns a paragraph starting with a Markdown prefix into a
// list item, quote, heading, code block or rule.
func (s *State) blockRule(p Pos, b *Block, before string) bool {
	if p.Offset != len([]rune(before)) {
		return false
	}
	var nb Block
	switch {
	case b.Kind == Paragraph && (before == "- " || before == "* " || before == "+ "):
		nb = Block{Kind: Bullet}
	case b.Kind == Paragraph && isOrderedPrefix(before):
		nb = Block{Kind: Numbered}
	case (b.Kind == Paragraph || b.Kind == Bullet || b.Kind == Numbered) && (before == "[] " || before == "[ ] "):
		nb = Block{Kind: Task, Level: b.Level}
	case (b.Kind == Paragraph || b.Kind == Bullet || b.Kind == Numbered) && (before == "[x] " || before == "[X] "):
		nb = Block{Kind: Task, Level: b.Level, Checked: true}
	case b.Kind == Paragraph && before == "> ":
		nb = Block{Kind: Quote}
	case b.Kind == Paragraph && strings.HasSuffix(before, " ") && strings.Trim(before, "#") == " " && len(before) <= 7:
		nb = Block{Kind: Heading, Level: len(before) - 1}
	case b.Kind == Paragraph && before == "```":
		nb = Block{Kind: Code}
	case b.Kind == Paragraph && before == "---" && b.Len() == 3:
		s.startRule()
		s.replace(p.Block, 0, 3, nil)
		s.setBlock(p.Block, Block{Kind: Rule})
		next := p.Block + 1
		if next >= len(s.Doc.Blocks) || !s.Doc.Blocks[next].Kind.HasText() {
			s.apply(Op{Kind: InsertBlocks, Block: next, Blocks: []Block{{}}})
		}
		s.Sel = Caret(Pos{Block: next})
		return true
	default:
		return false
	}
	s.startRule()
	n := len([]rune(before))
	s.replace(p.Block, 0, n, nil)
	nb.Spans = s.Doc.Blocks[p.Block].Spans
	if nb.Kind == Code {
		nb.Spans = []Span{{Text: spansText(nb.Spans)}}
		if spansLen(nb.Spans) == 0 {
			nb.Spans = nil
		}
	}
	s.setBlock(p.Block, nb)
	s.Sel = Caret(Pos{Block: p.Block})
	return true
}

func isOrderedPrefix(s string) bool {
	if len(s) < 3 || len(s) > 5 || s[len(s)-1] != ' ' {
		return false
	}
	if c := s[len(s)-2]; c != '.' && c != ')' {
		return false
	}
	for _, r := range s[:len(s)-2] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// inlineRules are delimiters closing around text, longest first.
var inlineRules = []struct {
	delim string
	mark  MarkKind
}{
	{"**", Bold}, {"__", Bold}, {"~~", Strike}, {"*", Italic}, {"_", Italic}, {"`", InlineCode},
}

func (s *State) inlineRule(p Pos, text []rune) {
	n := p.Offset
	for _, r := range inlineRules {
		d := []rune(r.delim)
		k := len(d)
		if n < 2*k+1 || string(text[n-k:n]) != r.delim {
			continue
		}
		// A single * or _ right after another is the start of ** or __.
		if k == 1 && n-2 >= 0 && text[n-2] == d[0] {
			continue
		}
		closeAt := n - k
		if unicode.IsSpace(text[closeAt-1]) {
			continue
		}
		for j := closeAt - 1; j >= k-1; j-- {
			open := j - k + 1
			if string(text[open:j+1]) != r.delim {
				continue
			}
			content := text[j+1 : closeAt]
			if len(content) == 0 || unicode.IsSpace(content[0]) {
				break
			}
			// The opening delimiter starts a word, and isn't part of a
			// longer one.
			if open > 0 && (isWord(text[open-1]) && r.delim != "`" || text[open-1] == d[0]) {
				break
			}
			s.startRule()
			b := &s.Doc.Blocks[p.Block]
			spans := sliceSpans(b.Spans, j+1, closeAt)
			for i := range spans {
				*r.mark.get(&spans[i].Marks) = true
			}
			s.replace(p.Block, open, n, normalize(spans))
			end := open + len(content)
			s.Sel = Caret(Pos{p.Block, end})
			// Typing on doesn't continue the mark.
			m := MarksAt(s.Doc.Blocks[p.Block].Spans, end)
			*r.mark.get(&m) = false
			s.Stored = &m
			return
		}
	}
}
