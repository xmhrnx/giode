package datetime

import "time"

// Persian is the Persian (Solar Hijri, Jalali) calendar used in Iran and
// Afghanistan: its year starts at the March equinox, its first six
// months have 31 days, the next five 30, and Esfand 29, or 30 in a leap
// year. Dates follow the 2820-year cycle's breaks, as in jalaali-js,
// from 1 to 3177 AP.
var Persian Calendar = persian{}

type persian struct{}

func (persian) ID() string { return "persian" }

func (persian) MonthsInYear(int) int { return 12 }

func (persian) DaysInMonth(year, month int) int {
	switch {
	case month <= 6:
		return 31
	case month <= 11:
		return 30
	}
	if leap, _, _ := jalCal(year); leap == 0 {
		return 30
	}
	return 29
}

func (persian) FromTime(t time.Time) Date {
	y, m, d := t.Date()
	jy, jm, jd := d2j(g2d(y, int(m), d))
	return Date{jy, jm, jd}
}

func (p persian) ToTime(d Date, loc *time.Location) time.Time {
	// Months and days out of range roll over, as time.Date does.
	for d.Month > 12 {
		d.Month -= 12
		d.Year++
	}
	for d.Month < 1 {
		d.Month += 12
		d.Year--
	}
	gy, gm, gd := d2g(j2d(d.Year, d.Month, 1))
	return StartOfDay(time.Date(gy, time.Month(gm), gd+d.Day-1, 12, 0, 0, 0, loc))
}

// The years where the 33-year leap pattern restarts.
var jalaliBreaks = [...]int{-61, 9, 38, 199, 426, 686, 756, 818, 1111, 1181, 1210, 1635, 2060, 2097, 2192, 2262, 2324, 2394, 2456, 3178}

// jalCal returns whether jy is a leap year (leap 0) and the Gregorian
// day of March its year starts on.
func jalCal(jy int) (leap, gy, march int) {
	gy = jy + 621
	leapJ := -14
	jp := jalaliBreaks[0]
	jump := 0
	for i := 1; i < len(jalaliBreaks); i++ {
		jm := jalaliBreaks[i]
		jump = jm - jp
		if jy < jm {
			break
		}
		leapJ += jump/33*8 + jump%33/4
		jp = jm
	}
	n := jy - jp
	leapJ += n/33*8 + (n%33+3)/4
	if jump%33 == 4 && jump-n == 4 {
		leapJ++
	}
	leapG := gy/4 - (gy/100+1)*3/4 - 150
	march = 20 + leapJ - leapG
	if jump-n < 6 {
		n = n - jump + (jump+4)/33*33
	}
	leap = ((n+1)%33 - 1) % 4
	if leap == -1 {
		leap = 4
	}
	return leap, gy, march
}

// j2d returns the Julian day number of a Persian date.
func j2d(jy, jm, jd int) int {
	_, gy, march := jalCal(jy)
	return g2d(gy, 3, march) + (jm-1)*31 - jm/7*(jm-7) + jd - 1
}

// d2j returns the Persian date of a Julian day number.
func d2j(jdn int) (jy, jm, jd int) {
	gy, _, _ := d2g(jdn)
	jy = gy - 621
	leap, _, march := jalCal(jy)
	k := jdn - g2d(gy, 3, march)
	if k >= 0 {
		if k <= 185 {
			return jy, 1 + k/31, k%31 + 1
		}
		k -= 186
	} else {
		jy--
		k += 179
		if leap == 1 {
			k++
		}
	}
	return jy, 7 + k/30, k%30 + 1
}

// g2d returns the Julian day number of a Gregorian date.
func g2d(gy, gm, gd int) int {
	d := (gy+(gm-8)/6+100100)*1461/4 + (153*((gm+9)%12)+2)/5 + gd - 34840408
	return d - (gy+100100+(gm-8)/6)/100*3/4 + 752
}

// d2g returns the Gregorian date of a Julian day number.
func d2g(jdn int) (gy, gm, gd int) {
	j := 4*jdn + 139361631
	j += (4*jdn+183187720)/146097*3/4*4 - 3908
	i := j%1461/4*5 + 308
	gd = i%153/5 + 1
	gm = i/153%12 + 1
	gy = j/1461 - 100100 + (8-gm)/6
	return gy, gm, gd
}
