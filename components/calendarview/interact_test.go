package calendarview

import (
	"image"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/datetime"
)

type harness struct {
	r   input.Router
	v   *CalendarView
	now time.Time
}

func (h *harness) frame() {
	h.now = h.now.Add(16 * time.Millisecond)
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Now: h.now, Source: h.r.Source(), Constraints: layout.Exact(image.Pt(1000, 900))}
	frame(gtx, h.v.Layout)
	h.r.Frame(&ops)
}

func (h *harness) drag(from, to f32.Point) {
	h.r.Queue(
		pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: from},
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: from, Buttons: pointer.ButtonPrimary},
	)
	h.frame()
	mid := from.Add(to).Mul(0.5)
	for _, p := range []f32.Point{mid, to} {
		h.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: p, Buttons: pointer.ButtonPrimary})
		h.frame()
	}
	h.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: to})
	h.frame()
	h.frame()
}

// at returns the window point of day column day at hour:minute in the
// week's hours.
func (h *harness) at(day, hour, minute int) f32.Point {
	g := &h.v.grid
	x := g.origin.X + g.colX(day) + g.colW/2
	y := g.origin.Y + (hour*60+minute)*g.hourH/60
	return f32.Pt(float32(x), float32(y))
}

func newWeek(t *testing.T) (*harness, *[]string) {
	var log []string
	ev := sample()
	v := New().Locale(datetime.NewLocale("en-US")).Zone(time.UTC).SetEvents(ev).HourHeight(30).Header(false)
	v.now = fixedNow
	v.Show(fixedNow())
	v.OnCreate(func(s, e time.Time, allDay bool) {
		log = append(log, "create "+s.Format("Mon 15:04")+"-"+e.Format("15:04"))
	}).OnChange(func(e Event, s, en time.Time) {
		log = append(log, "change "+e.Title+" "+s.Format("Mon 15:04")+"-"+en.Format("15:04"))
	}).OnEventClick(func(e Event) { log = append(log, "click "+e.Title) })
	h := &harness{v: v, now: time.Unix(0, 0)}
	h.frame()
	// Show the hours from midnight.
	v.grid.scroll.ScrollTo(0)
	h.frame()
	h.frame()
	return h, &log
}

func TestWeekCreateMoveResize(t *testing.T) {
	h, log := newWeek(t)
	// Sunday 15 is column 0 in the US. Create Wednesday 14:00-15:30.
	h.drag(h.at(3, 14, 2), h.at(3, 15, 28))
	// Move Lunch (Wednesday 12:00) to Thursday 13:00, grabbing it 10
	// minutes in.
	h.drag(h.at(3, 12, 10), h.at(4, 13, 10))
	// Resize Planning (Friday 9:00-10:00) to end at 10:30.
	h.drag(h.at(5, 9, 56), h.at(5, 10, 30))
	// Click Retro.
	h.drag(h.at(5, 11, 30), h.at(5, 11, 30))
	want := []string{
		"create Wed 14:00-15:30",
		"change Lunch Thu 13:00-14:00",
		"change Planning Fri 09:00-10:30",
		"click Retro",
	}
	if len(*log) != len(want) {
		t.Fatalf("log = %q", *log)
	}
	for i := range want {
		if (*log)[i] != want[i] {
			t.Errorf("%d: %q, want %q", i, (*log)[i], want[i])
		}
	}
}

func TestMonthMoveAndCreate(t *testing.T) {
	var log []string
	v := New().Locale(datetime.NewLocale("en-US")).Zone(time.UTC).SetEvents(sample()).Mode(Month).Header(false)
	v.now = fixedNow
	v.Show(fixedNow())
	v.OnChange(func(e Event, s, en time.Time) { log = append(log, "change "+e.Title+" "+s.Format("Jan 2 15:04")) }).
		OnCreate(func(s, e time.Time, allDay bool) {
			log = append(log, "create "+s.Format("Jan 2")+"-"+e.Format("Jan 2"))
		})
	h := &harness{v: v, now: time.Unix(0, 0)}
	h.frame()
	h.frame()
	m := &v.month
	cell := func(day, dy int) f32.Point {
		r := m.cellRect(day).Add(m.origin)
		return f32.Pt(float32(r.Min.X+r.Dx()/2), float32(r.Min.Y+dy))
	}
	// Nov 1 is the grid's first cell. Lunch is on Wednesday 18 (cell
	// 17), its line under the number row; drop it on Monday 23 (22).
	var lunch image.Rectangle
	for _, hh := range m.hits {
		if hh.ev.Title == "Lunch" {
			lunch = hh.r.Add(m.origin)
		}
	}
	if lunch.Empty() {
		t.Fatal("Lunch not drawn")
	}
	c := lunch.Min.Add(lunch.Size().Div(2))
	h.drag(f32.Pt(float32(c.X), float32(c.Y)), cell(22, m.cellH/2))
	// Drag across empty days: 3rd to 5th.
	h.drag(cell(2, m.cellH-6), cell(4, m.cellH-6))
	want := []string{"change Lunch Nov 23 12:00", "create Nov 3-Nov 6"}
	if len(log) != len(want) || log[0] != want[0] || log[1] != want[1] {
		t.Errorf("log = %q, want %q", log, want)
	}
}
