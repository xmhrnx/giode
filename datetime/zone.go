package datetime

import (
	"time"
)

// Time zones: IANA names ("Asia/Tehran", "America/New_York"), loaded
// from the database this package embeds.

// LoadZone returns the IANA zone name, or UTC and an error when it
// isn't one.
func LoadZone(name string) (*time.Location, error) {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC, err
	}
	return loc, nil
}

// ZoneLabel names t's zone as a UTC offset, "GMT+3:30" or "GMT-5", for
// zones without a common abbreviation; "GMT" at UTC.
func ZoneLabel(t time.Time) string {
	_, off := t.Zone()
	if off == 0 {
		return "GMT"
	}
	sign := "+"
	if off < 0 {
		sign, off = "-", -off
	}
	h, m := off/3600, off%3600/60
	s := "GMT" + sign + itoa(h)
	if m != 0 {
		s += ":" + pad(m, 2)
	}
	return s
}

// InZones formats t in the local zone and in another person's, for "3
// PM your time, 11:30 PM their time" displays. Each carries its zone
// label.
func (l Locale) InZones(t time.Time, theirs *time.Location, s Style) (yours, their string) {
	mine := t.In(time.Local)
	other := t.In(theirs)
	return l.Format(mine, s) + " " + ZoneLabel(mine), l.Format(other, s) + " " + ZoneLabel(other)
}

// OffsetBetween returns how far zone b's clocks are ahead of a's at t.
func OffsetBetween(t time.Time, a, b *time.Location) time.Duration {
	_, oa := t.In(a).Zone()
	_, ob := t.In(b).Zone()
	return time.Duration(ob-oa) * time.Second
}
