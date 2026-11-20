# Dates and times

The `datetime` package holds what dates need beyond Go's `time` package: calendar systems,
locale conventions, formatting, relative times, time zones and day arithmetic that daylight
saving doesn't break. Giode's date pickers and calendar views are built on it.

```go
import "github.com/xmhrnx/giode/datetime"

datetime.Format(t, datetime.DateLong)     // "Friday, November 20, 2026" in en-US
datetime.Format(t, datetime.Time)         // "3:04 PM", or "15:04" where clocks are 24-hour
datetime.Relative(t, time.Now())          // "in 2 days", "yesterday", "5 minutes ago"
```

## Locales

`datetime.Current()` follows the app's locale (`i18n.Locale()`); `datetime.NewLocale("fa-IR")`
gives any other. A locale knows:

| | |
| --- | --- |
| `FirstWeekday()` | Monday in most of Europe, Sunday in the US, Japan, Brazil, Saturday in Iran and much of the Arab world |
| `Hour12()` | a 12-hour clock in the US, India, Australia, Egypt, Korea...; 24 hours elsewhere |
| `Calendar()` | Persian in Iran and Afghanistan, Gregorian elsewhere |
| `MonthName`, `WeekdayName`, `DayPeriod` | names in its language, wide, abbreviated or narrow |
| `Digits(s)` | Persian digits in Persian, Arabic-Indic digits in Arabic |
| `Format(t, style)`, `FormatPattern(t, "d MMM y")` | the locale's date order, names, calendar and digits |

Names come built in for English, German, French, Spanish, Italian, Portuguese, Dutch,
Russian, Polish, Turkish, Persian, Arabic, Japanese, Chinese and Korean; other languages fall
back to English until an app gives them: i18n keys `giode.month.gregorian.1` to `.12`,
`giode.month_short.gregorian.N`, `giode.weekday.0` (Sunday) to `.6`, `giode.weekday_short.N`
and `giode.weekday_narrow.N`, or `datetime.SetMonthNames`.

`SetFirstWeekday`, `SetHour12` and `SetCalendar` override the locale's choices for the whole
app, as a user setting would.

## Styles

| Style | en-US | de-DE | fa-IR |
| --- | --- | --- | --- |
| `DateShort` | 11/20/2026 | 20.11.2026 | ۱۴۰۵/۸/۲۹ |
| `DateMedium` | Nov 20, 2026 | 20. Nov. 2026 | ۲۹ آبان ۱۴۰۵ |
| `DateLong` | Friday, November 20, 2026 | Freitag, 20. November 2026 | جمعه ۲۹ آبان ۱۴۰۵ |
| `MonthYear` | November 2026 | November 2026 | آبان ۱۴۰۵ |
| `Time` | 3:04 PM | 15:04 | ۱۵:۰۴ |

`DayMonth`, `TimeSeconds`, `DateTimeShort` and `DateTimeMedium` complete the set.
`FormatPattern` takes Unicode's pattern letters: `y` year, `M`/`MM` month number, `MMM` short
name, `MMMM` name, `LLLL` the name as a title, `d` day, `EEE`/`EEEE` weekday, `h`/`H` hours,
`mm` minutes, `ss` seconds, `a` AM/PM, `z` the zone; text in single quotes stays as it is.

## Calendars

`datetime.Gregorian` and `datetime.Persian` are built in. A `Calendar` converts between
`time.Time` and its `Date{Year, Month, Day}` and says how long its months and years are, so
another one (Hijri, Hebrew, Thai Buddhist) is an implementation of five methods and its month
names away:

```go
type Calendar interface {
	ID() string
	FromTime(t time.Time) datetime.Date
	ToTime(d datetime.Date, loc *time.Location) time.Time
	MonthsInYear(year int) int
	DaysInMonth(year, month int) int
}
```

`AddMonths(cal, date, n)` moves by months in any calendar, keeping the day or taking the
month's last.

## Day arithmetic

A day isn't always 24 hours: the night clocks change it's 23 or 25. These count calendar
days:

- `AddDays(t, n)` keeps the wall time (09:30 stays 09:30 across a DST change);
- `StartOfDay(t)` is midnight, or the first hour that exists where a change skips midnight;
- `DaysBetween(a, b)`, `SameDay`, `StartOfWeek(t, first)`, `StartOfMonth(cal, t)`.

## Time zones

The package embeds the IANA time zone database, so `datetime.LoadZone("Asia/Tehran")` works on
Windows, Android and iOS too (about 450 kB in the binary). `ZoneLabel(t)` names an offset
("GMT+3:30"), `OffsetBetween` says how far apart two zones are, and
`Locale.InZones(t, theirs, style)` formats a time "in your time" and "in their time".

## Relative times

`Relative(t, now)` says "now", "5 minutes ago", "in 3 hours", "yesterday", "tomorrow", "in 2
days", "3 weeks ago", "in 4 months", "2 years ago", counting days by the calendar. Its texts
are i18n plural keys (`giode.rel.minutes_ago`, `giode.rel.days_in`, ...), built in for English,
German, French, Spanish, Russian and Persian, and overridden by the app's messages.
