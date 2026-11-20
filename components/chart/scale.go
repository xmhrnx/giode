package chart

import (
	"math"
	"strconv"
	"strings"
	"time"
)

// scale maps data values to pixels.
type scale struct {
	min, max   float64
	from, to   float64 // pixels
	ticks      []float64
	time       bool
	timeUnit   timeUnit
	categories int // > 0 for a band scale of categories
}

func (s scale) px(v float64) float64 {
	if s.max == s.min {
		return (s.from + s.to) / 2
	}
	return s.from + (v-s.min)/(s.max-s.min)*(s.to-s.from)
}

func (s scale) value(px float64) float64 {
	if s.to == s.from {
		return s.min
	}
	return s.min + (px-s.from)/(s.to-s.from)*(s.max-s.min)
}

// band returns category i's band: its start and width in pixels.
func (s scale) band(i int) (float64, float64) {
	w := (s.to - s.from) / float64(max(s.categories, 1))
	return s.from + float64(i)*w, w
}

// niceNum rounds x to 1, 2, 2.5, 5 or 10 times a power of ten, up or to
// the nearest.
func niceNum(x float64, round bool) float64 {
	if x <= 0 || math.IsInf(x, 0) || math.IsNaN(x) {
		return 1
	}
	exp := math.Floor(math.Log10(x))
	f := x / math.Pow(10, exp)
	var nf float64
	if round {
		switch {
		case f < 1.5:
			nf = 1
		case f < 3:
			nf = 2
		case f < 7:
			nf = 5
		default:
			nf = 10
		}
	} else {
		switch {
		case f <= 1:
			nf = 1
		case f <= 2:
			nf = 2
		case f <= 2.5:
			nf = 2.5
		case f <= 5:
			nf = 5
		default:
			nf = 10
		}
	}
	return nf * math.Pow(10, exp)
}

// niceTicks returns about n round ticks covering [lo, hi], and the
// range they span.
func niceTicks(lo, hi float64, n int) (ticks []float64, lowest, highest float64) {
	if math.IsNaN(lo) || math.IsNaN(hi) {
		lo, hi = 0, 1
	}
	if lo == hi {
		if lo == 0 {
			hi = 1
		} else {
			d := math.Abs(lo) * 0.1
			lo, hi = lo-d, hi+d
		}
	}
	n = max(n, 2)
	step := niceNum((hi-lo)/float64(n-1), true)
	lowest = math.Floor(lo/step) * step
	highest = math.Ceil(hi/step) * step
	for v := lowest; v <= highest+step/2; v += step {
		if math.Abs(v) < step*1e-9 {
			v = 0
		}
		ticks = append(ticks, v)
	}
	return ticks, lowest, highest
}

// FormatNumber writes v compactly: 950, 1.2k, 3.4M, 2B, with up to
// decimals digits after the point for small numbers.
func FormatNumber(v float64) string {
	a := math.Abs(v)
	switch {
	case a >= 1e12:
		return trimFloat(v/1e12, 1) + "T"
	case a >= 1e9:
		return trimFloat(v/1e9, 1) + "B"
	case a >= 1e6:
		return trimFloat(v/1e6, 1) + "M"
	case a >= 1e4:
		return trimFloat(v/1e3, 1) + "k"
	case a >= 100 || a == 0:
		return trimFloat(v, 0)
	case a >= 1:
		return trimFloat(v, 2)
	}
	return trimFloat(v, 4)
}

func trimFloat(v float64, decimals int) string {
	s := strconv.FormatFloat(v, 'f', decimals, 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	if s == "-0" {
		s = "0"
	}
	return s
}

// timeUnit is a time axis' tick spacing.
type timeUnit struct {
	d      time.Duration
	months int // calendar months instead of d when > 0
	format string
}

var timeUnits = []timeUnit{
	{d: time.Second, format: "15:04:05"},
	{d: 5 * time.Second, format: "15:04:05"},
	{d: 15 * time.Second, format: "15:04:05"},
	{d: 30 * time.Second, format: "15:04:05"},
	{d: time.Minute, format: "15:04"},
	{d: 5 * time.Minute, format: "15:04"},
	{d: 15 * time.Minute, format: "15:04"},
	{d: 30 * time.Minute, format: "15:04"},
	{d: time.Hour, format: "15:04"},
	{d: 3 * time.Hour, format: "15:04"},
	{d: 6 * time.Hour, format: "Jan 2 15:04"},
	{d: 12 * time.Hour, format: "Jan 2 15:04"},
	{d: 24 * time.Hour, format: "Jan 2"},
	{d: 2 * 24 * time.Hour, format: "Jan 2"},
	{d: 7 * 24 * time.Hour, format: "Jan 2"},
	{months: 1, format: "Jan 2006"},
	{months: 3, format: "Jan 2006"},
	{months: 6, format: "Jan 2006"},
	{months: 12, format: "2006"},
	{months: 24, format: "2006"},
	{months: 60, format: "2006"},
	{months: 120, format: "2006"},
}

// timeTicks returns about n ticks on round times between lo and hi,
// seconds since the Unix epoch, in loc.
func timeTicks(lo, hi float64, n int, loc *time.Location) ([]float64, timeUnit) {
	span := time.Duration((hi - lo) * float64(time.Second))
	u := timeUnits[len(timeUnits)-1]
	for _, c := range timeUnits {
		d := c.d
		if c.months > 0 {
			d = time.Duration(c.months) * 30 * 24 * time.Hour
		}
		if span/d <= time.Duration(n) {
			u = c
			break
		}
	}
	start := time.Unix(int64(lo), 0).In(loc)
	var t time.Time
	if u.months > 0 {
		y, m := start.Year(), int(start.Month())-1
		total := y*12 + m
		total = (total + u.months - 1) / u.months * u.months
		t = time.Date(total/12, time.Month(total%12+1), 1, 0, 0, 0, 0, loc)
	} else if u.d >= 24*time.Hour {
		t = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc)
		if u.d == 7*24*time.Hour {
			for t.Weekday() != time.Monday {
				t = t.AddDate(0, 0, 1)
			}
		}
		if t.Before(start) {
			t = t.Add(u.d)
		}
	} else {
		t = start.Truncate(u.d)
		if t.Before(start) {
			t = t.Add(u.d)
		}
	}
	var out []float64
	for i := 0; i < 1000; i++ {
		v := float64(t.Unix())
		if v > hi {
			break
		}
		out = append(out, v)
		switch {
		case u.months > 0:
			t = t.AddDate(0, u.months, 0)
		case u.d >= 24*time.Hour:
			t = t.AddDate(0, 0, int(u.d/(24*time.Hour)))
		default:
			t = t.Add(u.d)
		}
	}
	return out, u
}
