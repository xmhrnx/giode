package richedit

import (
	"image"
	"strings"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/richdoc"
)

// frame runs a frame's steps as ui.Frame does.
func frame(gtx layout.Context, view elements.Element) {
	elements.BeginFrame()
	defer elements.EndFrame()
	elements.SetWindowSize(gtx.Constraints.Max)
	elements.SetMetric(gtx.Metric)
	view.Layout(gtx)
	elements.DrainContextMenus(gtx)
	elements.DrainPortals(gtx)
	elements.DispatchBack(gtx)
	elements.DispatchKeys(gtx)
	elements.RunAfterLayout()
}

type harness struct {
	r   input.Router
	e   *RichEditor
	now time.Time
	max image.Point
}

func newHarness(e *RichEditor) *harness {
	fonts.UseGofont()
	h := &harness{e: e, now: time.Unix(0, 0), max: image.Pt(600, 400)}
	e.Focus()
	h.frame()
	h.frame()
	return h
}

func (h *harness) frame() {
	h.now = h.now.Add(16 * time.Millisecond)
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Now: h.now, Source: h.r.Source(), Constraints: layout.Constraints{Max: h.max},
		Metric: unitMetric()}
	frame(gtx, h.e)
	h.r.Frame(&ops)
}

func (h *harness) typeText(s string) {
	for _, r := range s {
		p := h.e.st.Sel.Head.Offset
		h.r.Queue(key.EditEvent{Range: key.Range{Start: p, End: p}, Text: string(r)})
		h.frame()
	}
}

func (h *harness) key(name key.Name, mods key.Modifiers) {
	h.r.Queue(key.Event{Name: name, Modifiers: mods, State: key.Press}, key.Event{Name: name, Modifiers: mods, State: key.Release})
	h.frame()
}

func (h *harness) click(p f32.Point) {
	h.r.Queue(
		pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: p},
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: p, Buttons: pointer.ButtonPrimary},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: p},
	)
	h.frame()
	h.frame()
}

func TestTypingListsAndMarkdown(t *testing.T) {
	e := New("Write…")
	h := newHarness(e)
	if !e.Focused() {
		t.Fatal("not focused")
	}
	h.typeText("# Notes")
	h.key(key.NameReturn, 0)
	h.typeText("- one")
	h.key(key.NameReturn, 0)
	h.typeText("two **bold**")
	h.key(key.NameReturn, 0)
	h.key(key.NameReturn, 0)
	h.typeText("end")
	want := "# Notes\n\n- one\n- two **bold**\n\nend"
	if got := e.Markdown(); got != want {
		t.Fatalf("markdown\n%q\nwant\n%q", got, want)
	}
	// Undo takes the typing of "end" back, then the list's end.
	h.key("Z", key.ModShortcut)
	if got := e.Text(); strings.HasSuffix(got, "end") {
		t.Errorf("undo: %q", got)
	}
	h.key("Z", key.ModShortcut|key.ModShift)
	if got := e.Text(); !strings.HasSuffix(got, "end") {
		t.Errorf("redo: %q", got)
	}
}

func TestSelectionAndFormatting(t *testing.T) {
	e := New("")
	e.SetText("hello world")
	h := newHarness(e)
	h.key(key.NameHome, 0)
	for i := 0; i < 5; i++ {
		h.key(key.NameRightArrow, key.ModShift)
	}
	if s := e.Selection(); s.Anchor.Offset != 0 || s.Head.Offset != 5 {
		t.Fatalf("selection %+v", s)
	}
	h.key("B", key.ModShortcut)
	if got := e.Markdown(); got != "**hello** world" {
		t.Errorf("bold: %q", got)
	}
	// Word-wise moves.
	h.key(key.NameEnd, 0)
	h.key(key.NameLeftArrow, key.ModShortcutAlt)
	if p := e.Selection().Head; p.Offset != 6 {
		t.Errorf("word left: %v", p)
	}
	h.key(key.NameDeleteBackward, key.ModShortcutAlt)
	if got := e.Text(); got != "helloworld" && got != "world" {
		t.Errorf("word delete: %q", got)
	}
}

func TestClickPlacesCaretAndArrowsCrossBlocks(t *testing.T) {
	e := New("")
	e.SetText("first line\nsecond line")
	h := newHarness(e)
	// The end of the first line.
	h.click(f32.Pt(590, 10))
	if p := e.Selection().Head; p != (richdoc.Pos{Block: 0, Offset: 10}) {
		t.Fatalf("click: %v", p)
	}
	h.key(key.NameDownArrow, 0)
	if p := e.Selection().Head; p.Block != 1 || p.Offset < 5 {
		t.Errorf("down: %v", p)
	}
	h.key(key.NameUpArrow, 0)
	if p := e.Selection().Head; p.Block != 0 {
		t.Errorf("up: %v", p)
	}
	h.key(key.NameRightArrow, 0)
	h.key(key.NameRightArrow, 0)
	if p := e.Selection().Head; p != (richdoc.Pos{Block: 1, Offset: 0}) && p != (richdoc.Pos{Block: 1, Offset: 1}) {
		t.Errorf("right across blocks: %v", p)
	}
	// Double-click selects a word.
	h.r.Queue(
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: f32.Pt(10, 10), Buttons: pointer.ButtonPrimary},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: f32.Pt(10, 10)},
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: f32.Pt(10, 10), Buttons: pointer.ButtonPrimary},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: f32.Pt(10, 10)},
	)
	h.frame()
	if s := e.Selection(); s.Start().Offset != 0 || s.End().Offset != 5 {
		t.Errorf("double click: %+v", s)
	}
}

func TestMentionsAndSlash(t *testing.T) {
	e := New("")
	e.Suggest('@', func(q string) []Suggestion {
		var out []Suggestion
		for _, n := range []string{"Ada Lovelace", "Alan Turing", "Grace Hopper"} {
			if strings.HasPrefix(strings.ToLower(n), strings.ToLower(q)) {
				out = append(out, Suggestion{Label: n, Atom: &richdoc.Atom{Kind: "mention", ID: n[:3], Label: "@" + n}})
			}
		}
		return out
	})
	h := newHarness(e)
	h.typeText("hi @a")
	if !e.sugg.open || len(e.sugg.items) != 2 {
		t.Fatalf("suggestions %+v", e.sugg)
	}
	h.key(key.NameDownArrow, 0)
	h.key(key.NameReturn, 0)
	b := e.Doc().Blocks[0]
	if b.Text() != "hi @Alan Turing " || b.Len() != 5 || e.sugg.open {
		t.Fatalf("after picking: %q (%d), open %v", b.Text(), b.Len(), e.sugg.open)
	}
	if got := e.Markdown(); got != "hi [@Alan Turing](mention:Ala)" {
		t.Errorf("markdown %q", got)
	}
	// Slash commands.
	h.key(key.NameReturn, 0)
	h.typeText("/check")
	if !e.sugg.open || e.sugg.items[0].Label != "Checklist" {
		t.Fatalf("slash: %+v", e.sugg.items)
	}
	h.key(key.NameReturn, 0)
	if k, _ := e.Kind(); k != richdoc.Task || e.Doc().Blocks[1].Len() != 0 {
		t.Errorf("slash command: %v %q", k, e.Doc().Blocks[1].Text())
	}
	// Escape closes the list for that trigger.
	h.typeText("/x")
	h.typeText("")
	h.key(key.NameEscape, 0)
	if e.sugg.open {
		t.Error("escape didn't close")
	}
}

func TestInputMethodComposition(t *testing.T) {
	e := New("")
	h := newHarness(e)
	// A composing input method replaces the composition as it goes.
	h.r.Queue(key.EditEvent{Range: key.Range{Start: 0, End: 0}, Text: "ni"})
	h.frame()
	h.r.Queue(key.EditEvent{Range: key.Range{Start: 0, End: 2}, Text: "你"})
	h.frame()
	h.r.Queue(key.EditEvent{Range: key.Range{Start: 1, End: 1}, Text: "好"})
	h.frame()
	if got := e.Text(); got != "你好" {
		t.Errorf("composed %q", got)
	}
	if p := e.Selection().Head; p.Offset != 2 {
		t.Errorf("caret %v", p)
	}
}

func TestSubmitOnEnter(t *testing.T) {
	e := New("Message")
	sent := 0
	e.SubmitOn(SubmitEnter).OnSubmit(func() { sent++ })
	h := newHarness(e)
	h.typeText("hi")
	h.key(key.NameReturn, key.ModShift)
	h.typeText("there")
	h.key(key.NameReturn, 0)
	if sent != 1 || len(e.Doc().Blocks) != 2 {
		t.Errorf("sent %d, blocks %d", sent, len(e.Doc().Blocks))
	}
}

func TestTaskBoxToggles(t *testing.T) {
	e := New("")
	e.SetMarkdown("- [ ] buy milk")
	h := newHarness(e)
	if len(e.taskBoxes) != 1 {
		t.Fatalf("boxes %v", e.taskBoxes)
	}
	c := e.taskBoxes[0].r
	h.click(f32.Pt(float32(c.Min.X+c.Dx()/2), float32(c.Min.Y+c.Dy()/2)))
	if !e.Doc().Blocks[0].Checked {
		t.Error("not checked")
	}
}

// bigDoc is about 50,000 characters.
func bigDoc() *richdoc.Doc {
	var md strings.Builder
	for i := 0; i < 400; i++ {
		switch i % 10 {
		case 0:
			md.WriteString("## Section\n\n")
		case 5:
			md.WriteString("- an item with **bold** and `code`\n- another item\n\n")
		default:
			md.WriteString("Lorem ipsum dolor sit amet, *consectetur* adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam.\n\n")
		}
	}
	return richdoc.ParseMarkdown(md.String(), nil)
}

func TestLargeDocumentLaysOutOnlyWhatShows(t *testing.T) {
	e := New("")
	e.SetDoc(bigDoc())
	if n := e.Doc().Len(); n < 45000 {
		t.Fatalf("document of %d characters", n)
	}
	h := newHarness(e)
	laid := 0
	for _, v := range e.laid {
		if v != nil {
			laid++
		}
	}
	if laid > 30 {
		t.Errorf("%d blocks laid out", laid)
	}
	// The caret is at the end, in view.
	r, ok := e.caretRect(e.Selection().Head)
	if !ok || float32(r.Max.Y) > e.offY+float32(e.size.Y) || float32(r.Min.Y) < e.offY {
		t.Errorf("caret %v not in view at %v (%v)", r, e.offY, e.size)
	}
	start := time.Now()
	h.typeText("typing at the end of a long document")
	per := time.Since(start) / 36
	t.Logf("%v a keystroke's frame", per)
}

func BenchmarkTypingFrame(b *testing.B) {
	e := New("")
	e.SetDoc(bigDoc())
	h := newHarness(e)
	e.Select(richdoc.Caret(richdoc.Pos{Block: 200}))
	h.frame()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := e.st.Sel.Head.Offset
		h.r.Queue(key.EditEvent{Range: key.Range{Start: p, End: p}, Text: "x"})
		h.frame()
	}
}

func unitMetric() unit.Metric { return unit.Metric{PxPerDp: 1, PxPerSp: 1} }
