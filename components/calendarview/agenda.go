package calendarview

import (
	"time"

	"gioui.org/layout"

	"github.com/xmhrnx/giode/components/scroll"
	"github.com/xmhrnx/giode/datetime"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/i18n"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// agenda is the agenda view's state: its scroll and rows, kept by
// event and day so their press state lasts.
type agenda struct {
	scroll *scroll.Scroll
	rows   map[agendaKey]*elements.Pressable
	seen   map[agendaKey]bool
	build  func() elements.Element
}

type agendaKey struct {
	ev  *Event
	day int64
}

// layoutAgenda lists the days of the range that have events, each with
// its events.
func (v *CalendarView) layoutAgenda(gtx layout.Context, from, to time.Time, events []*Event) {
	a := &v.agendaState
	if a.rows == nil {
		a.rows = map[agendaKey]*elements.Pressable{}
	}
	loc := v.loc()
	a.seen = map[agendaKey]bool{}
	a.build = func() elements.Element {
		var sections []elements.Element
		for day := from; day.Before(to); day = datetime.AddDays(day, 1) {
			next := datetime.AddDays(day, 1)
			var todays []*Event
			for _, e := range events {
				end := e.End
				if !end.After(e.Start) {
					end = e.Start.Add(time.Minute)
				}
				if e.Start.Before(next) && end.After(day) {
					todays = append(todays, e)
				}
			}
			if len(todays) == 0 {
				continue
			}
			sortByStart(todays)
			rows := []elements.Element{
				elements.Text(loc.Format(day, datetime.DateLong), styles.Styles{FontSize: 13, FontWeight: properties.FontWeightSemiBold, Color: "theme.muted",
					Padding: properties.Inset{Top: 12, Bottom: 4}}),
			}
			for _, e := range todays {
				rows = append(rows, v.agendaRow(e, day))
			}
			sections = append(sections, elements.Box(styles.Styles{FlexDirection: properties.FlexDirectionColumn, Width: -1}, rows...))
		}
		if len(sections) == 0 {
			return elements.Text(i18n.T("giode.cal.no_events"), styles.Styles{Color: "theme.muted", Padding: properties.UniformInset(16)})
		}
		return elements.Box(styles.Styles{FlexDirection: properties.FlexDirectionColumn, Width: -1}, sections...)
	}
	if a.scroll == nil {
		a.scroll = scroll.New(func() elements.Element { return a.build() })
	}
	a.scroll.Layout(gtx)
	// Forget rows of events gone.
	for k := range a.rows {
		if !a.seen[k] {
			delete(a.rows, k)
		}
	}
}

// agendaRow is an event's row in a day: its color, times and title.
func (v *CalendarView) agendaRow(e *Event, day time.Time) elements.Element {
	a := &v.agendaState
	k := agendaKey{e, day.Unix()}
	a.seen[k] = true
	p := a.rows[k]
	if p == nil {
		ev := e
		p = elements.NewPressable(func(s elements.PressState) elements.Element {
			loc := v.loc()
			when := i18n.T("giode.cal.all_day")
			if !ev.AllDay {
				when = loc.Format(ev.Start.In(v.zone), datetime.Time) + " – " + loc.Format(ev.End.In(v.zone), datetime.Time)
			}
			bg := properties.Background("")
			if s.Hovered || s.Focused {
				bg = "theme.surface"
			}
			color := ev.Color
			if color == "" {
				color = "theme.primary"
			}
			return elements.Row(styles.Styles{Width: -1, Align: properties.AlignCenter, Gap: 10, Background: bg, BorderRadius: 6,
				Padding: properties.Inset{Top: 8, Bottom: 8, Left: 8, Right: 8}},
				elements.Box(styles.Styles{Width: 10, Height: 10, BorderRadius: 5, Background: properties.Background(color)}),
				elements.Box(styles.Styles{Width: 150}, elements.Text(when, styles.Styles{FontSize: 13, Color: "theme.muted"})),
				elements.Box(styles.Styles{FlexGrow: 1}, elements.Text(ev.Title, styles.Styles{FontSize: 14, Color: "theme.text"})),
			)
		}).OnClick(func(elements.ClickEvent) {
			if v.onClick != nil {
				v.onClick(*ev)
			}
		}).Label(e.Title)
		a.rows[k] = p
	}
	return p
}
