// Package calendarview provides calendar views of events: a month, a
// week, a day and an agenda, with overlapping events side by side,
// all-day and multi-day events in bars, dragging to create, move and
// resize events, the current time, and time zones.
package calendarview

import "time"

// Event is something on the calendar. Start and End are moments; an
// AllDay event covers the days from Start's to the day before End's
// (End the start of the day after, as in iCalendar), or Start's day
// alone when End isn't after it.
type Event struct {
	// ID identifies it to the app.
	ID     string
	Title  string
	Start  time.Time
	End    time.Time
	AllDay bool
	// Color is its color, a theme token or any color Styles take;
	// "theme.primary" when empty.
	Color string
	// ReadOnly events can't be dragged.
	ReadOnly bool
	// Data is the app's.
	Data any
}
