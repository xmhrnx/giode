package elements

import (
	"image"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/gesture"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

type selHarness struct {
	r     input.Router
	inner MemoCache
	build func() Element
}

func (h *selHarness) frame() {
	BeginFrame()
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Source: h.r.Source(), Constraints: layout.Exact(image.Pt(600, 300)), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
	h.build().Layout(gtx)
	h.r.Frame(&ops)
	EndFrame()
}

func (h *selHarness) mouse(kind pointer.Kind, x, y float32) {
	e := pointer.Event{Kind: kind, Source: pointer.Mouse, Position: f32.Pt(x, y)}
	if kind != pointer.Release {
		e.Buttons = pointer.ButtonPrimary
	}
	h.r.Queue(e)
}

func area(key any) *selArea {
	return Keyed("giode.selectionarea", key, newSelArea)
}

func TestSelectionArea_DragAcrossElements(t *testing.T) {
	fonts.UseGofont()
	innerBuilds := 0
	h := &selHarness{}
	h.build = func() Element {
		return SelectionArea("sel", Box(styles.Styles{FlexDirection: properties.FlexDirectionColumn, Width: -1, Height: -1},
			Text("Hello world", styles.Styles{FontSize: 20}),
			Raw(func(gtx layout.Context) layout.Dimensions {
				return h.inner.Layout(gtx, func() Element {
					innerBuilds++
					return Text("Second paragraph here", styles.Styles{FontSize: 20})
				})
			}),
			RichText(styles.Styles{FontSize: 20}, SpanText("Rich "), SpanText("text", styles.Styles{FontWeight: properties.FontWeightBold})),
			// A button's label isn't selectable.
			Keyed("t", "btn", func() *Pressable {
				return NewPressable(func(PressState) Element { return Text("Button", styles.Styles{FontSize: 20}) })
			}),
		))
	}
	for i := 0; i < 3; i++ {
		h.frame()
	}
	a := area("sel")
	if len(a.prev) != 3 {
		t.Fatalf("registered %d text parts, want 3", len(a.prev))
	}
	p0, p1 := a.prev[0], a.prev[1]
	if p1.origin.Y <= p0.origin.Y {
		t.Fatalf("second text at %v, first at %v", p1.origin, p0.origin)
	}

	// From the start of the first text to just past "Second".
	builds := innerBuilds
	h.mouse(pointer.Press, 1, float32(p0.origin.Y+p0.size.Y/2))
	h.frame()
	stops := p1.stopLines()
	x := 0
	for _, s := range stops[0].Stops {
		if s.Rune == len("Second") {
			x = s.X
		}
	}
	h.mouse(pointer.Move, float32(x), float32(p1.origin.Y+p1.size.Y/2))
	h.frame()
	h.mouse(pointer.Release, float32(x), float32(p1.origin.Y+p1.size.Y/2))
	h.frame()

	if got, want := a.selectedText(), "Hello world\nSecond"; got != want {
		t.Errorf("selected %q, want %q", got, want)
	}
	if innerBuilds == builds {
		t.Error("the cached text didn't rebuild to show the selection")
	}

	// Still registered when its memo replays.
	h.frame()
	h.frame()
	if len(a.prev) != 3 {
		t.Errorf("after replays: %d parts, want 3", len(a.prev))
	}

	// Double-click selects a word; in rich text too.
	p2 := a.prev[2]
	y := float32(p2.origin.Y + p2.size.Y/2)
	for i := 0; i < 2; i++ {
		h.mouse(pointer.Press, float32(p2.origin.X+p2.size.X-3), y)
		h.mouse(pointer.Release, float32(p2.origin.X+p2.size.X-3), y)
	}
	h.frame()
	if got := a.selectedText(); got != "text" {
		t.Errorf("double-click selected %q, want %q", got, "text")
	}

	// Shortcut+A selects everything.
	h.r.Queue(key.Event{Name: "A", Modifiers: key.ModShortcut, State: key.Press})
	h.frame()
	if got, want := a.selectedText(), "Hello world\nSecond paragraph here\nRich text"; got != want {
		t.Errorf("select all: %q, want %q", got, want)
	}
	// Escape clears.
	h.r.Queue(key.Event{Name: key.NameEscape, State: key.Press})
	h.frame()
	if !a.empty() {
		t.Errorf("after Escape: %q selected", a.selectedText())
	}
}

func TestSelectionArea_TouchLongPress(t *testing.T) {
	fonts.UseGofont()
	h := &selHarness{}
	h.build = func() Element {
		return SelectionArea("touch", Text("one two three", styles.Styles{FontSize: 20}))
	}
	h.frame()
	h.frame()
	a := area("touch")
	p := a.prev[0]
	line := p.stopLines()[0]
	stops := line.Stops
	// The middle of "two".
	x := float32(stops[4].X+stops[7].X) / 2
	y := float32(line.Top+line.Bottom) / 2
	h.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Touch, Position: f32.Pt(x, y)})
	h.frame()
	if !a.empty() {
		t.Fatal("selected before the long-press")
	}
	a.pressAt = a.pressAt.Add(-longPress)
	h.frame()
	if got := a.selectedText(); got != "two" {
		t.Errorf("long-press selected %q, want %q", got, "two")
	}
	h.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Touch, Position: f32.Pt(x, y)})
	h.frame()
	if !a.showCopy {
		t.Error("no Copy action after the long-press")
	}
}

// fakeList scrolls its child with a touch drag, as the scrolling
// components do.
type fakeList struct {
	scroll gesture.Scroll
	gest   kit.ScrollGesture
	off    int
	child  Element
}

func (l *fakeList) Layout(gtx layout.Context) layout.Dimensions {
	d := l.scroll.Update(gtx.Metric, gtx.Source, gtx.Now, gesture.Vertical, pointer.ScrollRange{}, pointer.ScrollRange{Min: -1000, Max: 1000})
	l.off += d
	l.gest.Track(&l.scroll, d)
	defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
	l.scroll.Add(gtx.Ops)
	st := op.Offset(image.Pt(0, -l.off)).Push(gtx.Ops)
	l.child.Layout(gtx)
	st.Pop()
	return layout.Dimensions{Size: gtx.Constraints.Max}
}

func TestSelectionArea_ScrollIsNotALongPress(t *testing.T) {
	fonts.UseGofont()
	list := &fakeList{child: Text("one two three four five six seven eight", styles.Styles{FontSize: 20})}
	h := &selHarness{build: func() Element { return SelectionArea("scrolling", list) }}
	now := time.Unix(0, 0)
	frame := func() {
		BeginFrame()
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Source: h.r.Source(), Now: now, Constraints: layout.Exact(image.Pt(600, 300)), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
		h.build().Layout(gtx)
		h.r.Frame(&ops)
		EndFrame()
	}
	touch := func(kind pointer.Kind, y float32) {
		h.r.Queue(pointer.Event{Kind: kind, Source: pointer.Touch, Position: f32.Pt(30, y), PointerID: 1, Time: now.Sub(time.Unix(0, 0))})
		frame()
	}
	frame()
	frame()
	a := area("scrolling")

	// Down, scroll a bit, then rest past the long-press time.
	touch(pointer.Press, 15)
	now = now.Add(50 * time.Millisecond)
	touch(pointer.Move, 10) // the list grabs the pointer
	now = now.Add(50 * time.Millisecond)
	touch(pointer.Move, 5)
	if list.off == 0 {
		t.Fatal("the list didn't scroll")
	}
	for i := 0; i < 4; i++ {
		now = now.Add(300 * time.Millisecond)
		frame()
	}
	if !a.empty() {
		t.Errorf("a resting finger mid-scroll selected %q", a.selectedText())
	}
	touch(pointer.Release, 5)

	// A real long-press still selects.
	touch(pointer.Press, 15)
	now = now.Add(600 * time.Millisecond)
	frame()
	if a.empty() {
		t.Error("a long-press without scrolling selected nothing")
	}
}
