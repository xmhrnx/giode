// Package numberinput provides a numeric field with − and + steppers,
// a range and a step.
package numberinput

import (
	"image"
	"math"
	"strconv"
	"strings"
	"time"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/xmhrnx/giode/components/input"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Repeat timing for a held stepper.
const (
	repeatDelay = 400 * time.Millisecond
	repeatEvery = 60 * time.Millisecond
)

// NumberInput is a stateful numeric field between − and + buttons.
// Create it once with New and lay it out every frame:
//
//	size := numberinput.New(12).Range(6, 96).OnChange(func(v float64) { setFontSize(v) })
//
// Typing changes the value as soon as the text is a number in range;
// leaving the field or pressing Enter puts anything else back in range
// (or back to the value, for text that isn't a number). The buttons
// step the value, repeating while held; Up and Down do too, by ten
// steps with Shift. Commas are read as decimal points.
type NumberInput struct {
	// Contextual gives it a context menu (Context).
	elements.Contextual

	field    *input.Input
	value    float64
	min, max float64
	step     float64
	decimals int
	onChange func(v float64)
	st       styles.Styles

	dec, inc  widget.Clickable
	held      int // -1 or +1 while a stepper is held, else 0
	nextStep  time.Time
	wasFocus  bool
	lastValid string
}

// New returns a NumberInput holding value, with no range, a step of 1
// and no decimals. The styles argument is optional: Width (the whole
// control, default 140dp; -1 fills), and the field's styles otherwise
// (FontSize, Color, BorderColor...).
func New(value float64, st ...styles.Styles) *NumberInput {
	n := &NumberInput{value: value, min: math.Inf(-1), max: math.Inf(1), step: 1}
	if len(st) > 0 {
		n.st = st[0]
	}
	n.field = input.New("").Keyboard(input.KeyboardNumeric)
	n.field.OnChange(n.typed).Submit(func(string) { n.commit() }).OnBlur(n.commit)
	n.field.OnKeys(n.arrow, key.NameUpArrow, key.NameDownArrow)
	n.field.Styles(n.fieldStyles())
	n.show()
	return n
}

// fieldStyles are the field's: the given ones but the width, which is
// the whole control's.
func (n *NumberInput) fieldStyles() styles.Styles {
	st := n.st
	st.Width = -1
	if st.TextAlign == "" {
		st.TextAlign = properties.TextAlignCenter
	}
	return st
}

// Value returns the current value.
func (n *NumberInput) Value() float64 {
	return n.value
}

// SetValue sets the value, put in range and rounded to the decimals,
// without calling OnChange.
func (n *NumberInput) SetValue(v float64) {
	n.value = n.clamp(v)
	n.show()
}

// Change sets the value and calls OnChange when it changed, as a user
// edit would.
func (n *NumberInput) Change(v float64) {
	v = n.clamp(v)
	changed := v != n.value
	n.value = v
	n.show()
	if changed && n.onChange != nil {
		n.onChange(v)
	}
}

// Range limits the value to [min, max]; math.Inf(-1) and math.Inf(1)
// leave a side open. It returns n for chaining.
func (n *NumberInput) Range(min, max float64) *NumberInput {
	n.min, n.max = min, max
	n.value = n.clamp(n.value)
	n.show()
	return n
}

// Step sets how much the buttons and arrow keys change the value. It
// returns n for chaining.
func (n *NumberInput) Step(step float64) *NumberInput {
	if step > 0 {
		n.step = step
	}
	return n
}

// Decimals sets how many decimals the value keeps and shows. It
// returns n for chaining.
func (n *NumberInput) Decimals(d int) *NumberInput {
	n.decimals = max(0, d)
	n.value = n.clamp(n.value)
	n.show()
	return n
}

// OnChange sets the handler called when the user changes the value. It
// returns n for chaining.
func (n *NumberInput) OnChange(fn func(v float64)) *NumberInput {
	n.onChange = fn
	return n
}

// Styles replaces the styles (see New). It returns n for chaining.
func (n *NumberInput) Styles(st styles.Styles) *NumberInput {
	n.st = st
	n.field.Styles(n.fieldStyles())
	return n
}

// Label sets the accessible name reported to automation. It returns n
// for chaining.
func (n *NumberInput) Label(label string) *NumberInput {
	n.field.Label(label)
	return n
}

// Field returns the text field, for focusing it (Focus, Blur) or
// adding handlers.
func (n *NumberInput) Field() *input.Input {
	return n.field
}

// clamp puts v in range, rounded to the decimals.
func (n *NumberInput) clamp(v float64) float64 {
	if math.IsNaN(v) {
		v = 0
	}
	p := math.Pow(10, float64(n.decimals))
	v = math.Round(v*p) / p
	return math.Min(math.Max(v, n.min), n.max)
}

func (n *NumberInput) format(v float64) string {
	return strconv.FormatFloat(v, 'f', n.decimals, 64)
}

// show puts the value in the field.
func (n *NumberInput) show() {
	n.lastValid = n.format(n.value)
	n.field.SetText(n.lastValid)
}

// parse reads text as a number, with a comma as a decimal point.
func parse(text string) (float64, bool) {
	text = strings.TrimSpace(strings.ReplaceAll(text, ",", "."))
	v, err := strconv.ParseFloat(text, 64)
	return v, err == nil && !math.IsInf(v, 0) && !math.IsNaN(v)
}

// typed applies text typed into the field when it's a number in range.
func (n *NumberInput) typed(text string) {
	v, ok := parse(text)
	if !ok || v < n.min || v > n.max {
		return
	}
	v = n.clamp(v)
	if v != n.value {
		n.value = v
		if n.onChange != nil {
			n.onChange(v)
		}
	}
}

// commit puts the field's text back in range, or back to the value.
func (n *NumberInput) commit() {
	if v, ok := parse(n.field.Text()); ok {
		n.Change(v)
		return
	}
	n.show()
}

// arrow steps with Up and Down, ten steps with Shift.
func (n *NumberInput) arrow(e key.Event) {
	steps := 1.0
	if e.Modifiers.Contain(key.ModShift) {
		steps = 10
	}
	if e.Name == key.NameDownArrow {
		steps = -steps
	}
	n.stepBy(steps)
}

// stepBy changes the value by steps steps, from the typed text if it's
// a number.
func (n *NumberInput) stepBy(steps float64) {
	v := n.value
	if t, ok := parse(n.field.Text()); ok {
		v = t
	}
	n.Change(v + steps*n.step)
}

// update handles the steppers: a click steps once, a hold repeats.
func (n *NumberInput) update(gtx layout.Context) {
	// Process this frame's presses first, so Pressed is current; the
	// clicks themselves aren't steps (a press is).
	for _, c := range []*widget.Clickable{&n.dec, &n.inc} {
		for {
			if _, ok := c.Update(gtx); !ok {
				break
			}
		}
	}
	held := 0
	switch {
	case n.dec.Pressed() && n.value > n.min:
		held = -1
	case n.inc.Pressed() && n.value < n.max:
		held = 1
	}
	now := gtx.Now
	if held != 0 && held != n.held {
		n.stepBy(float64(held))
		n.nextStep = now.Add(repeatDelay)
	} else if held != 0 && !now.Before(n.nextStep) {
		n.stepBy(float64(held))
		n.nextStep = now.Add(repeatEvery)
	}
	n.held = held
	if held != 0 {
		kit.RequestFrameAt(gtx, n.nextStep)
	}
}

// Layout lays out the field between its − and + buttons.
func (n *NumberInput) Layout(gtx layout.Context) layout.Dimensions {
	kit.OpaqueInput()
	n.update(gtx)

	w := gtx.Dp(140)
	switch {
	case n.st.Width < 0:
		w = gtx.Constraints.Max.X
	case n.st.Width > 0:
		w = gtx.Dp(unit.Dp(n.st.Width))
	}
	w = min(max(w, gtx.Constraints.Min.X), gtx.Constraints.Max.X)

	// The buttons are squares as tall as the field.
	fgtx := gtx
	fgtx.Constraints = layout.Constraints{Max: image.Pt(w, gtx.Constraints.Max.Y)}
	h := elements.Measure(fgtx, n.field).Size.Y
	gap := gtx.Dp(4)
	fw := max(0, w-2*h-2*gap)

	n.stepper(gtx, &n.dec, kit.GlyphMinus, h, n.value > n.min)
	off := op.Offset(image.Pt(h+gap, 0)).Push(gtx.Ops)
	pop := elements.PushOffset(image.Pt(h+gap, 0))
	fgtx.Constraints = layout.Exact(image.Pt(fw, h))
	n.field.Layout(fgtx)
	pop()
	off.Pop()
	off = op.Offset(image.Pt(w-h, 0)).Push(gtx.Ops)
	pop = elements.PushOffset(image.Pt(w-h, 0))
	n.stepper(gtx, &n.inc, kit.GlyphPlus, h, n.value < n.max)
	pop()
	off.Pop()

	size := image.Pt(w, h)
	n.AnchorContext(gtx, size)
	return layout.Dimensions{Size: size}
}

// stepper lays out a − or + button, h square, greyed when it can't
// step.
func (n *NumberInput) stepper(gtx layout.Context, c *widget.Clickable, g kit.Glyph, h int, enabled bool) {
	gtx.Constraints = layout.Exact(image.Pt(h, h))
	c.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		size := image.Pt(h, h)
		r := properties.BorderRadius(gtx.Dp(6))
		bg := properties.Color("theme.surface")
		if enabled && c.Pressed() {
			bg = kit.Alpha("theme.text", 0.16)
		} else if enabled && c.Hovered() {
			bg = kit.Alpha("theme.text", 0.08)
		}
		properties.PaintBackground(gtx.Ops, size, properties.Background(bg), r)
		bc := n.st.BorderColor
		if bc == "" {
			bc = "theme.border"
		}
		properties.PaintBorder(gtx.Ops, size, properties.BorderWidth(max(1, gtx.Dp(1))), properties.BorderColor(bc), r)
		col := properties.CalcColor(properties.Color("theme.text"))
		if !enabled {
			col = properties.CalcColor(properties.Color("theme.muted"))
		}
		gs := unit.Dp(16)
		at := image.Pt((h-gtx.Dp(gs))/2, (h-gtx.Dp(gs))/2)
		defer op.Offset(at).Push(gtx.Ops).Pop()
		kit.DrawGlyph(gtx, g, gs, col)
		return layout.Dimensions{Size: size}
	})
	kit.BlockDrag(gtx, image.Pt(h, h))
}
