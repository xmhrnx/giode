package box

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/elements/internal/base"
	"github.com/xmhrnx/giode/elements/raw"
	"github.com/xmhrnx/giode/styles"
)

// leaf is a CacheSafe element counting its layouts.
type leaf struct{ layouts *int }

func (l leaf) Layout(gtx layout.Context) layout.Dimensions {
	*l.layouts++
	return layout.Dimensions{Size: image.Pt(40, 20)}
}
func (leaf) CacheSafe() bool { return true }

type frames struct {
	r   input.Router
	win image.Point
}

func (f *frames) frame(el base.Element) {
	base.BeginFrame()
	defer base.EndFrame()
	var ops op.Ops
	el.Layout(layout.Context{Ops: &ops, Source: f.r.Source(), Constraints: layout.Exact(f.win)})
	f.r.Frame(&ops)
}

// A hoverable box made of CacheSafe content replays its output while
// nothing changes, and lays its content out again when hovered.
func TestHoverBoxCachesItsOutput(t *testing.T) {
	n := 0
	b := Box(styles.Styles{Width: -1, Height: -1}, leaf{&n}).OnHover(styles.Styles{Opacity: 0.5})
	f := &frames{win: image.Pt(100, 100)}
	for i := 0; i < 5; i++ {
		f.frame(b)
	}
	if n != 1 {
		t.Fatalf("content laid out %d times in 5 unchanged frames, want 1", n)
	}
	f.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: f32.Pt(50, 50)})
	f.frame(b)
	f.frame(b)
	if !b.Hovered() || n < 2 {
		t.Fatalf("after hovering: hovered=%v, %d layouts; want hovered and laid out again", b.Hovered(), n)
	}
	before := n
	f.frame(b)
	f.frame(b)
	if n != before {
		t.Errorf("hovered and still: %d more layouts, want none", n-before)
	}
}

// Content that may read any variable - a Raw drawing function - keeps
// the box laying it out every frame.
func TestBoxWithRawIsNotCached(t *testing.T) {
	n := 0
	b := Box(styles.Styles{}, raw.Raw(func(gtx layout.Context) layout.Dimensions {
		n++
		return layout.Dimensions{}
	})).OnHover(styles.Styles{Opacity: 0.5})
	if b.CacheSafe() {
		t.Fatal("a box holding a Raw element says it's cache-safe")
	}
	f := &frames{win: image.Pt(100, 100)}
	for i := 0; i < 3; i++ {
		f.frame(b)
	}
	if n != 3 {
		t.Errorf("Raw laid out %d times in 3 frames, want 3", n)
	}
}

// A box made each frame (no OnHover or OnClick) isn't cached itself:
// only kept elements are.
func TestPlainBoxIsNotCached(t *testing.T) {
	n := 0
	f := &frames{win: image.Pt(100, 100)}
	for i := 0; i < 3; i++ {
		f.frame(Box(styles.Styles{}, leaf{&n}))
	}
	if n != 3 {
		t.Errorf("content laid out %d times in 3 frames, want 3", n)
	}
}
