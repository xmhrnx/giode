package calendarview

import (
	"image"
	"time"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/datetime"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/i18n"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Sizes, in dp.
const (
	weekdayRowH = 24
	numberRowH  = 26
	laneH       = 20
)

// monthGrid draws the month view: six weeks of days, spanning events in
// bars across them, other events as lines, and "+N more" past the room.
type monthGrid struct {
	v *CalendarView

	from  time.Time
	cellW int
	cellH int
	top   int
	rtl   bool
	hits  []hit
	more  []moreHit
	nums  []image.Rectangle

	tag      struct{}
	origin   image.Point
	pressDay int
	pressEv  *Event
	pressed  bool
}

type moreHit struct {
	r   image.Rectangle
	day int
}

func (m *monthGrid) cellRect(day int) image.Rectangle {
	row, col := day/7, day%7
	if m.rtl {
		col = 6 - col
	}
	x := col * m.cellW
	y := m.top + row*m.cellH
	return image.Rect(x, y, x+m.cellW, y+m.cellH)
}

func (m *monthGrid) dayAt(p image.Point) int {
	if p.Y < m.top || m.cellW <= 0 || m.cellH <= 0 {
		return -1
	}
	col := p.X / m.cellW
	if m.rtl {
		col = 6 - col
	}
	row := (p.Y - m.top) / m.cellH
	if col < 0 || col > 6 || row < 0 || row > 5 {
		return -1
	}
	return row*7 + col
}

func (m *monthGrid) layout(gtx layout.Context, from time.Time, events []*Event) {
	v := m.v
	m.from = from
	m.rtl = kit.RTL(gtx)
	elements.TrackPosition(&m.origin)
	m.update(gtx)
	size := gtx.Constraints.Max
	loc := v.loc()
	cal := loc.Calendar()
	m.cellW = size.X / 7
	m.top = gtx.Dp(weekdayRowH)
	m.cellH = max((size.Y-m.top)/6, gtx.Dp(numberRowH+laneH))
	border := properties.CalcColor("theme.border")
	shown := cal.FromTime(v.anchor.In(v.zone))
	today := datetime.StartOfDay(v.now().In(v.zone))

	area := clip.Rect{Max: size}.Push(gtx.Ops)
	event.Op(gtx.Ops, &m.tag)
	area.Pop()

	// Weekday names.
	for i := range 7 {
		r := m.cellRect(i)
		wd := datetime.AddDays(from, i).Weekday()
		label(gtx, loc.WeekdayName(wd, datetime.Abbreviated), image.Rect(r.Min.X, 0, r.Max.X, m.top), styles.Styles{FontSize: 12, Color: "theme.muted"}, true)
	}
	m.hits, m.more, m.nums = m.hits[:0], m.more[:0], m.nums[:0]
	lh := gtx.Dp(laneH)
	numH := gtx.Dp(numberRowH)
	maxLanes := max((m.cellH-numH-gtx.Dp(2))/lh, 1)
	for week := range 6 {
		weekStart := datetime.AddDays(from, week*7)
		// Grid lines and day numbers.
		for d := range 7 {
			idx := week*7 + d
			r := m.cellRect(idx)
			fill(gtx, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+1), border)
			fill(gtx, image.Rect(r.Min.X, r.Min.Y, r.Min.X+1, r.Max.Y), border)
			day := datetime.AddDays(weekStart, d)
			dd := cal.FromTime(day)
			col := properties.Color("theme.text")
			if dd.Month != shown.Month {
				col = "theme.muted"
			}
			numR := image.Rect(r.Min.X+gtx.Dp(4), r.Min.Y+gtx.Dp(3), r.Min.X+gtx.Dp(30), r.Min.Y+numH-gtx.Dp(1))
			st := styles.Styles{FontSize: 13, Color: col}
			if day.Equal(today) {
				fillRound(gtx, numR, numR.Dy()/2, properties.CalcColor("theme.primary"))
				st.Color = "theme.onprimary"
				st.FontWeight = properties.FontWeightSemiBold
			}
			label(gtx, loc.Digits(itoa(dd.Day)), numR, st, true)
			m.nums = append(m.nums, numR)
		}
		// Spanning events in bars, by lane.
		used := [7]int{}
		bars := layoutWeek(weekStart, events, spans)
		hidden := [7]int{}
		for _, b := range bars {
			if b.lane >= maxLanes-1 && !(b.lane == maxLanes-1 && fitsAll(bars, maxLanes)) {
				for d := b.first; d <= b.last; d++ {
					hidden[d]++
				}
				continue
			}
			r0, r1 := m.cellRect(week*7+b.first), m.cellRect(week*7+b.last)
			x0, x1 := r0.Min.X, r1.Max.X
			if m.rtl {
				x0, x1 = r1.Min.X, r0.Max.X
			}
			y := r0.Min.Y + numH + b.lane*lh
			r := image.Rect(x0+gtx.Dp(2), y, x1-gtx.Dp(2), y+lh-gtx.Dp(2))
			drawBar(gtx, v, b.ev, r)
			m.hits = append(m.hits, hit{r: r, ev: b.ev, day: week*7 + b.first})
			for d := b.first; d <= b.last; d++ {
				used[d] = max(used[d], b.lane+1)
			}
		}
		// Other events as lines under the bars.
		for d := range 7 {
			day := datetime.AddDays(weekStart, d)
			var timed []*Event
			for _, e := range events {
				if !spans(e) && datetime.SameDay(e.Start.In(v.zone), day) {
					timed = append(timed, e)
				}
			}
			sortByStart(timed)
			r := m.cellRect(week*7 + d)
			lane := used[d]
			for i, e := range timed {
				last := lane == maxLanes-1
				if last && (len(timed)-i > 1 || hidden[d] > 0) {
					hidden[d] += len(timed) - i
					break
				}
				if lane >= maxLanes {
					hidden[d] += len(timed) - i
					break
				}
				y := r.Min.Y + numH + lane*lh
				lr := image.Rect(r.Min.X+gtx.Dp(2), y, r.Max.X-gtx.Dp(2), y+lh-gtx.Dp(2))
				m.drawLine(gtx, e, lr)
				m.hits = append(m.hits, hit{r: lr, ev: e, day: week*7 + d})
				lane++
			}
			if hidden[d] > 0 {
				y := r.Min.Y + numH + (maxLanes-1)*lh
				mr := image.Rect(r.Min.X+gtx.Dp(6), y, r.Max.X-gtx.Dp(2), y+lh)
				label(gtx, i18n.N("giode.cal.more", hidden[d]), mr, styles.Styles{FontSize: 11, FontWeight: properties.FontWeightSemiBold, Color: "theme.muted"}, false)
				m.more = append(m.more, moreHit{r: mr, day: week*7 + d})
			}
		}
	}
}

// fitsAll reports whether every bar's lane is below n.
func fitsAll(bars []bar, n int) bool {
	for _, b := range bars {
		if b.lane >= n {
			return false
		}
	}
	return true
}

// drawLine draws a timed event as a dot, its time and its title.
func (m *monthGrid) drawLine(gtx layout.Context, e *Event, r image.Rectangle) {
	v := m.v
	loc := v.loc()
	d := gtx.Dp(7)
	cy := r.Min.Y + r.Dy()/2
	dot := image.Rect(r.Min.X+gtx.Dp(3), cy-d/2, r.Min.X+gtx.Dp(3)+d, cy+d/2)
	if m.rtl {
		dot = image.Rect(r.Max.X-gtx.Dp(3)-d, cy-d/2, r.Max.X-gtx.Dp(3), cy+d/2)
	}
	fillRound(gtx, dot, d/2, eventColor(e))
	text := loc.Format(e.Start.In(v.zone), datetime.Time) + " " + e.Title
	tr := image.Rect(r.Min.X+gtx.Dp(14), r.Min.Y, r.Max.X, r.Max.Y)
	if m.rtl {
		tr = image.Rect(r.Min.X, r.Min.Y, r.Max.X-gtx.Dp(14), r.Max.Y)
	}
	labelMiddle(gtx, text, tr, styles.Styles{FontSize: 12, Color: "theme.text"})
	elements.RecordSemantic(gtx, elements.SemanticNode{
		Kind: elements.SemanticButton, Label: e.Title + ", " + loc.Format(e.Start.In(v.zone), datetime.DateTimeMedium),
		Bounds: r.Add(elements.AbsolutePosition()), Target: eventTarget{v, e},
	}, r.Size())
}

// update handles clicks on events, numbers and "+N more", and drags
// between days: an event to another day, or across empty days to create
// an all-day event.
func (m *monthGrid) update(gtx layout.Context) {
	v := m.v
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &m.tag, Kinds: pointer.Press | pointer.Release | pointer.Cancel})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		pt := image.Pt(int(e.Position.X), int(e.Position.Y))
		switch e.Kind {
		case pointer.Press:
			m.pressed = true
			m.pressDay = m.dayAt(pt)
			m.pressEv = nil
			for i := len(m.hits) - 1; i >= 0; i-- {
				if pt.In(m.hits[i].r) {
					m.pressEv = m.hits[i].ev
					break
				}
			}
		case pointer.Cancel:
			m.pressed = false
		case pointer.Release:
			if !m.pressed {
				continue
			}
			m.pressed = false
			day := m.dayAt(pt)
			if day < 0 || m.pressDay < 0 {
				continue
			}
			if day == m.pressDay {
				m.click(pt, day)
				continue
			}
			delta := day - m.pressDay
			switch {
			case m.pressEv != nil && !m.pressEv.ReadOnly && v.onChange != nil:
				s := datetime.AddDays(m.pressEv.Start.In(v.zone), delta)
				en := datetime.AddDays(m.pressEv.End.In(v.zone), delta)
				v.onChange(*m.pressEv, s, en)
			case m.pressEv == nil && v.onCreate != nil:
				a, b := min(m.pressDay, day), max(m.pressDay, day)
				v.onCreate(datetime.AddDays(m.from, a), datetime.AddDays(m.from, b+1), true)
			}
			kit.RequestFrame(gtx)
		}
	}
}

// click handles a press and release on one day.
func (m *monthGrid) click(pt image.Point, day int) {
	v := m.v
	if m.pressEv != nil {
		if v.onClick != nil {
			v.onClick(*m.pressEv)
		}
		return
	}
	for _, mh := range m.more {
		if pt.In(mh.r) {
			v.anchor = datetime.AddDays(m.from, mh.day)
			v.Mode(Day)
			v.scrolled = false
			return
		}
	}
	if day < len(m.nums) && pt.In(m.nums[day]) {
		d := datetime.AddDays(m.from, day)
		if v.onDay != nil {
			v.onDay(d)
		} else {
			v.anchor = d
			v.Mode(Day)
			v.scrolled = false
		}
		return
	}
	if v.onCreate != nil {
		d := datetime.AddDays(m.from, day)
		v.onCreate(d, datetime.AddDays(d, 1), true)
	}
}
