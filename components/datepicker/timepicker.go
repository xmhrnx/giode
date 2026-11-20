package datepicker

import (
	"image"
	"time"

	"gioui.org/io/key"
	"gioui.org/layout"

	"github.com/xmhrnx/giode/components/iconbutton"
	"github.com/xmhrnx/giode/components/input"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/components/scroll"
	"github.com/xmhrnx/giode/datetime"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/i18n"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// The list's size and rows, in dp.
const (
	listWidth  = 140
	listHeight = 240
	slotHeight = 32
)

// TimeOfDay is a time on a clock: Hour from 0 to 23, Minute from 0 to 59.
type TimeOfDay struct {
	Hour, Minute int
}

// On returns the time of day on day, in day's location.
func (t TimeOfDay) On(day time.Time) time.Time {
	y, m, d := day.Date()
	return time.Date(y, m, d, t.Hour, t.Minute, 0, 0, day.Location())
}

func (t TimeOfDay) minutes() int { return t.Hour*60 + t.Minute }

// TimePicker is a time field: type a time ("3pm", "15:30", "1530"), or
// open the list of times with its button (or Down in the field). It
// shows times in the locale's 12- or 24-hour clock. Create it once with
// NewTime and lay it out every frame.
type TimePicker struct {
	field  *input.Input
	button *iconbutton.Button
	pop    popover
	list   *scroll.Scroll
	slots  []*elements.Pressable
	slotOf []TimeOfDay

	locale   *datetime.Locale
	step     time.Duration
	value    TimeOfDay
	has      bool
	onChange func(t TimeOfDay, ok bool)
	shown    string
	st       styles.Styles
	reveal   bool
}

// NewTime returns an empty time field listing times every 30 minutes.
// The styles argument is optional (Width).
func NewTime(st ...styles.Styles) *TimePicker {
	p := &TimePicker{step: 30 * time.Minute}
	if len(st) > 0 {
		p.st = st[0]
	}
	p.field = input.New("").Autocorrect(false)
	p.field.OnChange(p.typed).Submit(func(string) { p.commit() }).OnBlur(p.commit)
	p.field.OnKeys(func(e key.Event) { p.openList() }, key.NameDownArrow)
	p.button = iconbutton.New(elements.Glyph(kit.GlyphClock, styles.Styles{Width: 18})).Variant(iconbutton.VariantGhost).
		Label(i18n.T("giode.date.open_times")).OnClick(func() {
		if p.pop.open {
			p.pop.close()
		} else {
			p.openList()
		}
	})
	p.list = scroll.New(p.listContent)
	return p
}

func (p *TimePicker) loc() datetime.Locale {
	if p.locale != nil {
		return *p.locale
	}
	return datetime.Current()
}

// Locale sets the locale it follows instead of the app's. It returns p
// for chaining.
func (p *TimePicker) Locale(l datetime.Locale) *TimePicker { p.locale = &l; return p }

// Step sets the interval of the listed times (30 minutes by default).
// It returns p for chaining.
func (p *TimePicker) Step(d time.Duration) *TimePicker {
	if d >= time.Minute {
		p.step = d
		p.slots = nil
	}
	return p
}

// Label names the field. It returns p for chaining.
func (p *TimePicker) Label(s string) *TimePicker { p.field.Label(s); return p }

// OnChange sets the handler of a new time; ok is false when cleared. It
// returns p for chaining.
func (p *TimePicker) OnChange(fn func(t TimeOfDay, ok bool)) *TimePicker { p.onChange = fn; return p }

// Value returns the time; ok is false when the field is empty.
func (p *TimePicker) Value() (t TimeOfDay, ok bool) { return p.value, p.has }

// SetValue sets the time without calling OnChange.
func (p *TimePicker) SetValue(t TimeOfDay) { p.set(t, true, false) }

// Clear empties the field without calling OnChange.
func (p *TimePicker) Clear() { p.set(TimeOfDay{}, false, false) }

// Field returns the text field.
func (p *TimePicker) Field() *input.Input { return p.field }

// Open reports whether the list is open.
func (p *TimePicker) Open() bool { return p.pop.open }

// format writes t in the locale's clock.
func (p *TimePicker) format(t TimeOfDay) string {
	return p.loc().Format(t.On(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)), datetime.Time)
}

func (p *TimePicker) set(t TimeOfDay, has, notify bool) {
	changed := has != p.has || t != p.value
	p.value, p.has = t, has
	p.shown = ""
	if has {
		p.shown = p.format(t)
	}
	if p.field.Text() != p.shown {
		p.field.SetText(p.shown)
	}
	p.field.Invalid("")
	if changed && notify && p.onChange != nil {
		p.onChange(t, has)
	}
}

func (p *TimePicker) typed(s string) {
	if s == p.shown {
		return
	}
	if h, m, err := p.loc().ParseTime(s); err == nil {
		t := TimeOfDay{h, m}
		changed := !p.has || t != p.value
		p.value, p.has = t, true
		p.field.Invalid("")
		if changed && p.onChange != nil {
			p.onChange(t, true)
		}
	}
}

func (p *TimePicker) commit() {
	s := p.field.Text()
	switch {
	case s == "":
		p.set(TimeOfDay{}, false, true)
	case s == p.shown:
	default:
		if h, m, err := p.loc().ParseTime(s); err == nil {
			p.set(TimeOfDay{h, m}, true, true)
		} else {
			p.field.Invalid(i18n.T("giode.time.invalid"))
		}
	}
}

func (p *TimePicker) openList() {
	p.pop.open = true
	p.reveal = true
}

// buildSlots makes the list's rows, once per step.
func (p *TimePicker) buildSlots() {
	if p.slots != nil {
		return
	}
	stepMin := int(p.step / time.Minute)
	for m := 0; m < 24*60; m += stepMin {
		t := TimeOfDay{m / 60, m % 60}
		p.slotOf = append(p.slotOf, t)
		p.slots = append(p.slots, elements.NewPressable(func(s elements.PressState) elements.Element {
			bg := ""
			col := properties.Color("theme.text")
			switch {
			case p.has && t == p.value:
				bg, col = "theme.primary", "theme.onprimary"
			case s.Hovered || s.Focused:
				bg = "theme.surface"
			}
			return elements.Box(styles.Styles{Width: -1, Height: slotHeight, Align: properties.AlignCenter,
				Padding:    properties.Inset{Left: 10, Right: 10},
				Background: properties.Background(bg), BorderRadius: 6},
				elements.Text(p.format(t), styles.Styles{Color: col}))
		}).OnClick(func(elements.ClickEvent) {
			p.set(t, true, true)
			p.pop.close()
			p.field.Focus()
		}).Label(p.format(t)))
	}
}

func (p *TimePicker) listContent() elements.Element {
	p.buildSlots()
	rows := make([]elements.Element, len(p.slots))
	for i, s := range p.slots {
		rows[i] = s
	}
	return elements.Box(styles.Styles{FlexDirection: properties.FlexDirectionColumn, Width: -1}, rows...)
}

// Layout lays out the field with its button, and the list when open.
func (p *TimePicker) Layout(gtx layout.Context) layout.Dimensions {
	if p.field.Text() == "" {
		p.field.Placeholder(p.format(TimeOfDay{9, 0}))
	}
	if p.reveal {
		// Open at the chosen time, or the slot nearest to now.
		p.reveal = false
		p.buildSlots()
		target := p.value
		if !p.has {
			now := time.Now()
			target = TimeOfDay{now.Hour(), now.Minute()}
		}
		idx := 0
		for i, t := range p.slotOf {
			if t.minutes() <= target.minutes() {
				idx = i
			}
		}
		// The chosen one third of the way down.
		p.list.ScrollToDp(float32(max(idx*slotHeight-listHeight/3, 0)))
	}
	fieldSt := styles.Styles{FlexGrow: 1}
	if p.st.Width != 0 {
		fieldSt.Width = -1
	}
	row := elements.Row(styles.Styles{Align: properties.AlignCenter, Gap: 4, Width: p.st.Width, Wrap: properties.WrapNoWrap},
		elements.Box(fieldSt, p.field), p.button)
	dims := row.Layout(gtx)
	p.pop.layout(gtx, dims.Size, func() elements.Element {
		return elements.Raw(func(gtx layout.Context) layout.Dimensions {
			size := image.Pt(gtx.Dp(listWidth), gtx.Dp(listHeight))
			gtx.Constraints = layout.Exact(size)
			return p.list.Layout(gtx)
		})
	})
	return dims
}

// DateTimeField is a date and a time of day together, in a zone.
// Create it once with NewDateTime.
type DateTimeField struct {
	date     *DatePicker
	clock    *TimePicker
	zone     *time.Location
	onChange func(t time.Time, ok bool)
}

// NewDateTime returns an empty date and time field, in time.Local.
func NewDateTime() *DateTimeField {
	f := &DateTimeField{date: New(), clock: NewTime(), zone: time.Local}
	f.date.OnChange(func(time.Time) { f.changed() })
	f.clock.OnChange(func(TimeOfDay, bool) { f.changed() })
	return f
}

func (f *DateTimeField) changed() {
	if f.onChange != nil {
		t, ok := f.Value()
		f.onChange(t, ok)
	}
}

// Locale sets the locale both follow. It returns f for chaining.
func (f *DateTimeField) Locale(l datetime.Locale) *DateTimeField {
	f.date.Locale(l)
	f.clock.Locale(l)
	return f
}

// Zone sets the zone the date and time are in. It returns f for
// chaining.
func (f *DateTimeField) Zone(z *time.Location) *DateTimeField {
	f.zone = z
	f.date.Zone(z)
	return f
}

// Step sets the interval of the listed times. It returns f for
// chaining.
func (f *DateTimeField) Step(d time.Duration) *DateTimeField { f.clock.Step(d); return f }

// Labels names the two fields. It returns f for chaining.
func (f *DateTimeField) Labels(date, clock string) *DateTimeField {
	f.date.Label(date)
	f.clock.Label(clock)
	return f
}

// OnChange sets the handler of a change; ok is false until both a date
// and a time are set. It returns f for chaining.
func (f *DateTimeField) OnChange(fn func(t time.Time, ok bool)) *DateTimeField {
	f.onChange = fn
	return f
}

// Value returns the date at the time, in the field's zone; ok is false
// unless both are set.
func (f *DateTimeField) Value() (time.Time, bool) {
	day := f.date.Value()
	clock, ok := f.clock.Value()
	if day.IsZero() || !ok {
		return time.Time{}, false
	}
	return clock.On(day.In(f.zone)), true
}

// SetValue sets both from t (in the field's zone) without calling
// OnChange.
func (f *DateTimeField) SetValue(t time.Time) {
	t = t.In(f.zone)
	f.date.SetValue(t)
	f.clock.SetValue(TimeOfDay{t.Hour(), t.Minute()})
}

// Date and Time return the two fields.
func (f *DateTimeField) Date() *DatePicker { return f.date }
func (f *DateTimeField) Time() *TimePicker { return f.clock }

// Layout lays out the two fields side by side.
func (f *DateTimeField) Layout(gtx layout.Context) layout.Dimensions {
	return elements.Row(styles.Styles{Align: properties.AlignCenter, Gap: 8},
		elements.Box(styles.Styles{FlexGrow: 3}, f.date),
		elements.Box(styles.Styles{FlexGrow: 2}, f.clock),
	).Layout(gtx)
}
