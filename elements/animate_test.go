package elements

import (
	"image"
	"testing"
	"time"

	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// animFrame lays el out at now and returns its size.
func animFrame(el Element, now time.Time) image.Point {
	BeginFrame()
	defer EndFrame()
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Now: now, Constraints: layout.Constraints{Max: image.Pt(1000, 1000)}}
	return el.Layout(gtx).Size
}

func TestAnimateInterpolatesStyleChanges(t *testing.T) {
	now := time.Unix(100, 0)
	tr := properties.Transition{Duration: 100 * time.Millisecond, Easing: properties.EaseLinear}
	box := func(h int) Element {
		return Box(styles.Styles{Width: 50, Height: properties.Height(h), Transition: tr}).Animate("panel")
	}
	if s := animFrame(box(100), now); s.Y != 100 {
		t.Fatalf("first frame: height %d, want 100 at once", s.Y)
	}
	animFrame(box(200), now)
	now = now.Add(50 * time.Millisecond)
	if s := animFrame(box(200), now); s.Y != 150 {
		t.Errorf("half way: height %d, want 150", s.Y)
	}
	// Changing course mid-way starts from where it is.
	animFrame(box(100), now)
	now = now.Add(50 * time.Millisecond)
	if s := animFrame(box(100), now); s.Y != 125 {
		t.Errorf("turned back half way: height %d, want 125", s.Y)
	}
	now = now.Add(60 * time.Millisecond)
	if s := animFrame(box(100), now); s.Y != 100 {
		t.Errorf("done: height %d, want 100", s.Y)
	}

	kit.SetReducedMotion(true)
	defer kit.SetReducedMotion(false)
	animFrame(box(300), now)
	if s := animFrame(box(300), now.Add(time.Millisecond)); s.Y != 300 {
		t.Errorf("reduced motion: height %d, want 300 at once", s.Y)
	}
}

func TestKeyframes(t *testing.T) {
	now := time.Unix(200, 0)
	done := 0
	kf := Keyframes(100*time.Millisecond,
		Keyframe{At: 0, Styles: styles.Styles{Width: 100}},
		Keyframe{At: 0.5, Styles: styles.Styles{Width: 200}},
		Keyframe{At: 1, Styles: styles.Styles{Width: 120}},
	).Easing(properties.EaseLinear).OnDone(func() { done++ })
	box := Box(styles.Styles{Width: 10, Height: 10}).Animation(kf)
	at := func(ms int) int { return animFrame(box, now.Add(time.Duration(ms)*time.Millisecond)).X }
	if w := at(0); w != 100 {
		t.Errorf("start: %d", w)
	}
	if w := at(25); w != 150 {
		t.Errorf("a quarter: %d, want 150", w)
	}
	if w := at(75); w != 160 {
		t.Errorf("three quarters: %d, want 160", w)
	}
	if w := at(150); w != 120 || done != 1 || !kf.Done() {
		t.Errorf("after: width %d, done %d", w, done)
	}
	at(200)
	if done != 1 {
		t.Errorf("OnDone ran %d times", done)
	}

	// Repeating and alternating.
	now = now.Add(time.Second)
	kf2 := Keyframes(100*time.Millisecond, Keyframe{At: 0, Styles: styles.Styles{Width: 100}}, Keyframe{At: 1, Styles: styles.Styles{Width: 200}}).
		Easing(properties.EaseLinear).Repeat(0).Alternate(true)
	box2 := Box(styles.Styles{Width: 10, Height: 10}).Animation(kf2)
	at2 := func(ms int) int { return animFrame(box2, now.Add(time.Duration(ms)*time.Millisecond)).X }
	at2(0)
	if w := at2(125); w != 175 {
		t.Errorf("second, backwards run at a quarter: %d, want 175", w)
	}
	if w := at2(1050); w != 150 {
		t.Errorf("eleventh run, half way: %d, want 150", w)
	}

	// Reduced motion: a repeating animation doesn't play.
	kit.SetReducedMotion(true)
	defer kit.SetReducedMotion(false)
	if w := at2(1075); w != 10 {
		t.Errorf("reduced motion, repeating: width %d, want the box's own 10", w)
	}
}
