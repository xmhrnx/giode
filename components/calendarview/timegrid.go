package calendarview

import (
	"image"
	"time"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/components/scroll"
	"github.com/xmhrnx/giode/datetime"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/i18n"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Sizes, in dp.
const (
	gutterW     = 56
	secondW     = 48
	dayHeaderH  = 52
	barH        = 22
	resizeZone  = 6
	minEventH   = 18
	longPress   = 400 * time.Millisecond
	touchSlopDp = 8
)

// dragKind is what a drag in the time grid does.
type dragKind int

const (
	dragNone dragKind = iota
	dragCreate
	dragMove
	dragResize
)

// hit is an event's rectangle in the hours, last frame.
type hit struct {
	r  image.Rectangle
	ev *Event
	// day is the column it's drawn in.
	day int
}

// timeGrid draws the week and day views: a header of days, a row of
// all-day events, and the hours.
type timeGrid struct {
	v      *CalendarView
	scroll *scroll.Scroll

	// This frame's period and events, for the scrolled content.
	from   time.Time
	days   int
	events []*Event
	colW   int
	left   int // the hours' x
	hourH  int
	rtl    bool
	hits   []hit

	tag struct{}
	// origin is the hours' window position, last frame.
	origin image.Point
	// The drag: what it does, the event, where it started (day and
	// minutes), and where it is.
	kind               dragKind
	ev                 *Event
	startDay, startMin int
	curDay, curMin     int
	grabOffset         int // minutes from the event's start to the press
	moved              bool
	pressAt            time.Time
	pressPos           f32.Point
	touch, pending     bool
}

func (g *timeGrid) layout(gtx layout.Context, from time.Time, days int, events []*Event) {
	v := g.v
	g.from, g.days, g.events = from, days, events
	g.rtl = kit.RTL(gtx)
	size := gtx.Constraints.Max
	gutter := gtx.Dp(gutterW)
	if v.second != nil {
		gutter += gtx.Dp(secondW)
	}
	g.left = gutter
	g.colW = max((size.X-gutter)/days, 1)
	g.hourH = gtx.Dp(v.hourHeight)
	border := properties.CalcColor("theme.border")
	loc := v.loc()
	today := datetime.StartOfDay(v.now().In(v.zone))

	// The days' header.
	hdrH := gtx.Dp(dayHeaderH)
	for i := range days {
		day := datetime.AddDays(from, i)
		x := g.colX(i)
		r := image.Rect(x, 0, x+g.colW, hdrH)
		isToday := day.Equal(today)
		wd := styles.Styles{FontSize: 12, Color: "theme.muted"}
		label(gtx, loc.WeekdayName(day.Weekday(), datetime.Abbreviated), image.Rect(r.Min.X, r.Min.Y+gtx.Dp(4), r.Max.X, r.Min.Y+gtx.Dp(22)), wd, true)
		numR := image.Rect(r.Min.X, r.Min.Y+gtx.Dp(22), r.Max.X, r.Max.Y-gtx.Dp(2))
		numSt := styles.Styles{FontSize: 18, Color: "theme.text"}
		if isToday {
			d := gtx.Dp(28)
			c := numR.Min.Add(numR.Size().Div(2))
			fillRound(gtx, image.Rect(c.X-d/2, c.Y-d/2, c.X+d/2, c.Y+d/2), d/2, properties.CalcColor("theme.primary"))
			numSt.Color = "theme.onprimary"
		}
		label(gtx, loc.Digits(itoa(loc.Calendar().FromTime(day).Day)), numR, numSt, true)
	}
	if v.second != nil {
		label(gtx, datetime.ZoneLabel(v.now().In(v.second)), image.Rect(g.gutterX(0), hdrH-gtx.Dp(18), g.gutterX(0)+gtx.Dp(secondW), hdrH),
			styles.Styles{FontSize: 10, Color: "theme.muted"}, true)
		label(gtx, datetime.ZoneLabel(v.now().In(v.zone)), image.Rect(g.gutterX(1), hdrH-gtx.Dp(18), g.gutterX(1)+gtx.Dp(gutterW), hdrH),
			styles.Styles{FontSize: 10, Color: "theme.muted"}, true)
	}
	fill(gtx, image.Rect(0, hdrH-1, size.X, hdrH), border)

	// The all-day row.
	bars := layoutWeek(from, events, inAllDayRow)
	lanes := 0
	for i := range bars {
		if bars[i].first < days {
			lanes = max(lanes, bars[i].lane+1)
		}
	}
	bh := gtx.Dp(barH)
	pad := gtx.Dp(4)
	allDayH := max(lanes*(bh+2)+2*pad, gtx.Dp(24))
	label(gtx, i18n.T("giode.cal.all_day"), image.Rect(g.gutterX(0), hdrH+pad, g.gutterX(0)+gutter-pad, hdrH+allDayH),
		styles.Styles{FontSize: 11, Color: "theme.muted"}, false)
	for _, b := range bars {
		if b.first >= days {
			continue
		}
		last := min(b.last, days-1)
		x0, x1 := g.colX(b.first), g.colX(last)+g.colW
		if g.rtl {
			x0, x1 = g.colX(last), g.colX(b.first)+g.colW
		}
		y := hdrH + pad + b.lane*(bh+2)
		r := image.Rect(x0+2, y, x1-2, y+bh)
		g.drawBar(gtx, b.ev, r)
	}
	top := hdrH + allDayH
	fill(gtx, image.Rect(0, top-1, size.X, top), border)

	// The hours, scrolling.
	if g.scroll == nil {
		g.scroll = scroll.New(func() elements.Element { return elements.Raw(g.layoutHours) })
	}
	if !v.scrolled {
		v.scrolled = true
		first := 8
		if now := v.now().In(v.zone); now.Hour() > 1 {
			first = min(first, now.Hour()-1)
		}
		g.scroll.ScrollToDp(float32(first) * float32(v.hourHeight))
	}
	off := op.Offset(image.Pt(0, top)).Push(gtx.Ops)
	pop := elements.PushOffset(image.Pt(0, top))
	sg := gtx
	sg.Constraints = layout.Exact(image.Pt(size.X, max(size.Y-top, 0)))
	g.scroll.Layout(sg)
	pop()
	off.Pop()
}

// colX returns the x of day column i, right to left mirrored.
func (g *timeGrid) colX(i int) int {
	if g.rtl {
		return (g.days - 1 - i) * g.colW
	}
	return g.left + i*g.colW
}

// gutterX returns the x of gutter column c (0 the second zone's, or the
// only one; 1 the view's when there's a second).
func (g *timeGrid) gutterX(c int) int {
	x := 0
	if c == 1 {
		x = g.left - int(float32(g.left)*float32(gutterW)/float32(gutterW+secondW))
	}
	if g.rtl {
		return g.days*g.colW + x
	}
	return x
}

// dayAt returns the day column at x, clamped.
func (g *timeGrid) dayAt(x float32) int {
	i := int((x - float32(g.left)) / float32(g.colW))
	if g.rtl {
		i = g.days - 1 - int(x/float32(g.colW))
	}
	return max(0, min(i, g.days-1))
}

func (g *timeGrid) minutesAt(y float32) int {
	return max(0, min(int(y*60/float32(g.hourH)), 24*60))
}

// moment returns minutes past midnight on day column i, by the wall
// clock.
func (g *timeGrid) moment(i, minutes int) time.Time {
	d := datetime.AddDays(g.from, i)
	y, m, dd := d.Date()
	return time.Date(y, m, dd, 0, minutes, 0, 0, d.Location())
}

// layoutHours draws the hours' grid, the events and the drag, and
// handles the pointer.
func (g *timeGrid) layoutHours(gtx layout.Context) layout.Dimensions {
	v := g.v
	elements.TrackPosition(&g.origin)
	g.update(gtx)
	width := gtx.Constraints.Max.X
	height := 24 * g.hourH
	size := image.Pt(width, height)
	border := properties.CalcColor("theme.border")
	loc := v.loc()

	area := clip.Rect{Max: size}.Push(gtx.Ops)
	event.Op(gtx.Ops, &g.tag)
	area.Pop()

	// Hour lines and labels.
	day0 := g.from
	for h := 0; h < 24; h++ {
		y := h * g.hourH
		if h > 0 {
			fill(gtx, image.Rect(g.colX(0)-boolInt(g.rtl)*0, y, g.colX(0)+g.days*g.colW, y+1), border)
		}
		t := time.Date(day0.Year(), day0.Month(), day0.Day(), h, 0, 0, 0, v.zone)
		pattern := "HH:mm"
		if loc.Hour12() {
			pattern = "h a"
		}
		if h > 0 {
			lab := image.Rect(g.gutterX(boolInt(v.second != nil)), y-gtx.Dp(8), g.gutterX(boolInt(v.second != nil))+gtx.Dp(gutterW)-gtx.Dp(6), y+gtx.Dp(8))
			label(gtx, loc.FormatPattern(t, pattern), lab, styles.Styles{FontSize: 11, Color: "theme.muted", TextAlign: properties.TextAlignEnd}, false)
			if v.second != nil {
				o := t.In(v.second)
				lab2 := image.Rect(g.gutterX(0), y-gtx.Dp(8), g.gutterX(0)+gtx.Dp(secondW)-gtx.Dp(4), y+gtx.Dp(8))
				label(gtx, loc.FormatPattern(o, pattern), lab2, styles.Styles{FontSize: 10, Color: "theme.muted", TextAlign: properties.TextAlignEnd}, false)
			}
		}
	}
	// Day separators.
	for i := 0; i <= g.days; i++ {
		x := g.left + i*g.colW
		if g.rtl {
			x = i * g.colW
		}
		fill(gtx, image.Rect(x, 0, x+1, height), border)
	}

	// Events.
	g.hits = g.hits[:0]
	for i := range g.days {
		day := datetime.AddDays(g.from, i)
		for _, p := range layoutDay(day, g.events) {
			if g.kind != dragNone && p.ev == g.ev && g.moved {
				continue // drawn at its new place below
			}
			x := g.colX(i) + g.colW*p.col/p.cols
			w := g.colW*p.span/p.cols - gtx.Dp(2)
			y0 := p.start * g.hourH / 60
			y1 := max(p.end*g.hourH/60, y0+gtx.Dp(minEventH))
			r := image.Rect(x+1, y0+1, x+1+w, y1-1)
			g.drawEvent(gtx, p.ev, r, p.ev.Start, p.ev.End)
			g.hits = append(g.hits, hit{r: r, ev: p.ev, day: i})
		}
	}

	// The current time.
	now := v.now().In(v.zone)
	for i := range g.days {
		if datetime.SameDay(datetime.AddDays(g.from, i), now) {
			y := (now.Hour()*60 + now.Minute()) * g.hourH / 60
			red := properties.CalcColor("theme.danger")
			fill(gtx, image.Rect(g.colX(i), y-1, g.colX(i)+g.colW, y+1), red)
			d := gtx.Dp(10)
			fillRound(gtx, image.Rect(g.colX(i)-d/2, y-d/2, g.colX(i)+d/2, y+d/2), d/2, red)
			// Again at the next minute.
			next := now.Truncate(time.Minute).Add(time.Minute)
			gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(next.Sub(now))})
		}
	}

	// The drag's preview.
	if g.kind != dragNone && g.moved {
		if s, e, day, ok := g.preview(); ok {
			y0 := minutesSince(s, g.moment(day, 0)) * g.hourH / 60
			y1 := max(minutesSince(e, g.moment(day, 0))*g.hourH/60, y0+gtx.Dp(minEventH))
			r := image.Rect(g.colX(day)+1, y0+1, g.colX(day)+g.colW-gtx.Dp(2), y1-1)
			ev := g.ev
			if ev == nil {
				ev = &Event{Title: ""}
			}
			g.drawEvent(gtx, ev, r, s, e)
		}
	}
	return layout.Dimensions{Size: size}
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// minutesSince returns the minutes on the wall clock from day's start
// to t, 24*60 at most.
func minutesSince(t, dayStart time.Time) int {
	if datetime.DaysBetween(dayStart, t) > 0 {
		return 24 * 60
	}
	return t.Hour()*60 + t.Minute()
}

// preview returns where the dragged event would go: start, end and day
// column.
func (g *timeGrid) preview() (start, end time.Time, day int, ok bool) {
	step := g.v.snapMin
	switch g.kind {
	case dragCreate:
		a, b := snap(g.startMin, step), snap(g.curMin, step)
		day = g.startDay
		if b < a {
			a, b = b, a
		}
		if b == a {
			b = a + 60
		}
		return g.moment(day, a), g.moment(day, min(b, 24*60)), day, true
	case dragMove:
		dur := g.ev.End.Sub(g.ev.Start)
		m := snap(g.curMin-g.grabOffset, step)
		m = max(0, min(m, 24*60-step))
		start = g.moment(g.curDay, m)
		return start, start.Add(dur), g.curDay, true
	case dragResize:
		s := g.ev.Start.In(g.v.zone)
		day = datetime.DaysBetween(g.from, s)
		if day < 0 || day >= g.days {
			day = g.startDay
		}
		e := g.moment(g.startDay, snap(g.curMin, step))
		if !e.After(s.Add(time.Duration(step) * time.Minute)) {
			e = s.Add(time.Duration(step) * time.Minute)
		}
		return s, e, day, true
	}
	return
}

// update handles the pointer: clicks on events, and drags.
func (g *timeGrid) update(gtx layout.Context) {
	v := g.v
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &g.tag, Kinds: pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		switch e.Kind {
		case pointer.Press:
			if e.Buttons != 0 && e.Buttons != pointer.ButtonPrimary {
				continue
			}
			g.pressAt, g.pressPos = gtx.Now, e.Position
			g.touch = e.Source == pointer.Touch
			g.moved = false
			g.startDay, g.startMin = g.dayAt(e.Position.X), g.minutesAt(e.Position.Y)
			g.curDay, g.curMin = g.startDay, g.startMin
			g.kind, g.ev = dragCreate, nil
			for i := len(g.hits) - 1; i >= 0; i-- {
				h := g.hits[i]
				pt := image.Pt(int(e.Position.X), int(e.Position.Y))
				if !pt.In(h.r) {
					continue
				}
				g.ev = h.ev
				if h.ev.ReadOnly || v.onChange == nil {
					g.kind = dragNone
				} else if pt.Y >= h.r.Max.Y-gtx.Dp(resizeZone) {
					g.kind = dragResize
				} else {
					g.kind = dragMove
					g.grabOffset = g.startMin - minutesSince(h.ev.Start.In(v.zone), g.moment(h.day, 0))
				}
				break
			}
			if g.kind == dragCreate && v.onCreate == nil {
				g.kind = dragNone
			}
			g.pending = g.touch
			if !g.touch && g.kind != dragNone {
				gtx.Execute(pointer.GrabCmd{Tag: &g.tag, ID: e.PointerID})
			}
		case pointer.Drag:
			if g.pending {
				// On touch, a long press starts the drag; moving first
				// scrolls.
				if gtx.Now.Sub(g.pressAt) < longPress {
					if e.Position.Sub(g.pressPos).Y*e.Position.Sub(g.pressPos).Y > float32(gtx.Dp(touchSlopDp)*gtx.Dp(touchSlopDp)) {
						g.kind, g.pending = dragNone, false
					}
					continue
				}
				g.pending = false
				gtx.Execute(pointer.GrabCmd{Tag: &g.tag, ID: e.PointerID})
			}
			if g.kind == dragNone {
				continue
			}
			g.curDay, g.curMin = g.dayAt(e.Position.X), g.minutesAt(e.Position.Y)
			if !g.moved {
				d := e.Position.Sub(g.pressPos)
				g.moved = d.X*d.X+d.Y*d.Y > float32(gtx.Dp(4)*gtx.Dp(4))
			}
			kit.MarkLive()
			kit.RequestFrame(gtx)
		case pointer.Release:
			g.finish(gtx)
		case pointer.Cancel:
			g.kind, g.ev, g.moved, g.pending = dragNone, nil, false, false
		}
	}
}

// finish ends a press: a click or a drop.
func (g *timeGrid) finish(gtx layout.Context) {
	v := g.v
	kind, ev, moved := g.kind, g.ev, g.moved
	g.kind, g.moved, g.pending = dragNone, false, false
	if !moved {
		if ev != nil && v.onClick != nil {
			v.onClick(*ev)
		} else if ev == nil && kind == dragCreate && v.onCreate != nil {
			s := g.moment(g.startDay, g.startMin/v.snapMin*v.snapMin)
			v.onCreate(s, s.Add(time.Hour), false)
		}
		return
	}
	g.kind = kind
	s, e, _, ok := g.preview()
	g.kind = dragNone
	if !ok {
		return
	}
	switch kind {
	case dragCreate:
		if v.onCreate != nil {
			v.onCreate(s, e, false)
		}
	case dragMove, dragResize:
		if v.onChange != nil && (!s.Equal(ev.Start) || !e.Equal(ev.End)) {
			v.onChange(*ev, s, e)
		}
	}
	kit.RequestFrame(gtx)
}

// drawEvent draws a timed event in r, showing start to end.
func (g *timeGrid) drawEvent(gtx layout.Context, e *Event, r image.Rectangle, start, end time.Time) {
	c := eventColor(e)
	rad := gtx.Dp(4)
	fillRound(gtx, r, rad, tint(c, 52))
	fill(gtx, image.Rect(r.Min.X, r.Min.Y+rad/2, r.Min.X+gtx.Dp(3), r.Max.Y-rad/2), c)
	loc := g.v.loc()
	inner := image.Rect(r.Min.X+gtx.Dp(6), r.Min.Y+gtx.Dp(2), r.Max.X-gtx.Dp(2), r.Max.Y)
	h := label(gtx, e.Title, inner, styles.Styles{FontSize: 12, FontWeight: properties.FontWeightSemiBold, Color: "theme.text"}, false)
	if inner.Dy() > h.Y+gtx.Dp(12) {
		times := loc.Format(start.In(g.v.zone), datetime.Time) + " – " + loc.Format(end.In(g.v.zone), datetime.Time)
		label(gtx, times, image.Rect(inner.Min.X, inner.Min.Y+h.Y, inner.Max.X, inner.Max.Y), styles.Styles{FontSize: 11, Color: "theme.muted"}, false)
	}
	elements.RecordSemantic(gtx, elements.SemanticNode{
		Kind:   elements.SemanticButton,
		Label:  e.Title + ", " + loc.Format(start.In(g.v.zone), datetime.DateTimeMedium) + " – " + loc.Format(end.In(g.v.zone), datetime.Time),
		Bounds: r.Add(elements.AbsolutePosition()), Target: eventTarget{g.v, e},
	}, r.Size())
}

// drawBar draws a spanning event's bar in r.
func (g *timeGrid) drawBar(gtx layout.Context, e *Event, r image.Rectangle) {
	drawBar(gtx, g.v, e, r)
}

// drawBar draws a spanning event's bar in r.
func drawBar(gtx layout.Context, v *CalendarView, e *Event, r image.Rectangle) {
	c := eventColor(e)
	fillRound(gtx, r, gtx.Dp(4), c)
	labelMiddle(gtx, e.Title, image.Rect(r.Min.X+gtx.Dp(6), r.Min.Y, r.Max.X-gtx.Dp(4), r.Max.Y),
		styles.Styles{FontSize: 12, FontWeight: properties.FontWeightSemiBold, Color: "theme.onprimary"})
	elements.RecordSemantic(gtx, elements.SemanticNode{
		Kind: elements.SemanticButton, Label: e.Title + ", " + v.loc().Format(e.Start.In(v.zone), datetime.DateMedium),
		Bounds: r.Add(elements.AbsolutePosition()), Target: eventTarget{v, e},
	}, r.Size())
}

// eventTarget lets a screen reader open an event.
type eventTarget struct {
	v *CalendarView
	e *Event
}

func (t eventTarget) Click() {
	if t.v.onClick != nil {
		t.v.onClick(*t.e)
	}
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}
