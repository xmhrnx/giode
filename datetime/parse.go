package datetime

import (
	"errors"
	"strings"
	"unicode"
)

// ErrNoDate and ErrNoTime are what the parsers return for text they
// can't read.
var (
	ErrNoDate = errors.New("datetime: not a date")
	ErrNoTime = errors.New("datetime: not a time")
)

// asciiDigits writes Persian, Arabic-Indic and other Unicode digits as
// ASCII ones.
func asciiDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) && (r < '0' || r > '9') {
			switch {
			case r >= '۰' && r <= '۹':
				r = '0' + (r - '۰')
			case r >= '٠' && r <= '٩':
				r = '0' + (r - '٠')
			}
		}
		b.WriteRune(r)
	}
	return b.String()
}

// fieldOrder returns the order of year, month and day in the locale's
// short date: "ymd", "mdy" or "dmy".
func (l Locale) fieldOrder() string {
	var order []byte
	for _, c := range l.patterns().short {
		switch c {
		case 'y', 'M', 'd':
			b := byte(unicode.ToLower(c))
			if len(order) == 0 || order[len(order)-1] != b {
				order = append(order, b)
			}
		}
	}
	return string(order)
}

// ParseDate reads a date typed in the locale's order ("11/20/2026" in
// the US, "20.11.2026" in Germany, "1405/8/29" in Iran), with any
// separators, native digits, a two-digit year (the nearest century), a
// month by name ("20 Nov 2026"), or a missing year (this year's, from
// thisYear).
func (l Locale) ParseDate(s string, thisYear int) (Date, error) {
	s = strings.ToLower(asciiDigits(strings.TrimSpace(s)))
	if s == "" {
		return Date{}, ErrNoDate
	}
	cal := l.Calendar()
	// Split into numbers and words.
	var nums []string
	month := 0
	for _, f := range strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		if unicode.IsDigit([]rune(f)[0]) {
			// "20nov" or "2026-11-20" glued: split digits from letters.
			digits := strings.TrimRightFunc(f, unicode.IsLetter)
			nums = append(nums, digits)
			f = f[len(digits):]
			if f == "" {
				continue
			}
		}
		if m := l.monthByName(cal, f); m > 0 {
			month = m
		}
	}
	// Digits alone: "20261120" or "112026".
	if len(nums) == 1 && month == 0 && len(nums[0]) == 8 {
		n := nums[0]
		switch l.fieldOrder() {
		case "ymd":
			nums = []string{n[:4], n[4:6], n[6:]}
		case "mdy", "dmy":
			nums = []string{n[:2], n[2:4], n[4:]}
		}
	}
	var y, m, d int
	order := l.fieldOrder()
	switch {
	case month > 0 && len(nums) >= 1:
		m = month
		d = atoi(nums[0])
		if len(nums) >= 2 {
			// "2026 Nov 20" puts the year first.
			if len(nums[0]) >= 3 {
				y, d = atoi(nums[0]), atoi(nums[1])
			} else {
				y = atoi(nums[1])
			}
		}
	case len(nums) == 3:
		vals := map[byte]int{}
		for i := range 3 {
			vals[order[i]] = atoi(nums[i])
		}
		// A four-digit first number is a year whatever the order.
		if len(nums[0]) == 4 {
			vals = map[byte]int{'y': atoi(nums[0]), 'm': atoi(nums[1]), 'd': atoi(nums[2])}
		}
		y, m, d = vals['y'], vals['m'], vals['d']
	case len(nums) == 2:
		// Month and day, this year.
		vals := map[byte]int{}
		i := 0
		for _, f := range order {
			if f != 'y' && i < 2 {
				vals[byte(f)] = atoi(nums[i])
				i++
			}
		}
		m, d = vals['m'], vals['d']
	default:
		return Date{}, ErrNoDate
	}
	if y == 0 {
		y = thisYear
	} else if y < 100 {
		// The nearest century to this year.
		c := thisYear / 100 * 100
		y += c
		if y > thisYear+50 {
			y -= 100
		} else if y < thisYear-50 {
			y += 100
		}
	}
	if m < 1 || m > cal.MonthsInYear(y) || d < 1 || d > cal.DaysInMonth(y, m) {
		return Date{}, ErrNoDate
	}
	return Date{y, m, d}, nil
}

// monthByName returns the month a word names, wide or short, in the
// locale's language or English; 0 if none.
func (l Locale) monthByName(cal Calendar, w string) int {
	if len([]rune(w)) < 3 {
		return 0
	}
	for _, lang := range []string{l.lang, "en"} {
		names := lookupNames(cal.ID(), lang)
		if names == nil {
			continue
		}
		for i := range names.wide {
			for _, list := range [][]string{names.wide, names.short, names.genitive} {
				if i < len(list) && strings.HasPrefix(strings.ToLower(strings.TrimSuffix(list[i], ".")), w) {
					return i + 1
				}
			}
		}
	}
	return 0
}

// ParseTime reads a time of day: "15:04", "3:04 pm", "3pm", "1504",
// "3.04", "۱۵:۰۴", with the locale's or English AM/PM. It returns hours
// (0 to 23) and minutes.
func (l Locale) ParseTime(s string) (h, m int, err error) {
	s = strings.ToLower(asciiDigits(strings.TrimSpace(s)))
	if s == "" {
		return 0, 0, ErrNoTime
	}
	pm, am := false, false
	p := dayPeriods[l.lang]
	for _, mark := range []struct {
		text string
		pm   bool
	}{{strings.ToLower(p[0]), false}, {strings.ToLower(p[1]), true}, {"am", false}, {"pm", true}, {"a", false}, {"p", true}} {
		if mark.text == "" {
			continue
		}
		if i := strings.Index(s, mark.text); i >= 0 && !(len(mark.text) == 1 && i+1 < len(s) && unicode.IsLetter(rune(s[i+1]))) {
			if mark.pm {
				pm = true
			} else {
				am = true
			}
			s = s[:i] + s[i+len(mark.text):]
			break
		}
	}
	var nums []string
	for _, f := range strings.FieldsFunc(s, func(r rune) bool { return !unicode.IsDigit(r) }) {
		nums = append(nums, f)
	}
	switch {
	case len(nums) == 1 && len(nums[0]) <= 2:
		h = atoi(nums[0])
	case len(nums) == 1 && len(nums[0]) <= 4:
		n := nums[0]
		h, m = atoi(n[:len(n)-2]), atoi(n[len(n)-2:])
	case len(nums) >= 2:
		h, m = atoi(nums[0]), atoi(nums[1])
	default:
		return 0, 0, ErrNoTime
	}
	if pm || am {
		if h < 1 || h > 12 {
			return 0, 0, ErrNoTime
		}
		h %= 12
		if pm {
			h += 12
		}
	}
	if h > 23 || m > 59 {
		return 0, 0, ErrNoTime
	}
	return h, m, nil
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return -1
		}
		n = n*10 + int(c-'0')
		if n > 1<<20 {
			return -1
		}
	}
	return n
}
