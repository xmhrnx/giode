package calendarview

import (
	"image"
	"image/color"
	"sort"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/components/button"
	"github.com/xmhrnx/giode/components/iconbutton"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/components/segmented"
	"github.com/xmhrnx/giode/datetime"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/i18n"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Mode is what a CalendarView shows.
type Mode int

const (
	// Month is a month's weeks.
	Month Mode = iota
	// Week is seven days' hours.
	Week
	// Day is a day's hours.
	Day
	// Agenda lists the events of the coming days.
	Agenda
)

// CalendarView shows events by month, week, day or as an agenda. Create
// it once with New, give it events (SetEvents, or Source for a range at
// a time) and lay it out every frame:
//
//	cal := calendarview.New().SetEvents(events).
//		OnEventClick(func(e calendarview.Event) { open(e) }).
//		OnCreate(func(start, end time.Time, allDay bool) { draft(start, end, allDay) }).
//		OnChange(func(e calendarview.Event, start, end time.Time) { reschedule(e.ID, start, end) })
//
// Dragging an empty stretch of hours creates an event, dragging one
// moves it (to another day too), dragging its bottom edge resizes it;
// on touch, a long press starts them. Times snap to Snap (15 minutes).
type CalendarView struct {
	mode   Mode
	anchor time.Time
	zone   *time.Location
	locale *datetime.Locale

	events   []Event
	source   func(from, to time.Time) []Event
	onClick  func(Event)
	onCreate func(start, end time.Time, allDay bool)
	onChange func(e Event, start, end time.Time)
	onDay    func(day time.Time)
	onRange  func(from, to time.Time)

	second     *time.Location
	hourHeight unit.Dp
	snapMin    int
	agendaDays int
	showHeader bool
	modes      []Mode

	prev, next *iconbutton.Button
	today      *button.Button
	switcher   *segmented.Control

	grid        timeGrid
	month       monthGrid
	agendaState agenda
	scrolled    bool
	lastRange   [2]time.Time
	now         func() time.Time
}

// New returns a week view of this week.
func New() *CalendarView {
	v := &CalendarView{mode: Week, zone: time.Local, hourHeight: 48, snapMin: 15, agendaDays: 30, showHeader: true,
		modes: []Mode{Month, Week, Day, Agenda}, now: time.Now}
	v.anchor = time.Now()
	v.prev = iconbutton.New(elements.Glyph(kit.GlyphChevronBack, styles.Styles{Width: 18})).Variant(iconbutton.VariantGhost).
		Label(i18n.T("giode.cal.previous")).OnClick(func() { v.step(-1) })
	v.next = iconbutton.New(elements.Glyph(kit.GlyphChevronForward, styles.Styles{Width: 18})).Variant(iconbutton.VariantGhost).
		Label(i18n.T("giode.cal.next")).OnClick(func() { v.step(1) })
	v.today = button.New(i18n.T("giode.date.today")).Variant(button.VariantOutlined).OnClick(func() { v.Show(v.now()) })
	v.grid.v = v
	v.month.v = v
	return v
}

func (v *CalendarView) loc() datetime.Locale {
	if v.locale != nil {
		return *v.locale
	}
	return datetime.Current()
}

// Locale sets the locale whose conventions it follows. It returns v for
// chaining.
func (v *CalendarView) Locale(l datetime.Locale) *CalendarView { v.locale = &l; return v }

// Zone sets the zone it shows times in (time.Local by default). It
// returns v for chaining.
func (v *CalendarView) Zone(z *time.Location) *CalendarView { v.zone = z; return v }

// SecondZone adds a column of hours in another zone beside the week and
// day views': a colleague's, a customer's. It returns v for chaining.
func (v *CalendarView) SecondZone(z *time.Location) *CalendarView { v.second = z; return v }

// Mode sets what it shows. It returns v for chaining.
func (v *CalendarView) Mode(m Mode) *CalendarView {
	v.mode = m
	if v.switcher != nil {
		v.switcher.SetSelected(v.modeIndex())
	}
	return v
}

// Modes sets the modes its switcher offers, in order; none hides the
// switcher. It returns v for chaining.
func (v *CalendarView) Modes(ms ...Mode) *CalendarView { v.modes = ms; v.switcher = nil; return v }

// Header shows or hides its header (arrows, today, title, switcher). It
// returns v for chaining.
func (v *CalendarView) Header(on bool) *CalendarView { v.showHeader = on; return v }

// HourHeight sets the height of an hour in the week and day views (48dp
// by default). It returns v for chaining.
func (v *CalendarView) HourHeight(h unit.Dp) *CalendarView { v.hourHeight = h; return v }

// Snap sets the minutes dragged times snap to (15 by default). It
// returns v for chaining.
func (v *CalendarView) Snap(minutes int) *CalendarView { v.snapMin = max(minutes, 1); return v }

// AgendaDays sets how many days the agenda lists (30 by default). It
// returns v for chaining.
func (v *CalendarView) AgendaDays(n int) *CalendarView { v.agendaDays = max(n, 1); return v }

// SetEvents sets the events it shows. It returns v for chaining.
func (v *CalendarView) SetEvents(es []Event) *CalendarView { v.events = es; return v }

// Source sets a function returning the events of a range, asked for each
// range shown, in place of SetEvents. It returns v for chaining.
func (v *CalendarView) Source(fn func(from, to time.Time) []Event) *CalendarView {
	v.source = fn
	return v
}

// OnEventClick sets the handler of a click on an event. It returns v for
// chaining.
func (v *CalendarView) OnEventClick(fn func(Event)) *CalendarView { v.onClick = fn; return v }

// OnCreate sets the handler of a dragged out new event. It returns v for
// chaining.
func (v *CalendarView) OnCreate(fn func(start, end time.Time, allDay bool)) *CalendarView {
	v.onCreate = fn
	return v
}

// OnChange sets the handler of an event dragged to a new time or
// resized; the app moves it. It returns v for chaining.
func (v *CalendarView) OnChange(fn func(e Event, start, end time.Time)) *CalendarView {
	v.onChange = fn
	return v
}

// OnDayClick sets the handler of a click on a day's number. It returns v
// for chaining.
func (v *CalendarView) OnDayClick(fn func(day time.Time)) *CalendarView { v.onDay = fn; return v }

// OnRange sets the handler called when it shows another range, to load
// its events. It returns v for chaining.
func (v *CalendarView) OnRange(fn func(from, to time.Time)) *CalendarView { v.onRange = fn; return v }

// Show shows the period holding t.
func (v *CalendarView) Show(t time.Time) { v.anchor = t; v.scrolled = false }

// Anchor returns the moment the shown period holds.
func (v *CalendarView) Anchor() time.Time { return v.anchor }

// CurrentMode returns what it shows.
func (v *CalendarView) CurrentMode() Mode { return v.mode }

// Range returns the start of the first day shown and of the day after
// the last.
func (v *CalendarView) Range() (from, to time.Time) {
	loc := v.loc()
	a := v.anchor.In(v.zone)
	switch v.mode {
	case Month:
		cal := loc.Calendar()
		first := datetime.StartOfMonth(cal, a)
		from = datetime.StartOfWeek(first, loc.FirstWeekday())
		return from, datetime.AddDays(from, 42)
	case Week:
		from = datetime.StartOfWeek(a, loc.FirstWeekday())
		return from, datetime.AddDays(from, 7)
	case Day:
		from = datetime.StartOfDay(a)
		return from, datetime.AddDays(from, 1)
	}
	from = datetime.StartOfDay(a)
	return from, datetime.AddDays(from, v.agendaDays)
}

// step moves by a period.
func (v *CalendarView) step(n int) {
	switch v.mode {
	case Month:
		cal := v.loc().Calendar()
		d := datetime.AddMonths(cal, cal.FromTime(v.anchor.In(v.zone)), n)
		v.anchor = cal.ToTime(d, v.zone)
	case Week:
		v.anchor = datetime.AddDays(v.anchor.In(v.zone), 7*n)
	case Day:
		v.anchor = datetime.AddDays(v.anchor.In(v.zone), n)
	case Agenda:
		v.anchor = datetime.AddDays(v.anchor.In(v.zone), v.agendaDays*n)
	}
}

// visible returns the events overlapping the range, in the view's zone.
func (v *CalendarView) visible(from, to time.Time) []*Event {
	src := v.events
	if v.source != nil {
		src = v.source(from, to)
	}
	out := make([]*Event, 0, len(src))
	for i := range src {
		e := &src[i]
		end := e.End
		if !end.After(e.Start) {
			end = e.Start.Add(time.Minute)
			if e.AllDay {
				end = datetime.AddDays(datetime.StartOfDay(e.Start.In(v.zone)), 1)
			}
		}
		if e.Start.Before(to) && end.After(from) {
			out = append(out, e)
		}
	}
	return out
}

func (v *CalendarView) modeIndex() int {
	for i, m := range v.modes {
		if m == v.mode {
			return i
		}
	}
	return 0
}

func modeName(m Mode) string {
	switch m {
	case Month:
		return i18n.T("giode.cal.month")
	case Week:
		return i18n.T("giode.cal.week")
	case Day:
		return i18n.T("giode.cal.day")
	}
	return i18n.T("giode.cal.agenda")
}

// title names the shown period.
func (v *CalendarView) title(from, to time.Time) string {
	loc := v.loc()
	switch v.mode {
	case Month:
		return loc.Format(v.anchor.In(v.zone), datetime.MonthYear)
	case Day:
		return loc.Format(from, datetime.DateLong)
	}
	last := datetime.AddDays(to, -1)
	return loc.Format(from, datetime.DayMonth) + " – " + loc.Format(last, datetime.DateMedium)
}

// Layout lays the view out in the space it's given.
func (v *CalendarView) Layout(gtx layout.Context) layout.Dimensions {
	size := gtx.Constraints.Max
	if size.Y >= 1<<20 {
		size.Y = gtx.Dp(640)
	}
	from, to := v.Range()
	if r := [2]time.Time{from, to}; r != v.lastRange {
		v.lastRange = r
		if v.onRange != nil {
			v.onRange(from, to)
		}
	}
	events := v.visible(from, to)
	y := 0
	if v.showHeader {
		y = v.layoutHeader(gtx, size.X, from, to)
	}
	body := image.Rect(0, y, size.X, size.Y)
	off := op.Offset(body.Min).Push(gtx.Ops)
	pop := elements.PushOffset(body.Min)
	bgtx := gtx
	bgtx.Constraints = layout.Exact(body.Size())
	switch v.mode {
	case Month:
		v.month.layout(bgtx, from, events)
	case Week, Day:
		days := 7
		if v.mode == Day {
			days = 1
		}
		v.grid.layout(bgtx, from, days, events)
	case Agenda:
		v.layoutAgenda(bgtx, from, to, events)
	}
	pop()
	off.Pop()
	kit.BlockDrag(gtx, size)
	return layout.Dimensions{Size: size}
}

// layoutHeader draws the arrows, today, the title and the switcher, and
// returns its height.
func (v *CalendarView) layoutHeader(gtx layout.Context, width int, from, to time.Time) int {
	if v.switcher == nil && len(v.modes) > 1 {
		names := make([]string, len(v.modes))
		for i, m := range v.modes {
			names[i] = modeName(m)
		}
		v.switcher = segmented.New(names...).OnChange(func(i int) {
			v.mode = v.modes[i]
			v.scrolled = false
		})
		v.switcher.SetSelected(v.modeIndex())
	}
	title := elements.Text(v.title(from, to), styles.Styles{FontSize: 18, FontWeight: properties.FontWeightSemiBold, Color: "theme.text", MaxLines: 1})
	nav := []elements.Element{v.today, v.prev, v.next}
	noWrap := styles.Styles{Align: properties.AlignCenter, Gap: 4, Wrap: properties.WrapNoWrap}
	var h elements.Element
	if width >= gtx.Dp(640) {
		// One row: the buttons, the title, and the switcher at the end.
		items := append(nav, elements.Box(styles.Styles{FlexGrow: 1, Padding: properties.Inset{Start: 8}}, title))
		if v.switcher != nil {
			items = append(items, v.switcher)
		}
		st := noWrap
		st.Width, st.Padding = -1, properties.Inset{Bottom: 8}
		h = elements.Row(st, items...)
	} else {
		// Narrow: the buttons and the switcher, then the title.
		items := append(nav, elements.Box(styles.Styles{FlexGrow: 1}))
		if v.switcher != nil {
			items = append(items, v.switcher)
		}
		st := noWrap
		st.Width = -1
		h = elements.Box(styles.Styles{FlexDirection: properties.FlexDirectionColumn, Width: -1, Gap: 6, Padding: properties.Inset{Bottom: 8}},
			elements.Row(st, items...), title)
	}
	g := gtx
	g.Constraints = layout.Constraints{Min: image.Pt(width, 0), Max: image.Pt(width, gtx.Constraints.Max.Y)}
	return h.Layout(g).Size.Y
}

// --- Drawing helpers ---

func fill(gtx layout.Context, r image.Rectangle, c color.NRGBA) {
	paint.FillShape(gtx.Ops, c, clip.Rect(r).Op())
}

func fillRound(gtx layout.Context, r image.Rectangle, radius int, c color.NRGBA) {
	paint.FillShape(gtx.Ops, c, clip.UniformRRect(r, radius).Op(gtx.Ops))
}

// label draws text in r, clipped to it, from its top-left corner (or
// centered when center), and returns its size.
func label(gtx layout.Context, s string, r image.Rectangle, st styles.Styles, center bool) image.Point {
	if r.Empty() {
		return image.Point{}
	}
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints = layout.Constraints{Max: r.Size()}
	if st.MaxLines == 0 {
		st.MaxLines = 1
	}
	dims := elements.Text(s, st).Layout(g)
	call := m.Stop()
	at := r.Min
	if center {
		at = r.Min.Add(r.Size().Sub(dims.Size).Div(2))
	}
	cl := clip.Rect(r).Push(gtx.Ops)
	o := op.Offset(at).Push(gtx.Ops)
	call.Add(gtx.Ops)
	o.Pop()
	cl.Pop()
	return dims.Size
}

// labelMiddle draws text in r, clipped to it, centered vertically from
// its start edge (left, or right to left the right).
func labelMiddle(gtx layout.Context, s string, r image.Rectangle, st styles.Styles) {
	if r.Empty() {
		return
	}
	if st.MaxLines == 0 {
		st.MaxLines = 1
	}
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints = layout.Constraints{Max: r.Size()}
	dims := elements.Text(s, st).Layout(g)
	call := m.Stop()
	x := r.Min.X
	if kit.RTL(gtx) {
		x = r.Max.X - dims.Size.X
	}
	cl := clip.Rect(r).Push(gtx.Ops)
	o := op.Offset(image.Pt(x, r.Min.Y+(r.Dy()-dims.Size.Y)/2)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	o.Pop()
	cl.Pop()
}

func eventColor(e *Event) color.NRGBA {
	c := e.Color
	if c == "" {
		c = "theme.primary"
	}
	return properties.CalcColor(properties.Color(c))
}

// tint is c faded for an event's background.
func tint(c color.NRGBA, a uint8) color.NRGBA { c.A = a; return c }

// sortByStart orders events by start, longer first at a tie.
func sortByStart(es []*Event) {
	sort.SliceStable(es, func(i, j int) bool {
		if !es[i].Start.Equal(es[j].Start) {
			return es[i].Start.Before(es[j].Start)
		}
		return es[i].End.After(es[j].End)
	})
}

func init() {
	i18n.AddBuiltin("en", map[string]any{
		"giode.cal.previous":  "Previous",
		"giode.cal.next":      "Next",
		"giode.cal.month":     "Month",
		"giode.cal.week":      "Week",
		"giode.cal.day":       "Day",
		"giode.cal.agenda":    "Agenda",
		"giode.cal.more":      map[string]string{"one": "+{n} more", "other": "+{n} more"},
		"giode.cal.all_day":   "All day",
		"giode.cal.no_events": "No events",
	})
	for lang, t := range map[string][8]string{
		"de": {"Zurück", "Weiter", "Monat", "Woche", "Tag", "Terminübersicht", "Ganztägig", "Keine Termine"},
		"fr": {"Précédent", "Suivant", "Mois", "Semaine", "Jour", "Planning", "Toute la journée", "Aucun événement"},
		"es": {"Anterior", "Siguiente", "Mes", "Semana", "Día", "Agenda", "Todo el día", "No hay eventos"},
		"ru": {"Назад", "Вперёд", "Месяц", "Неделя", "День", "Повестка", "Весь день", "Нет событий"},
		"fa": {"قبلی", "بعدی", "ماه", "هفته", "روز", "برنامه", "تمام روز", "رویدادی نیست"},
	} {
		more := map[string]string{"one": "+{n}", "other": "+{n}"}
		switch lang {
		case "fa":
			more = map[string]string{"one": "{n} مورد دیگر", "other": "{n} مورد دیگر"}
		case "de":
			more = map[string]string{"one": "+{n} weitere", "other": "+{n} weitere"}
		case "fr":
			more = map[string]string{"one": "+{n} autre", "other": "+{n} autres"}
		case "es":
			more = map[string]string{"one": "+{n} más", "other": "+{n} más"}
		case "ru":
			more = map[string]string{"one": "ещё {n}", "few": "ещё {n}", "many": "ещё {n}", "other": "ещё {n}"}
		}
		i18n.AddBuiltin(lang, map[string]any{
			"giode.cal.previous": t[0], "giode.cal.next": t[1], "giode.cal.month": t[2], "giode.cal.week": t[3],
			"giode.cal.day": t[4], "giode.cal.agenda": t[5], "giode.cal.all_day": t[6], "giode.cal.no_events": t[7],
			"giode.cal.more": more,
		})
	}
}
