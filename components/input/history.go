package input

import (
	"time"
	"unicode"

	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
)

// Input keeps its own undo history, and does its own word movement:
// the editor's history has no API (no Undo item in the menu, no
// Ctrl+Y), and its word steps don't move the caret in Gio v0.10.

// snap is a state to go back to.
type snap struct {
	text string
	a, b int // selection
}

// history is the undo stack: states[idx] is the current one.
type history struct {
	states []snap
	idx    int
	last   time.Time
	kind   int // the last change: 1 typed a letter, 2 deleted one, 0 other
	pos    int // where it happened
	init   bool
}

const (
	maxHistory  = 200
	coalesceFor = time.Second
)

func runeLen(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}

// note records the field's state after a frame's edits; typing and
// deleting letter by letter merge into one step per word.
func (h *history) note(text string, a, b int, now time.Time) {
	if !h.init {
		h.init = true
		h.states, h.idx = []snap{{text, a, b}}, 0
		return
	}
	cur := &h.states[h.idx]
	if text == cur.text {
		cur.a, cur.b = a, b
		return
	}
	kind, pos := classify(cur.text, text)
	merge := kind != 0 && kind == h.kind && now.Sub(h.last) < coalesceFor &&
		(kind == 1 && pos == h.pos+1 || kind == 2 && pos == h.pos-1)
	h.kind, h.pos, h.last = kind, pos, now
	if kind == 1 {
		// A space ends its word's step: the next letter starts anew.
		if r := []rune(text); pos < len(r) && unicode.IsSpace(r[pos]) {
			h.kind = 0
		}
	}
	if merge {
		*cur = snap{text, a, b}
		return
	}
	h.states = append(h.states[:h.idx+1], snap{text, a, b})
	if len(h.states) > maxHistory {
		h.states = h.states[len(h.states)-maxHistory:]
	}
	h.idx = len(h.states) - 1
}

// classify tells a one-letter insertion (1) or deletion (2) from other
// edits, and where it happened.
func classify(old, new string) (kind, pos int) {
	o, n := []rune(old), []rune(new)
	if d := len(n) - len(o); d != 1 && d != -1 {
		return 0, 0
	}
	i := 0
	for i < len(o) && i < len(n) && o[i] == n[i] {
		i++
	}
	if len(n) > len(o) {
		if string(n[:i])+string(n[i+1:]) == old {
			return 1, i
		}
		return 0, 0
	}
	if string(o[:i])+string(o[i+1:]) == new {
		return 2, i
	}
	return 0, 0
}

// reset starts the history over from the current state.
func (h *history) reset() { h.init = false; h.kind = 0 }

func (h *history) canUndo() bool { return h.init && h.idx > 0 }
func (h *history) canRedo() bool { return h.init && h.idx < len(h.states)-1 }

// CanUndo and CanRedo report whether there's an edit to undo or redo.
func (i *Input) CanUndo() bool { return i.hist.canUndo() }
func (i *Input) CanRedo() bool { return i.hist.canRedo() }

// Undo undoes the last edit: a word typed, a paste, a cut.
func (i *Input) Undo() { i.step(-1) }

// Redo redoes the last undone edit.
func (i *Input) Redo() { i.step(1) }

func (i *Input) step(d int) {
	h := &i.hist
	if d < 0 && !h.canUndo() || d > 0 && !h.canRedo() || i.editor.ReadOnly {
		return
	}
	h.idx += d
	h.kind = 0
	s := h.states[h.idx]
	i.editor.SetText(s.text)
	i.editor.SetCaret(s.b, s.a)
	if i.onChange != nil {
		i.onChange(s.text)
	}
}

// isWord tells letters, marks and digits of any script from spaces and
// punctuation, for word steps: Persian and Arabic letters, joiners and
// combining marks included.
func isWord(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r) || r == '_' || r == '‌' || r == '‍'
}

// wordBoundary is the position a word step from p in direction d (1 or
// -1) reaches: past any spaces and punctuation, then the word.
func wordBoundary(rs []rune, p, d int) int {
	if d > 0 {
		for p < len(rs) && !isWord(rs[p]) {
			p++
		}
		for p < len(rs) && isWord(rs[p]) {
			p++
		}
		return p
	}
	for p > 0 && !isWord(rs[p-1]) {
		p--
	}
	for p > 0 && isWord(rs[p-1]) {
		p--
	}
	return p
}

// historyKeys handles undo, redo and word movement and deletion before
// the editor sees the keys.
func (i *Input) historyKeys(gtx layout.Context) bool {
	changed := false
	word := key.ModShortcutAlt
	filters := []event.Filter{
		key.Filter{Focus: &i.editor, Name: "Z", Required: key.ModShortcut, Optional: key.ModShift},
		key.Filter{Focus: &i.editor, Name: "Y", Required: key.ModShortcut},
		key.Filter{Focus: &i.editor, Name: key.NameLeftArrow, Required: word, Optional: key.ModShift},
		key.Filter{Focus: &i.editor, Name: key.NameRightArrow, Required: word, Optional: key.ModShift},
		key.Filter{Focus: &i.editor, Name: key.NameDeleteBackward, Required: word},
		key.Filter{Focus: &i.editor, Name: key.NameDeleteForward, Required: word},
	}
	for {
		ev, ok := gtx.Event(filters...)
		if !ok {
			break
		}
		e, ok := ev.(key.Event)
		if !ok || e.State != key.Press {
			continue
		}
		changed = true
		switch e.Name {
		case "Z":
			if e.Modifiers.Contain(key.ModShift) {
				i.Redo()
			} else {
				i.Undo()
			}
		case "Y":
			i.Redo()
		default:
			i.wordKey(e)
		}
	}
	return changed
}

func (i *Input) wordKey(e key.Event) {
	rs := []rune(i.editor.Text())
	caret, anchor := i.editor.Selection() // the caret moves; the anchor stays
	d := 1
	if e.Name == key.NameLeftArrow || e.Name == key.NameDeleteBackward {
		d = -1
	}
	// In right-to-left text the arrows go the other way, as the
	// editor's own do.
	if (e.Name == key.NameLeftArrow || e.Name == key.NameRightArrow) && i.rtl {
		d = -d
	}
	to := wordBoundary(rs, caret, d)
	switch e.Name {
	case key.NameLeftArrow, key.NameRightArrow:
		if e.Modifiers.Contain(key.ModShift) {
			i.editor.SetCaret(to, anchor)
		} else {
			i.editor.SetCaret(to, to)
		}
	default:
		if i.editor.ReadOnly {
			return
		}
		a, b := min(caret, anchor), max(caret, anchor)
		if a == b {
			a, b = min(caret, to), max(caret, to)
		}
		i.editor.SetCaret(b, a)
		i.editor.Insert("")
		if i.onChange != nil {
			i.onChange(i.editor.Text())
		}
	}
}
