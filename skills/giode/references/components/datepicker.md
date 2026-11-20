# Date and time fields

Date fields, time fields and a month calendar, in the locale's calendar (Persian in Iran),
order, first day of the week, names, digits and 12- or 24-hour clock. They're built on the
[`datetime`](../datetime.md) package.

```go
due := giode.Input.Date(giode.Styles{Width: 220}).Label("Due date").
	OnChange(func(day time.Time) { task.Due = day })

start := giode.Input.Time().Label("Starts").Step(15 * time.Minute).
	OnChange(func(t giode.TimeOfDay, ok bool) { meeting.Start = t })

stay := giode.Input.DateRange().Labels("Check-in", "Check-out").
	Limits(time.Now(), time.Time{}).
	OnChange(func(in, out time.Time) { booking.In, booking.Out = in, out })
```

## Date fields

`Input.Date` is a text field and a calendar button. A date typed in the locale's order
("11/20/2026" in the US, "20.11.2026" in Germany, "۱۴۰۵/۸/۲۹" in Iran) applies as soon as it
reads as one; separators don't matter, a two-digit year takes the nearest century, a month can
be named ("20 Nov"), and the year can be left out. Enter or leaving the field formats it, or
marks it invalid; an emptied field clears the date. The button, or Down in the field, opens the
calendar; picking a day closes it.

| | |
| --- | --- |
| `Value()`, `SetValue(day)` | the date, the start of its day; zero for none |
| `OnChange(func(time.Time))` | a new date, or zero when cleared |
| `Limits(first, last)` | the earliest and latest days it takes (zero leaves an end open) |
| `Disabled(func(day) bool)` | days it doesn't take: weekends, booked days |
| `Label`, `Required` | as on any field; `Field()` gives the text field itself |
| `Locale`, `Calendar`, `Zone` | another locale, calendar system or zone than the app's |

`Input.DateRange` is two date fields, start and end, with a two-month calendar: the first
click picks the start, the second the end, with the days between shown as the pointer moves.

## Time fields

`Input.Time` takes "3pm", "15:30", "1530", "3.30" or "۱۵:۳۰", and shows times in the locale's
clock. Its button opens a list of times (every 30 minutes; `Step` changes it), scrolled to the
chosen time or the present one. `Value()` returns a `TimeOfDay{Hour, Minute}` and whether one is
set; `TimeOfDay.On(day)` puts it on a date.

`Input.DateTime` is a date field and a time field together: `Value()` is the moment in its
`Zone`, built by the calendar, so 9:30 on the day clocks go forward is still 9:30.

## The calendar

`giode.DateCalendar()` is the calendar the fields open, for laying out on its own: a booking
page's month view, a sidebar's.

```go
cal := giode.DateCalendar().Months(2).Range(true).
	Disabled(func(day time.Time) bool { return booked[day] }).
	OnRange(func(start, end time.Time) { quote(start, end) }).
	OnMonth(func(first time.Time) { loadBookings(first) })
```

- **Keyboard**: arrows move by days and weeks, Page Up and Page Down by months (with Shift, by
  years), Home and End to the week's ends, Enter or Space picks. Right to left, the arrows and
  months mirror.
- **Screen readers** hear each day as a button named with its full date, selected or not,
  unavailable or not.
- Today has a ring; days of the neighboring months show faded and can be picked.
