package dropdown

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/pointer"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/styles"
)

func (h *sheetHarness) drag(x, y0, y1 int) {
	h.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Touch, Position: f32.Pt(float32(x), float32(y0)), Buttons: pointer.ButtonPrimary})
	h.frame()
	for y := y0; y != y1; {
		step := 10
		if y1 < y0 {
			step = -10
		}
		y += step
		if (step > 0 && y > y1) || (step < 0 && y < y1) {
			y = y1
		}
		h.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Touch, Position: f32.Pt(float32(x), float32(y)), Buttons: pointer.ButtonPrimary})
		h.frame()
	}
	h.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Touch, Position: f32.Pt(float32(x), float32(y1))})
	h.frame()
	h.frame()
}

func TestSelect_TouchDragScrollsList(t *testing.T) {
	fonts.UseGofont()
	defer elements.SetWindowSize(image.Point{})
	h := &sheetHarness{win: image.Pt(400, 300), s: New(styles.Styles{}, manyOptions(60)...)}
	h.frame()
	h.tap(20, 10)
	if h.s.list == nil {
		t.Fatal("not scrolling")
	}
	before := h.s.list.Offset()
	h.drag(100, 250, 80)
	if h.s.list.Offset() <= before {
		t.Errorf("drag up didn't scroll: offset %v, was %v", h.s.list.Offset(), before)
	}
	if !h.s.Opened() {
		t.Error("the drag closed the list")
	}
}

func TestSelect_TouchDragScrollsSheet(t *testing.T) {
	fonts.UseGofont()
	defer elements.SetWindowSize(image.Point{})
	h := &sheetHarness{win: image.Pt(400, 800), s: New(styles.Styles{}, manyOptions(60)...).Sheet(SheetAlways)}
	h.frame()
	h.tap(20, 10)
	before := h.s.sheet.list.Offset()
	h.drag(200, 750, 450)
	if h.s.sheet.list.Offset() <= before {
		t.Errorf("drag up didn't scroll the sheet: offset %v, was %v", h.s.sheet.list.Offset(), before)
	}
	h.r.Queue(pointer.Event{Kind: pointer.Scroll, Source: pointer.Mouse, Position: f32.Pt(200, 600), Scroll: f32.Pt(0, 200)})
	h.frame()
	h.frame()
	if h.s.sheet.list.Offset() <= before {
		t.Errorf("wheel didn't scroll the sheet")
	}
}
