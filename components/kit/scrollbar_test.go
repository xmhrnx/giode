package kit

import (
	"image"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/styles/properties"
)

func TestScrollbar_Hidden(t *testing.T) {
	defer func(f func() (int, string)) { Screen = f }(Screen)
	width, bp := 0, ""
	Screen = func() (int, string) { return width, bp }
	gtx := layout.Context{Ops: new(op.Ops), Metric: unit.Metric{PxPerDp: 2, PxPerSp: 2}}

	shown := func(sb properties.ScrollBar) bool {
		var s Scrollbar
		off := float32(0)
		s.Layout(gtx, image.Pt(200, 100), 1000, &off, sb)
		return s.shown
	}
	cases := []struct {
		name  string
		width int
		bp    string
		sb    properties.ScrollBar
		want  bool
	}{
		{"default", 1600, "", properties.ScrollBar{}, true},
		{"hidden", 1600, "", properties.ScrollBar{Hidden: true}, false},
		// 600dp is 1200px at 2px per dp.
		{"phone width", 1000, "", properties.ScrollBar{HiddenMaxWidth: 600}, false},
		{"desktop width", 1600, "", properties.ScrollBar{HiddenMaxWidth: 600}, true},
		{"compact breakpoint", 1600, "compact", properties.ScrollBar{HiddenOn: "compact"}, false},
		{"one of several", 1600, "medium", properties.ScrollBar{HiddenOn: "compact, medium"}, false},
		{"other breakpoint", 1600, "wide", properties.ScrollBar{HiddenOn: "compact,medium"}, true},
	}
	for _, c := range cases {
		width, bp = c.width, c.bp
		if got := shown(c.sb); got != c.want {
			t.Errorf("%s: shown = %v, want %v", c.name, got, c.want)
		}
	}
}

// Dragging the thumb moves the content by the pointer's movement from
// the press - also where other handlers share the thumb's events (the
// list under it, a window-wide listener), so that the press and the
// first drags don't come with Gio's grab.
func TestScrollbar_DragSharedThumb(t *testing.T) {
	var r input.Router
	var s Scrollbar
	var scroll gesture.Scroll
	window := new(int)
	off := float32(0)
	viewport := image.Pt(200, 100)
	const contentH = 1000
	now := time.Unix(0, 0)
	frame := func() {
		var ops op.Ops
		now = now.Add(16 * time.Millisecond)
		gtx := layout.Context{Ops: &ops, Now: now, Source: r.Source(), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(viewport)}
		// The list's own scrolling, under the bar.
		scroll.Update(gtx.Metric, gtx.Source, gtx.Now, gesture.Vertical, pointer.ScrollRange{}, pointer.ScrollRange{Min: -1e6, Max: 1e6})
		cl := clip.Rect{Max: viewport}.Push(gtx.Ops)
		scroll.Add(gtx.Ops)
		cl.Pop()
		s.Layout(gtx, viewport, contentH, &off, properties.ScrollBar{})
		// A window-wide pass-through listener over everything.
		for {
			if _, ok := gtx.Event(pointer.Filter{Target: window, Kinds: pointer.Press | pointer.Release}); !ok {
				break
			}
		}
		ps := pointer.PassOp{}.Push(gtx.Ops)
		cl = clip.Rect{Max: viewport}.Push(gtx.Ops)
		event.Op(gtx.Ops, window)
		cl.Pop()
		ps.Pop()
		r.Frame(&ops)
	}
	frame()
	frame()
	// The thumb: 8px wide at the right edge, 20px tall (100*100/1000,
	// at least the 20px minimum) at the top; 80px of track to move in
	// for 900px of content.
	x := float32(viewport.X - 4)
	at := func(y float32) f32.Point { return f32.Pt(x, y) }
	r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: at(10)})
	frame()
	r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: at(10)})
	frame()
	for i := 1; i <= 8; i++ {
		r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: at(10 + float32(i)*2)})
		frame()
		want := float32(i*2) * 900 / 80
		if off < want-1 || off > want+1 {
			t.Fatalf("after dragging %dpx: offset %v, want %v", i*2, off, want)
		}
	}
	r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: at(26)})
	frame()
	// A second drag starts from its own press, not the first one's end.
	start := off
	r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: at(30)})
	frame()
	r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: at(34)})
	frame()
	if want := start + 4*900/80; off < want-1 || off > want+1 {
		t.Errorf("second drag of 4px: offset %v, want %v", off, want)
	}
}
