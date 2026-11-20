package datepicker

import (
	"image"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/datetime"
	"github.com/xmhrnx/giode/ui"
)

type harness struct {
	r     input.Router
	el    layout.Widget
	now   time.Time
	frame func()
}

func newHarness(w layout.Widget) *harness {
	h := &harness{el: w, now: time.Unix(0, 0)}
	h.frame = func() {
		h.now = h.now.Add(16 * time.Millisecond)
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Now: h.now, Source: h.r.Source(), Constraints: layout.Constraints{Max: image.Pt(800, 600)}}
		ui.Frame(gtx, rawElement(h.el))
		h.r.Frame(&ops)
	}
	return h
}

type rawElement layout.Widget

func (r rawElement) Layout(gtx layout.Context) layout.Dimensions { return r(gtx) }

func (h *harness) click(p f32.Point) {
	h.r.Queue(
		pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: p},
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: p, Buttons: pointer.ButtonPrimary},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: p},
	)
	h.frame()
	h.frame()
}

func (h *harness) key(name key.Name, mods key.Modifiers) {
	h.r.Queue(key.Event{Name: name, Modifiers: mods, State: key.Press}, key.Event{Name: name, Modifiers: mods, State: key.Release})
	h.frame()
}

// cellCenter returns the middle of day's cell in the calendar's first
// month, laid out at the window's origin.
func cellCenter(c *DateCalendar, day time.Time) f32.Point {
	g := c.grids[0]
	n := datetime.DaysBetween(g.first, day)
	row, col := n/7, n%7
	return f32.Pt(float32(g.rect.Min.X+col*c.cell+c.cell/2), float32(g.rect.Min.Y+row*c.cell+c.cell/2))
}

func TestCalendarPicksADay(t *testing.T) {
	var picked time.Time
	c := NewCalendar().Locale(datetime.NewLocale("en-US")).Zone(time.UTC).OnSelect(func(d time.Time) { picked = d })
	c.ShowMonth(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC))
	h := newHarness(c.Layout)
	h.frame()
	// November 2026 starts on a Sunday: its 1st is the top-left cell
	// in the US.
	if g := c.grids[0].first; !g.Equal(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("grid starts %s", g)
	}
	h.click(cellCenter(c, time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)))
	if !picked.Equal(time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("picked %s", picked)
	}
	// The keyboard: right to the 21st, down a week to the 28th, Page
	// Down to December 28th, Enter.
	h.key(key.NameRightArrow, 0)
	h.key(key.NameDownArrow, 0)
	h.key(key.NamePageDown, 0)
	h.key(key.NameReturn, 0)
	if !picked.Equal(time.Date(2026, 12, 28, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("keyboard picked %s", picked)
	}
	if !c.Shown().Equal(time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("shows %s", c.Shown())
	}
}

func TestCalendarRangeAndLimits(t *testing.T) {
	var start, end time.Time
	c := NewCalendar().Locale(datetime.NewLocale("de-DE")).Zone(time.UTC).Range(true).
		Limits(time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC), time.Time{}).
		OnRange(func(s, e time.Time) { start, end = s, e })
	c.ShowMonth(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC))
	h := newHarness(c.Layout)
	h.frame()
	// Weeks start on Monday in Germany: the grid starts on October 26.
	if g := c.grids[0].first; !g.Equal(time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("grid starts %s", g)
	}
	day := func(d int) time.Time { return time.Date(2026, 11, d, 0, 0, 0, 0, time.UTC) }
	h.click(cellCenter(c, day(3))) // before the limit: ignored
	h.click(cellCenter(c, day(18)))
	h.click(cellCenter(c, day(9)))
	if !start.Equal(day(9)) || !end.Equal(day(18)) {
		t.Errorf("range %s - %s", start, end)
	}
}

func TestCalendarPersian(t *testing.T) {
	c := NewCalendar().Locale(datetime.NewLocale("fa-IR")).Zone(time.UTC)
	c.ShowMonth(time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC))
	h := newHarness(c.Layout)
	h.frame()
	// Aban 1405 starts on Friday October 23; weeks start on Saturday,
	// so the grid starts on Saturday October 17.
	if g := c.grids[0].first; !g.Equal(time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("grid starts %s", g)
	}
	if m := c.grids[0].month; m != (datetime.Date{Year: 1405, Month: 8, Day: 1}) {
		t.Errorf("month %v", m)
	}
}

func TestCalendarArrows(t *testing.T) {
	c := NewCalendar().Locale(datetime.NewLocale("en-US")).Zone(time.UTC)
	c.ShowMonth(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC))
	h := newHarness(c.Layout)
	h.frame()
	w := 7 * c.cell
	// The next arrow sits at the header's right end.
	h.click(f32.Pt(float32(w-c.cell/2), float32(c.cell/2)))
	if !c.Shown().Equal(time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("after next: %s", c.Shown())
	}
	h.click(f32.Pt(float32(c.cell/2), float32(c.cell/2)))
	h.click(f32.Pt(float32(c.cell/2), float32(c.cell/2)))
	if !c.Shown().Equal(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("after previous twice: %s", c.Shown())
	}
}
