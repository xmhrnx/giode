package datetime

import (
	"testing"
	"time"

	"github.com/xmhrnx/giode/i18n"
)

func TestFormat(t *testing.T) {
	ts := time.Date(2026, 11, 20, 15, 4, 5, 0, time.UTC)
	for _, c := range []struct {
		tag   string
		style Style
		want  string
	}{
		{"en-US", DateShort, "11/20/2026"},
		{"en-US", DateLong, "Friday, November 20, 2026"},
		{"en-US", Time, "3:04 PM"},
		{"en-GB", DateMedium, "20 Nov 2026"},
		{"en-GB", Time, "15:04"},
		{"de-DE", DateLong, "Freitag, 20. November 2026"},
		{"ru-RU", DateLong, "пятница, 20 ноября 2026 г."},
		{"ru-RU", MonthYear, "ноябрь 2026"},
		{"ja-JP", DateMedium, "2026年11月20日"},
		{"fa-IR", DateShort, "۱۴۰۵/۸/۲۹"},
		{"fa-IR", DateLong, "جمعه ۲۹ آبان ۱۴۰۵"},
		{"fa-IR", Time, "۱۵:۰۴"},
		{"ar-EG", Time, "٣:٠٤ م"},
	} {
		if got := NewLocale(c.tag).Format(ts, c.style); got != c.want {
			t.Errorf("%s style %d = %q, want %q", c.tag, c.style, got, c.want)
		}
	}
}

func TestLocaleConventions(t *testing.T) {
	for _, c := range []struct {
		tag    string
		first  time.Weekday
		hour12 bool
		cal    string
	}{
		{"en-US", time.Sunday, true, "gregorian"},
		{"en-GB", time.Monday, false, "gregorian"},
		{"en", time.Sunday, true, "gregorian"},
		{"fa", time.Saturday, false, "persian"},
		{"fa-IR", time.Saturday, false, "persian"},
		{"ar-AE", time.Saturday, true, "gregorian"},
		{"de_DE.UTF-8", time.Monday, false, "gregorian"},
		{"zh-Hant-TW", time.Sunday, true, "gregorian"},
	} {
		l := NewLocale(c.tag)
		if l.FirstWeekday() != c.first || l.Hour12() != c.hour12 || l.Calendar().ID() != c.cal {
			t.Errorf("%s: first %v, 12h %v, %s", c.tag, l.FirstWeekday(), l.Hour12(), l.Calendar().ID())
		}
	}
	SetFirstWeekday(time.Wednesday)
	defer SetFirstWeekday(-1)
	if NewLocale("en-US").FirstWeekday() != time.Wednesday {
		t.Error("override ignored")
	}
}

func TestRelative(t *testing.T) {
	defer i18n.SetLocale(i18n.Locale())
	i18n.SetLocale("en")
	now := time.Date(2026, 11, 20, 1, 0, 0, 0, time.UTC)
	l := NewLocale("en")
	for _, c := range []struct {
		t    time.Time
		want string
	}{
		{now.Add(10 * time.Second), "now"},
		{now.Add(-5 * time.Minute), "5 minutes ago"},
		{now.Add(-1 * time.Minute), "1 minute ago"},
		{now.Add(2 * time.Hour), "in 2 hours"},
		{now.Add(-2 * time.Hour), "yesterday"},
		{AddDays(now, 1), "tomorrow"},
		{AddDays(now, 3), "in 3 days"},
		{AddDays(now, -15), "2 weeks ago"},
		{AddDays(now, 70), "in 2 months"},
		{AddDays(now, -800), "2 years ago"},
	} {
		if got := l.Relative(c.t, now); got != c.want {
			t.Errorf("%s from now = %q, want %q", c.t.Sub(now), got, c.want)
		}
	}
	i18n.SetLocale("ru")
	if got := NewLocale("ru").Relative(AddDays(now, -5), now); got != "5 дней назад" {
		t.Errorf("ru 5 days ago = %q", got)
	}
	i18n.SetLocale("fa")
	if got := NewLocale("fa").Relative(AddDays(now, 3), now); got != "۳ روز بعد" {
		t.Errorf("fa in 3 days = %q", got)
	}
}

func TestZones(t *testing.T) {
	teh, err := LoadZone("Asia/Tehran")
	if err != nil {
		t.Fatal(err)
	}
	ny, _ := LoadZone("America/New_York")
	ts := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	if l := ZoneLabel(ts.In(teh)); l != "GMT+3:30" {
		t.Errorf("Tehran = %s", l)
	}
	if l := ZoneLabel(ts.In(ny)); l != "GMT-5" {
		t.Errorf("New York = %s", l)
	}
	if d := OffsetBetween(ts, ny, teh); d != 8*time.Hour+30*time.Minute {
		t.Errorf("New York to Tehran = %v", d)
	}
}
