package input

import (
	"image"
	"strings"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/state"
)

// KeyboardInput takes the keyboard for content drawn by the app itself
// - a terminal, a game, a canvas, a custom editor: it gets focus, the
// text typed while it has it and the other keys, and asks for the
// on-screen keyboard on phones, while content draws what that changes.
// Create it once with NewKeyboardInput and lay it out every frame.
//
// Typed text arrives at OnText as the platform composes it: in the
// keyboard layout in effect, with Shift and Caps Lock applied, dead
// keys and compose sequences combined ("´" then "e" gives "é"), and
// input-method text (Chinese, Japanese, Korean; a phone keyboard's
// swipes and suggestions) once it's committed. The text being composed
// meanwhile is Composition; Caret says where the app draws its caret,
// so the input method's candidate window opens there.
//
// The other keys - arrows, Enter, Tab, Backspace, Escape, function
// keys, and any key with Ctrl, Alt or Cmd held - go to OnKey. Enter,
// Space and Tab do nothing of their own here: Enter and Tab go to OnKey
// (Tab doesn't move the focus while it has it), Space is typed text.
type KeyboardInput struct {
	content elements.Element

	onText  func(string)
	onKey   func(key.Event) bool
	names   []key.Name
	onFocus func()
	onBlur  func()

	keyboard    Keyboard
	keyboardSet bool
	autocorrect bool
	label       string

	requestFocus, requestBlur bool
	focused                   bool

	// buf is the text the input method sees: what was typed since it
	// was last cleared. Its first reported runes went to OnText.
	buf      []rune
	sel      key.Range
	comp     key.Range
	reported int
	// sent is the snippet and selection last told to the platform.
	sentSnippet string
	sentSel     key.Range
	sentCaret   image.Rectangle
	sentComp    bool
	// caret is where the app draws its caret, relative to the element.
	caret    image.Rectangle
	caretSet bool

	// delivered is set when a handler got text or a key this frame: what
	// it changed may be laid out already, so another frame follows.
	delivered bool

	pass   func(gtx layout.Context) bool
	origin image.Point
	size   image.Point
}

// NewKeyboardInput returns a KeyboardInput showing content.
func NewKeyboardInput(content elements.Element) *KeyboardInput {
	return &KeyboardInput{content: content, comp: key.Range{Start: -1, End: -1}}
}

// Content replaces what it shows. It returns k for chaining.
func (k *KeyboardInput) Content(content elements.Element) *KeyboardInput {
	k.content = content
	return k
}

// OnText has fn take the text typed while it has focus, as committed:
// usually a character at a time, a word or more from an input method or
// a paste through the keyboard. A new line in it ("\n", sent by some
// phone keyboards) goes to OnKey as an Enter press instead, and a
// correction of text fn already got (a phone keyboard's autocorrect)
// arrives as Backspace presses at OnKey and the corrected text here. It
// returns k for chaining.
func (k *KeyboardInput) OnText(fn func(text string)) *KeyboardInput {
	k.onText = fn
	return k
}

// OnKey has fn take the keys that don't type text, pressed and released
// while it has focus (check e.State): arrows, Enter, Tab and Shift+Tab,
// Backspace, Delete, Escape, Home, End, Page Up and Down, function keys,
// the modifier keys themselves, and any key with Ctrl, Alt, Cmd or
// Super held (Ctrl+C included); Android's Back goes on to HandleBack.
// With names it takes only those keys, with any modifiers held, and
// the rest go on to the app's OnKey shortcuts: a game names "W", "A",
// "S" and "D" (as key.Event names them: capitals, key.NameSpace) to get
// their presses and releases, whose text also arrives at OnText.
// fn returns whether it used the key; one it didn't use goes on to the
// app's OnKey shortcuts too. It returns k for chaining.
func (k *KeyboardInput) OnKey(fn func(e key.Event) bool, names ...key.Name) *KeyboardInput {
	k.onKey, k.names = fn, names
	return k
}

// OnFocus has fn run when it gains keyboard focus. It returns k for
// chaining.
func (k *KeyboardInput) OnFocus(fn func()) *KeyboardInput {
	k.onFocus = fn
	return k
}

// OnBlur has fn run when it loses keyboard focus. It returns k for
// chaining.
func (k *KeyboardInput) OnBlur(fn func()) *KeyboardInput {
	k.onBlur = fn
	return k
}

// Keyboard has it ask for an on-screen keyboard of kind kb on phones and
// tablets while it has focus; without it, none shows. Desktop platforms
// ignore it. It returns k for chaining.
func (k *KeyboardInput) Keyboard(kb Keyboard) *KeyboardInput {
	k.keyboard, k.keyboardSet = kb, true
	return k
}

// Autocorrect lets a phone keyboard correct, capitalize and suggest
// words; off by default, as a terminal or a game wants the keys as
// typed. Corrections reach OnKey as Backspace presses and OnText as the
// new text. It returns k for chaining.
func (k *KeyboardInput) Autocorrect(on bool) *KeyboardInput {
	k.autocorrect = on
	return k
}

// Label sets the accessible name reported to screen readers and
// automation. It returns k for chaining.
func (k *KeyboardInput) Label(label string) *KeyboardInput {
	k.label = label
	return k
}

// Caret tells the input method where the app draws its caret: r in
// pixels, relative to the element's top-left corner. The candidate
// window of a Chinese or Japanese input method opens at it, and a phone
// keyboard keeps it in view. Call it as the caret moves, also while
// laying out content. It returns k for chaining.
func (k *KeyboardInput) Caret(r image.Rectangle) *KeyboardInput {
	k.caret, k.caretSet = r.Canon(), true
	return k
}

// Composition returns the text an input method is composing, not yet
// committed to OnText, for the app to draw at its caret; "" when none.
func (k *KeyboardInput) Composition() string {
	if k.comp.Start < 0 {
		return ""
	}
	s, e := k.clamp(k.comp)
	return string(k.buf[max(s, k.reported):max(e, k.reported)])
}

// Focus gives it keyboard focus in the next frame. Call it on the UI
// goroutine, also right after showing it for the first time.
func (k *KeyboardInput) Focus() {
	k.requestFocus, k.requestBlur = true, false
	state.RequestRedraw()
}

// Blur takes keyboard focus away from it, if it has it, in the next
// frame, and closes the on-screen keyboard.
func (k *KeyboardInput) Blur() {
	k.requestBlur, k.requestFocus = true, false
	state.RequestRedraw()
}

// Focused reports whether it had keyboard focus in the last frame.
func (k *KeyboardInput) Focused() bool {
	return k.focused
}

// KeepsKeyboard reports whether a tap at p, in window coordinates,
// lands on it: a tap elsewhere closes the on-screen keyboard.
func (k *KeyboardInput) KeepsKeyboard(p image.Point) bool {
	return p.In(image.Rectangle{Min: k.origin, Max: k.origin.Add(k.size)})
}

// Layout lays out content and handles the keyboard.
func (k *KeyboardInput) Layout(gtx layout.Context) layout.Dimensions {
	k.update(gtx)
	if k.pass == nil {
		k.pass = k.update
	}
	kit.HandleInput(k.pass)
	elements.TrackPosition(&k.origin)

	var dims layout.Dimensions
	if k.content != nil {
		dims = k.content.Layout(gtx)
	} else {
		dims = layout.Dimensions{Size: gtx.Constraints.Min}
	}
	k.size = dims.Size

	area := clip.Rect{Max: dims.Size}.Push(gtx.Ops)
	pass := pointer.PassOp{}.Push(gtx.Ops)
	event.Op(gtx.Ops, k)
	pass.Pop()
	key.InputHintOp{Tag: k, Hint: k.hint()}.Add(gtx.Ops)
	area.Pop()
	kit.NoteFocusable(k)

	if k.focused {
		kit.MarkLive()
		// Keys that type into it aren't app shortcuts.
		elements.MarkTyping()
		if kit.SetReturnKeyDone != nil {
			kit.SetReturnKeyDone(false)
		}
		if gtx.Source.Enabled() {
			if k.keyboardSet {
				elements.SetKeyboardOwner(k)
			}
			k.sync(gtx)
		}
	}
	elements.RecordSemantic(gtx, elements.SemanticNode{
		Kind: elements.SemanticInput, Label: k.label, Focused: k.focused, Target: k,
	}, dims.Size)
	return dims
}

func (k *KeyboardInput) hint() key.InputHint {
	h := k.keyboard.hint()
	if !k.autocorrect {
		h |= hintPlain
	}
	return h
}

// keyMods are the modifiers a key may be pressed with.
const keyMods = key.ModCtrl | key.ModShift | key.ModAlt | key.ModSuper | key.ModCommand

// commandKeys are the keys that don't type text, which it takes by
// default. Back isn't one: it goes to the app's HandleBack.
var commandKeys = []key.Name{
	key.NameLeftArrow, key.NameRightArrow, key.NameUpArrow, key.NameDownArrow,
	key.NameReturn, key.NameEnter, key.NameEscape, key.NameTab,
	key.NameHome, key.NameEnd, key.NamePageUp, key.NamePageDown,
	key.NameDeleteBackward, key.NameDeleteForward,
	key.NameCtrl, key.NameShift, key.NameAlt, key.NameSuper, key.NameCommand,
	key.NameF1, key.NameF2, key.NameF3, key.NameF4, key.NameF5, key.NameF6,
	key.NameF7, key.NameF8, key.NameF9, key.NameF10, key.NameF11, key.NameF12,
}

// filters are the events it takes.
func (k *KeyboardInput) filters() []event.Filter {
	fs := []event.Filter{
		key.FocusFilter{Target: k},
		pointer.Filter{Target: k, Kinds: pointer.Press},
	}
	names := k.names
	if len(names) == 0 {
		names = commandKeys
		// Any key with a command modifier held: Ctrl+C, Alt+F.
		for _, m := range []key.Modifiers{key.ModCtrl, key.ModAlt, key.ModSuper, key.ModCommand} {
			fs = append(fs, key.Filter{Focus: k, Required: m, Optional: keyMods})
		}
	}
	for _, n := range names {
		fs = append(fs, key.Filter{Focus: k, Name: n, Optional: keyMods})
	}
	return fs
}

// typesWithAlt reports whether e may also type a character: Option on
// a Mac and AltGr (Ctrl+Alt on Windows) type ones like "@" and "é".
func typesWithAlt(e key.Event) bool {
	return e.State == key.Press && e.Modifiers.Contain(key.ModAlt) &&
		!e.Modifiers.Contain(key.ModCommand) && !e.Modifiers.Contain(key.ModSuper)
}

// update handles focus, pointer presses, keys and text, reporting
// whether its focus changed.
func (k *KeyboardInput) update(gtx layout.Context) (changed bool) {
	if gtx.Source.Enabled() {
		if k.requestFocus {
			k.requestFocus = false
			gtx.Execute(key.FocusCmd{Tag: k})
			k.showKeyboard(gtx, true)
			changed = true
		}
		if k.requestBlur {
			k.requestBlur = false
			if gtx.Source.Focused(k) {
				gtx.Execute(key.FocusCmd{})
				k.showKeyboard(gtx, false)
				changed = true
			}
		}
	}
	enterPressed := false
	// Alt presses wait for the end of the batch: one that typed a
	// character (its text came after it) isn't a command.
	var altKeys []key.Event
	typedAfterAlt := false
	for {
		ev, ok := gtx.Event(k.filters()...)
		if !ok {
			break
		}
		switch e := ev.(type) {
		case pointer.Event:
			if e.Source == pointer.Touch || e.Buttons.Contain(pointer.ButtonPrimary) {
				gtx.Execute(key.FocusCmd{Tag: k})
				k.showKeyboard(gtx, true)
				changed = true
			}
		case key.FocusEvent:
			if e.Focus == k.focused {
				break
			}
			k.focused = e.Focus
			k.reset()
			changed = true
			if e.Focus && k.onFocus != nil {
				k.onFocus()
			} else if !e.Focus && k.onBlur != nil {
				k.onBlur()
			}
		case key.Event:
			if !gtx.Focused(k) {
				break
			}
			elements.SeenKey(e)
			enter := e.State == key.Press && (e.Name == key.NameReturn || e.Name == key.NameEnter)
			if enter {
				enterPressed = true
			}
			if typesWithAlt(e) && len(e.Name) == 1 {
				altKeys = append(altKeys, e)
				break
			}
			// The text typed before the key goes first: "foo(" then
			// Enter in one frame mustn't reach the app as Enter, then
			// "foo(". A new line in it is this Enter, sent as text too.
			if k.comp.Start < 0 {
				k.commit(enter)
			}
			k.key(e)
		case key.EditEvent:
			if len(altKeys) > 0 && e.Text != "" {
				typedAfterAlt = true
			}
			k.edit(e)
		case key.SelectionEvent:
			k.sel = key.Range(e)
		case key.CompositionEvent:
			k.comp = key.Range(e)
			if len(altKeys) > 0 {
				// A dead key (Option+E) started composing.
				typedAfterAlt = true
			}
		case key.SnippetEvent:
			// The whole buffer is the snippet; sync sends it.
		}
	}
	k.commit(enterPressed)
	if !typedAfterAlt {
		for _, e := range altKeys {
			k.key(e)
		}
	}
	if k.delivered {
		k.delivered = false
		kit.RequestFrame(gtx)
		changed = true
	}
	return changed
}

// key hands a key to OnKey, or on to the app's shortcuts.
func (k *KeyboardInput) key(e key.Event) {
	k.delivered = true
	if k.onKey != nil && k.onKey(e) {
		return
	}
	elements.ForwardKey(e)
}

// showKeyboard opens or closes the on-screen keyboard, when it asks for
// one.
func (k *KeyboardInput) showKeyboard(gtx layout.Context, show bool) {
	if k.keyboardSet || !show {
		gtx.Execute(key.SoftKeyboardCmd{Show: show})
	}
}

func (k *KeyboardInput) clamp(r key.Range) (start, end int) {
	s, e := r.Start, r.End
	if s > e {
		s, e = e, s
	}
	return max(0, min(s, len(k.buf))), max(0, min(e, len(k.buf)))
}

// edit applies an input method's edit to the buffer. Text OnText
// already got that the edit takes back goes out as Backspace presses.
func (k *KeyboardInput) edit(e key.EditEvent) {
	s, end := k.clamp(e.Range)
	if s < k.reported {
		for range k.reported - s {
			k.key(key.Event{Name: key.NameDeleteBackward, State: key.Press})
			k.key(key.Event{Name: key.NameDeleteBackward, State: key.Release})
		}
		k.reported = s
	}
	ins := []rune(e.Text)
	buf := make([]rune, 0, len(k.buf)-(end-s)+len(ins))
	buf = append(buf, k.buf[:s]...)
	buf = append(buf, ins...)
	buf = append(buf, k.buf[end:]...)
	k.buf = buf
	c := s + len(ins)
	k.sel = key.Range{Start: c, End: c}
}

// commit hands the text no longer being composed to OnText, with its
// new lines as Enter presses (unless an Enter key press came with it).
func (k *KeyboardInput) commit(enterPressed bool) {
	end := len(k.buf)
	if k.comp.Start >= 0 {
		end, _ = k.clamp(k.comp)
	}
	if end <= k.reported {
		return
	}
	text := string(k.buf[k.reported:end])
	k.reported = end
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	for i, line := range strings.Split(text, "\n") {
		if i > 0 && !enterPressed {
			k.key(key.Event{Name: key.NameReturn, State: key.Press})
			k.key(key.Event{Name: key.NameReturn, State: key.Release})
		}
		if line != "" && k.onText != nil {
			k.delivered = true
			k.onText(line)
		}
	}
}

// keepRunes is how much committed text the buffer keeps for the input
// method's context (capitals after a full stop, corrections) before
// it's cleared: clearing makes a phone keyboard restart.
const keepRunes = 48

// reset clears the buffer.
func (k *KeyboardInput) reset() {
	k.buf, k.reported = nil, 0
	k.sel = key.Range{}
	k.comp = key.Range{Start: -1, End: -1}
}

// sync tells the platform the buffer, the selection and where the caret
// is, when they changed.
func (k *KeyboardInput) sync(gtx layout.Context) {
	if k.comp.Start < 0 && k.reported == len(k.buf) && len(k.buf) > keepRunes {
		k.reset()
	}
	caret := k.caret
	if !k.caretSet {
		caret = image.Rect(0, 0, 1, k.size.Y)
	}
	snippet := string(k.buf)
	composing := k.comp.Start >= 0
	if snippet != k.sentSnippet {
		gtx.Execute(key.SnippetCmd{Tag: k, Snippet: key.Snippet{Range: key.Range{End: len(k.buf)}, Text: snippet}})
		k.sentSnippet = snippet
	}
	if k.sel != k.sentSel || caret != k.sentCaret || composing != k.sentComp {
		cmd := key.SelectionCmd{
			Tag:   k,
			Range: k.sel,
			Caret: key.Caret{
				Pos:    f32.Pt(float32(caret.Min.X), float32(caret.Max.Y)),
				Ascent: float32(caret.Dy()),
			},
		}
		if composing {
			cmd.CompositionBounds = caret
		}
		gtx.Execute(cmd)
		k.sentSel, k.sentCaret, k.sentComp = k.sel, caret, composing
	}
}
