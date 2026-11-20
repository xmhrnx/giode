package richdoc

import (
	"encoding/json"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"time"
)

func para(spans ...Span) Block { return Block{Spans: spans} }
func txt(s string) Span        { return Span{Text: s} }
func bold(s string) Span       { return Span{Text: s, Marks: Marks{Bold: true}} }

func docOf(blocks ...Block) *Doc { return &Doc{Blocks: blocks} }

func kinds(d *Doc) string {
	var out []string
	for _, b := range d.Blocks {
		out = append(out, b.Kind.String()+":"+b.Text())
	}
	return strings.Join(out, "|")
}

// apply runs a command, checks undoing its operations restores the
// document, and redoes them.
func check(t *testing.T, s *State, cmd func() *Tx) *Tx {
	t.Helper()
	before := s.Doc.Clone()
	tx := cmd()
	if tx == nil {
		return nil
	}
	after := s.Doc.Clone()
	undo := s.Doc.Clone()
	for i := len(tx.Ops) - 1; i >= 0; i-- {
		if err := undo.Apply(tx.Ops[i].Invert()); err != nil {
			t.Fatalf("invert %d (%v): %v", i, tx.Ops[i].Kind, err)
		}
	}
	if !reflect.DeepEqual(norm(undo), norm(before)) {
		t.Fatalf("undo doesn't restore:\n%s\n%s\nops %s", dump(undo), dump(before), mustJSON(tx.Ops))
	}
	for _, op := range tx.Ops {
		if err := undo.Apply(op); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(norm(undo), norm(after)) {
		t.Fatalf("redo differs")
	}
	return tx
}

func norm(d *Doc) *Doc {
	d = d.Clone()
	for i := range d.Blocks {
		d.Blocks[i].Spans = normalize(d.Blocks[i].Spans)
		if len(d.Blocks[i].Spans) == 0 {
			d.Blocks[i].Spans = nil
		}
	}
	return d
}

func typeText(t *testing.T, s *State, text string) {
	for _, r := range text {
		check(t, s, func() *Tx { return s.InsertText(string(r)) })
	}
}

func TestTypingAndEnter(t *testing.T) {
	s := NewState(nil)
	typeText(t, s, "Hello")
	check(t, s, s.Enter)
	typeText(t, s, "world")
	if got := s.Doc.Text(); got != "Hello\nworld" {
		t.Fatalf("text %q", got)
	}
	if s.Sel.Head != (Pos{1, 5}) {
		t.Errorf("caret %v", s.Sel.Head)
	}
	// Backspace at the start joins.
	s.Select(Caret(Pos{1, 0}))
	check(t, s, func() *Tx { return s.Delete(true, Char) })
	if got := s.Doc.Text(); got != "Helloworld" || s.Sel.Head != (Pos{0, 5}) {
		t.Errorf("join: %q at %v", got, s.Sel.Head)
	}
	// Word delete.
	s.Select(Caret(s.Doc.End()))
	check(t, s, func() *Tx { return s.Delete(true, Word) })
	if got := s.Doc.Text(); got != "" {
		t.Errorf("word delete: %q", got)
	}
}

func TestListsAndRules(t *testing.T) {
	s := NewState(nil)
	typeText(t, s, "- one")
	if kinds(s.Doc) != "bullet:one" {
		t.Fatalf("bullet rule: %s", kinds(s.Doc))
	}
	check(t, s, s.Enter)
	typeText(t, s, "two")
	check(t, s, s.Enter)
	check(t, s, s.Enter) // an empty item ends the list
	typeText(t, s, "1. first")
	check(t, s, s.Enter)
	check(t, s, func() *Tx { return s.Indent(false) })
	typeText(t, s, "[x] done")
	if got := kinds(s.Doc); got != "bullet:one|bullet:two|numbered:first|task:done" {
		t.Fatalf("kinds %s", got)
	}
	if b := s.Doc.Blocks[3]; !b.Checked || b.Level != 1 {
		t.Errorf("task %+v", b)
	}
	check(t, s, s.Enter)
	typeText(t, s, "## Title")
	// "## " in a task isn't a heading: only paragraphs take block rules.
	if got := s.Doc.Blocks[4]; got.Kind != Task || got.Checked {
		t.Errorf("new task %+v", got)
	}
	s.Select(Caret(Pos{4, 0}))
	check(t, s, func() *Tx { return s.Delete(true, Char) }) // outdent
	check(t, s, func() *Tx { return s.Delete(true, Char) }) // paragraph
	if b := s.Doc.Blocks[4]; b.Kind != Paragraph || b.Text() != "## Title" {
		t.Errorf("backspace at item start: %+v", b)
	}
	s2 := NewState(nil)
	typeText(t, s2, "### Heading")
	typeText(t, s2, "")
	if b := s2.Doc.Blocks[0]; b.Kind != Heading || b.Level != 3 || b.Text() != "Heading" {
		t.Errorf("heading rule %+v", b)
	}
	check(t, s2, s2.Enter)
	if b := s2.Doc.Blocks[1]; b.Kind != Paragraph {
		t.Errorf("after a heading %+v", b)
	}
	typeText(t, s2, "```")
	if b := s2.Doc.Blocks[1]; b.Kind != Code {
		t.Errorf("code rule %+v", b)
	}
	typeText(t, s2, "x := 1")
	check(t, s2, s2.Enter)
	typeText(t, s2, "**y**")
	check(t, s2, s2.Enter)
	check(t, s2, s2.Enter) // leaves the block
	if got := s2.Doc.Blocks[1].Text(); got != "x := 1\n**y**" || s2.Doc.Blocks[2].Kind != Paragraph {
		t.Errorf("code block %q, then %s", got, kinds(s2.Doc))
	}
	s3 := NewState(nil)
	typeText(t, s3, "---")
	if got := kinds(s3.Doc); got != "rule:|paragraph:" || s3.Sel.Head != (Pos{1, 0}) {
		t.Errorf("rule: %s at %v", got, s3.Sel.Head)
	}
}

func TestInlineRules(t *testing.T) {
	cases := []struct {
		in, text string
		mark     MarkKind
	}{
		{"a **bold**", "a bold", Bold},
		{"a *it*", "a it", Italic},
		{"say `x+1`", "say x+1", InlineCode},
		{"~~gone~~", "gone", Strike},
		{"__b__", "b", Bold},
	}
	for _, c := range cases {
		s := NewState(nil)
		typeText(t, s, c.in)
		b := s.Doc.Blocks[0]
		if b.Text() != c.text {
			t.Errorf("%q: text %q", c.in, b.Text())
			continue
		}
		last := b.Spans[len(b.Spans)-1]
		if !*c.mark.get(&last.Marks) {
			t.Errorf("%q: spans %+v", c.in, b.Spans)
		}
		typeText(t, s, "z")
		if m := s.Doc.Blocks[0].Spans[len(s.Doc.Blocks[0].Spans)-1]; *c.mark.get(&m.Marks) {
			t.Errorf("%q: typing continues the mark", c.in)
		}
	}
	// Not rules: a space inside the delimiter, snake_case.
	for _, in := range []string{"a ** b**", "snake_case_name", "2*3*4"} {
		s := NewState(nil)
		typeText(t, s, in)
		if b := s.Doc.Blocks[0]; b.Text() != in {
			t.Errorf("%q became %q", in, b.Text())
		}
	}
}

func TestHistory(t *testing.T) {
	s := NewState(nil)
	h := &History{}
	clock := time.Unix(0, 0)
	h.now = func() time.Time { return clock }
	do := func(tx *Tx) { h.Record(tx); clock = clock.Add(100 * time.Millisecond) }
	for _, r := range "**hi" {
		do(s.InsertText(string(r)))
	}
	do(s.InsertText("*"))
	do(s.InsertText("*")) // the rule makes "hi" bold
	if b := s.Doc.Blocks[0]; b.Text() != "hi" || !b.Spans[0].Marks.Bold {
		t.Fatalf("rule: %+v", b)
	}
	h.Undo(s) // the rule, leaving what was typed
	if got := s.Doc.Text(); got != "**hi**" {
		t.Errorf("undo rule: %q", got)
	}
	h.Undo(s) // all the typing, one group
	if got := s.Doc.Text(); got != "" || s.Sel.Head != (Pos{}) {
		t.Errorf("undo typing: %q at %v", got, s.Sel.Head)
	}
	h.Redo(s)
	h.Redo(s)
	if got := s.Doc.Text(); got != "hi" {
		t.Errorf("redo: %q", got)
	}
	// A pause starts a new group.
	clock = clock.Add(2 * time.Second)
	do(s.InsertText("!"))
	h.Undo(s)
	if got := s.Doc.Text(); got != "hi" {
		t.Errorf("separate group: %q", got)
	}
}

func TestDeleteAcrossBlocks(t *testing.T) {
	d := docOf(
		Block{Kind: Heading, Level: 1, Spans: []Span{txt("Title")}},
		para(txt("one "), bold("two")),
		Block{Kind: Image, Src: "a.png"},
		Block{Kind: Bullet, Spans: []Span{txt("item")}},
	)
	s := NewState(d)
	s.Select(Selection{Pos{0, 2}, Pos{3, 2}})
	frag := s.Fragment()
	if got := kinds(frag); got != "heading:tle|paragraph:one two|image:|bullet:it" {
		t.Errorf("fragment %s", got)
	}
	check(t, s, func() *Tx { return s.Delete(true, Char) })
	if got := kinds(s.Doc); got != "heading:Tiem" || s.Sel.Head != (Pos{0, 2}) {
		t.Errorf("after delete: %s at %v", got, s.Sel.Head)
	}
	// Pasting the fragment back restores the text.
	check(t, s, func() *Tx { return s.InsertDoc(frag) })
	if got := s.Doc.Text(); got != "Title\none two\n\nitem" {
		t.Errorf("after paste: %q (%s)", got, kinds(s.Doc))
	}
	// Deleting everything leaves an empty paragraph.
	s.SelectAll()
	check(t, s, func() *Tx { return s.Delete(false, Char) })
	if got := kinds(s.Doc); got != "heading:" && got != "paragraph:" {
		t.Errorf("all deleted: %s", got)
	}
}

func TestMarksAndAtoms(t *testing.T) {
	s := NewState(docOf(para(txt("hello world"))))
	s.Select(Selection{Pos{0, 0}, Pos{0, 5}})
	check(t, s, func() *Tx { return s.ToggleMark(Bold) })
	if !s.HasMark(Bold) {
		t.Error("bold not on")
	}
	s.Select(Selection{Pos{0, 3}, Pos{0, 8}})
	check(t, s, func() *Tx { return s.ToggleMark(Bold) }) // partly bold: all on
	if !s.HasMark(Bold) || len(s.Doc.Blocks[0].Spans) != 2 {
		t.Errorf("spans %+v", s.Doc.Blocks[0].Spans)
	}
	check(t, s, func() *Tx { return s.SetLink("javascript:alert(1)") })
	s.Select(Caret(Pos{0, 11}))
	typeText(t, s, " @ad")
	trig, q, at, ok := s.Query("@#/", false)
	if !ok || trig != '@' || q != "ad" || at != (Pos{0, 12}) {
		t.Fatalf("query %c %q %v %v", trig, q, at, ok)
	}
	check(t, s, func() *Tx { return s.InsertAtom(Atom{Kind: "mention", ID: "7", Label: "@Ada"}, &at, " ") })
	b := s.Doc.Blocks[0]
	if b.Text() != "hello world @Ada " || b.Len() != 14 || s.Sel.Head.Offset != 14 {
		t.Errorf("atom: %q len %d caret %v", b.Text(), b.Len(), s.Sel.Head)
	}
	// Backspace deletes the space, then the whole atom.
	check(t, s, func() *Tx { return s.Delete(true, Char) })
	check(t, s, func() *Tx { return s.Delete(true, Char) })
	if got := s.Doc.Blocks[0].Text(); got != "hello world " {
		t.Errorf("atom delete: %q", got)
	}
	// Stored marks at a caret.
	check(t, s, func() *Tx { return s.ToggleMark(Italic) })
	typeText(t, s, "x")
	if m := s.Doc.Blocks[0].Spans[len(s.Doc.Blocks[0].Spans)-1].Marks; !m.Italic {
		t.Errorf("stored mark: %+v", m)
	}
}

func TestEmojiDelete(t *testing.T) {
	s := NewState(docOf(para(txt("a👍🏽🇮🇷👨‍👩‍👧"))))
	s.Select(Caret(s.Doc.End()))
	want := []string{"a👍🏽🇮🇷", "a👍🏽", "a", ""}
	for _, w := range want {
		check(t, s, func() *Tx { return s.Delete(true, Char) })
		if got := s.Doc.Text(); got != w {
			t.Fatalf("got %q, want %q", got, w)
		}
	}
}

func TestMapThroughRemoteOps(t *testing.T) {
	s := NewState(docOf(para(txt("hello world"))))
	s.Select(Caret(Pos{0, 6}))
	// A collaborator types before the caret and splits the block.
	ops := []Op{
		{Kind: Replace, Block: 0, From: 0, To: 0, New: []Span{txt(">> ")}},
		{Kind: Split, Block: 0, From: 3, Attrs: &Block{}},
	}
	if err := s.Apply(ops); err != nil {
		t.Fatal(err)
	}
	if s.Sel.Head != (Pos{1, 6}) || s.Doc.Blocks[1].Text()[6:] != "world" {
		t.Errorf("caret %v in %s", s.Sel.Head, kinds(s.Doc))
	}
}

func TestRandomOpsInvert(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	s := NewState(docOf(para(txt("The quick brown fox")), Block{Kind: Bullet, Spans: []Span{bold("jumps")}}))
	h := &History{}
	var snapshots []*Doc
	for i := 0; i < 400; i++ {
		snapshots = append(snapshots, s.Doc.Clone())
		a := Pos{rng.Intn(len(s.Doc.Blocks)), 0}
		a.Offset = rng.Intn(s.Doc.BlockLen(a.Block) + 1)
		b := Pos{rng.Intn(len(s.Doc.Blocks)), 0}
		b.Offset = rng.Intn(s.Doc.BlockLen(b.Block) + 1)
		s.Select(Selection{a, b})
		var tx *Tx
		switch rng.Intn(9) {
		case 0, 1:
			tx = check(t, s, func() *Tx { return s.InsertText([]string{"x", "yz", "a\nb", "*"}[rng.Intn(4)]) })
		case 2:
			tx = check(t, s, s.Enter)
		case 3:
			tx = check(t, s, func() *Tx { return s.Delete(rng.Intn(2) == 0, Unit(rng.Intn(3))) })
		case 4:
			tx = check(t, s, func() *Tx { return s.ToggleMark(MarkKind(rng.Intn(5))) })
		case 5:
			tx = check(t, s, func() *Tx { return s.SetKind(Kind(rng.Intn(7)), 1) })
		case 6:
			tx = check(t, s, func() *Tx { return s.InsertBlock(Block{Kind: Rule}) })
		case 7:
			frag := s.Fragment()
			tx = check(t, s, func() *Tx { return s.InsertDoc(frag) })
		case 8:
			tx = check(t, s, func() *Tx { return s.Indent(rng.Intn(2) == 0) })
		}
		h.Break()
		if tx == nil || len(tx.Ops) == 0 {
			snapshots = snapshots[:len(snapshots)-1]
		}
		h.Record(tx)
		for bi := range s.Doc.Blocks {
			if s.Doc.Blocks[bi].Kind == Image {
				t.Fatal("no images were inserted")
			}
		}
	}
	// Undoing everything gets back to the start, step by step.
	for i := len(snapshots) - 1; i >= 0; i-- {
		if h.Undo(s) == nil {
			// Rule steps undo separately: more undos than snapshots.
			break
		}
	}
	for h.CanUndo() {
		h.Undo(s)
	}
	if !reflect.DeepEqual(norm(s.Doc), norm(snapshots[0])) {
		t.Errorf("undo all: %s, want %s", kinds(s.Doc), kinds(snapshots[0]))
	}
}

func TestMarkdownRoundTrip(t *testing.T) {
	d := docOf(
		Block{Kind: Heading, Level: 2, Spans: []Span{txt("Notes")}},
		para(txt("Some "), bold("bold "), Span{Text: "and both", Marks: Marks{Bold: true, Italic: true}}, txt(", "),
			Span{Text: "code", Marks: Marks{Code: true}}, txt(", a "), Span{Text: "link", Marks: Marks{Link: "https://x.dev/a"}},
			txt(", "), Span{Text: "under", Marks: Marks{Underline: true}}, txt(" "), Span{Text: "red", Marks: Marks{Color: "#e00"}},
			txt(" "), Span{Text: "lit", Marks: Marks{Highlight: "#ff0"}}, txt(" "), Span{Atom: &Atom{Kind: "mention", ID: "42", Label: "@Ada"}},
			txt(" 2*3 [x] ~")),
		Block{Kind: Bullet, Spans: []Span{txt("one")}},
		Block{Kind: Bullet, Level: 1, Spans: []Span{txt("nested")}},
		Block{Kind: Task, Checked: true, Spans: []Span{txt("done")}},
		Block{Kind: Numbered, Spans: []Span{txt("first")}},
		Block{Kind: Numbered, Spans: []Span{txt("second")}},
		Block{Kind: Quote, Spans: []Span{txt("quoted")}},
		Block{Kind: Code, Lang: "go", Spans: []Span{txt("x := `a`\n\ny")}},
		Block{Kind: Table, Cells: [][][]Span{{{txt("A")}, {txt("B|C")}}, {{bold("1")}, {txt("2")}}}},
		Block{Kind: Image, Src: "https://x.dev/p.png", Alt: "pic"},
		Block{Kind: Rule},
		para(txt("# not a heading")),
		para(),
		para(txt("end")),
	)
	md := d.Markdown()
	back := ParseMarkdown(md, nil)
	if !reflect.DeepEqual(norm(back), norm(d)) {
		t.Errorf("round trip differs\n%s\n--- got\n%s\n--- want\n%s", md, dump(back), dump(d))
	}
}

func dump(d *Doc) string {
	b, _ := json.MarshalIndent(d, "", " ")
	return string(b)
}

func TestJSONRoundTrip(t *testing.T) {
	d := docOf(Block{Kind: Task, Level: 2, Checked: true, Spans: []Span{bold("a"), {Atom: &Atom{Kind: "ref", ID: "T-1", Label: "#T-1"}}}},
		Block{Kind: Table, Cells: [][][]Span{{{txt("h")}}}})
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"type":"task"`) || strings.Contains(string(b), `"marks":{}`) {
		t.Errorf("json %s", b)
	}
	var back Doc
	if err := json.Unmarshal(b, &back); err != nil || !reflect.DeepEqual(norm(&back), norm(d)) {
		t.Errorf("json round trip: %v\n%s", err, dump(&back))
	}
	// Transactions serialize, for sending to collaborators.
	s := NewState(d)
	tx := s.InsertText("x")
	if b, err := json.Marshal(tx); err != nil || !strings.Contains(string(b), `"kind":0`) {
		t.Errorf("tx json %s %v", b, err)
	}
}

func TestHTMLPaste(t *testing.T) {
	src := `<meta charset="utf-8"><b style="font-weight:normal" id="docs-internal-guid-1">
	<h1>Title <script>alert(1)</script></h1>
	<p>Plain <strong>bold</strong> <span style="font-style:italic;color:#ff0000">red</span>
	<a href="javascript:alert(1)">bad</a> <a href="https://ok.dev" onclick="x()">good</a></p>
	<ul><li>one</li><li><p>two</p><ul><li>deep</li></ul></li></ul>
	<ol><li><input type="checkbox" checked> done</li></ol>
	<blockquote><p>q</p></blockquote>
	<pre><code class="language-go">a := 1
b := 2
</code></pre>
	<table><tr><th>H</th></tr><tr><td><p>c</p></td></tr></table>
	<img src="https://x.dev/i.png" alt="i"><img src="data:image/png;base64,AA" alt="no">
	<style>p{}</style></b>`
	d := ParseHTML(src)
	got := kinds(d)
	want := "heading:Title|paragraph:Plain bold red bad good|bullet:one|bullet:two|bullet:deep|task:done|quote:q|code:a := 1\nb := 2|table:|image:|paragraph:no"
	if got != want {
		t.Fatalf("kinds\n%s\nwant\n%s", got, want)
	}
	p := d.Blocks[1].Spans
	var red, bad, good *Span
	for i := range p {
		switch p[i].Text {
		case "red":
			red = &p[i]
		case "bad":
			bad = &p[i]
		case "good":
			good = &p[i]
		}
	}
	if red == nil || !red.Marks.Italic || red.Marks.Color != "#ff0000" || red.Marks.Bold {
		t.Errorf("red %+v", red)
	}
	if bad != nil || good == nil || good.Marks.Link != "https://ok.dev" || strings.Contains(mustJSON(d), "javascript") {
		t.Errorf("links %+v %+v", bad, good) // the unsafe link is plain text
	}
	if d.Blocks[4].Level != 1 || !d.Blocks[5].Checked || d.Blocks[7].Lang != "go" {
		t.Errorf("attrs %+v %+v %+v", d.Blocks[4], d.Blocks[5], d.Blocks[7])
	}
	if h := d.HTML(); strings.Contains(h, "script") || !strings.Contains(h, "<li>deep</li>") {
		t.Errorf("html %s", h)
	}
	// HTML written reads back the same.
	if back := ParseHTML(d.HTML()); kinds(back) != got {
		t.Errorf("html round trip %s", kinds(back))
	}
}

func BenchmarkTypingInLargeDoc(b *testing.B) {
	var blocks []Block
	for i := 0; i < 500; i++ {
		blocks = append(blocks, para(txt(strings.Repeat("lorem ipsum ", 9))))
	}
	s := NewState(docOf(blocks...))
	s.Select(Caret(Pos{250, 50}))
	h := &History{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Record(s.InsertText("x"))
	}
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func TestTableEdits(t *testing.T) {
	s := NewState(docOf(para(txt("a")), Block{Kind: Table, Cells: [][][]Span{{{txt("h1")}, {txt("h2")}}, {{txt("x")}, nil}}}))
	check(t, s, func() *Tx { return s.SetCell(1, 1, 1, []Span{bold("y")}) })
	check(t, s, func() *Tx { return s.InsertRow(1, 2) })
	check(t, s, func() *Tx { return s.InsertColumn(1, 0) })
	tb := s.Doc.Blocks[1]
	if len(tb.Cells) != 3 || len(tb.Cells[0]) != 3 || SpansText(tb.Cells[1][2]) != "y" {
		t.Fatalf("table %s", mustJSON(tb))
	}
	check(t, s, func() *Tx { return s.DeleteColumn(1, 0) })
	check(t, s, func() *Tx { return s.DeleteRow(1, 2) })
	if got := s.Doc.Markdown(); got != "a\n\n| h1 | h2 |\n| --- | --- |\n| x | **y** |" {
		t.Errorf("markdown %q", got)
	}
	if got := ReplaceSpans([]Span{txt("hello")}, 1, 3, []Span{txt("EY")}); SpansText(got) != "hEYlo" {
		t.Errorf("replace %v", got)
	}
}
