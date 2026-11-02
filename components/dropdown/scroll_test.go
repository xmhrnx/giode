package dropdown

import (
	"fmt"
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/pointer"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/styles"
)

func manyOptions(n int) []string {
	opts := make([]string, n)
	for i := range opts {
		opts[i] = fmt.Sprintf("Option %d", i)
	}
	return opts
}

func TestSelect_LongListScrolls(t *testing.T) {
	fonts.UseGofont()
	defer elements.SetWindowSize(image.Point{})
	h := &sheetHarness{win: image.Pt(400, 300), s: New(styles.Styles{}, manyOptions(60)...)}
	h.s.selected = 45
	h.frame()
	h.tap(20, 10) // opens the list below the field
	h.frame()
	if !h.s.Opened() {
		t.Fatal("the list didn't open")
	}
	if h.s.list == nil {
		t.Fatal("60 options in a 300px window didn't scroll")
	}
	if v := h.s.list.ViewportHeight(); v <= 0 || v > 300 {
		t.Fatalf("viewport %d, want within the window", v)
	}
	// The selected option, far down the list, is scrolled into view.
	off := h.s.list.Offset()
	if off <= 0 {
		t.Fatalf("selected option 45 not revealed: offset %v", off)
	}
	row := h.s.list.ContentHeight() / 60
	top := 45*row - int(off)
	if top < 0 || top+row > h.s.list.ViewportHeight() {
		t.Errorf("option 45 at %d..%d, outside the %dpx viewport", top, top+row, h.s.list.ViewportHeight())
	}

	// The wheel scrolls it.
	h.r.Queue(pointer.Event{Kind: pointer.Scroll, Source: pointer.Mouse, Position: f32.Pt(100, 150), Scroll: f32.Pt(0, -200)})
	h.frame()
	h.frame()
	if h.s.list.Offset() >= off {
		t.Errorf("wheel up didn't scroll: offset %v, was %v", h.s.list.Offset(), off)
	}
}

func TestSelect_ShortListHugs(t *testing.T) {
	fonts.UseGofont()
	defer elements.SetWindowSize(image.Point{})
	h := &sheetHarness{win: image.Pt(400, 800), s: New(styles.Styles{}, "a", "b", "c")}
	h.frame()
	h.tap(20, 10)
	if !h.s.Opened() || h.s.list != nil {
		t.Errorf("three options: open=%v scrolling=%v, want open without scrolling", h.s.Opened(), h.s.list != nil)
	}
}

func TestListHeight(t *testing.T) {
	gtx := (&sheetHarness{}).ctx()
	win := image.Pt(400, 600)
	// A field near the top: the space below.
	if got, want := listHeight(gtx, image.Rect(0, 10, 100, 40), win), 600-40-listGap-8; got != want {
		t.Errorf("field at top: %d, want %d", got, want)
	}
	// Near the bottom: the space above.
	if got, want := listHeight(gtx, image.Rect(0, 550, 100, 580), win), 550-listGap-8; got != want {
		t.Errorf("field at bottom: %d, want %d", got, want)
	}
}
