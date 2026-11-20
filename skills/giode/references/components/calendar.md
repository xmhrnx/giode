# Calendar views

`giode.CalendarView()` shows events by month, week, day or as an agenda, in the locale's
calendar, first day of the week, names, digits and clock, and lets people create, move and
resize events by dragging.

```go
cal := giode.CalendarView().
	SetEvents(events).
	OnEventClick(func(e giode.CalendarEvent) { openEvent(e.ID) }).
	OnCreate(func(start, end time.Time, allDay bool) { draft(start, end, allDay) }).
	OnChange(func(e giode.CalendarEvent, start, end time.Time) {
		reschedule(e.ID, start, end) // then SetEvents with the new times
	})
```

An event is a `CalendarEvent{ID, Title, Start, End, AllDay, Color, ReadOnly, Data}`. An
all-day event's `End` is the start of the day after its last, as in iCalendar.

## Views

| | |
| --- | --- |
| `CalendarMonth` | six weeks of days: events of a day or more, and those crossing midnight, as bars across their days, in lanes; others as lines with their time; "+N more" past the room, which opens that day |
| `CalendarWeek` | seven days of hours under a row of all-day events; overlapping events side by side, each cluster sharing its width, an event widening into free columns; the current time as a line |
| `CalendarDay` | one day of hours |
| `CalendarAgenda` | the events of the coming days (`AgendaDays`, 30 by default), grouped by day |

The header has Today, back and forward, the period's title, and a switcher of the `Modes` it
offers; `Header(false)` hides it for an app's own. `Mode`, `Show(t)`, `Range()` and
`CurrentMode()` drive it from code.

## Creating, moving, resizing

- **Week and day**: dragging over empty hours creates an event (a click, an hour); dragging an
  event moves it, to another day too; dragging its bottom edge resizes it. Times snap to
  `Snap` minutes (15 by default). On touch a long press starts each, so a swipe still scrolls.
- **Month**: dragging an event to another day moves it by days; dragging across empty days
  creates an all-day event over them; clicking a day's number opens it (`OnDayClick` takes
  that over).

The view doesn't change events itself: `OnCreate` and `OnChange` say what was asked, and the
app saves it and passes the new events. `ReadOnly` events don't move.

## Lots of events

`Source(func(from, to time.Time) []CalendarEvent)` asks for each period's events as it's
shown, in place of `SetEvents`, and `OnRange` reports a new period, to fetch it from a server.

## Time zones

Times show in `Zone` (`time.Local` by default), so the same events read right for people in
other places. `SecondZone(z)` adds a column of hours in another zone beside the week and day
views, labelled with both zones' offsets: a customer's hours beside yours. Days are counted by
the calendar, so a meeting at 10:00 sits at 10:00 on the day clocks change.

## Accessibility

Each event is a button for screen readers, named with its title, date and times; clicking it
from a screen reader calls `OnEventClick`. Right to left, days run from the right and the
arrows mirror.
