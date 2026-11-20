package kit

import (
	"image"
	"sync"
	"sync/atomic"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/styles/properties"
)

// Focus rings.
//
// A focused control shows a ring when the focus came from the keyboard
// (Tab, arrows), not from a click or a tap - what browsers call
// :focus-visible - so keyboard users see where they are without rings
// following every click.

var (
	// keyboardNav is set when focus last moved without a pointer press.
	keyboardNav atomic.Bool
	// pressFrame is the frame of the last pointer press; frameCount
	// counts frames (NextFrame).
	pressFrame atomic.Int64
	frameCount atomic.Int64

	ringMu sync.Mutex
	// ringFocused holds the tags focused last frame, to see focus move.
	ringFocused = map[any]int64{}
)

// NotePointerPress records a pointer press anywhere in the window: focus
// that follows it came from the pointer. The app calls it.
func NotePointerPress() {
	// Frames count from 1 here, so 0 means no press yet.
	pressFrame.Store(frameCount.Load() + 1)
	keyboardNav.Store(false)
}

// FocusVisible reports whether tag's focus should show: it's focused,
// and the focus came from the keyboard.
func FocusVisible(gtx layout.Context, tag any) bool {
	NoteFocusable(tag)
	focused := gtx.Focused(tag)
	if focused {
		lastFocused = tag
	}
	frame := frameCount.Load()
	ringMu.Lock()
	last, was := ringFocused[tag]
	if focused {
		ringFocused[tag] = frame
		if !was || frame-last > 1 {
			// Focus arrived: by keyboard unless a press came just before.
			p := pressFrame.Load()
			keyboardNav.Store(p == 0 || frame+1-p > 2)
		}
	} else if was {
		delete(ringFocused, tag)
	}
	if len(ringFocused) > 64 {
		for k, f := range ringFocused {
			if frame-f > 2 {
				delete(ringFocused, k)
			}
		}
	}
	ringMu.Unlock()
	return focused && keyboardNav.Load()
}

// FocusRing draws the focus ring around a size control at the current
// position, rounded by radius pixels, when FocusVisible.
func FocusRing(gtx layout.Context, tag any, size image.Point, radius int) {
	if !FocusVisible(gtx, tag) {
		return
	}
	PaintFocusRing(gtx, size, radius)
}

// PaintFocusRing draws a focus ring around a size area at the current
// position: 2 dp in the theme's primary color, just outside it.
func PaintFocusRing(gtx layout.Context, size image.Point, radius int) {
	w := max(gtx.Dp(unit.Dp(2)), 1)
	gap := max(gtx.Dp(unit.Dp(1)), 1)
	r := image.Rectangle{Max: size}.Inset(-(gap + w/2))
	rr := clip.RRect{Rect: r, SE: radius + gap, SW: radius + gap, NW: radius + gap, NE: radius + gap}
	col := properties.CalcColor("theme.primary")
	paint.FillShape(gtx.Ops, col, clip.Stroke{Path: rr.Path(gtx.Ops), Width: float32(w)}.Op())
}
