// Package datepicker provides date and time inputs: a month calendar
// (DateCalendar), a date field with a calendar popover (DatePicker), a
// range of dates (DateRangePicker), a time field (TimePicker) and a date
// and time together (DateTimeField). They follow the locale's calendar,
// first day of the week, names, digits and 12- or 24-hour clock (see
// package datetime).
package datepicker

import (
	"image"
	"image/color"
	"time"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/components/iconbutton"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/datetime"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/i18n"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Sizes, in dp.
const (
	cellSize   = 36
	headerH    = 40
	weekdayH   = 28
	monthGap   = 20
	todayRingW = 1.5
)

// DateCalendar is a month calendar to pick a day, or a range of days,
// from. Create it once with NewCalendar and lay it out every frame:
//
//	cal := datepicker.NewCalendar().OnSelect(func(day time.Time) { due = day })
//
// Days are the start of each day in the calendar's zone (time.Local by
// default). The keyboard moves among days with the arrows, by months
// with Page Up and Page Down (by years with Shift), to the week's start
// and end with Home and End, and picks with Enter or Space.
type DateCalendar struct {
	locale *datetime.Locale
	cal    datetime.Calendar
	zone   *time.Location
	st     styles.Styles

	// shown is the first month shown (its Day unused); months how many.
	shown  datetime.Date
	months int

	selected  time.Time
	rangeMode bool
	start     time.Time
	end       time.Time
	min, max  time.Time
	disabled  func(day time.Time) bool
	onSelect  func(day time.Time)
	onRange   func(start, end time.Time)
	onMonth   func(first time.Time)

	// focusDay is the day the keyboard is on; hover the day under the
	// pointer, for a range's preview.
	focusDay time.Time
	hover    time.Time
	focused  bool
	focusReq bool

	prev, next *iconbutton.Button
	// grids are where each month's days were drawn last frame.
	grids []monthGrid
	cell  int
	rtl   bool
	// origin is its window position, last frame.
	origin  image.Point
	pressed time.Time
}

type monthGrid struct {
	rect  image.Rectangle
	first time.Time // the day in the grid's top-left cell
	month datetime.Date
}

// NewCalendar returns a calendar showing this month, picking single
// days. The styles argument is optional: Width sets the cell size as
// the whole calendar's width (default 7 cells of 36dp).
func NewCalendar(st ...styles.Styles) *DateCalendar {
	c := &DateCalendar{months: 1, zone: time.Local}
	if len(st) > 0 {
		c.st = st[0]
	}
	c.prev = iconbutton.New(elements.Glyph(kit.GlyphChevronBack, styles.Styles{Width: 18})).Variant(iconbutton.VariantGhost).
		OnClick(func() { c.ShowMonth(c.monthOffset(-1)) })
	c.next = iconbutton.New(elements.Glyph(kit.GlyphChevronForward, styles.Styles{Width: 18})).Variant(iconbutton.VariantGhost).
		OnClick(func() { c.ShowMonth(c.monthOffset(1)) })
	c.ShowMonth(time.Now())
	return c
}

func (c *DateCalendar) loc() datetime.Locale {
	if c.locale != nil {
		return *c.locale
	}
	return datetime.Current()
}

func (c *DateCalendar) calendar() datetime.Calendar {
	if c.cal != nil {
		return c.cal
	}
	return c.loc().Calendar()
}

// Locale sets the locale whose conventions it follows instead of the
// app's. It returns c for chaining.
func (c *DateCalendar) Locale(l datetime.Locale) *DateCalendar {
	c.locale = &l
	return c
}

// Calendar sets the calendar system instead of the locale's. It returns
// c for chaining.
func (c *DateCalendar) Calendar(cal datetime.Calendar) *DateCalendar {
	c.cal = cal
	return c
}

// Zone sets the zone days are in (time.Local by default). It returns c
// for chaining.
func (c *DateCalendar) Zone(z *time.Location) *DateCalendar {
	c.zone = z
	return c
}

// Months shows n months side by side (1 or 2; a range picks better with
// 2). It returns c for chaining.
func (c *DateCalendar) Months(n int) *DateCalendar {
	c.months = max(1, min(n, 3))
	return c
}

// Range makes it pick a range: the first click its start, the second its
// end. It returns c for chaining.
func (c *DateCalendar) Range(on bool) *DateCalendar {
	c.rangeMode = on
	return c
}

// Limits sets the first and last days that can be picked; a zero time
// leaves that end open. It returns c for chaining.
func (c *DateCalendar) Limits(first, last time.Time) *DateCalendar {
	c.min, c.max = first, last
	return c
}

// Disabled sets which days can't be picked (weekends, booked days). It
// returns c for chaining.
func (c *DateCalendar) Disabled(fn func(day time.Time) bool) *DateCalendar {
	c.disabled = fn
	return c
}

// OnSelect sets the handler of a picked day. It returns c for chaining.
func (c *DateCalendar) OnSelect(fn func(day time.Time)) *DateCalendar {
	c.onSelect = fn
	return c
}

// OnRange sets the handler of a picked range, called once both ends are
// picked. It returns c for chaining.
func (c *DateCalendar) OnRange(fn func(start, end time.Time)) *DateCalendar {
	c.onRange = fn
	return c
}

// OnMonth sets the handler called when it shows another month, with the
// month's first day: to load that month's events or busy days. It
// returns c for chaining.
func (c *DateCalendar) OnMonth(fn func(first time.Time)) *DateCalendar {
	c.onMonth = fn
	return c
}

// day returns the start of t's day in the calendar's zone.
func (c *DateCalendar) day(t time.Time) time.Time {
	if t.IsZero() {
		return t
	}
	return datetime.StartOfDay(t.In(c.zone))
}

// Selected returns the picked day, zero if none.
func (c *DateCalendar) Selected() time.Time { return c.selected }

// SetSelected picks day without calling OnSelect, and shows its month.
func (c *DateCalendar) SetSelected(day time.Time) {
	c.selected = c.day(day)
	if !day.IsZero() {
		c.ShowMonth(day)
	}
}

// SelectedRange returns the picked range; end is zero while only the
// start is picked.
func (c *DateCalendar) SelectedRange() (start, end time.Time) { return c.start, c.end }

// SetRange picks a range without calling OnRange.
func (c *DateCalendar) SetRange(start, end time.Time) {
	c.start, c.end = c.day(start), c.day(end)
	if !start.IsZero() {
		c.ShowMonth(start)
	}
}

// ShowMonth shows the month holding t (first of several).
func (c *DateCalendar) ShowMonth(t time.Time) {
	d := c.calendar().FromTime(t.In(c.zone))
	m := datetime.Date{Year: d.Year, Month: d.Month, Day: 1}
	if m == c.shown {
		return
	}
	c.shown = m
	if c.focusDay.IsZero() || c.calendar().FromTime(c.focusDay).Month != d.Month {
		c.focusDay = c.day(t)
	}
	if c.onMonth != nil {
		c.onMonth(c.calendar().ToTime(m, c.zone))
	}
}

// Shown returns the first day of the first month shown.
func (c *DateCalendar) Shown() time.Time {
	return c.calendar().ToTime(c.shown, c.zone)
}

// monthOffset returns a day in the month n months from the shown one.
func (c *DateCalendar) monthOffset(n int) time.Time {
	return c.calendar().ToTime(datetime.AddMonths(c.calendar(), c.shown, n), c.zone)
}

// enabled reports whether day can be picked.
func (c *DateCalendar) enabled(day time.Time) bool {
	if !c.min.IsZero() && day.Before(c.day(c.min)) {
		return false
	}
	if !c.max.IsZero() && day.After(c.day(c.max)) {
		return false
	}
	return c.disabled == nil || !c.disabled(day)
}

// pick picks day, as a click or Enter does.
func (c *DateCalendar) pick(day time.Time) {
	if !c.enabled(day) {
		return
	}
	c.focusDay = day
	if !c.rangeMode {
		c.selected = day
		if c.onSelect != nil {
			c.onSelect(day)
		}
		return
	}
	if c.start.IsZero() || !c.end.IsZero() {
		c.start, c.end = day, time.Time{}
		return
	}
	if day.Before(c.start) {
		c.start, c.end = day, c.start
	} else {
		c.end = day
	}
	if c.onRange != nil {
		c.onRange(c.start, c.end)
	}
}

// Click picks the focused day, as Enter does.
func (c *DateCalendar) Click() { c.pick(c.focusDay) }

// Focus gives the calendar keyboard focus.
func (c *DateCalendar) Focus() { c.focusReq = true }

// moveFocus moves the keyboard's day, showing its month.
func (c *DateCalendar) moveFocus(day time.Time) {
	c.focusDay = c.day(day)
	cal := c.calendar()
	d := cal.FromTime(c.focusDay)
	first := c.shown
	last := datetime.AddMonths(cal, c.shown, c.months-1)
	m := datetime.Date{Year: d.Year, Month: d.Month, Day: 1}
	if m.Before(first) || last.Before(m) {
		if m.Before(first) {
			c.ShowMonth(c.focusDay)
		} else {
			c.ShowMonth(cal.ToTime(datetime.AddMonths(cal, m, -(c.months-1)), c.zone))
		}
	}
}

// update handles the keyboard and the pointer.
func (c *DateCalendar) update(gtx layout.Context) {
	if c.focusReq {
		c.focusReq = false
		gtx.Execute(key.FocusCmd{Tag: c})
	}
	first := c.loc().FirstWeekday()
	for {
		ev, ok := gtx.Event(
			key.FocusFilter{Target: c},
			key.Filter{Focus: c, Name: key.NameLeftArrow, Optional: key.ModShift},
			key.Filter{Focus: c, Name: key.NameRightArrow, Optional: key.ModShift},
			key.Filter{Focus: c, Name: key.NameUpArrow},
			key.Filter{Focus: c, Name: key.NameDownArrow},
			key.Filter{Focus: c, Name: key.NamePageUp, Optional: key.ModShift},
			key.Filter{Focus: c, Name: key.NamePageDown, Optional: key.ModShift},
			key.Filter{Focus: c, Name: key.NameHome},
			key.Filter{Focus: c, Name: key.NameEnd},
			key.Filter{Focus: c, Name: key.NameReturn},
			key.Filter{Focus: c, Name: key.NameEnter},
			key.Filter{Focus: c, Name: key.NameSpace},
		)
		if !ok {
			break
		}
		switch e := ev.(type) {
		case key.FocusEvent:
			c.focused = e.Focus
			if e.Focus && c.focusDay.IsZero() {
				c.focusDay = c.day(time.Now())
			}
		case key.Event:
			if e.State != key.Press {
				continue
			}
			f := c.focusDay
			if f.IsZero() {
				f = c.day(time.Now())
			}
			back, fwd := -1, 1
			if c.rtl {
				back, fwd = 1, -1
			}
			cal := c.calendar()
			months := 1
			if e.Modifiers.Contain(key.ModShift) {
				months = cal.MonthsInYear(cal.FromTime(f).Year)
			}
			switch e.Name {
			case key.NameLeftArrow:
				c.moveFocus(datetime.AddDays(f, back))
			case key.NameRightArrow:
				c.moveFocus(datetime.AddDays(f, fwd))
			case key.NameUpArrow:
				c.moveFocus(datetime.AddDays(f, -7))
			case key.NameDownArrow:
				c.moveFocus(datetime.AddDays(f, 7))
			case key.NamePageUp:
				c.moveFocus(cal.ToTime(datetime.AddMonths(cal, cal.FromTime(f), -months), c.zone))
			case key.NamePageDown:
				c.moveFocus(cal.ToTime(datetime.AddMonths(cal, cal.FromTime(f), months), c.zone))
			case key.NameHome:
				c.moveFocus(datetime.StartOfWeek(f, first))
			case key.NameEnd:
				c.moveFocus(datetime.AddDays(datetime.StartOfWeek(f, first), 6))
			case key.NameReturn, key.NameEnter, key.NameSpace:
				c.pick(f)
			}
			kit.RequestFrame(gtx)
		}
	}
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: c, Kinds: pointer.Press | pointer.Release | pointer.Enter | pointer.Move | pointer.Leave | pointer.Cancel})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		day, hit := c.dayAt(e.Position)
		switch e.Kind {
		case pointer.Move:
			if day != c.hover {
				c.hover = day
				if c.rangeMode && !c.start.IsZero() && c.end.IsZero() {
					kit.RequestFrame(gtx)
				}
			}
		case pointer.Leave, pointer.Cancel:
			c.hover = time.Time{}
			c.pressed = time.Time{}
		case pointer.Press:
			c.pressed = day
			if hit {
				gtx.Execute(key.FocusCmd{Tag: c})
			}
		case pointer.Release:
			if hit && day.Equal(c.pressed) {
				c.pick(day)
				kit.RequestFrame(gtx)
			}
			c.pressed = time.Time{}
		}
	}
}

// dayAt returns the day under p (relative to the calendar).
func (c *DateCalendar) dayAt(p f32.Point) (time.Time, bool) {
	pt := image.Pt(int(p.X), int(p.Y))
	for _, g := range c.grids {
		if !pt.In(g.rect) || c.cell <= 0 {
			continue
		}
		col := (pt.X - g.rect.Min.X) / c.cell
		row := (pt.Y - g.rect.Min.Y) / c.cell
		if c.rtl {
			col = 6 - col
		}
		day := datetime.AddDays(g.first, row*7+col)
		return day, true
	}
	return time.Time{}, false
}

// Layout lays the calendar out: a header with the month and arrows, the
// weekdays, and the days.
func (c *DateCalendar) Layout(gtx layout.Context) layout.Dimensions {
	c.rtl = kit.RTL(gtx)
	elements.TrackPosition(&c.origin)
	c.update(gtx)
	loc := c.loc()
	cal := c.calendar()
	c.cell = gtx.Dp(cellSize)
	if c.st.Width > 0 {
		c.cell = gtx.Dp(unit.Dp(c.st.Width)) / 7
	}
	monthW := 7 * c.cell
	gap := gtx.Dp(monthGap)
	width := c.months*monthW + (c.months-1)*gap
	hdr := gtx.Dp(headerH)
	wdH := gtx.Dp(weekdayH)
	height := hdr + wdH + 6*c.cell
	size := image.Pt(width, height)
	kit.BlockDrag(gtx, size)
	// One input area under it all, for the keyboard and the days; the
	// arrow buttons, laid out after, take their own presses.
	area := clip.Rect{Max: size}.Push(gtx.Ops)
	event.Op(gtx.Ops, c)
	area.Pop()

	sem := elements.BeginSemanticGroup(elements.SemanticNode{Kind: elements.SemanticGroup, Label: loc.Format(c.Shown(), datetime.MonthYear)})

	// The arrows at the header's ends; months' names between.
	at := func(x, y int, f func(gtx layout.Context) layout.Dimensions) layout.Dimensions {
		defer op.Offset(image.Pt(x, y)).Push(gtx.Ops).Pop()
		pop := elements.PushOffset(image.Pt(x, y))
		defer pop()
		g := gtx
		g.Constraints.Min = image.Point{}
		return f(g)
	}
	muted := properties.CalcColor("theme.muted")
	text := properties.CalcColor("theme.text")
	primary := properties.CalcColor("theme.primary")
	onPrimary := properties.CalcColor("theme.onprimary")

	btn := gtx.Dp(32)
	startX, endX := 0, width-btn
	if c.rtl {
		startX, endX = endX, startX
	}
	at(startX, (hdr-btn)/2, c.prev.Label(i18n.T("giode.date.previous_month")).Layout)
	at(endX, (hdr-btn)/2, c.next.Label(i18n.T("giode.date.next_month")).Layout)

	first := loc.FirstWeekday()
	today := c.day(time.Now())
	c.grids = c.grids[:0]
	for i := range c.months {
		m := datetime.AddMonths(cal, c.shown, i)
		mx := i * (monthW + gap)
		if c.rtl {
			mx = width - monthW - mx
		}
		firstOfMonth := cal.ToTime(m, c.zone)
		title := elements.Text(loc.Format(firstOfMonth, datetime.MonthYear), styles.Styles{Color: "theme.text", FontWeight: properties.FontWeightSemiBold, FontSize: 15})
		centered(gtx, title, image.Rect(mx, 0, mx+monthW, hdr))
		// Weekday initials.
		for col := range 7 {
			wd := time.Weekday((int(first) + col) % 7)
			x := mx + col*c.cell
			if c.rtl {
				x = mx + (6-col)*c.cell
			}
			centered(gtx, elements.Text(loc.WeekdayName(wd, datetime.Narrow), styles.Styles{Color: "theme.muted", FontSize: 12}),
				image.Rect(x, hdr, x+c.cell, hdr+wdH))
		}
		gridTop := hdr + wdH
		lead := (int(firstOfMonth.Weekday()) - int(first) + 7) % 7
		gridFirst := datetime.AddDays(firstOfMonth, -lead)
		c.grids = append(c.grids, monthGrid{rect: image.Rect(mx, gridTop, mx+monthW, gridTop+6*c.cell), first: gridFirst, month: m})
		for idx := range 42 {
			row, col := idx/7, idx%7
			day := datetime.AddDays(gridFirst, idx)
			x := mx + col*c.cell
			if c.rtl {
				x = mx + (6-col)*c.cell
			}
			r := image.Rect(x, gridTop+row*c.cell, x+c.cell, gridTop+(row+1)*c.cell)
			c.layoutDay(gtx, loc, cal, day, m, r, today, text, muted, primary, onPrimary)
		}
	}
	for _, g := range c.grids {
		area := clip.Rect(g.rect).Push(gtx.Ops)
		pointer.CursorPointer.Add(gtx.Ops)
		area.Pop()
	}
	kit.NoteFocusable(c)
	sem.End(gtx, size)
	return layout.Dimensions{Size: size}
}

// layoutDay draws one day's cell.
func (c *DateCalendar) layoutDay(gtx layout.Context, loc datetime.Locale, cal datetime.Calendar, day time.Time, month datetime.Date,
	r image.Rectangle, today time.Time, text, muted, primary, onPrimary color.NRGBA) {
	d := cal.FromTime(day)
	inMonth := d.Month == month.Month && d.Year == month.Year
	enabled := c.enabled(day)
	selected := !c.rangeMode && !c.selected.IsZero() && day.Equal(c.selected)
	inRange, edge := false, false
	if c.rangeMode && !c.start.IsZero() {
		end := c.end
		if end.IsZero() && !c.hover.IsZero() && !c.hover.Before(c.start) {
			end = c.hover
		}
		if end.IsZero() {
			end = c.start
		}
		inRange = !day.Before(c.start) && !day.After(end)
		edge = day.Equal(c.start) || day.Equal(end)
	}
	pad := max(gtx.Dp(2), 1)
	inner := r.Inset(pad)
	radius := inner.Dy() / 2
	switch {
	case inRange && !edge && inMonth:
		band := primary
		band.A = 40
		paint.FillShape(gtx.Ops, band, clip.Rect(image.Rect(r.Min.X, inner.Min.Y, r.Max.X, inner.Max.Y)).Op())
	case (selected || edge) && inMonth:
		paint.FillShape(gtx.Ops, primary, clip.UniformRRect(inner, radius).Op(gtx.Ops))
	}
	col := text
	switch {
	case (selected || edge) && inMonth:
		col = onPrimary
	case !inMonth:
		col = muted
		col.A = col.A / 2
	case !enabled:
		col = muted
	}
	if day.Equal(today) && !(selected || edge) {
		w := float32(gtx.Dp(todayRingW))
		paint.FillShape(gtx.Ops, primary, clip.Stroke{Path: clip.UniformRRect(inner, radius).Path(gtx.Ops), Width: w}.Op())
	}
	if c.focused && day.Equal(c.focusDay) && kit.FocusVisible(gtx, c) {
		ring := op.Offset(inner.Min).Push(gtx.Ops)
		kit.PaintFocusRing(gtx, inner.Size(), radius)
		ring.Pop()
	}
	weight := properties.FontWeight("")
	if day.Equal(today) {
		weight = properties.FontWeightSemiBold
	}
	label := elements.Text(loc.Digits(itoa(d.Day)), styles.Styles{Color: properties.Color(hexColor(col)), FontSize: 14, FontWeight: weight})
	centered(gtx, label, r)
	if inMonth {
		elements.RecordSemantic(gtx, elements.SemanticNode{
			Kind: elements.SemanticButton, Label: loc.Format(day, datetime.DateLong),
			Selected: selected || (inRange && c.rangeMode), Disabled: !enabled,
			Focused: c.focused && day.Equal(c.focusDay), Target: dayTarget{c, day},
			Bounds: r.Add(elements.AbsolutePosition()),
		}, r.Size())
	}
}

func init() {
	i18n.AddBuiltin("en", map[string]any{
		"giode.date.previous_month": "Previous month",
		"giode.date.next_month":     "Next month",
		"giode.date.open_calendar":  "Choose date",
		"giode.date.open_times":     "Choose time",
		"giode.date.invalid":        "Not a date",
		"giode.time.invalid":        "Not a time",
	})
}

// dayTarget lets a screen reader pick a day.
type dayTarget struct {
	c   *DateCalendar
	day time.Time
}

func (t dayTarget) Click() { t.c.pick(t.day) }

// centered lays el out centered in r.
func centered(gtx layout.Context, el elements.Element, r image.Rectangle) {
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints = layout.Constraints{Max: r.Size()}
	dims := el.Layout(g)
	call := m.Stop()
	off := r.Min.Add(r.Size().Sub(dims.Size).Div(2))
	defer op.Offset(off).Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
}

func hexColor(c color.NRGBA) string {
	const digits = "0123456789abcdef"
	b := []byte{'#', 0, 0, 0, 0, 0, 0, 0, 0}
	for i, v := range []uint8{c.R, c.G, c.B, c.A} {
		b[1+2*i] = digits[v>>4]
		b[2+2*i] = digits[v&15]
	}
	return string(b)
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}
