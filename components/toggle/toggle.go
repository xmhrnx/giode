// Package toggle provides an on/off switch with an animated knob, the
// standard settings control on iOS and Android.
package toggle

import (
	"image"
	"time"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

const animDuration = 150 * time.Millisecond

// Switch is a stateful on/off switch with an optional label. Create it
// once with New and lay it out every frame.
type Switch struct {
	// Contextual gives it a context menu (Context).
	elements.Contextual

	label    string
	st       styles.Styles
	click    widget.Clickable
	pass     func(gtx layout.Context) bool
	on       bool
	onChange func(on bool)
	disabled bool
	// knobOff and knobOn color the knob; white when unset.
	knobOff, knobOn properties.Color

	// pos animates the knob, 0 off to 1 on.
	pos  float32
	last time.Time
	// cache keeps the output between frames; see Layout.
	cache   *elements.MemoCache
	build   func() elements.Element
	watched elements.Watch[drawn]
}

// drawn is what a Switch draws from its own fields.
type drawn struct {
	label           string
	st              styles.Styles
	on, disabled    bool
	knobOff, knobOn properties.Color
	pos             float32
}

// CacheSafe reports true: what it draws comes from its own fields,
// which the cache watches.
func (s *Switch) CacheSafe() bool { return true }

// watch is what it draws from its own fields.
func (s *Switch) watch() drawn {
	return drawn{s.label, s.st, s.on, s.disabled, s.knobOff, s.knobOn, s.pos}
}

// New returns a switch. label is optional; it shows left of the track
// and fills the row, as in a settings list. The styles argument is
// optional: Background is the track when on (default theme.primary),
// BorderColor the track when off (default theme.border), Color the
// label (default theme.text), FontSize the label size.
func New(label string, st ...styles.Styles) *Switch {
	s := &Switch{label: label}
	if len(st) > 0 {
		s.st = st[0]
	}
	return s
}

// KnobColor sets the knob's color when off and when on (white by
// default), so it stays visible on a light track: a white "on" track
// in a monochrome theme hides a white knob. It returns s for chaining.
func (s *Switch) KnobColor(off, on properties.Color) *Switch {
	s.knobOff, s.knobOn = off, on
	return s
}

// OnChange sets the handler called when the user flips the switch. It
// returns s for chaining.
func (s *Switch) OnChange(fn func(on bool)) *Switch {
	s.onChange = fn
	return s
}

// Disabled greys the switch out and ignores input. It returns s for
// chaining.
func (s *Switch) Disabled(v bool) *Switch {
	s.disabled = v
	return s
}

// Checked reports whether the switch is on.
func (s *Switch) Checked() bool { return s.on }

// SetChecked sets the state without invoking OnChange.
func (s *Switch) SetChecked(v bool) { s.on = v }

// Toggle flips the switch and invokes OnChange, as a tap would.
func (s *Switch) Toggle() {
	if s.disabled {
		return
	}
	s.on = !s.on
	if s.onChange != nil {
		s.onChange(s.on)
	}
}

// Layout lays out and updates the switch.
// update handles clicks, reporting whether one toggled the switch.
func (s *Switch) update(gtx layout.Context) bool {
	if s.click.Clicked(gtx) {
		s.Toggle()
		kit.RequestFrame(gtx)
		return true
	}
	return false
}

func (s *Switch) Layout(gtx layout.Context) layout.Dimensions {
	dims := s.layoutSelf(gtx)
	// A right-click or long-press anywhere on it opens its context menu.
	s.AnchorContext(gtx, dims.Size)
	return dims
}

// layoutSelf lays it out without its context menu area.
func (s *Switch) layoutSelf(gtx layout.Context) layout.Dimensions {
	// The output recorded last time is replayed until the label, the
	// value, the styles, the knob's animation, hover, press,
	// constraints or direction change.
	if s.cache == nil {
		s.cache = elements.NewMemoCache()
		uncached := elements.Raw(s.layoutUncached)
		s.build = func() elements.Element { return uncached }
		s.watched.Snap = s.watch
	}
	return s.cache.LayoutOwn(gtx, s.build, &s.watched)
}

// layoutUncached lays it out without the cache.
func (s *Switch) layoutUncached(gtx layout.Context) layout.Dimensions {
	if s.disabled {
		gtx = gtx.Disabled()
	}
	s.update(gtx)
	if s.pass == nil {
		s.pass = kit.ClickablePass(&s.click, func(gtx layout.Context) bool {
			if s.disabled {
				gtx = gtx.Disabled()
			}
			return s.update(gtx)
		})
	}
	kit.HandleInput(s.pass)
	target := float32(0)
	if s.on {
		target = 1
	}
	dt := gtx.Now.Sub(s.last)
	if s.last.IsZero() || dt < 0 || dt > animDuration {
		dt = animDuration
	}
	s.last = gtx.Now
	if s.pos != target {
		step := float32(dt) / float32(animDuration)
		if s.pos < target {
			s.pos = min(target, s.pos+step)
		} else {
			s.pos = max(target, s.pos-step)
		}
		if s.pos != target {
			kit.RequestFrame(gtx)
		}
	}

	st := styles.Merge(styles.Styles{Color: "theme.text", Background: "theme.primary", BorderColor: "theme.border"}, s.st)
	track := elements.Raw(elements.RawFunc(s.track(st)))
	var content elements.Element = track
	if s.label != "" {
		// The row must not wrap: FlexGrow is ignored on wrapped lines, so
		// the switch would sit right after the label, and a long label
		// would push it onto a line of its own. The label wraps instead.
		content = elements.Row(styles.Styles{Width: -1, Align: properties.AlignCenter, Gap: 12, Wrap: properties.WrapNoWrap},
			elements.Box(styles.Styles{FlexGrow: 1, Width: -1}, elements.Text(s.label, styles.Styles{Color: st.Color, FontSize: st.FontSize})),
			track)
	}
	dims := s.click.Layout(gtx, kit.Unselectable(func(gtx layout.Context) layout.Dimensions {
		if s.click.Hovered() && !s.disabled {
			pointer.CursorPointer.Add(gtx.Ops)
		}
		return elements.MinTouchTarget(gtx, content.Layout)
	}))
	kit.BlockDrag(gtx, dims.Size)
	if s.pos != target {
		kit.MarkLive()
	}
	value := "false"
	if s.on {
		value = "true"
	}
	elements.RecordSemantic(gtx, elements.SemanticNode{Kind: elements.SemanticCheckbox, Label: s.label, Value: value, Disabled: s.disabled, Target: s}, dims.Size)
	return dims
}

// track draws the rounded track and its knob.
func (s *Switch) track(st styles.Styles) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		size := image.Pt(gtx.Dp(44), gtx.Dp(26))
		inset := gtx.Dp(3)
		knob := size.Y - 2*inset
		c := properties.CalcColor(properties.LerpColor(st.BorderColor, st.Background, s.pos))
		if s.disabled {
			c.A /= 2
		}
		func() {
			defer clip.UniformRRect(image.Rectangle{Max: size}, size.Y/2).Push(gtx.Ops).Pop()
			paint.Fill(gtx.Ops, c)
		}()
		// Right to left, on is the knob at the left.
		x := inset + int(float32(size.X-2*inset-knob)*s.pos)
		if kit.RTL(gtx) {
			x = size.X - x - knob
		}
		r := image.Rect(x, inset, x+knob, inset+knob)
		defer clip.Ellipse(r).Push(gtx.Ops).Pop()
		off, on := s.knobOff, s.knobOn
		if off == "" {
			off = "#ffffff"
		}
		if on == "" {
			on = "#ffffff"
		}
		paint.Fill(gtx.Ops, properties.CalcColor(properties.LerpColor(off, on, s.pos)))
		return layout.Dimensions{Size: size}
	}
}
