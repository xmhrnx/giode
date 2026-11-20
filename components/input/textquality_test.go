package input

import (
	"image"
	"testing"

	"gioui.org/io/key"

	"github.com/xmhrnx/giode/fonts"
)

func newFocused(t *testing.T) (*Input, *harness) {
	fonts.UseGofont()
	in := New("Text")
	h := &harness{in: in, max: image.Pt(400, 200)}
	in.Focus()
	h.frame()
	h.frame()
	if !in.Focused() {
		t.Fatal("not focused")
	}
	return in, h
}

func (h *harness) edit(start, end int, text string) {
	h.r.Queue(key.EditEvent{Range: key.Range{Start: start, End: end}, Text: text})
	h.frame()
}

func (h *harness) press(name key.Name, mods key.Modifiers) {
	h.r.Queue(key.Event{Name: name, Modifiers: mods, State: key.Press}, key.Event{Name: name, Modifiers: mods, State: key.Release})
	h.frame()
}

// IME composition replaces its composing range with each update, as
// Pinyin, Korean, Japanese and predictive keyboards do.
func TestInput_IMEComposition(t *testing.T) {
	in, h := newFocused(t)
	// Pinyin: "ni" composes, then becomes 你; "hao" becomes 好.
	h.edit(0, 0, "n")
	h.edit(0, 1, "ni")
	h.edit(0, 2, "你")
	h.edit(1, 1, "hao")
	h.edit(1, 4, "好")
	if in.Text() != "你好" {
		t.Errorf("pinyin: %q", in.Text())
	}
	// Korean: jamo combine into a syllable as they're typed.
	in.SetText("")
	h.frame()
	for _, step := range []string{"ㅎ", "하", "한"} {
		h.edit(0, len([]rune(in.Text())), step)
	}
	h.edit(1, 1, "ㄱ")
	h.edit(1, 2, "그")
	h.edit(1, 2, "글")
	if in.Text() != "한글" {
		t.Errorf("hangul: %q", in.Text())
	}
	// Predictive text replaces the word being typed.
	in.SetText("I am goin")
	h.frame()
	h.edit(5, 9, "going ")
	if in.Text() != "I am going " {
		t.Errorf("prediction: %q", in.Text())
	}
	// Persian, and Arabic shaping, go in as typed.
	in.SetText("")
	h.frame()
	h.edit(0, 0, "سلام ")
	h.edit(5, 5, "دنیا")
	if in.Text() != "سلام دنیا" {
		t.Errorf("persian: %q", in.Text())
	}
}

func TestInput_UndoRedo(t *testing.T) {
	in, h := newFocused(t)
	h.edit(0, 0, "hello")
	h.edit(5, 5, " world")
	h.press("Z", key.ModShortcut)
	if in.Text() != "hello" {
		t.Fatalf("undo: %q", in.Text())
	}
	h.press("Z", key.ModShortcut)
	if in.Text() != "" {
		t.Fatalf("second undo: %q", in.Text())
	}
	h.press("Z", key.ModShortcut|key.ModShift)
	if in.Text() != "hello" {
		t.Fatalf("redo: %q", in.Text())
	}
	h.press("Y", key.ModShortcut)
	if in.Text() != "hello world" {
		t.Errorf("Ctrl+Y redo: %q", in.Text())
	}
}

// The caret moves through mixed Persian and English text visiting every
// position, and selections cover what they should.
func TestInput_BidiCaret(t *testing.T) {
	in, h := newFocused(t)
	text := "Go سلام دنیا is fun"
	in.SetText(text)
	h.frame()
	n := len([]rune(text))
	in.SetCaret(0)
	h.frame()
	seen := map[int]bool{in.Caret(): true}
	for i := 0; i < n+3; i++ {
		h.press(key.NameRightArrow, 0)
		seen[in.Caret()] = true
	}
	for p := 0; p <= n; p++ {
		if !seen[p] {
			t.Errorf("Right never reached position %d of %d (visited %v)", p, n, seen)
			break
		}
	}
	// Shift+End from the start selects it all; typing replaces it.
	in.SetCaret(0)
	h.frame()
	h.press(key.NameEnd, key.ModShift)
	if s, e := in.Selection(); s != 0 || e != n {
		t.Errorf("Shift+End selected %d-%d of %d", s, e, n)
	}
	// Word steps stop at the Persian words too.
	in.SetCaret(0)
	h.frame()
	var stops []int
	for i := 0; i < 6; i++ {
		h.press(key.NameRightArrow, key.ModShortcut)
		stops = append(stops, in.Caret())
	}
	t.Logf("word stops %v", stops)
	if stops[len(stops)-1] != n {
		t.Errorf("word steps didn't reach the end: %v", stops)
	}
}

func TestInput_WordEditsAndHistory(t *testing.T) {
	in, h := newFocused(t)
	// Typed letter by letter, undo goes back a word at a time.
	for i, r := range []rune("one two") {
		h.edit(i, i, string(r))
	}
	if !in.CanUndo() || in.CanRedo() {
		t.Fatal("history flags")
	}
	in.Undo()
	if in.Text() != "one " {
		t.Errorf("undo a word: %q", in.Text())
	}
	in.Redo()
	if in.Text() != "one two" {
		t.Errorf("redo: %q", in.Text())
	}
	// Ctrl+Backspace deletes the Persian word before the caret.
	in.SetText("سلام دنیا")
	h.frame()
	h.press(key.NameDeleteBackward, key.ModShortcutAlt)
	if in.Text() != "سلام " {
		t.Errorf("word delete: %q", in.Text())
	}
	in.Undo()
	if in.Text() != "سلام دنیا" {
		t.Errorf("undo word delete: %q", in.Text())
	}
	// SetText starts the history over.
	in.SetText("fresh")
	h.frame()
	if in.CanUndo() {
		t.Error("SetText kept the history")
	}
	// The touch menu offers Undo when there's something to undo.
	h.edit(5, 5, "!")
	acts := in.menu.actions(in)
	found := false
	for _, a := range acts {
		found = found || a == "undo"
	}
	if !found {
		t.Errorf("menu actions %v lack undo", acts)
	}
}
