// Package toaster shows small notifications that pop up in a corner of
// the window and disappear on their own.

package toaster

import (
	"sync"
	"time"

	"gioui.org/layout"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/components/transition"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/i18n"
	"github.com/xmhrnx/giode/state"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
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
