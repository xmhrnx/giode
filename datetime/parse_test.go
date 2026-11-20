package datetime

import "testing"

func TestParseDate(t *testing.T) {
	for _, c := range []struct {
		tag, in string
		want    Date
	}{
		{"en-US", "11/20/2026", Date{2026, 11, 20}},
		{"en-US", "11-20-26", Date{2026, 11, 20}},
		{"en-US", "11/20", Date{2026, 11, 20}},
		{"en-US", "Nov 20 2026", Date{2026, 11, 20}},
		{"en-US", "2026-11-20", Date{2026, 11, 20}},
		{"en-GB", "20/11/2026", Date{2026, 11, 20}},
		{"en-GB", "20 november", Date{2026, 11, 20}},
		{"de-DE", "20.11.2026", Date{2026, 11, 20}},
		{"de-DE", "20112026", Date{2026, 11, 20}},
		{"ru-RU", "20 ноября 2026", Date{2026, 11, 20}},
		{"fa-IR", "۱۴۰۵/۸/۲۹", Date{1405, 8, 29}},
		{"fa-IR", "29 آبان", Date{1405, 8, 29}},
	} {
		year := 2026
		if c.tag == "fa-IR" {
			year = 1405
		}
		got, err := NewLocale(c.tag).ParseDate(c.in, year)
		if err != nil || got != c.want {
			t.Errorf("%s %q = %v, %v; want %v", c.tag, c.in, got, err, c.want)
		}
	}
	for _, bad := range []string{"", "13/40/2026", "hello", "2/30/2026"} {
		if _, err := NewLocale("en-US").ParseDate(bad, 2026); err == nil {
			t.Errorf("%q parsed", bad)
		}
	}
}

func TestParseTime(t *testing.T) {
	for _, c := range []struct {
		tag, in string
		h, m    int
	}{
		{"en-US", "15:04", 15, 4},
		{"en-US", "3:04 pm", 15, 4},
		{"en-US", "3pm", 15, 0},
		{"en-US", "12am", 0, 0},
		{"en-US", "12:30 PM", 12, 30},
		{"en-US", "1504", 15, 4},
		{"de-DE", "9.30", 9, 30},
		{"fa-IR", "۱۵:۰۴", 15, 4},
		{"ja-JP", "午後3:04", 15, 4},
	} {
		h, m, err := NewLocale(c.tag).ParseTime(c.in)
		if err != nil || h != c.h || m != c.m {
			t.Errorf("%s %q = %d:%02d, %v", c.tag, c.in, h, m, err)
		}
	}
	for _, bad := range []string{"", "25:00", "13pm", "10:75"} {
		if _, _, err := NewLocale("en-US").ParseTime(bad); err == nil {
			t.Errorf("%q parsed", bad)
		}
	}
}
