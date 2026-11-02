package ui

import (
	"image"
	"testing"

	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/mehran9675/giode/fonts"
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
