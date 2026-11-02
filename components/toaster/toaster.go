// Package toaster shows small notifications that pop up in a corner of
// the window and disappear on their own.
package toaster

import (
	"image"
	"sync"
	"time"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/components/transition"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/i18n"
	"github.com/xmhrnx/giode/state"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Kind selects a toast's accent color.
type Kind int

const (
	Info Kind = iota
	Success
	Warning
	Error
)

func (k Kind) color() properties.Color {
	switch k {
	case Success:
		return "theme.success"
	case Warning:
		return "theme.warning"
	case Error:
		return "theme.danger"
	default:
		return "theme.info"
	}
}

// Placement selects the window corner or edge toasts appear at.
type Placement int

const (
	BottomRight Placement = iota
	BottomLeft
	BottomCenter
	TopRight
	TopLeft
	TopCenter
	// BottomEnd, BottomStart, TopEnd and TopStart follow the reading
	// direction: BottomEnd is the bottom right, or the bottom left in a
	// right-to-left locale. BottomEnd is the default.
	BottomEnd
	BottomStart
	TopEnd
	TopStart
)

func (p Placement) top() bool {
	switch p {
	case TopRight, TopLeft, TopCenter, TopEnd, TopStart:
		return true
	}
	return false
}

// physical resolves the start and end placements for a layout reading
// right to left or not.
func (p Placement) physical(rtl bool) Placement {
	switch p {
	case BottomEnd, BottomStart, TopEnd, TopStart:
		end := p == BottomEnd || p == TopEnd
		right := end != rtl
		switch {
		case p.top() && right:
			return TopRight
		case p.top():
			return TopLeft
		case right:
			return BottomRight
		}
		return BottomLeft
	}
	return p
}

// DefaultDuration is how long a toast stays up, not counting time
// spent hovered.
const DefaultDuration = 5 * time.Second

const (
	toastWidth = 340 // dp
	margin     = 16  // dp, from the window edges
	gap        = 8   // dp, between toasts
)

// Toaster shows toasts. Create it once, lay it out every frame anywhere
// in the view (it takes no space and paints above everything), and
// call Show from anywhere:
//
//	toasts := giode.Toaster()
//	app.Run(func() giode.Element {
//		return giode.Stack(giode.Styles{Width: -1, Height: -1}, content(), toasts)
//	})
//	...
//	toasts.Show(giode.ToastSuccess, "Saved")
//
// Toasts expire after Duration (5s by default) - the timer pauses while
// the pointer is over one - and at most Max (3) show at once, the rest
// waiting their turn. They animate in and out and take their colors
// from the theme.
type Toaster struct {
	st        styles.Styles
	placement Placement
	duration  time.Duration
	max       int
	// inset keeps toasts clear of bars along the window's edges, in dp,
	// added to the margin.
	inset properties.Inset

	mu      sync.Mutex
	nextID  int
	pending []*toast
	dismiss map[int]bool

	toasts []*toast // UI goroutine only
}

type toast struct {
	id     int
	kind   Kind
	msg    string
	label  string
	action func()

	started   bool
	deadline  time.Time
	remaining time.Duration
	paused    bool
	leaving   bool

	trans *transition.Transition
	body  *elements.Pressable
	close *elements.Pressable
	act   *elements.Pressable
}

// New returns a Toaster. The styles argument is optional: Background
// (default theme.surface), Color (text, default theme.text),
// BorderColor (default theme.border), BorderRadius, Width (dp, default
// 340), FontSize.
func New(st ...styles.Styles) *Toaster {
	t := &Toaster{duration: DefaultDuration, max: 3, dismiss: map[int]bool{}, placement: BottomEnd}
	if len(st) > 0 {
		t.st = st[0]
	}
	return t
}

// Placement sets where toasts appear (default BottomEnd: the bottom
// right, or bottom left right to left). It returns
// t for chaining.
func (t *Toaster) Placement(p Placement) *Toaster {
	t.placement = p
	return t
}

// Inset keeps toasts this many dp further from each window edge, clear
// of a custom title bar at the top or a control bar at the bottom. It
// returns t for chaining.
func (t *Toaster) Inset(in properties.Inset) *Toaster {
	t.inset = in
	return t
}

// Duration sets how long toasts stay up (default 5s); zero or less
// keeps them until dismissed. It returns t for chaining.
func (t *Toaster) Duration(d time.Duration) *Toaster {
	t.duration = d
	return t
}

// Max sets how many toasts show at once (default 3, minimum 1). It
// returns t for chaining.
func (t *Toaster) Max(n int) *Toaster {
	t.max = max(1, n)
	return t
}

// Show shows a toast and returns its id. It is safe to call from any
// goroutine.
func (t *Toaster) Show(kind Kind, msg string) int {
	return t.add(&toast{kind: kind, msg: msg})
}

// ShowAction shows a toast with an action button labeled label;
// clicking it runs fn (on the UI goroutine) and dismisses the toast.
// It returns the toast's id and is safe to call from any goroutine.
func (t *Toaster) ShowAction(kind Kind, msg, label string, fn func()) int {
	return t.add(&toast{kind: kind, msg: msg, label: label, action: fn})
}

// Dismiss hides a toast early. It is safe to call from any goroutine.
func (t *Toaster) Dismiss(id int) {
	t.mu.Lock()
	t.dismiss[id] = true
	t.mu.Unlock()
	state.RequestRedraw()
}

// DismissAll hides every toast, including those waiting their turn.
func (t *Toaster) DismissAll() {
	t.mu.Lock()
	t.dismiss[-1] = true
	t.mu.Unlock()
	state.RequestRedraw()
}

// Count returns the number of toasts showing or waiting, as of the
// last frame.
func (t *Toaster) Count() int {
	return len(t.toasts)
}

func (t *Toaster) add(ts *toast) int {
	t.mu.Lock()
	t.nextID++
	ts.id = t.nextID
	t.pending = append(t.pending, ts)
	t.mu.Unlock()
	state.RequestRedraw()
	return ts.id
}

// sync moves toasts queued from other goroutines in and applies
// dismissals.
func (t *Toaster) sync() {
	t.mu.Lock()
	pending := t.pending
	t.pending = nil
	dismiss := t.dismiss
	t.dismiss = map[int]bool{}
	t.mu.Unlock()
	for _, ts := range pending {
		t.prepare(ts)
		t.toasts = append(t.toasts, ts)
	}
	if len(dismiss) == 0 {
		return
	}
	for _, ts := range t.toasts {
		if dismiss[-1] || dismiss[ts.id] {
			t.leave(ts)
		}
	}
}

func (t *Toaster) leave(ts *toast) {
	if !ts.started {
		// Never shown: drop it without an animation.
		ts.leaving = true
		ts.trans.Hidden()
		return
	}
	ts.leaving = true
	ts.trans.Hide()
}

// prepare builds a toast's persistent widgets.
func (t *Toaster) prepare(ts *toast) {
	kind := transition.Fade | transition.SlideUp | transition.Collapse
	if t.placement.top() {
		kind = transition.Fade | transition.SlideDown | transition.Collapse
	}
	ts.body = elements.NewPressable(nil, styles.Styles{Cursor: properties.CursorDefault}).Focusable(false)
	ts.close = elements.NewPressable(func(s elements.PressState) elements.Element {
		st := styles.Styles{Padding: properties.UniformInset(4), BorderRadius: 4}
		if s.Hovered {
			st.Background = "theme.border"
		}
		return elements.Box(st, elements.Glyph(kit.GlyphClose, styles.Styles{Width: 12}))
	}).Label(i18n.T("giode.dismiss")).OnClick(func(elements.ClickEvent) { t.leave(ts) })
	if ts.label != "" {
		ts.act = elements.NewPressable(func(s elements.PressState) elements.Element {
			st := styles.Styles{Padding: properties.SymmetricInset(8, 4), BorderRadius: 4}
			if s.Hovered {
				st.Background = kit.Alpha(ts.kind.color(), 0.15)
			}
			return elements.Box(st, elements.Text(ts.label, styles.Styles{
				Color: ts.kind.color(), FontWeight: properties.FontWeightBold, FontSize: 13,
			}))
		}).Label(ts.label).OnClick(func(elements.ClickEvent) {
			if ts.action != nil {
				ts.action()
			}
			t.leave(ts)
		})
	}
	ts.trans = transition.New(nil, kind, 220*time.Millisecond, properties.EaseOut)
	ts.trans.Child(elements.RawFunc(func(gtx layout.Context) layout.Dimensions {
		return t.toastView(ts).Layout(gtx)
	}))
}

// toastView builds a toast's face.
func (t *Toaster) toastView(ts *toast) elements.Element {
	st := styles.Merge(styles.Styles{
		Background:   "theme.surface",
		Color:        "theme.text",
		BorderColor:  "theme.border",
		BorderWidth:  1,
		BorderRadius: 8,
		FontSize:     14,
	}, t.st)
	box := styles.Styles{
		Width:        -1,
		Background:   st.Background,
		BorderColor:  st.BorderColor,
		BorderWidth:  st.BorderWidth,
		BorderRadius: st.BorderRadius,
		Color:        st.Color,
		Overflow:     properties.OverflowHidden,
		BoxShadow:    properties.BoxShadow{Y: 4, Blur: 14, Color: "#00000040"},
		Align:        properties.AlignCenter,
		Wrap:         properties.WrapNoWrap,
	}
	children := []elements.Element{
		elements.Box(styles.Styles{Width: 4}), // room for the accent stripe
		elements.Box(styles.Styles{FlexGrow: 1, Padding: properties.SymmetricInset(12, 10)},
			elements.Text(ts.msg, styles.Styles{FontSize: st.FontSize})),
	}
	if ts.act != nil {
		children = append(children, ts.act)
	}
	children = append(children, elements.Box(styles.Styles{Padding: properties.Inset{End: 8, Start: 4}}, ts.close))
	row := elements.Row(box, children...)
	// The kind's accent stripe runs down the left edge at the toast's
	// own height, painted once that height is known.
	face := elements.RawFunc(func(gtx layout.Context) layout.Dimensions {
		dims := row.Layout(gtx)
		corners := properties.UniformCorners(gtx.Dp(unit.Dp(st.BorderRadius)))
		cl := properties.RRect(image.Rectangle{Max: dims.Size}, corners).Push(gtx.Ops)
		paint.FillShape(gtx.Ops, properties.CalcColor(ts.kind.color()), clip.Rect{Max: image.Pt(gtx.Dp(4), dims.Size.Y)}.Op())
		cl.Pop()
		return dims
	})
	return ts.body.Content(func(elements.PressState) elements.Element { return face })
}

// Layout queues the toasts above everything else. It takes no space.
func (t *Toaster) Layout(gtx layout.Context) layout.Dimensions {
	// Handles its input without an input pass; see kit.OpaqueInput.
	kit.OpaqueInput()
	t.sync()

	// Start the timers of toasts coming into view, pause hovered ones,
	// and retire expired ones.
	shown := 0
	var next time.Time
	alive := t.toasts[:0]
	for _, ts := range t.toasts {
		if ts.leaving && !ts.trans.Visible() && !ts.trans.Animating() {
			continue // gone
		}
		alive = append(alive, ts)
		if ts.leaving {
			continue
		}
		if shown >= t.max {
			continue // waiting its turn
		}
		shown++
		if !ts.started {
			ts.started = true
			ts.remaining = t.duration
			ts.deadline = gtx.Now.Add(t.duration)
		}
		if t.duration <= 0 {
			continue
		}
		hovered := ts.body.Hovered()
		switch {
		case hovered && !ts.paused:
			ts.paused = true
			ts.remaining = max(0, ts.deadline.Sub(gtx.Now))
		case !hovered && ts.paused:
			ts.paused = false
			ts.deadline = gtx.Now.Add(ts.remaining)
		}
		if ts.paused {
			continue
		}
		if !gtx.Now.Before(ts.deadline) {
			t.leave(ts)
			continue
		}
		if next.IsZero() || ts.deadline.Before(next) {
			next = ts.deadline
		}
	}
	clear(t.toasts[len(alive):])
	t.toasts = alive
	if !next.IsZero() {
		kit.RequestFrameAt(gtx, next)
	}
	if len(t.toasts) == 0 {
		return layout.Dimensions{}
	}
	kit.MarkLive()
	elements.RegisterPortal(gtx, image.Point{}, func() elements.Element {
		return elements.RawFunc(t.layoutStack)
	})
	return layout.Dimensions{}
}

// layoutStack lays the visible toasts out in their corner. It runs as
// a portal covering the window; only the toasts take input.
func (t *Toaster) layoutStack(gtx layout.Context) layout.Dimensions {
	win := gtx.Constraints.Max
	w := gtx.Dp(toastWidth)
	if t.st.Width > 0 {
		w = gtx.Dp(unit.Dp(t.st.Width))
	}
	m := gtx.Dp(margin)
	in := image.Rect(gtx.Dp(unit.Dp(t.inset.Left)), gtx.Dp(unit.Dp(t.inset.Top)), gtx.Dp(unit.Dp(t.inset.Right)), gtx.Dp(unit.Dp(t.inset.Bottom)))
	w = min(w, win.X-2*m-in.Min.X-in.Max.X)
	if w <= 0 {
		return layout.Dimensions{}
	}
	// Newest nearest the edge the toasts come from.
	var visible []*toast
	n := 0
	for _, ts := range t.toasts {
		if !ts.started {
			continue
		}
		visible = append(visible, ts)
		n++
	}
	if t.placement.top() {
		for i, j := 0, len(visible)-1; i < j; i, j = i+1, j-1 {
			visible[i], visible[j] = visible[j], visible[i]
		}
	}
	children := make([]elements.Element, 0, n)
	for _, ts := range visible {
		children = append(children, elements.SizedPx(w, 0, ts.trans))
	}
	stack := elements.Stack(styles.Styles{Gap: gap}, children...)

	// Measure, then place in the corner.
	cgtx := gtx
	cgtx.Constraints = layout.Constraints{Max: image.Pt(w, max(0, win.Y-2*m-in.Min.Y-in.Max.Y))}
	dims := elements.Measure(cgtx, stack)
	var x, y int
	switch t.placement.physical(kit.RTL(gtx)) {
	case BottomLeft, TopLeft:
		x = m + in.Min.X
	case BottomCenter, TopCenter:
		x = in.Min.X + (win.X-in.Min.X-in.Max.X-w)/2
	default:
		x = win.X - w - m - in.Max.X
	}
	if t.placement.top() {
		y = m + in.Min.Y
	} else {
		y = win.Y - m - in.Max.Y - dims.Size.Y
	}
	return elements.OffsetPx(image.Pt(x, y), elements.RawFunc(func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints = cgtx.Constraints
		return stack.Layout(gtx)
	})).Layout(gtx)
}
