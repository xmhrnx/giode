package calendarview

import (
	"sort"
	"time"

	"github.com/xmhrnx/giode/datetime"
)

// Event placement, apart from drawing: which column of a day's time grid
// a timed event takes among those it overlaps, and which lane of a
// week's all-day row a spanning event takes.

// placed is an event's place in a day's time grid: column col of cols,
// from start to end minutes past midnight.
type placed struct {
	ev         *Event
	start, end int
	col, cols  int
	// span is how many columns it may widen into, free ones to its
	// right.
	span int
}

// layoutDay places the timed events of day (those overlapping it, cut
// to it) in columns: overlapping events side by side, each cluster of
// overlapping events sharing its width.
func layoutDay(day time.Time, events []*Event) []placed {
	return layoutDayFiltered(day, events, func(e *Event) bool { return !inAllDayRow(e) })
}

// inAllDayRow reports whether e shows in the week and day views' all-day
// row rather than among the hours: all-day events and those of a day or
// more. An overnight event stays among the hours of both days.
func inAllDayRow(e *Event) bool {
	return e.AllDay || e.End.Sub(e.Start) >= 24*time.Hour
}

func layoutDayFiltered(day time.Time, events []*Event, keep func(*Event) bool) []placed {
	dayStart := datetime.StartOfDay(day)
	dayEnd := datetime.StartOfDay(datetime.AddDays(day, 1))
	var ps []placed
	for _, e := range events {
		if !keep(e) || !e.Start.Before(dayEnd) || !e.End.After(dayStart) {
			continue
		}
		s, en := e.Start.In(day.Location()), e.End.In(day.Location())
		if s.Before(dayStart) {
			s = dayStart
		}
		if en.After(dayEnd) {
			en = dayEnd
		}
		// Minutes on the day's clock, so a 23- or 25-hour day keeps its
		// hours where the grid draws them.
		sm := minutesOf(dayStart, s)
		em := minutesOf(dayStart, en)
		if em <= sm {
			em = sm + 15 // a zero-length event still shows
		}
		ps = append(ps, placed{ev: e, start: sm, end: em})
	}
	sort.SliceStable(ps, func(i, j int) bool {
		if ps[i].start != ps[j].start {
			return ps[i].start < ps[j].start
		}
		return ps[i].end-ps[i].start > ps[j].end-ps[j].start
	})
	// Clusters: runs of events each overlapping the run so far.
	for i := 0; i < len(ps); {
		j, clusterEnd := i, ps[i].end
		var colEnds []int
		for ; j < len(ps) && ps[j].start < clusterEnd; j++ {
			clusterEnd = max(clusterEnd, ps[j].end)
			col := -1
			for c, end := range colEnds {
				if end <= ps[j].start {
					col = c
					break
				}
			}
			if col < 0 {
				col = len(colEnds)
				colEnds = append(colEnds, 0)
			}
			colEnds[col] = ps[j].end
			ps[j].col = col
		}
		for k := i; k < j; k++ {
			ps[k].cols = len(colEnds)
			// Widen into the free columns to the right.
			ps[k].span = 1
			for c := ps[k].col + 1; c < len(colEnds); c++ {
				free := true
				for m := i; m < j; m++ {
					if ps[m].col == c && ps[m].start < ps[k].end && ps[m].end > ps[k].start {
						free = false
						break
					}
				}
				if !free {
					break
				}
				ps[k].span++
			}
		}
		i = j
	}
	return ps
}

// minutesOf returns the minutes on dayStart's clock at t: 9:30 is 570
// whatever daylight saving did that night.
func minutesOf(dayStart, t time.Time) int {
	if datetime.DaysBetween(dayStart, t) > 0 {
		return 24 * 60
	}
	return t.Hour()*60 + t.Minute()
}

// bar is an event's place in a week's all-day row: from day first to
// day last (0 to 6, inclusive) in lane.
type bar struct {
	ev          *Event
	first, last int
	lane        int
	// cutStart and cutEnd mark an event that goes on before or after the
	// week.
	cutStart, cutEnd bool
}

// spans reports whether e shows in the all-day row or the month's bars:
// all-day events and those crossing midnight.
func spans(e *Event) bool {
	if e.AllDay {
		return true
	}
	end := e.End.Add(-time.Nanosecond)
	return !datetime.SameDay(e.Start, end)
}

// layoutWeek places the events keep takes (nil: all) of the seven days
// from start in lanes: each in the first lane free over all its days.
func layoutWeek(start time.Time, events []*Event, keep func(*Event) bool) []bar {
	weekEnd := datetime.AddDays(start, 7)
	var bars []bar
	for _, e := range events {
		if keep != nil && !keep(e) {
			continue
		}
		s := e.Start.In(start.Location())
		en := e.End.In(start.Location())
		if e.AllDay {
			// An all-day event's end is the start of the day after.
			en = datetime.StartOfDay(en)
			if !en.After(datetime.StartOfDay(s)) {
				en = datetime.AddDays(datetime.StartOfDay(s), 1)
			}
		}
		if !s.Before(weekEnd) || !en.After(start) {
			continue
		}
		b := bar{ev: e}
		if s.Before(start) {
			b.first, b.cutStart = 0, true
		} else {
			b.first = datetime.DaysBetween(start, s)
		}
		last := en.Add(-time.Nanosecond)
		if !last.Before(weekEnd) {
			b.last, b.cutEnd = 6, true
		} else {
			b.last = datetime.DaysBetween(start, last)
		}
		if b.last < b.first {
			b.last = b.first
		}
		bars = append(bars, b)
	}
	// Longer bars first, then by start, so they take the top lanes.
	sort.SliceStable(bars, func(i, j int) bool {
		li, lj := bars[i].last-bars[i].first, bars[j].last-bars[j].first
		if li != lj {
			return li > lj
		}
		if bars[i].first != bars[j].first {
			return bars[i].first < bars[j].first
		}
		return bars[i].ev.Start.Before(bars[j].ev.Start)
	})
	var lanes [][7]bool
	for i := range bars {
		b := &bars[i]
		lane := 0
		for ; ; lane++ {
			if lane == len(lanes) {
				lanes = append(lanes, [7]bool{})
			}
			free := true
			for d := b.first; d <= b.last; d++ {
				if lanes[lane][d] {
					free = false
					break
				}
			}
			if free {
				break
			}
		}
		for d := b.first; d <= b.last; d++ {
			lanes[lane][d] = true
		}
		b.lane = lane
	}
	return bars
}

// snap rounds minutes to step.
func snap(minutes, step int) int {
	if step <= 1 {
		return minutes
	}
	return (minutes + step/2) / step * step
}
