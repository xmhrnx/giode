package toggle

import (
	"image"
	"strings"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/styles"
)

func TestSwitch(t *testing.T) {
	var changes []bool
	s := New("").OnChange(func(on bool) { changes = append(changes, on) })
	var r input.Router
	now := time.Unix(0, 0)
	var dims layout.Dimensions
	frame := func() {
		now = now.Add(50 * time.Millisecond)
		var ops op.Ops
		dims = s.Layout(layout.Context{Ops: &ops, Now: now, Source: r.Source(), Constraints: layout.Constraints{Max: image.Pt(400, 100)}})
		r.Frame(&ops)
	}
	frame()
	if dims.Size.X < 40 || dims.Size.Y < 20 {
		t.Fatalf("switch size %v", dims.Size)
	}
	p := f32.Pt(10, 10)
	r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: p, Buttons: pointer.ButtonPrimary},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: p})
	for i := 0; i < 6; i++ {
		frame()
	}
	if !s.Checked() || len(changes) != 1 || !changes[0] {
		t.Fatalf("after a tap: checked=%v changes=%v", s.Checked(), changes)
	}
	if s.pos != 1 {
		t.Errorf("knob at %v after the animation, want 1", s.pos)
	}
	s.Disabled(true).Toggle()
	if !s.Checked() {
		t.Error("a disabled switch toggled")
	}
}

func TestSwitch_LongLabelKeepsSwitchBeside(t *testing.T) {
	fonts.UseGofont()
	measure := func(e interface {
		Layout(layout.Context) layout.Dimensions
	}, w int) layout.Dimensions {
		var ops op.Ops
		return e.Layout(layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: image.Pt(w, 2000)}})
	}
	track := measure(New(""), 400)
	label := strings.Repeat("Remember this device and sign in without a code ", 4)
	got := measure(New(label), 300)
	if got.Size.X != 300 {
		t.Errorf("labelled switch is %dpx wide, want the full 300", got.Size.X)
	}
	// The label wraps beside the switch: the row is as tall as the label
	// at the width left of the switch, not that plus a line for the
	// switch.
	text := measure(elements.Text(label, styles.Styles{}), 300-track.Size.X-12)
	if got.Size.Y > text.Size.Y+4 {
		t.Errorf("row is %dpx tall, the label alone %dpx: the switch went onto its own line", got.Size.Y, text.Size.Y)
	}
}
