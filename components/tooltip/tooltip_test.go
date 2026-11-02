package tooltip

import (
	"image"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
)

func TestTooltip_HoveredFollowsPointer(t *testing.T) {
	elements.SetWindowSize(image.Pt(800, 600))
	defer elements.SetWindowSize(image.Point{})

	var ops op.Ops
	var router input.Router
	gtx := layout.Context{Ops: &ops, Source: router.Source(), Constraints: layout.Exact(image.Pt(100, 40))}

	trigger := elements.Box(styles.Styles{Width: 100, Height: 40})
	tip := New(trigger, "hint")

	moveTo := func(p f32.Point) {
		router.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: p})
	}
	frame := func() {
		ops.Reset()
		tip.Layout(gtx)
		router.Frame(&ops)
	}

	frame() // register the hover region

	moveTo(f32.Pt(10, 10))
	frame()
	frame()
	if !tip.Hovered() {
		t.Fatalf("pointer inside trigger: Hovered() = false, want true")
	}

	moveTo(f32.Pt(500, 500))
	frame()
	frame()
	if tip.Hovered() {
		t.Fatalf("pointer outside trigger: Hovered() = true, want false")
	}
}

func TestTooltip_LayoutReturnsTriggerDimensions(t *testing.T) {
	elements.SetWindowSize(image.Pt(800, 600))
	defer elements.SetWindowSize(image.Point{})

	var ops op.Ops
	var router input.Router
	gtx := layout.Context{Ops: &ops, Source: router.Source(), Constraints: layout.Exact(image.Pt(100, 40))}

	trigger := elements.Box(styles.Styles{Width: 100, Height: 40})
	tip := New(trigger, "hint")

	dims := tip.Layout(gtx)
	if dims.Size != (image.Point{X: 100, Y: 40}) {
		t.Errorf("Layout dims = %v, want the trigger's own 100x40", dims.Size)
	}
}

func TestTooltip_LongPressOnTouch(t *testing.T) {
	elements.SetWindowSize(image.Pt(800, 600))
	defer elements.SetWindowSize(image.Point{})

	var ops op.Ops
	var router input.Router
	now := time.Unix(100, 0)
	tip := New(elements.Box(styles.Styles{Width: 100, Height: 40}), "hint")
	frame := func(dt time.Duration) {
		now = now.Add(dt)
		ops.Reset()
		gtx := layout.Context{Ops: &ops, Now: now, Source: router.Source(), Constraints: layout.Exact(image.Pt(100, 40))}
		tip.Layout(gtx)
		router.Frame(&ops)
	}
	touch := func(kind pointer.Kind, x float32) {
		router.Queue(pointer.Event{Kind: kind, Source: pointer.Touch, Position: f32.Pt(x, 10)})
	}

	frame(0)
	// A quick tap doesn't show it.
	touch(pointer.Press, 10)
	frame(16 * time.Millisecond)
	if tip.Visible() {
		t.Fatal("visible right after a touch press")
	}
	touch(pointer.Release, 10)
	frame(100 * time.Millisecond)
	if tip.Visible() {
		t.Fatal("visible after a quick tap")
	}

	// A long-press does, and it lingers after the finger lifts.
	touch(pointer.Press, 10)
	frame(16 * time.Millisecond)
	frame(longPress)
	if !tip.Visible() {
		t.Fatal("not visible after a long-press")
	}
	touch(pointer.Release, 10)
	frame(16 * time.Millisecond)
	if !tip.Visible() {
		t.Fatal("hidden right after the long-press ended")
	}
	frame(linger)
	if tip.Visible() {
		t.Fatal("still visible after lingering")
	}

	// Sliding off before the long-press completes cancels it.
	touch(pointer.Press, 10)
	frame(16 * time.Millisecond)
	touch(pointer.Move, 500)
	frame(longPress)
	if tip.Visible() {
		t.Error("visible although the finger slid off the trigger")
	}
	if tip.Hovered() {
		t.Error("touch reported as hover")
	}
}
