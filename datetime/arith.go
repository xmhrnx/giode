package datetime

import "time"

// Day arithmetic by the calendar, not by 24-hour durations: across a
// daylight saving change a day is 23 or 25 hours, and adding 24 hours
// lands on the wrong wall time.

// StartOfDay returns the first instant of t's day in t's location
// (midnight, or later where a DST change skips midnight).
func StartOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	s := time.Date(y, m, d, 0, 0, 0, 0, t.Location())
	if s.Day() != d {
		// Midnight doesn't exist that day: the first hour that does.
		for h := 1; h < 24; h++ {
			if s = time.Date(y, m, d, h, 0, 0, 0, t.Location()); s.Day() == d {
				break
			}
		}
	}
	return s
}

// AddDays returns t moved by n calendar days, its wall time kept.
func AddDays(t time.Time, n int) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d+n, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}

// DaysBetween returns the number of calendar days from a's day to b's,
// in a's location: 1 from 23:00 to 01:00 the next day.
func DaysBetween(a, b time.Time) int {
	b = b.In(a.Location())
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	ua := time.Date(ay, am, ad, 0, 0, 0, 0, time.UTC)
	ub := time.Date(by, bm, bd, 0, 0, 0, 0, time.UTC)
	return int(ub.Sub(ua).Hours() / 24)
}

// SameDay reports whether a and b fall on the same day in a's location.
func SameDay(a, b time.Time) bool { return DaysBetween(a, b) == 0 }

// StartOfWeek returns the start of the week holding t, weeks starting
// on first.
func StartOfWeek(t time.Time, first time.Weekday) time.Time {
	back := (int(t.Weekday()) - int(first) + 7) % 7
	return StartOfDay(AddDays(t, -back))
}

// StartOfMonth returns the start of the month holding t in cal.
func StartOfMonth(cal Calendar, t time.Time) time.Time {
	d := cal.FromTime(t)
	return cal.ToTime(Date{d.Year, d.Month, 1}, t.Location())
}
