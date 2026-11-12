package giode

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"

	"github.com/xmhrnx/giode/elements"
)

// fakeField is a focused text field covering field.
type fakeField struct{ field image.Rectangle }

func (f *fakeField) KeepsKeyboard(p image.Point) bool { return p.In(f.field) }

// A tap outside the focused field takes its focus (closing the
// keyboard); a tap on it, or a drag (a scroll), doesn't.
func TestTapOutsideDismissesKeyboard(t *testing.T) {
	for _, tc := range []struct {
		name          string
		press, finish f32.Point
		blur          bool
	}{
		{"tap outside", f32.Pt(150, 150), f32.Pt(151, 150), true},
		{"tap on the field", f32.Pt(20, 20), f32.Pt(20, 20), false},
		{"drag outside", f32.Pt(150, 150), f32.Pt(150, 60), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var (
				a     App
				ops   op.Ops
				r     input.Router
				tag   = new(int)
				field = &fakeField{image.Rect(0, 0, 100, 40)}
			)
			gtx := layout.Context{Ops: &ops, Source: r.Source(), Constraints: layout.Exact(image.Pt(200, 200))}
			frame := func() {
				ops.Reset()
				cl := clip.Rect(field.field).Push(gtx.Ops)
				event.Op(gtx.Ops, tag)
				cl.Pop()
				for {
					if _, ok := gtx.Source.Event(key.FocusFilter{Target: tag}); !ok {
						break
					}
				}
				if gtx.Focused(tag) {
					elements.SetKeyboardOwner(field)
				}
				a.winPointer.layout(&a, gtx)
				r.Frame(&ops)
			}
			frame()
			gtx.Execute(key.FocusCmd{Tag: tag})
			frame()
			if !gtx.Focused(tag) {
				t.Fatal("the field didn't take focus")
			}
			r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Touch, Position: tc.press, Buttons: pointer.ButtonPrimary})
			frame()
			if tc.press != tc.finish {
				r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Touch, Position: tc.finish, Buttons: pointer.ButtonPrimary})
			}
			r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Touch, Position: tc.finish})
			frame()
			frame()
			if blurred := !gtx.Focused(tag); blurred != tc.blur {
				t.Errorf("focus lost: %v, want %v", blurred, tc.blur)
			}
		})
	}
}
