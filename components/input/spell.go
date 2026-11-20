package input

import (
	"strings"
	"sync"
	"time"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/spellcheck"
	"github.com/xmhrnx/giode/state"
	"github.com/xmhrnx/giode/styles/properties"
)

// spellState is a field's spell checking: the last result and the
// check waiting to run.
type spellState struct {
	on   bool
	lang string

	queued string // the text a check is due for
	due    time.Time

	mu      sync.Mutex
	checked string // the text results are for
	results []spellcheck.Misspelling
	running bool
}

// spellDebounce is how long typing must pause before a check.
const spellDebounce = 400 * time.Millisecond

// SpellCheck underlines misspelled words in red, with suggestions in
// the edit menu (right-click, or a long press on a touch screen). It
// uses the platform's spell checker (see package spellcheck) in the
// user's language, or the one SpellLanguage sets. It returns i for
// chaining.
func (i *Input) SpellCheck(on bool) *Input {
	i.spell.on = on
	return i
}

// SpellLanguage sets the language to check, a BCP 47 tag ("en-US",
// "fa"); the user's by default. It returns i for chaining.
func (i *Input) SpellLanguage(tag string) *Input {
	i.spell.lang = tag
	i.spell.queued = "\x00" // check again
	return i
}

// Misspellings are the misspelled words found in the current text.
func (i *Input) Misspellings() []spellcheck.Misspelling {
	s := &i.spell
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.checked != i.editor.Text() {
		return nil
	}
	return append([]spellcheck.Misspelling(nil), s.results...)
}

// updateSpell schedules a check after edits, and runs it when due.
func (i *Input) updateSpell(gtx layout.Context) {
	s := &i.spell
	if !s.on || i.editor.Mask != 0 || i.editor.ReadOnly {
		return
	}
	now := gtx.Now
	if now.IsZero() {
		now = time.Now()
	}
	text := i.editor.Text()
	if text != s.queued {
		s.queued = text
		s.due = now.Add(spellDebounce)
	}
	s.mu.Lock()
	done := s.checked == text
	running := s.running
	s.mu.Unlock()
	if done || running {
		return
	}
	if now.Before(s.due) {
		kit.RequestFrameAt(gtx, s.due)
		return
	}
	s.mu.Lock()
	s.running = true
	s.mu.Unlock()
	lang := s.lang
	go func() {
		res := spellcheck.Check(text, lang)
		s.mu.Lock()
		s.checked, s.results, s.running = text, res, false
		s.mu.Unlock()
		state.RequestRedraw()
	}()
}

// misspellingAt is the misspelled word around rune p.
func (i *Input) misspellingAt(p int) (spellcheck.Misspelling, bool) {
	for _, m := range i.Misspellings() {
		if p >= m.Start && p <= m.End {
			return m, true
		}
	}
	return spellcheck.Misspelling{}, false
}

// drawSpelling draws a red wave under each misspelled word, in the
// editor's coordinates.
func (i *Input) drawSpelling(gtx layout.Context) {
	ms := i.Misspellings()
	if len(ms) == 0 {
		return
	}
	col := properties.ResolveColor("theme.danger")
	amp := float32(gtx.Dp(1)) + 0.5
	period := float32(gtx.Dp(4))
	width := float32(max(gtx.Dp(1), 1))
	for _, m := range ms {
		for _, r := range i.editor.Regions(m.Start, m.End, nil) {
			b := r.Bounds
			if b.Empty() {
				continue
			}
			y := float32(b.Max.Y) - amp - 1
			var p clip.Path
			p.Begin(gtx.Ops)
			p.MoveTo(f32.Pt(float32(b.Min.X), y))
			up := true
			for x := float32(b.Min.X); x < float32(b.Max.X); x += period / 2 {
				dy := amp
				if up {
					dy = -amp
				}
				p.LineTo(f32.Pt(min(x+period/2, float32(b.Max.X)), y+dy))
				up = !up
			}
			st := clip.Stroke{Path: p.End(), Width: width}.Op().Push(gtx.Ops)
			paint.ColorOp{Color: col}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			st.Pop()
		}
	}
}

// spellActions are the edit menu's corrections for a misspelled word
// at the caret: "fix:" and a suggestion, then "learn".
func (m *editMenu) spellActions(in *Input) []string {
	if !in.spell.on {
		return nil
	}
	a, b := in.editor.Selection()
	w, ok := in.misspellingAt(min(a, b))
	if !ok || max(a, b) > w.End {
		return nil
	}
	if m.suggWord != w.Word {
		m.suggWord = w.Word
		m.sugg = spellcheck.Suggest(w.Word, in.spell.lang)
	}
	var out []string
	for k, s := range m.sugg {
		if k == 3 {
			break
		}
		out = append(out, "fix:"+s)
	}
	return append(out, "learn")
}

// fix replaces the misspelled word at the caret with s.
func (m *editMenu) fix(in *Input, s string) {
	a, _ := in.editor.Selection()
	w, ok := in.misspellingAt(a)
	if !ok {
		return
	}
	in.editor.SetCaret(w.End, w.Start)
	in.editor.Insert(s)
	if in.onChange != nil {
		in.onChange(in.editor.Text())
	}
}

// learn adds the misspelled word at the caret to the dictionary.
func (m *editMenu) learn(in *Input) {
	a, _ := in.editor.Selection()
	if w, ok := in.misspellingAt(a); ok {
		spellcheck.Learn(w.Word)
		s := &in.spell
		s.mu.Lock()
		kept := s.results[:0]
		for _, r := range s.results {
			if !strings.EqualFold(r.Word, w.Word) {
				kept = append(kept, r)
			}
		}
		s.results = kept
		s.mu.Unlock()
	}
}
