package input

import (
	"image"
	"reflect"
	"testing"

	"gioui.org/io/event"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/elements"
)

type kbHarness struct {
	r       input.Router
	k       *KeyboardInput
	text    []string
	keys    []string
	app     []string
	watched []string
	use     func(key.Event) bool
}

func newKBHarness(names ...key.Name) *kbHarness {
	h := &kbHarness{}
	h.k = NewKeyboardInput(nil).
		OnText(func(s string) { h.text = append(h.text, s) }).
		OnKey(func(e key.Event) bool {
			if e.State == key.Press {
				h.keys = append(h.keys, e.Modifiers.String()+string(e.Name))
			}
			if h.use != nil {
				return h.use(e)
			}
			return true
		}, names...)
	h.frame()
	h.k.Focus()
	h.frame()
	h.frame()
	if !h.k.Focused() {
		panic("not focused")
	}
	return h
}

func (h *kbHarness) frame() {
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Source: h.r.Source(), Constraints: layout.Exact(image.Pt(200, 100))}
	elements.OnKey(func(e key.Event) bool {
		if e.State == key.Press {
			h.app = append(h.app, e.Modifiers.String()+string(e.Name))
		}
		return true
	})
	elements.WatchKeys(func(e key.Event) {
		if e.State == key.Press {
			h.watched = append(h.watched, string(e.Name))
		}
	})
	h.k.Layout(gtx)
	elements.DispatchBack(gtx)
	elements.DispatchKeys(gtx)
	h.r.Frame(&ops)
}

func (h *kbHarness) send(evs ...event.Event) {
	h.r.Queue(evs...)
	h.frame()
}

func (h *kbHarness) reset() { h.text, h.keys, h.app, h.watched = nil, nil, nil, nil }

func check(t *testing.T, what string, got, want []string) {
	t.Helper()
	if len(got) == 0 && len(want) == 0 {
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s = %q, want %q", what, got, want)
	}
}

func TestKeyboardInputText(t *testing.T) {
	h := newKBHarness()
	h.send(key.EditEvent{Range: key.Range{}, Text: "a"})
	h.send(key.EditEvent{Range: key.Range{Start: 1, End: 1}, Text: "bc"})
	check(t, "text", h.text, []string{"a", "bc"})
	check(t, "keys", h.keys, nil)
}

func TestKeyboardInputComposition(t *testing.T) {
	h := newKBHarness()
	h.send(key.EditEvent{Text: "ni"}, key.CompositionEvent{Start: 0, End: 2})
	if len(h.text) != 0 {
		t.Fatalf("text %q while composing", h.text)
	}
	if got := h.k.Composition(); got != "ni" {
		t.Errorf("Composition() = %q, want %q", got, "ni")
	}
	h.send(key.EditEvent{Range: key.Range{End: 2}, Text: "你"}, key.CompositionEvent{Start: -1, End: -1})
	check(t, "text", h.text, []string{"你"})
	if got := h.k.Composition(); got != "" {
		t.Errorf("Composition() after commit = %q", got)
	}
}

// A correction of text already reported comes as Backspace presses and
// the new text.
func TestKeyboardInputCorrection(t *testing.T) {
	h := newKBHarness()
	h.send(key.EditEvent{Text: "teh"})
	h.send(key.EditEvent{Range: key.Range{Start: 0, End: 3}, Text: "the"})
	check(t, "text", h.text, []string{"teh", "the"})
	check(t, "keys", h.keys, []string{"⌫", "⌫", "⌫"})
}

func TestKeyboardInputKeys(t *testing.T) {
	h := newKBHarness()
	h.use = func(e key.Event) bool { return e.Name != key.NameEscape }
	for _, e := range []key.Event{
		{Name: key.NameTab},
		{Name: key.NameReturn},
		{Name: "C", Modifiers: key.ModCtrl},
		{Name: key.NameEscape},
		{Name: "H"},
		{Name: key.NameShift},
	} {
		e.State = key.Press
		h.send(e)
	}
	check(t, "keys", h.keys, []string{"Tab", "⏎", "CtrlC", "⎋", "Shift"})
	// Escape went unused, so on to the app; "H" types text, so it isn't
	// a shortcut while the input has focus.
	check(t, "app", h.app, []string{"⎋"})
	check(t, "watched", h.watched, []string{"Tab", "⏎", "C", "⎋", "H", "Shift"})
}

// A new line in the text is an Enter press, once.
func TestKeyboardInputNewline(t *testing.T) {
	h := newKBHarness()
	h.send(key.EditEvent{Text: "ls\n"})
	check(t, "text", h.text, []string{"ls"})
	check(t, "keys", h.keys, []string{"⏎"})
	h.reset()
	h.send(key.Event{Name: key.NameReturn, State: key.Press}, key.EditEvent{Range: key.Range{Start: 3, End: 3}, Text: "\n"})
	check(t, "keys", h.keys, []string{"⏎"})
}

// AltGr (Ctrl+Alt) typing "@" is text, not a Ctrl+Alt+Q shortcut.
func TestKeyboardInputAltGr(t *testing.T) {
	h := newKBHarness()
	h.send(key.Event{Name: "Q", Modifiers: key.ModCtrl | key.ModAlt, State: key.Press}, key.EditEvent{Text: "@"})
	check(t, "text", h.text, []string{"@"})
	check(t, "keys", h.keys, nil)
	h.send(key.Event{Name: "F", Modifiers: key.ModAlt, State: key.Press})
	check(t, "keys", h.keys, []string{"AltF"})
}

// Named keys only: the rest go on to the app.
func TestKeyboardInputNames(t *testing.T) {
	h := newKBHarness("W", key.NameUpArrow)
	h.send(key.Event{Name: "W", State: key.Press}, key.Event{Name: key.NameUpArrow, State: key.Press}, key.Event{Name: key.NameF5, State: key.Press})
	check(t, "keys", h.keys, []string{"W", "↑"})
	check(t, "app", h.app, []string{"F5"})
}

func TestKeyboardInputBlur(t *testing.T) {
	h := newKBHarness()
	blurred := false
	h.k.OnBlur(func() { blurred = true })
	h.k.Blur()
	h.frame()
	h.frame()
	if h.k.Focused() || !blurred {
		t.Errorf("Focused() = %v, OnBlur ran = %v after Blur", h.k.Focused(), blurred)
	}
}
