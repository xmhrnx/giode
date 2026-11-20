package datepicker

import (
	"time"

	"gioui.org/io/key"
	"gioui.org/layout"

	"github.com/xmhrnx/giode/components/iconbutton"
	"github.com/xmhrnx/giode/components/input"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/datetime"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/i18n"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// DatePicker is a date field: type a date in the locale's order, or open
// the calendar with its button (or Down in the field) and pick one.
// Create it once with New and lay it out every frame:
//
//	due := datepicker.New().Label("Due date").OnChange(func(d time.Time) { task.Due = d })
//
// A typed date applies as soon as it reads as one; Enter or leaving the
// field formats it, or marks it invalid. An emptied field clears it.
type DatePicker struct {
	field  *input.Input
	button *iconbutton.Button
	cal    *DateCalendar
	pop    popover

	value    time.Time
	onChange func(time.Time)
	st       styles.Styles
	// shown is the text the field shows for value, to tell typing from
	// it.
	shown string
}

// New returns an empty date field. The styles argument is optional and
// styles the field (Width sets its width).
func New(st ...styles.Styles) *DatePicker {
	p := &DatePicker{}
	if len(st) > 0 {
		p.st = st[0]
	}
	p.cal = NewCalendar().OnSelect(func(day time.Time) {
		p.set(day, true)
		p.pop.close()
		p.field.Focus()
	})
	p.field = input.New("").Keyboard(input.KeyboardText).Autocorrect(false)
	p.field.OnChange(p.typed).Submit(func(string) { p.commit() }).OnBlur(p.commit)
	p.field.OnKeys(func(e key.Event) { p.openCalendar() }, key.NameDownArrow)
	p.button = iconbutton.New(elements.Glyph(kit.GlyphCalendar, styles.Styles{Width: 18})).Variant(iconbutton.VariantGhost).
		Label(i18n.T("giode.date.open_calendar")).OnClick(func() {
		if p.pop.open {
			p.pop.close()
		} else {
			p.openCalendar()
		}
	})
	return p
}

// Label names the field. It returns p for chaining.
func (p *DatePicker) Label(s string) *DatePicker { p.field.Label(s); return p }

// Required marks the field required. It returns p for chaining.
func (p *DatePicker) Required(on bool) *DatePicker { p.field.Required(on); return p }

// Locale sets the locale whose conventions it follows instead of the
// app's. It returns p for chaining.
func (p *DatePicker) Locale(l datetime.Locale) *DatePicker { p.cal.Locale(l); return p }

// Calendar sets the calendar system. It returns p for chaining.
func (p *DatePicker) Calendar(c datetime.Calendar) *DatePicker { p.cal.Calendar(c); return p }

// Zone sets the zone its days are in (time.Local by default). It
// returns p for chaining.
func (p *DatePicker) Zone(z *time.Location) *DatePicker { p.cal.Zone(z); return p }

// Limits sets the first and last days it takes. It returns p for
// chaining.
func (p *DatePicker) Limits(first, last time.Time) *DatePicker { p.cal.Limits(first, last); return p }

// Disabled sets which days it doesn't take. It returns p for chaining.
func (p *DatePicker) Disabled(fn func(day time.Time) bool) *DatePicker {
	p.cal.Disabled(fn)
	return p
}

// OnChange sets the handler of a new date (zero when cleared). It
// returns p for chaining.
func (p *DatePicker) OnChange(fn func(day time.Time)) *DatePicker { p.onChange = fn; return p }

// Value returns the date, the start of its day; zero when empty.
func (p *DatePicker) Value() time.Time { return p.value }

// SetValue sets the date without calling OnChange; zero empties it.
func (p *DatePicker) SetValue(day time.Time) { p.set(day, false) }

// Field returns the text field, for its other settings.
func (p *DatePicker) Field() *input.Input { return p.field }

// Calendar returns the calendar the button opens, for its settings.
func (p *DatePicker) CalendarView() *DateCalendar { return p.cal }

// Open reports whether the calendar is open.
func (p *DatePicker) Open() bool { return p.pop.open }

func (p *DatePicker) openCalendar() {
	if !p.value.IsZero() {
		p.cal.SetSelected(p.value)
		p.cal.focusDay = p.cal.day(p.value)
	}
	p.pop.open = true
	p.cal.Focus()
}

// set sets the value, the field's text and the calendar's selection.
func (p *DatePicker) set(day time.Time, notify bool) {
	if !day.IsZero() {
		day = p.cal.day(day)
	}
	changed := !day.Equal(p.value)
	p.value = day
	p.shown = ""
	if !day.IsZero() {
		p.shown = p.cal.loc().Format(day, datetime.DateShort)
		p.cal.SetSelected(day)
	}
	if p.field.Text() != p.shown {
		p.field.SetText(p.shown)
	}
	p.field.Invalid("")
	if changed && notify && p.onChange != nil {
		p.onChange(day)
	}
}

// parse reads the field's text as a date the picker takes.
func (p *DatePicker) parse(s string) (time.Time, bool) {
	loc, cal := p.cal.loc(), p.cal.calendar()
	thisYear := cal.FromTime(time.Now().In(p.cal.zone)).Year
	d, err := loc.ParseDate(s, thisYear)
	if err != nil {
		return time.Time{}, false
	}
	day := cal.ToTime(d, p.cal.zone)
	return day, p.cal.enabled(day)
}

// typed applies a date as soon as the text reads as one.
func (p *DatePicker) typed(s string) {
	if s == p.shown {
		return
	}
	if day, ok := p.parse(s); ok {
		changed := !day.Equal(p.value)
		p.value = day
		p.cal.SetSelected(day)
		p.field.Invalid("")
		if changed && p.onChange != nil {
			p.onChange(day)
		}
	}
}

// commit formats the field's date, or marks it invalid; an empty field
// clears it.
func (p *DatePicker) commit() {
	s := p.field.Text()
	switch {
	case s == "":
		p.set(time.Time{}, true)
	case s == p.shown:
	default:
		if day, ok := p.parse(s); ok {
			p.set(day, true)
		} else {
			p.field.Invalid(i18n.T("giode.date.invalid"))
		}
	}
}

// Layout lays out the field with its button, and the calendar when open.
func (p *DatePicker) Layout(gtx layout.Context) layout.Dimensions {
	if p.field.Text() == "" && !p.field.Focused() {
		// An example date, in the locale's order, as the hint.
		p.field.Placeholder(p.cal.loc().Format(time.Now(), datetime.DateShort))
	}
	fieldSt := styles.Styles{FlexGrow: 1}
	if p.st.Width != 0 {
		fieldSt.Width = -1
	}
	rowSt := styles.Styles{Align: properties.AlignCenter, Gap: 4, Width: p.st.Width, Wrap: properties.WrapNoWrap}
	row := elements.Row(rowSt, elements.Box(fieldSt, p.field), p.button)
	dims := row.Layout(gtx)
	p.pop.layout(gtx, dims.Size, func() elements.Element { return p.cal })
	return dims
}

// DateRangePicker is two date fields, a range's start and end, with a
// calendar of two months to pick both in. Create it once with NewRange.
type DateRangePicker struct {
	start, end *DatePicker
	button     *iconbutton.Button
	cal        *DateCalendar
	pop        popover
	onChange   func(start, end time.Time)
	st         styles.Styles
}

// NewRange returns an empty range picker. The styles argument is
// optional (Width).
func NewRange(st ...styles.Styles) *DateRangePicker {
	r := &DateRangePicker{start: New(), end: New()}
	if len(st) > 0 {
		r.st = st[0]
	}
	r.cal = NewCalendar().Range(true).Months(2).OnRange(func(s, e time.Time) {
		r.start.set(s, false)
		r.end.set(e, false)
		r.pop.close()
		r.changed()
	})
	r.start.OnChange(func(time.Time) { r.syncCalendar(); r.changed() })
	r.end.OnChange(func(time.Time) { r.syncCalendar(); r.changed() })
	r.button = iconbutton.New(elements.Glyph(kit.GlyphCalendar, styles.Styles{Width: 18})).Variant(iconbutton.VariantGhost).
		Label(i18n.T("giode.date.open_calendar")).OnClick(func() {
		if r.pop.open {
			r.pop.close()
			return
		}
		r.syncCalendar()
		r.pop.open = true
		r.cal.Focus()
	})
	return r
}

func (r *DateRangePicker) syncCalendar() {
	r.cal.SetRange(r.start.Value(), r.end.Value())
}

func (r *DateRangePicker) changed() {
	if r.onChange != nil {
		r.onChange(r.start.Value(), r.end.Value())
	}
}

// Labels names the two fields. It returns r for chaining.
func (r *DateRangePicker) Labels(start, end string) *DateRangePicker {
	r.start.Label(start)
	r.end.Label(end)
	return r
}

// Locale sets the locale it follows. It returns r for chaining.
func (r *DateRangePicker) Locale(l datetime.Locale) *DateRangePicker {
	r.start.Locale(l)
	r.end.Locale(l)
	r.cal.Locale(l)
	return r
}

// Limits sets the first and last days it takes. It returns r for
// chaining.
func (r *DateRangePicker) Limits(first, last time.Time) *DateRangePicker {
	r.start.Limits(first, last)
	r.end.Limits(first, last)
	r.cal.Limits(first, last)
	return r
}

// OnChange sets the handler of a new start or end. It returns r for
// chaining.
func (r *DateRangePicker) OnChange(fn func(start, end time.Time)) *DateRangePicker {
	r.onChange = fn
	return r
}

// Value returns the range; either end may be zero.
func (r *DateRangePicker) Value() (start, end time.Time) { return r.start.Value(), r.end.Value() }

// SetValue sets the range without calling OnChange.
func (r *DateRangePicker) SetValue(start, end time.Time) {
	r.start.SetValue(start)
	r.end.SetValue(end)
	r.syncCalendar()
}

// Layout lays out the two fields with the button, and the calendar when
// open.
func (r *DateRangePicker) Layout(gtx layout.Context) layout.Dimensions {
	for _, p := range []*DatePicker{r.start, r.end} {
		if p.field.Text() == "" {
			p.field.Placeholder(p.cal.loc().Format(time.Now(), datetime.DateShort))
		}
	}
	grow := styles.Styles{FlexGrow: 1}
	if r.st.Width != 0 {
		grow.Width = -1
	}
	row := elements.Row(styles.Styles{Align: properties.AlignCenter, Gap: 6, Width: r.st.Width, Wrap: properties.WrapNoWrap},
		elements.Box(grow, r.start.field),
		elements.Text("–", styles.Styles{Color: "theme.muted"}),
		elements.Box(grow, r.end.field),
		r.button,
	)
	dims := row.Layout(gtx)
	r.pop.layout(gtx, dims.Size, func() elements.Element { return r.cal })
	return dims
}
