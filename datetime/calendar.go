// Package datetime holds what dates and times need beyond the time
// package: calendar systems (Gregorian, Persian, or an app's own),
// locale conventions (first day of the week, 12 or 24 hours, month and
// weekday names, date patterns, digits), formatting, relative times
// ("in 2 days", "yesterday"), time zones, and day arithmetic that
// daylight saving doesn't break. Giode's date pickers and calendar views
// are built on it.
//
// Importing it embeds the IANA time zone database (time/tzdata), so
// zones load on every platform, Windows and phones included.
package datetime

import (
	"time"
	_ "time/tzdata"
)

// Date is a day in a calendar system: Month from 1, Day from 1.
type Date struct {
	Year, Month, Day int
}

// Calendar is a calendar system. Gregorian and Persian are built in; an
// app adds another (Hijri, Hebrew, Buddhist...) by implementing it and
// passing it to SetCalendar, with its month names (SetMonthNames).
type Calendar interface {
	// ID names it: "gregorian", "persian".
	ID() string
	// FromTime returns the date of t, in t's location.
	FromTime(t time.Time) Date
	// ToTime returns the start of d in loc. Days past the end of a
	// month roll over into the next, as time.Date does.
	ToTime(d Date, loc *time.Location) time.Time
	// MonthsInYear and DaysInMonth describe the calendar.
	MonthsInYear(year int) int
	DaysInMonth(year, month int) int
}

// Gregorian is the Gregorian calendar.
var Gregorian Calendar = gregorian{}

type gregorian struct{}

func (gregorian) ID() string { return "gregorian" }

func (gregorian) FromTime(t time.Time) Date {
	y, m, d := t.Date()
	return Date{y, int(m), d}
}

func (gregorian) ToTime(d Date, loc *time.Location) time.Time {
	return StartOfDay(time.Date(d.Year, time.Month(d.Month), d.Day, 12, 0, 0, 0, loc))
}

func (gregorian) MonthsInYear(int) int { return 12 }

func (gregorian) DaysInMonth(year, month int) int {
	return time.Date(year, time.Month(month)+1, 0, 12, 0, 0, 0, time.UTC).Day()
}

// AddMonths returns d moved by n months in cal, its day kept or, past the
// end of the new month, its last day (January 31 + 1 month is February
// 28 or 29).
func AddMonths(cal Calendar, d Date, n int) Date {
	y, m := d.Year, d.Month+n
	for m > cal.MonthsInYear(y) {
		m -= cal.MonthsInYear(y)
		y++
	}
	for m < 1 {
		y--
		m += cal.MonthsInYear(y)
	}
	return Date{y, m, min(d.Day, cal.DaysInMonth(y, m))}
}

// Before reports whether d is before e.
func (d Date) Before(e Date) bool {
	if d.Year != e.Year {
		return d.Year < e.Year
	}
	if d.Month != e.Month {
		return d.Month < e.Month
	}
	return d.Day < e.Day
}
