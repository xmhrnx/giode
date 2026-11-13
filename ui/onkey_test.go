package ui

import (
	"image"
	"strings"
	"testing"

	"gioui.org/io/event"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/fonts"
)

// OnKey gets only the keys nothing else took, though the view
// registers it before the field and the dialog.
func TestOnKeyGetsLeftovers(t *testing.T) {
	fonts.UseGofont()
	var r input.Router
	var got []string
	field := Input.Text("Name")
	dialogOpen := false
	dlg := Dialog(&dialogOpen, func() Element { return Text("Sure?") })
	frame := func() {
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Source: r.Source(), Constraints: layout.Exact(image.Pt(640, 480))}
		Frame(gtx, Raw(func(gtx layout.Context) layout.Dimensions {
			OnKey(func(e key.Event) bool {
				if e.State == key.Press {
					got = append(got, string(e.Name))
				}
				return true
			})
			field.Layout(gtx)
			dlg.Layout(gtx)
			return layout.Dimensions{Size: gtx.Constraints.Max}
		}))
		r.Frame(&ops)
	}
	press := func(name key.Name, mods key.Modifiers) {
		r.Queue(key.Event{Name: name, Modifiers: mods, State: key.Press}, key.Event{Name: name, Modifiers: mods, State: key.Release})
		frame()
		frame()
	}
	frame()

	// A shortcut with nothing focused.
	press("S", key.ModShortcut)
	// Escape with a dialog open closes the dialog, not the shortcut.
	dialogOpen = true
	frame()
	press(key.NameEscape, 0)
	if dialogOpen {
		t.Error("Escape didn't close the dialog")
	}
	// Escape with nothing open is the app's.
	press(key.NameEscape, 0)
	// Typing goes to the focused field.
	field.Focus()
	frame()
	r.Queue(key.EditEvent{Text: "xy"})
	frame()
	frame()
	press(key.NameDeleteBackward, 0)

	want := []string{"S", string(key.NameEscape)}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("OnKey got %q, want %q", got, want)
	}
	// The field took the typing and Backspace (its caret is at the
	// start, so Backspace deletes nothing); OnKey saw neither.
	if field.Text() != "xy" {
		t.Errorf("the field has %q, want the typed xy", field.Text())
	}
}

// Shortcuts fire while a text field has focus, unless it uses the key.
func TestShortcutsWithFocusedField(t *testing.T) {
	fonts.UseGofont()
	var r input.Router
	var ran []string
	field := Input.Text("Search")
	frame := func() {
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Source: r.Source(), Constraints: layout.Exact(image.Pt(640, 480))}
		Frame(gtx, Raw(func(gtx layout.Context) layout.Dimensions {
			Shortcuts(
				Shortcut("Mod+T", func() { ran = append(ran, "tab") }),
				Shortcut("F5", func() { ran = append(ran, "refresh") }),
				Shortcut("Mod+1", func() { ran = append(ran, "page1") }),
			)
			return field.Layout(gtx)
		}))
		r.Frame(&ops)
	}
	press := func(name key.Name, mods key.Modifiers) {
		r.Queue(key.Event{Name: name, Modifiers: mods, State: key.Press}, key.Event{Name: name, Modifiers: mods, State: key.Release})
		frame()
		frame()
	}
	frame()
	field.Focus()
	frame()
	r.Queue(key.EditEvent{Text: "abc"})
	frame()
	press("T", key.ModShortcut)
	press(key.NameF5, 0)
	press("1", key.ModShortcut)
	if len(ran) != 3 || ran[0] != "tab" || ran[1] != "refresh" || ran[2] != "page1" {
		t.Errorf("shortcuts ran %v with a field focused", ran)
	}
	if field.Text() != "abc" {
		t.Errorf("field %q: typing should stay in the field", field.Text())
	}
}

// Tab and Shift+Tab shortcuts fire, and the focus stays. A window hands
// Gio those keys as system events, which go only to a listener that
// names the key; otherwise the window moves the focus.
func TestShortcutsTab(t *testing.T) {
	fonts.UseGofont()
	var r input.Router
	a, b := Input.Text("A"), Input.Text("B")
	var fired []string
	frame := func() {
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Source: r.Source(), Constraints: layout.Exact(image.Pt(640, 480))}
		Frame(gtx, Raw(func(gtx layout.Context) layout.Dimensions {
			Shortcuts(
				Shortcut("Tab", func() { fired = append(fired, "Tab") }),
				Shortcut("Shift+Tab", func() { fired = append(fired, "Shift+Tab") }),
			)
			return Stack(Styles{Width: -1}, a, b).Layout(gtx)
		}))
		r.Frame(&ops)
	}
	// press sends Tab as the window does: as a system event when it
	// would move the focus (no modifiers, or Shift).
	press := func(mods key.Modifiers) {
		for _, st := range []key.State{key.Press, key.Release} {
			var e event.Event = key.Event{Name: key.NameTab, Modifiers: mods, State: st}
			if mods == 0 || mods == key.ModShift {
				e = input.SystemEvent{Event: e}
			}
			r.Queue(e)
		}
		frame()
		frame()
	}
	frame()
	a.Focus()
	frame()
	frame()
	press(0)
	press(key.ModShift)
	if len(fired) != 2 || fired[0] != "Tab" || fired[1] != "Shift+Tab" {
		t.Errorf("shortcuts fired %q, want Tab then Shift+Tab", fired)
	}
	if !a.Focused() || b.Focused() {
		t.Errorf("the focus moved (A %v, B %v)", a.Focused(), b.Focused())
	}
	// Ctrl+Tab isn't a shortcut here.
	press(key.ModCtrl)
	if len(fired) != 2 {
		t.Errorf("Ctrl+Tab fired a shortcut: %q", fired)
	}
}

// Keys that type into a focused field don't run shortcuts: Gio sends
// each as a key event too, which the field doesn't take. Once the field
// is blurred they do.
func TestShortcutsNotWhileTyping(t *testing.T) {
	fonts.UseGofont()
	var r input.Router
	var ran []string
	field := Input.Text("Name")
	frame := func() {
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Source: r.Source(), Constraints: layout.Exact(image.Pt(640, 480))}
		Frame(gtx, Raw(func(gtx layout.Context) layout.Dimensions {
			Shortcuts(
				Shortcut("H", func() { ran = append(ran, "H") }),
				Shortcut("Shift+H", func() { ran = append(ran, "Shift+H") }),
				Shortcut("Space", func() { ran = append(ran, "Space") }),
				Shortcut("Delete", func() { ran = append(ran, "Delete") }),
				Shortcut("Mod+S", func() { ran = append(ran, "Mod+S") }),
				Shortcut("Esc", func() { ran = append(ran, "Esc") }),
			)
			return field.Layout(gtx)
		}))
		r.Frame(&ops)
	}
	// typeKey sends a key as a window does while a field has focus: the
	// key event, then the text it types.
	typeKey := func(name key.Name, mods key.Modifiers, text string) {
		r.Queue(key.Event{Name: name, Modifiers: mods, State: key.Press})
		if text != "" {
			r.Queue(key.EditEvent{Text: text})
		}
		r.Queue(key.Event{Name: name, Modifiers: mods, State: key.Release})
		frame()
		frame()
	}
	frame()
	field.Focus()
	frame()
	frame()
	typeKey("H", 0, "h")
	typeKey("H", key.ModShift, "H")
	typeKey(key.NameSpace, 0, " ")
	typeKey(key.NameDeleteForward, 0, "")
	typeKey("S", key.ModShortcut, "")
	typeKey(key.NameEscape, 0, "")
	if got, want := strings.Join(ran, ","), "Mod+S,Esc"; got != want {
		t.Errorf("while typing, shortcuts ran %q, want %q", got, want)
	}
	if !strings.Contains(field.Text(), "h") || !strings.Contains(field.Text(), "H") {
		t.Errorf("field %q: the typed h and H should be in it", field.Text())
	}

	ran = nil
	field.Blur()
	frame()
	frame()
	if field.Focused() {
		t.Fatal("Blur left the field focused")
	}
	typeKey("H", 0, "")
	typeKey(key.NameSpace, 0, "")
	typeKey(key.NameDeleteForward, 0, "")
	if got, want := strings.Join(ran, ","), "H,Space,Delete"; got != want {
		t.Errorf("after Blur, shortcuts ran %q, want %q", got, want)
	}
}
