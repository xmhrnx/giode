package datetime

import (
	"testing"
	"time"
)

func TestPersian(t *testing.T) {
	for _, c := range []struct {
		g    time.Time
		date Date
	}{
		{time.Date(2024, 3, 20, 10, 0, 0, 0, time.UTC), Date{1403, 1, 1}},
		{time.Date(2025, 3, 21, 10, 0, 0, 0, time.UTC), Date{1404, 1, 1}},
		{time.Date(2025, 3, 20, 10, 0, 0, 0, time.UTC), Date{1403, 12, 30}},
		{time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC), Date{1405, 8, 29}},
		{time.Date(1979, 2, 11, 10, 0, 0, 0, time.UTC), Date{1357, 11, 22}},
	} {
		if got := Persian.FromTime(c.g); got != c.date {
			t.Errorf("%s = %v, want %v", c.g.Format("2006-01-02"), got, c.date)
		}
		if back := Persian.ToTime(c.date, time.UTC); !SameDay(back, c.g) {
			t.Errorf("%v back = %s", c.date, back)
		}
	}
	if Persian.DaysInMonth(1403, 12) != 30 || Persian.DaysInMonth(1404, 12) != 29 || Persian.DaysInMonth(1404, 7) != 30 {
		t.Error("month lengths")
	}
	if d := AddMonths(Persian, Date{1403, 6, 31}, 1); d != (Date{1403, 7, 30}) {
		t.Errorf("Shahrivar 31 + 1 month = %v", d)
	}
}

func TestDayArithmetic(t *testing.T) {
	ny, _ := time.LoadLocation("America/New_York")
	// The night clocks went forward: a 23-hour day.
	before := time.Date(2026, 3, 7, 9, 30, 0, 0, ny)
	after := AddDays(before, 1)
	if after.Hour() != 9 || after.Day() != 8 {
		t.Errorf("a day after 9:30 = %s", after)
	}
	if DaysBetween(before, after) != 1 || after.Sub(before) != 23*time.Hour {
		t.Errorf("between = %d, %v", DaysBetween(before, after), after.Sub(before))
	}
	// Midnight skipped (Santiago, DST at 00:00).
	scl, _ := time.LoadLocation("America/Santiago")
	s := StartOfDay(time.Date(2026, 9, 6, 15, 0, 0, 0, scl))
	if s.Day() != 6 {
		t.Errorf("start of a day without midnight = %s", s)
	}
	if w := StartOfWeek(time.Date(2026, 11, 20, 15, 0, 0, 0, time.UTC), time.Monday); w.Day() != 16 {
		t.Errorf("week of Friday the 20th starts %s", w)
	}
	if d := AddMonths(Gregorian, Date{2026, 1, 31}, 1); d != (Date{2026, 2, 28}) {
		t.Errorf("January 31 + 1 month = %v", d)
	}
	if d := AddMonths(Gregorian, Date{2026, 1, 15}, -2); d != (Date{2025, 11, 15}) {
		t.Errorf("January 15 - 2 months = %v", d)
	}
}
