package input

import (
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/widget"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Keyboard selects the on-screen keyboard a field asks for on touch
// devices. Desktop platforms ignore it.
type Keyboard uint8

const (
	// KeyboardText is the plain text keyboard, the default.
	KeyboardText Keyboard = iota
	// KeyboardNumeric shows digits, "." and ",".
	KeyboardNumeric
	// KeyboardEmail adds shortcuts such as "@" and ".com".
	KeyboardEmail
	// KeyboardURL adds shortcuts such as "/" and ".com".
	KeyboardURL
	// KeyboardPhone shows digits, "#" and "*".
	KeyboardPhone
)

// hint maps k to Gio's keyboard hint.
func (k Keyboard) hint() key.InputHint {
	switch k {
	case KeyboardNumeric:
		return key.HintNumeric
	case KeyboardEmail:
		return key.HintEmail
	case KeyboardURL:
		return key.HintURL
	case KeyboardPhone:
		return key.HintTelephone
	}
	return key.HintAny
}

// SubmitMode selects which key presses submit a field.
type SubmitMode int

const (
	// SubmitDefault submits on Enter in a single-line field; a
	// multiline field inserts a newline and never submits.
	SubmitDefault SubmitMode = iota
	// SubmitEnter submits on Enter, also in a multiline field.
	SubmitEnter
	// SubmitEnterUnlessShift submits on Enter and inserts a newline on
	// Shift+Enter, as chat composers do. On a touch screen the
	// on-screen keyboard's Enter key inserts a newline instead, since
	// phone apps send with a button.
	SubmitEnterUnlessShift
	// SubmitNever never submits; Enter inserts a newline in a
	// multiline field.
	SubmitNever
)

// Input is a stateful text field, single-line by default. Create it
// once with New and lay it out every frame.
type Input struct {
	pass         func(gtx layout.Context) bool
	editor       widget.Editor
	placeholder  string
	label        string
	keyboard     Keyboard
	st           styles.Styles
	onSubmit     func(text string)
	onChange     func(text string)
	requestFocus bool
	focused      bool
	submitOn     SubmitMode
	minLines     int
	maxLines     int
	onFocus      func()
	onBlur       func()
	// tap is the event tag for presses anywhere on the field, padding
	// and border included, which focus the editor.
	tap int
	// menu is the long-press (and right-click) editing menu.
	menu editMenu
}

// New returns an Input with the given placeholder. The styles
// argument is optional.
func New(placeholder string, st ...styles.Styles) *Input {
	i := &Input{placeholder: placeholder}
	if len(st) > 0 {
		i.st = st[0]
	}
	i.editor.SingleLine = true
	i.editor.Submit = true
	return i
}

// Text returns the current content.
func (i *Input) Text() string {
	return i.editor.Text()
}

// SetText replaces the content without invoking OnChange.
func (i *Input) SetText(s string) {
	i.editor.SetText(s)
}

// Submit sets the handler invoked when Enter is pressed in a
// single-line field. It returns i for chaining.
func (i *Input) Submit(fn func(text string)) *Input {
	i.onSubmit = fn
	return i
}

// OnChange sets the handler invoked after every edit. It returns i for
// chaining.
func (i *Input) OnChange(fn func(text string)) *Input {
	i.onChange = fn
	return i
}

// Password masks the content with bullets and asks touch devices for
// the password keyboard. It returns i for chaining.
func (i *Input) Password(on bool) *Input {
	if on {
		i.editor.Mask = '•'
	} else {
		i.editor.Mask = 0
	}
	i.updateHint()
	return i
}

// Keyboard selects the on-screen keyboard shown while the field is
// focused on a touch device; Password overrides it with the password
// keyboard. It returns i for chaining.
func (i *Input) Keyboard(k Keyboard) *Input {
	i.keyboard = k
	i.updateHint()
	return i
}

func (i *Input) updateHint() {
	if i.editor.Mask != 0 {
		i.editor.InputHint = key.HintPassword
	} else {
		i.editor.InputHint = i.keyboard.hint()
	}
}

// Multiline lets the field hold several lines; Enter inserts a newline
// instead of submitting (see SubmitOn). It returns i for chaining.
func (i *Input) Multiline(on bool) *Input {
	i.editor.SingleLine = !on
	i.editor.Submit = !on
	return i
}

// SubmitOn selects which key presses call the Submit handler; see
// SubmitMode. It returns i for chaining.
func (i *Input) SubmitOn(m SubmitMode) *Input {
	i.submitOn = m
	return i
}

// Lines makes a multiline field grow with its content from min to max
// lines, then scroll; 0 for max means no limit. It also turns
// Multiline on. It returns i for chaining.
func (i *Input) Lines(min, max int) *Input {
	i.minLines, i.maxLines = min, max
	return i.Multiline(true)
}

// OnFocus sets a handler called when the field gains keyboard focus.
// It returns i for chaining.
func (i *Input) OnFocus(fn func()) *Input {
	i.onFocus = fn
	return i
}

// OnBlur sets a handler called when the field loses keyboard focus.
// It returns i for chaining.
func (i *Input) OnBlur(fn func()) *Input {
	i.onBlur = fn
	return i
}

// Placeholder replaces the text shown while the field is empty. It
// returns i for chaining.
func (i *Input) Placeholder(s string) *Input {
	i.placeholder = s
	return i
}

// Styles replaces the field's styles, from the next frame, keeping its
// text and focus. It returns i for chaining.
func (i *Input) Styles(st styles.Styles) *Input {
	i.st = st
	return i
}

// SelectAll selects the whole content.
func (i *Input) SelectAll() {
	i.editor.SetCaret(i.editor.Len(), 0)
}

// InsertText replaces the selection (or inserts at the caret) with s
// and invokes OnChange, as typing would.
func (i *Input) InsertText(s string) {
	i.editor.Insert(s)
	if i.onChange != nil {
		i.onChange(i.editor.Text())
	}
}

// applySubmit sets the editor's submit behavior for this frame.
func (i *Input) applySubmit() {
	switch i.submitOn {
	case SubmitEnter:
		i.editor.Submit = true
	case SubmitEnterUnlessShift:
		i.editor.Submit = !elements.TouchInput()
	case SubmitNever:
		i.editor.Submit = false
	default:
		i.editor.Submit = i.editor.SingleLine
	}
}

// ReadOnly makes the content selectable but not editable. It returns i
// for chaining.
func (i *Input) ReadOnly(on bool) *Input {
	i.editor.ReadOnly = on
	return i
}

// MaxLength limits the content to n runes; 0 removes the limit. It
// returns i for chaining.
func (i *Input) MaxLength(n int) *Input {
	i.editor.MaxLen = n
	return i
}

// Label sets the accessible name reported to automation (the semantic
// tree, the MCP server); the placeholder is used when unset. It
// returns i for chaining.
func (i *Input) Label(label string) *Input {
	i.label = label
	return i
}

// ChangeText replaces the content and invokes OnChange, as typing
// would.
func (i *Input) ChangeText(s string) {
	i.editor.SetText(s)
	if i.onChange != nil {
		i.onChange(s)
	}
}

// SubmitText invokes the Submit handler with the current content, as
// pressing Enter would.
func (i *Input) SubmitText() {
	if i.onSubmit != nil {
		i.onSubmit(i.editor.Text())
	}
}

// Focus requests keyboard focus.
func (i *Input) Focus() {
	i.requestFocus = true
}

// Focused reports whether the field had keyboard focus in the last
// frame.
func (i *Input) Focused() bool {
	return i.focused
}

// update handles taps, editing and focus, reporting whether anything
// happened that changes how the field looks.
func (i *Input) update(gtx layout.Context) (changed bool) {
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &i.tap, Kinds: pointer.Press})
		if !ok {
			break
		}
		if e, ok := ev.(pointer.Event); ok && (e.Source == pointer.Touch || e.Buttons.Contain(pointer.ButtonPrimary)) {
			changed = true
			gtx.Execute(key.FocusCmd{Tag: &i.editor})
			if !i.editor.ReadOnly {
				gtx.Execute(key.SoftKeyboardCmd{Show: true})
			}
		}
	}
	if i.menu.update(gtx, i) {
		changed = true
	}
	i.applySubmit()
	for {
		ev, ok := i.editor.Update(gtx)
		if !ok {
			break
		}
		changed = true
		switch e := ev.(type) {
		case widget.SubmitEvent:
			if i.onSubmit != nil {
				i.onSubmit(e.Text)
				kit.RequestFrame(gtx)
			}
		case widget.ChangeEvent:
			if i.onChange != nil {
				i.onChange(i.editor.Text())
				kit.RequestFrame(gtx)
			}
		}
	}
	focused := gtx.Source.Focused(&i.editor)
	if focused != i.focused {
		changed = true
		if focused && i.onFocus != nil {
			i.onFocus()
		}
		if !focused && i.onBlur != nil {
			i.onBlur()
		}
	}
	i.focused = focused
	return changed
}

// Layout lays out and updates the field.
func (i *Input) Layout(gtx layout.Context) layout.Dimensions {
	if i.requestFocus {
		i.requestFocus = false
		gtx.Execute(key.FocusCmd{Tag: &i.editor})
	}
	i.update(gtx)
	if i.pass == nil {
		i.pass = i.update
	}
	kit.HandleInput(i.pass)
	if i.focused {
		// A focused field receives key events and blinks its caret, so
		// cached ancestors must keep laying it out.
		kit.MarkLive()
	}

	st := styles.Merge(styles.Styles{
		Padding:      properties.SymmetricInset(12, 8),
		BorderWidth:  1,
		BorderRadius: 6,
		BorderColor:  "theme.border",
		Color:        "theme.text",
	}, i.st)
	if st.Width == 0 {
		st.Width = -1
	}

	dims := elements.Box(st, &editorEl{input: i, st: st}).Layout(gtx)
	// A press anywhere on the field focuses it, not only on the text:
	// an empty or multiline field is mostly padding and blank space,
	// which on a touch screen otherwise makes it feel unresponsive.
	// The area passes presses through so the editor beneath still
	// places the caret and starts selections.
	pass := pointer.PassOp{}.Push(gtx.Ops)
	area := clip.Rect{Max: dims.Size}.Push(gtx.Ops)
	event.Op(gtx.Ops, &i.tap)
	area.Pop()
	pass.Pop()
	label := i.label
	if label == "" {
		label = i.placeholder
	}
	value := i.editor.Text()
	if i.editor.Mask != 0 {
		value = ""
	}
	elements.RecordSemantic(gtx, elements.SemanticNode{Kind: elements.SemanticInput, Label: label, Value: value, Target: i}, dims.Size)
	return dims
}
