package datetime

import (
	"strings"
	"time"
)

// Style is a date or time format of a locale.
type Style int

const (
	// DateShort is all numbers: 11/20/2026, 20.11.2026, ۱۴۰۵/۸/۲۹.
	DateShort Style = iota
	// DateMedium names the month briefly: Nov 20, 2026.
	DateMedium
	// DateLong names the weekday and month: Friday, November 20, 2026.
	DateLong
	// MonthYear is a calendar's header: November 2026.
	MonthYear
	// DayMonth is a date without the year: Nov 20.
	DayMonth
	// Time is hours and minutes: 3:04 PM or 15:04.
	Time
	// TimeSeconds adds seconds.
	TimeSeconds
	// DateTimeShort is DateShort and Time.
	DateTimeShort
	// DateTimeMedium is DateMedium and Time.
	DateTimeMedium
)

// patterns are a language's date patterns, in the letters of Unicode's
// date format patterns: y year, M month (MMM short name, MMMM name),
// L month standing alone (LLLL its name as a title), d day, EEE and
// EEEE weekday, h and H hours, mm minutes, ss seconds, a AM/PM; text
// in single quotes is kept.
type patterns struct {
	short, medium, long, monthYear, dayMonth string
}

var datePatterns = map[string]patterns{
	"en":    {"dd/MM/y", "d MMM y", "EEEE, d MMMM y", "MMMM y", "d MMM"},
	"en-US": {"M/d/y", "MMM d, y", "EEEE, MMMM d, y", "MMMM y", "MMM d"},
	"en-CA": {"y-MM-dd", "MMM d, y", "EEEE, MMMM d, y", "MMMM y", "MMM d"},
	"en-PH": {"M/d/y", "MMM d, y", "EEEE, MMMM d, y", "MMMM y", "MMM d"},
	"de":    {"dd.MM.y", "d. MMM y", "EEEE, d. MMMM y", "MMMM y", "d. MMM"},
	"fr":    {"dd/MM/y", "d MMM y", "EEEE d MMMM y", "MMMM y", "d MMM"},
	"es":    {"d/M/y", "d MMM y", "EEEE, d 'de' MMMM 'de' y", "MMMM 'de' y", "d MMM"},
	"it":    {"dd/MM/y", "d MMM y", "EEEE d MMMM y", "MMMM y", "d MMM"},
	"pt":    {"dd/MM/y", "d 'de' MMM 'de' y", "EEEE, d 'de' MMMM 'de' y", "MMMM 'de' y", "d 'de' MMM"},
	"nl":    {"dd-MM-y", "d MMM y", "EEEE d MMMM y", "MMMM y", "d MMM"},
	"ru":    {"dd.MM.y", "d MMM y 'г'.", "EEEE, d MMMM y 'г'.", "LLLL y", "d MMM"},
	"pl":    {"dd.MM.y", "d MMM y", "EEEE, d MMMM y", "LLLL y", "d MMM"},
	"tr":    {"dd.MM.y", "d MMM y", "d MMMM y EEEE", "MMMM y", "d MMM"},
	"fa":    {"y/M/d", "d MMMM y", "EEEE d MMMM y", "MMMM y", "d MMMM"},
	"ar":    {"d/M/y", "d MMMM y", "EEEE، d MMMM y", "MMMM y", "d MMMM"},
	"ja":    {"y/MM/dd", "y年M月d日", "y年M月d日EEEE", "y年M月", "M月d日"},
	"zh":    {"y/M/d", "y年M月d日", "y年M月d日EEEE", "y年M月", "M月d日"},
	"ko":    {"y. M. d.", "y년 M월 d일", "y년 M월 d일 EEEE", "y년 M월", "M월 d일"},
	"sv":    {"y-MM-dd", "d MMM y", "EEEE d MMMM y", "MMMM y", "d MMM"},
}

func (l Locale) patterns() patterns {
	if p, ok := datePatterns[l.lang+"-"+l.region]; ok {
		return p
	}
	if p, ok := datePatterns[l.lang]; ok {
		return p
	}
	return datePatterns["en"]
}

// timePattern is the locale's hours and minutes, with seconds.
func (l Locale) timePattern(seconds bool) string {
	sec := ""
	if seconds {
		sec = ":ss"
	}
	if l.Hour12() {
		switch l.lang {
		case "ja", "zh", "ko":
			return "a h:mm" + sec
		}
		return "h:mm" + sec + " a"
	}
	return "HH:mm" + sec
}

// Format formats t in the locale's style, in its calendar and digits.
func (l Locale) Format(t time.Time, s Style) string {
	p := l.patterns()
	switch s {
	case DateShort:
		return l.FormatPattern(t, p.short)
	case DateMedium:
		return l.FormatPattern(t, p.medium)
	case DateLong:
		return l.FormatPattern(t, p.long)
	case MonthYear:
		return l.FormatPattern(t, p.monthYear)
	case DayMonth:
		return l.FormatPattern(t, p.dayMonth)
	case Time:
		return l.FormatPattern(t, l.timePattern(false))
	case TimeSeconds:
		return l.FormatPattern(t, l.timePattern(true))
	case DateTimeShort:
		return l.FormatPattern(t, p.short+" "+l.timePattern(false))
	case DateTimeMedium:
		return l.FormatPattern(t, p.medium+" "+l.timePattern(false))
	}
	return t.String()
}

// FormatPattern formats t with a Unicode date pattern (see Style's
// letters), in the locale's calendar, names and digits.
func (l Locale) FormatPattern(t time.Time, pattern string) string {
	cal := l.Calendar()
	d := cal.FromTime(t)
	inDate := strings.ContainsRune(pattern, 'd')
	var b strings.Builder
	r := []rune(pattern)
	for i := 0; i < len(r); {
		c := r[i]
		if c == '\'' {
			j := i + 1
			for j < len(r) && r[j] != '\'' {
				j++
			}
			if j == i+1 {
				b.WriteRune('\'')
			} else {
				b.WriteString(string(r[i+1 : j]))
			}
			i = j + 1
			continue
		}
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z') {
			b.WriteRune(c)
			i++
			continue
		}
		n := 1
		for i+n < len(r) && r[i+n] == c {
			n++
		}
		i += n
		switch c {
		case 'y':
			if n == 2 {
				b.WriteString(l.Digits(pad(d.Year%100, 2)))
			} else {
				b.WriteString(l.Digits(itoa(d.Year)))
			}
		case 'M', 'L':
			switch {
			case n >= 4:
				b.WriteString(l.monthName(cal, d.Month, Wide, inDate && c == 'M'))
			case n == 3:
				b.WriteString(l.monthName(cal, d.Month, Abbreviated, false))
			default:
				b.WriteString(l.Digits(pad(d.Month, n)))
			}
		case 'd':
			b.WriteString(l.Digits(pad(d.Day, n)))
		case 'E':
			w := Abbreviated
			switch {
			case n >= 5:
				w = Narrow
			case n == 4:
				w = Wide
			}
			b.WriteString(l.WeekdayName(t.Weekday(), w))
		case 'H':
			b.WriteString(l.Digits(pad(t.Hour(), n)))
		case 'h':
			h := t.Hour() % 12
			if h == 0 {
				h = 12
			}
			b.WriteString(l.Digits(pad(h, n)))
		case 'm':
			b.WriteString(l.Digits(pad(t.Minute(), n)))
		case 's':
			b.WriteString(l.Digits(pad(t.Second(), n)))
		case 'a':
			b.WriteString(l.DayPeriod(t.Hour()))
		case 'z':
			b.WriteString(ZoneLabel(t))
		default:
			b.WriteString(strings.Repeat(string(c), n))
		}
	}
	return b.String()
}

func pad(v, n int) string {
	s := itoa(v)
	for len(s) < n {
		s = "0" + s
	}
	return s
}

// Format formats t in the current locale (Current).
func Format(t time.Time, s Style) string { return Current().Format(t, s) }
