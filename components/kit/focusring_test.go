package kit

import (
	"image"
	"testing"

	"gioui.org/io/event"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
)

func TestFocusVisible(t *testing.T) {
	var r input.Router
	a, b := new(int), new(int)
	frame := func() (va, vb bool) {
		NextFrame()
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Source: r.Source(), Constraints: layout.Exact(image.Pt(100, 100))}
		for _, tag := range []any{a, b} {
			for {
				if _, ok := gtx.Event(key.FocusFilter{Target: tag}); !ok {
					break
				}
			}
			cl := clip.Rect{Max: image.Pt(10, 10)}.Push(gtx.Ops)
			event.Op(gtx.Ops, tag)
			cl.Pop()
		}
		va, vb = FocusVisible(gtx, a), FocusVisible(gtx, b)
		r.Frame(&ops)
		return
	}
	frame()
	// Focus moved by the keyboard (no press): the ring shows.
	r.Source().Execute(key.FocusCmd{Tag: a})
	frame()
	if va, _ := frame(); !va {
		t.Error("keyboard focus shows no ring")
	}
	// A press, then focus: no ring.
	NotePointerPress()
	r.Source().Execute(key.FocusCmd{Tag: b})
	frame()
	if _, vb := frame(); vb {
		t.Error("focus from a press shows a ring")
	}
	// Tabbing on from there shows it again.
	for range 4 {
		frame()
	}
	r.Source().Execute(key.FocusCmd{Tag: a})
	frame()
	if va, _ := frame(); !va {
		t.Error("keyboard focus after a click shows no ring")
	}
}
