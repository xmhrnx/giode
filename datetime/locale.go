package datetime

import (
	"strings"
	"sync"
	"time"

	"github.com/xmhrnx/giode/i18n"
)

// Locale is the date and time conventions of a language and region,
// from a tag such as "en-US", "fa-IR" or "de".
type Locale struct {
	tag, lang, region string
}

// NewLocale returns the conventions of a locale tag.
func NewLocale(tag string) Locale {
	tag = strings.ReplaceAll(tag, "_", "-")
	if i := strings.IndexAny(tag, ".@"); i >= 0 {
		tag = tag[:i]
	}
	lang, region, _ := strings.Cut(tag, "-")
	lang = strings.ToLower(lang)
	region = strings.ToUpper(region)
	// "zh-Hant-TW": the region is the last two-letter part.
	if parts := strings.Split(tag, "-"); len(parts) > 2 {
		region = strings.ToUpper(parts[len(parts)-1])
	}
	if len(region) != 2 {
		region = ""
	}
	if region == "" {
		region = defaultRegion[lang]
	}
	return Locale{tag: tag, lang: lang, region: region}
}

// Current returns the conventions of the app's locale (i18n.Locale),
// with the overrides set here (SetFirstWeekday, SetHour12, SetCalendar).
func Current() Locale { return NewLocale(i18n.Locale()) }

// Tag returns the locale's tag.
func (l Locale) Tag() string { return l.tag }

// Language returns its language code.
func (l Locale) Language() string { return l.lang }

// The regions a bare language means.
var defaultRegion = map[string]string{
	"en": "US", "fa": "IR", "ar": "EG", "he": "IL", "ja": "JP", "ko": "KR", "zh": "CN",
	"pt": "BR", "hi": "IN", "de": "DE", "fr": "FR", "es": "ES", "it": "IT", "nl": "NL",
	"ru": "RU", "pl": "PL", "tr": "TR", "sv": "SE", "uk": "UA",
}

var (
	overrideMu    sync.RWMutex
	firstOverride = -1
	hourOverride  = -1
	calOverride   Calendar
)

// SetFirstWeekday makes weeks start on d whatever the locale; -1 goes
// back to the locale's.
func SetFirstWeekday(d time.Weekday) {
	overrideMu.Lock()
	firstOverride = int(d)
	overrideMu.Unlock()
}

// SetHour12 makes times use 12 hours (1) or 24 (0) whatever the locale;
// -1 goes back to the locale's.
func SetHour12(v int) {
	overrideMu.Lock()
	hourOverride = v
	overrideMu.Unlock()
}

// SetCalendar makes dates use cal whatever the locale; nil goes back to
// the locale's (Persian in Iran and Afghanistan, Gregorian elsewhere).
func SetCalendar(cal Calendar) {
	overrideMu.Lock()
	calOverride = cal
	overrideMu.Unlock()
}

// The regions whose weeks start on Sunday or Saturday; others, Monday.
var (
	sundayRegions   = setOf("US CA MX BR JP KR TW HK MO IL PH IN ZA SA PK TH ID GT HN PA DO PE VE CO PR BZ JM KH LA MM NP BT ZW")
	saturdayRegions = setOf("IR AF EG DZ IQ JO KW LY OM QA SY AE BH SD")
	// hour12Regions use a 12-hour clock.
	hour12Regions = setOf("US CA AU NZ IN PH PK BD EG SA JO KW QA AE OM BH IQ SY LY SD DZ MY KR TW CO MX SV HN NI")
)

func setOf(s string) map[string]bool {
	m := map[string]bool{}
	for _, f := range strings.Fields(s) {
		m[f] = true
	}
	return m
}

// FirstWeekday returns the day weeks start on.
func (l Locale) FirstWeekday() time.Weekday {
	overrideMu.RLock()
	o := firstOverride
	overrideMu.RUnlock()
	if o >= 0 {
		return time.Weekday(o)
	}
	switch {
	case saturdayRegions[l.region]:
		return time.Saturday
	case sundayRegions[l.region]:
		return time.Sunday
	}
	return time.Monday
}

// Hour12 reports whether times use a 12-hour clock.
func (l Locale) Hour12() bool {
	overrideMu.RLock()
	o := hourOverride
	overrideMu.RUnlock()
	if o >= 0 {
		return o == 1
	}
	if l.region == "CA" && l.lang == "fr" {
		return false
	}
	return hour12Regions[l.region]
}

// Calendar returns the calendar dates use: Persian for Persian in Iran
// and Afghanistan, Gregorian elsewhere, unless SetCalendar chose one.
func (l Locale) Calendar() Calendar {
	overrideMu.RLock()
	c := calOverride
	overrideMu.RUnlock()
	if c != nil {
		return c
	}
	if l.lang == "fa" && (l.region == "IR" || l.region == "AF") {
		return Persian
	}
	return Gregorian
}

// Width is how long a name is.
type Width int

const (
	// Wide is the full name: "January", "Monday".
	Wide Width = iota
	// Abbreviated is short: "Jan", "Mon".
	Abbreviated
	// Narrow is a letter or so: "J", "M".
	Narrow
)

// MonthName returns month's name in cal, in the locale's language, as a
// title (a calendar's header); i18n keys
// "giode.month.<calendar>.<n>" and "giode.month_short.<calendar>.<n>"
// override it.
func (l Locale) MonthName(cal Calendar, month int, w Width) string {
	return l.monthName(cal, month, w, false)
}

func (l Locale) monthName(cal Calendar, month int, w Width, inDate bool) string {
	key := "giode.month."
	if w != Wide {
		key = "giode.month_short."
	}
	key += cal.ID() + "." + itoa(month)
	if s := i18n.T(key); s != key {
		return s
	}
	names := lookupNames(cal.ID(), l.lang)
	if names == nil || month < 1 || month > len(names.wide) {
		return itoa(month)
	}
	switch {
	case w == Narrow:
		r := []rune(names.wide[month-1])
		if len(r) > 0 {
			return string(r[0])
		}
	case w == Abbreviated && names.short != nil:
		return names.short[month-1]
	case inDate && names.genitive != nil:
		return names.genitive[month-1]
	}
	return names.wide[month-1]
}

// WeekdayName returns d's name in the locale's language; i18n keys
// "giode.weekday.<n>" (0 Sunday) and "giode.weekday_short.<n>"
// override it.
func (l Locale) WeekdayName(d time.Weekday, w Width) string {
	key := "giode.weekday."
	switch w {
	case Abbreviated:
		key = "giode.weekday_short."
	case Narrow:
		key = "giode.weekday_narrow."
	}
	key += itoa(int(d))
	if s := i18n.T(key); s != key {
		return s
	}
	names, ok := weekdayNames[l.lang]
	if !ok {
		names = weekdayNames["en"]
	}
	switch w {
	case Abbreviated:
		return names[1][d]
	case Narrow:
		return names[2][d]
	}
	return names[0][d]
}

// DayPeriod returns the 12-hour clock's "AM" or "PM" for hour h.
func (l Locale) DayPeriod(h int) string {
	p, ok := dayPeriods[l.lang]
	if !ok {
		p = dayPeriods["en"]
	}
	if h < 12 {
		return p[0]
	}
	return p[1]
}

// Digits writes the ASCII digits in s in the locale's own: Persian
// digits in Persian, Arabic-Indic in Arabic (except the Maghreb).
func (l Locale) Digits(s string) string {
	var zero rune
	switch {
	case l.lang == "fa":
		zero = '۰'
	case l.lang == "ar" && l.region != "MA" && l.region != "DZ" && l.region != "TN" && l.region != "LY":
		zero = '٠'
	default:
		return s
	}
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			r = zero + (r - '0')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// monthNames are a calendar's months in a language.
type monthNames struct {
	wide, short, genitive []string
}

var (
	namesMu    sync.RWMutex
	extraNames = map[string]*monthNames{}
)

// SetMonthNames gives the month names of a calendar in a language, for
// a calendar an app adds or a language Giode lacks; short and genitive
// (the form after a day number, as in Russian) may be nil.
func SetMonthNames(calendarID, lang string, wide, short, genitive []string) {
	namesMu.Lock()
	extraNames[calendarID+"/"+lang] = &monthNames{wide, short, genitive}
	namesMu.Unlock()
}

func lookupNames(cal, lang string) *monthNames {
	namesMu.RLock()
	n := extraNames[cal+"/"+lang]
	namesMu.RUnlock()
	if n != nil {
		return n
	}
	if n := builtinMonths[cal+"/"+lang]; n != nil {
		return n
	}
	if lang != "en" {
		return lookupNames(cal, "en")
	}
	return nil
}

func itoa(n int) string {
	if n < 0 {
		return "-" + itoa(-n)
	}
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}
