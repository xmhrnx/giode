package button

import (
	"image"
	"testing"

	"gioui.org/io/input"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/fonts"
)

func cacheFrame(r *input.Router, el elements.Element) layout.Dimensions {
	elements.BeginFrame()
	defer elements.EndFrame()
	var ops op.Ops
	dims := el.Layout(layout.Context{Ops: &ops, Source: r.Source(), Constraints: layout.Constraints{Max: image.Pt(800, 200)}})
	r.Frame(&ops)
	return dims
}

// A label changed through the setter shows even while an enclosing
// Memo replays the button without laying it out: the button's own
// cache reports the change to the Memo.
func TestLabelSetterReachesThroughMemo(t *testing.T) {
	fonts.UseGofont()
	b := New("OK")
	var r input.Router
	memo := elements.Memo("toolbar", func() elements.Element { return b })
	short := cacheFrame(&r, memo).Size.X
	cacheFrame(&r, memo)
	b.Label("A much longer label")
	cacheFrame(&r, memo) // the Memo's pass notices the change
	long := cacheFrame(&r, memo).Size.X
	if long <= short {
		t.Fatalf("width %d after the label grew, was %d: the change didn't show", long, short)
	}
	_, misses := b.cache.Stats()
	for i := 0; i < 3; i++ {
		cacheFrame(&r, memo)
	}
	if _, m := b.cache.Stats(); m != misses {
		t.Errorf("unchanged button rebuilt %d times in 3 frames", m-misses)
	}
}
