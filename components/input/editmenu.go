package input

import (
	"image"
	"time"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
)

// editMenu is a field's touch editing: a long-press places the caret or
// selects the word under the finger and opens a menu - Cut, Copy,
// Paste, Select all - with handles to adjust the selection. A
// right-click opens the same menu with a mouse.
type editMenu struct {
	// tag receives presses over the editor, passed through to it.
	tag int

	pressing bool
	pressAt  time.Time
	pressPos f32.Point
	pressID  pointer.ID

	// open is set while the menu shows; handles while the selection
	// handles do (touch only).
	open, handles bool
	// hadFocus is set once the field had focus while the menu showed,
	// so losing it closes the menu.
	hadFocus bool
	// origin is the editor's window position, exact from the last
	// frame.
	origin image.Point

	// outside observes presses anywhere, to close the menu.
	outside int
	buttons map[string]*elements.Pressable
	action  string

	start, end handle
}

// longPress is how long a finger rests on a field to open its menu.
const longPress = 500 * time.Millisecond

// update processes the menu's events and reports whether the field's
// look changed.
func (m *editMenu) update(gtx layout.Context, in *Input) bool {
	changed := false
	ed := &in.editor
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &m.tag, Kinds: pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		switch {
		case e.Kind == pointer.Cancel:
			// The editor or a list took the pointer to scroll. Gio's
			// cancel carries no source.
			m.pressing = false
		case e.Kind == pointer.Press && e.Source == pointer.Touch:
			if m.open {
				// A tap in the field closes the menu; the editor
				// places the caret.
				m.close()
				changed = true
			}
			m.pressing, m.pressAt, m.pressPos, m.pressID = true, gtx.Now, e.Position, e.PointerID
			kit.RequestFrameAt(gtx, gtx.Now.Add(longPress))
		case e.Kind == pointer.Press && e.Buttons.Contain(pointer.ButtonSecondary):
			// The desktop context menu: keep a selection the click is
			// in, otherwise place the caret.
			p := e.Position.Round()
			c := caretAt(ed, p)
			if s, t := ed.Selection(); c < min(s, t) || c > max(s, t) || s == t {
				ed.SetCaret(c, c)
			}
			m.focus(gtx, in)
			m.open, m.handles = true, false
			changed = true
		case e.Kind == pointer.Drag && m.pressing:
			d := e.Position.Sub(m.pressPos)
			if slop := float32(gtx.Dp(8)); d.X*d.X+d.Y*d.Y > slop*slop {
				m.pressing = false
			}
		case e.Kind == pointer.Release:
			m.pressing = false
		}
	}
	if m.pressing && kit.Scrolling() {
		m.pressing = false
	}
	if m.pressing {
		if gtx.Now.Sub(m.pressAt) >= longPress {
			m.pressing = false
			m.longPress(gtx, in)
			changed = true
		} else {
			kit.RequestFrameAt(gtx, m.pressAt.Add(longPress))
		}
	}

	if m.open {
		// A press anywhere outside the menu and handles closes it.
		for {
			ev, ok := gtx.Event(pointer.Filter{Target: &m.outside, Kinds: pointer.Press})
			if !ok {
				break
			}
			if _, ok := ev.(pointer.Event); ok {
				m.close()
				changed = true
			}
		}
		if m.dragHandles(gtx, in) {
			changed = true
		}
		if m.action != "" {
			m.do(gtx, in, m.action)
			m.action = ""
			changed = true
		}
		if gtx.Source.Focused(ed) {
			m.hadFocus = true
		} else if m.hadFocus {
			m.close()
			changed = true
		}
	}
	return changed
}

// longPress selects the word under the finger, or places the caret
// where there's none, and opens the menu.
func (m *editMenu) longPress(gtx layout.Context, in *Input) {
	ed := &in.editor
	c := caretAt(ed, m.pressPos.Round())
	if ed.Mask == 0 {
		if s, e, ok := wordAt([]rune(ed.Text()), c); ok {
			ed.SetCaret(s, e)
		} else {
			ed.SetCaret(c, c)
		}
	} else {
		// A password field shows no words.
		ed.SetCaret(c, c)
	}
	m.focus(gtx, in)
	// Keep the finger's release from reaching the editor as a tap,
	// which would collapse the selection.
	gtx.Execute(pointer.GrabCmd{Tag: &m.tag, ID: m.pressID})
	m.open, m.handles = true, true
}

func (m *editMenu) focus(gtx layout.Context, in *Input) {
	gtx.Execute(key.FocusCmd{Tag: &in.editor})
	if !in.editor.ReadOnly {
		gtx.Execute(key.SoftKeyboardCmd{Show: true})
	}
}

func (m *editMenu) close() {
	m.open, m.handles, m.hadFocus = false, false, false
}

// do runs a menu action.
func (m *editMenu) do(gtx layout.Context, in *Input, action string) {
	ed := &in.editor
	switch action {
	case "cut":
		if t := ed.SelectedText(); t != "" && ed.Mask == 0 {
			kit.WriteClipboard(gtx, t)
			ed.Delete(1)
		}
		m.close()
	case "copy":
		if t := ed.SelectedText(); t != "" && ed.Mask == 0 {
			kit.WriteClipboard(gtx, t)
		}
		s, e := ed.Selection()
		c := max(s, e)
		ed.SetCaret(c, c)
		m.close()
	case "paste":
		in.paste(gtx)
		m.close()
	case "select_all":
		ed.SetCaret(ed.Len(), 0)
		// Stay open for Cut or Copy.
	}
	kit.RequestFrame(gtx)
}

// actions returns the menu's actions for the field's state.
func (m *editMenu) actions(in *Input) []string {
	ed := &in.editor
	var out []string
	sel := ed.SelectionLen() > 0
	if sel && !ed.ReadOnly && ed.Mask == 0 {
		out = append(out, "cut")
	}
	if sel && ed.Mask == 0 {
		out = append(out, "copy")
	}
	if !ed.ReadOnly {
		out = append(out, "paste")
	}
	if n := ed.Len(); n > 0 && ed.SelectionLen() < n {
		out = append(out, "select_all")
	}
	return out
}
