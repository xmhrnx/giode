package calendarview

import (
	"testing"
	"time"
)

func at(d, h, m int) time.Time { return time.Date(2026, 11, d, h, m, 0, 0, time.UTC) }

func TestLayoutDay(t *testing.T) {
	a := &Event{Title: "a", Start: at(20, 9, 0), End: at(20, 10, 0)}
	b := &Event{Title: "b", Start: at(20, 9, 30), End: at(20, 11, 0)}
	c := &Event{Title: "c", Start: at(20, 10, 0), End: at(20, 10, 30)}
	d := &Event{Title: "d", Start: at(20, 13, 0), End: at(20, 14, 0)}
	night := &Event{Title: "night", Start: at(19, 22, 0), End: at(20, 2, 0)}
	ps := layoutDay(at(20, 0, 0), []*Event{d, c, b, a, night})
	got := map[string]placed{}
	for _, p := range ps {
		got[p.ev.Title] = p
	}
	// night (0:00-2:00) alone; a, b, c a cluster of two columns, c
	// reusing a's; d alone.
	if p := got["night"]; p.start != 0 || p.end != 120 || p.cols != 1 {
		t.Errorf("night = %+v", p)
	}
	if got["a"].cols != 2 || got["b"].cols != 2 || got["c"].cols != 2 {
		t.Errorf("cluster columns: a %d b %d c %d", got["a"].cols, got["b"].cols, got["c"].cols)
	}
	if got["a"].col != 0 || got["b"].col != 1 || got["c"].col != 0 {
		t.Errorf("columns: a %d b %d c %d", got["a"].col, got["b"].col, got["c"].col)
	}
	if p := got["d"]; p.cols != 1 || p.span != 1 || p.start != 13*60 {
		t.Errorf("d = %+v", p)
	}
}

func TestLayoutDayDST(t *testing.T) {
	ny, _ := time.LoadLocation("America/New_York")
	// November 1, 2026: clocks go back at 2:00; a 10:00 meeting still
	// sits at 10:00 on the grid.
	e := &Event{Start: time.Date(2026, 11, 1, 10, 0, 0, 0, ny), End: time.Date(2026, 11, 1, 11, 0, 0, 0, ny)}
	ps := layoutDay(time.Date(2026, 11, 1, 0, 0, 0, 0, ny), []*Event{e})
	if len(ps) != 1 || ps[0].start != 600 || ps[0].end != 660 {
		t.Errorf("placed %+v", ps)
	}
}

func TestLayoutWeek(t *testing.T) {
	monday := at(16, 0, 0)
	conf := &Event{Title: "conf", AllDay: true, Start: at(17, 0, 0), End: at(20, 0, 0)}       // Tue-Thu
	trip := &Event{Title: "trip", AllDay: true, Start: at(12, 0, 0), End: at(18, 0, 0)}       // until Tue, from before
	holiday := &Event{Title: "holiday", AllDay: true, Start: at(19, 0, 0), End: at(19, 0, 0)} // Thu
	late := &Event{Title: "late", Start: at(21, 22, 0), End: at(23, 1, 0)}                    // Sat night to Mon: past the week
	short := &Event{Title: "short", Start: at(18, 9, 0), End: at(18, 10, 0)}
	bars := layoutWeek(monday, []*Event{conf, trip, holiday, late, short}, spans)
	got := map[string]bar{}
	for _, b := range bars {
		got[b.ev.Title] = b
	}
	if _, ok := got["short"]; ok {
		t.Error("a timed event within a day got a bar")
	}
	if b := got["conf"]; b.first != 1 || b.last != 3 {
		t.Errorf("conf = %+v", b)
	}
	if b := got["trip"]; b.first != 0 || b.last != 1 || !b.cutStart {
		t.Errorf("trip = %+v", b)
	}
	if b := got["late"]; b.first != 5 || b.last != 6 || !b.cutEnd {
		t.Errorf("late = %+v", b)
	}
	// conf and trip overlap on Tuesday; holiday overlaps conf on
	// Thursday: three bars in two lanes.
	if got["conf"].lane == got["trip"].lane || got["holiday"].lane == got["conf"].lane {
		t.Errorf("lanes: conf %d trip %d holiday %d", got["conf"].lane, got["trip"].lane, got["holiday"].lane)
	}
	if snap(67, 15) != 60 || snap(68, 15) != 75 {
		t.Error("snap")
	}
}

func TestOvernightStaysInHours(t *testing.T) {
	night := &Event{Title: "night", Start: at(19, 22, 0), End: at(20, 2, 0)}
	long := &Event{Title: "long", Start: at(19, 9, 0), End: at(20, 9, 0)}
	if inAllDayRow(night) || !inAllDayRow(long) {
		t.Error("all-day row choice")
	}
	if ps := layoutDay(at(19, 0, 0), []*Event{night, long}); len(ps) != 1 || ps[0].ev != night || ps[0].start != 22*60 || ps[0].end != 24*60 {
		t.Errorf("the 19th's hours = %+v", ps)
	}
}
