package transition

import (
	"image"
	"testing"
	"time"

	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// frames lays t out at the given times, returning the sizes reported.
func frames(t *Transition, start time.Time, at ...time.Duration) []image.Point {
	var out []image.Point
	for _, d := range at {
		gtx := layout.Context{
			Ops:         new(op.Ops),
			Now:         start.Add(d),
			Constraints: layout.Constraints{Max: image.Pt(500, 500)},
		}
		out = append(out, t.Layout(gtx).Size)
	}
	return out
}

func box() elements.Element {
	return elements.Box(styles.Styles{Width: 100, Height: 40})
}

func TestAppearsAndSettles(t *testing.T) {
	tr := New(box(), FadeUp, 100*time.Millisecond, properties.EaseLinear)
	start := time.Unix(0, 0)
	frames(tr, start, 0)
	if !tr.Animating() || tr.Progress() != 0 {
		t.Fatalf("first frame: progress %v, animating %v - want the appear animation to start at 0", tr.Progress(), tr.Animating())
	}
	frames(tr, start, 50*time.Millisecond)
	if p := tr.Progress(); p < 0.45 || p > 0.55 {
		t.Errorf("halfway: progress %v, want ~0.5", p)
	}
	frames(tr, start, 200*time.Millisecond)
	if tr.Animating() || tr.Progress() != 1 {
		t.Errorf("after the duration: progress %v, want settled at 1", tr.Progress())
	}
}

func TestHiddenTakesNoSpace(t *testing.T) {
	tr := New(box(), Fade, Fast, "").Hidden()
	start := time.Unix(0, 0)
	if s := frames(tr, start, 0)[0]; s != (image.Point{}) {
		t.Errorf("hidden size = %v, want zero", s)
	}
	tr.Show()
	sizes := frames(tr, start, 10*time.Millisecond, time.Second)
	if sizes[1] != image.Pt(100, 40) {
		t.Errorf("shown size = %v, want the child's", sizes[1])
	}
	tr.Hide()
	frames(tr, start, 2*time.Second, 3*time.Second)
	if s := frames(tr, start, 4*time.Second)[0]; s != (image.Point{}) || tr.Visible() {
		t.Errorf("after hiding: size %v, want zero", s)
	}
}

func TestCollapseAnimatesLayoutSize(t *testing.T) {
	tr := New(box(), Collapse, 100*time.Millisecond, properties.EaseLinear)
	start := time.Unix(0, 0)
	sizes := frames(tr, start, 0, 50*time.Millisecond, 200*time.Millisecond)
	if sizes[0].Y != 0 || sizes[1].Y < 18 || sizes[1].Y > 22 || sizes[2].Y != 40 {
		t.Errorf("collapse heights = %v, want 0, ~20, 40", sizes)
	}
	if sizes[1].X != 100 {
		t.Errorf("collapse changed the width: %v", sizes[1])
	}
}

func TestNonCollapseKeepsFullSizeWhileAnimating(t *testing.T) {
	tr := New(box(), Zoom|SlideLeft|Rotate, 100*time.Millisecond, properties.EaseOutBack)
	start := time.Unix(0, 0)
	for _, s := range frames(tr, start, 0, 30*time.Millisecond, 60*time.Millisecond) {
		if s != image.Pt(100, 40) {
			t.Errorf("size while animating = %v, want the child's, so layout doesn't jump", s)
		}
	}
}

func TestInterruptReverses(t *testing.T) {
	tr := New(box(), Fade, 100*time.Millisecond, properties.EaseLinear)
	start := time.Unix(0, 0)
	frames(tr, start, 0, 60*time.Millisecond)
	tr.Hide()
	frames(tr, start, 80*time.Millisecond)
	if p := tr.Progress(); p < 0.35 || p > 0.45 {
		t.Errorf("after reversing for 20ms from 0.6: progress %v, want ~0.4", p)
	}
}

func TestOnDoneAndAppearFalse(t *testing.T) {
	var done []bool
	tr := New(box(), Fade, 50*time.Millisecond, "").Appear(false).OnDone(func(v bool) { done = append(done, v) })
	start := time.Unix(0, 0)
	frames(tr, start, 0)
	if tr.Animating() {
		t.Error("Appear(false): animating on first layout")
	}
	tr.Hide()
	frames(tr, start, 10*time.Millisecond, 200*time.Millisecond)
	if len(done) != 1 || done[0] {
		t.Errorf("OnDone calls = %v, want [false]", done)
	}
}

func TestIdleStartDoesNotJump(t *testing.T) {
	tr := New(box(), Fade, 100*time.Millisecond, properties.EaseLinear).Appear(false)
	start := time.Unix(0, 0)
	frames(tr, start, 0)
	tr.Hide()
	// The first frame after a long idle period starts the animation
	// rather than finishing it.
	frames(tr, start, time.Hour)
	if tr.Progress() != 1 {
		t.Errorf("first frame after idle: progress %v, want 1 (animation just starting)", tr.Progress())
	}
}
