package elements

import (
	"image"
	"time"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/i18n"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// longPress is how long a finger must rest on selectable text to
// select it.
const longPress = 500 * time.Millisecond

// selectState is the persistent state of a selectable Text or
// RichText, kept in the keyed store under its key.
type selectState struct {
	// sel does mouse selection for Text: drag to select, double-click
	// for a word, Ctrl/Cmd+C to copy.
	sel widget.Selectable

	// tap observes presses, to detect a long-press or right-click.
	tap      int
	pressing bool
	pressAt  time.Time
	pressPos f32.Point

	// all is set while the whole text is selected and the Copy action
	// shows.
	all      bool
	copyText string
	copy     *copyAction
	// applied is set once a select-all reached sel (mouse).
	applied bool
	// origin is the text's position, exact from the last frame, to
	// place the Copy action.
	origin image.Point
	pass   func(gtx layout.Context) bool
}

func newSelectState() *selectState {
	return &selectState{copy: newCopyAction()}
}

// update processes presses on the text: a touch held still for
// longPress, or a right-click, selects it all.
func (s *selectState) update(gtx layout.Context) {
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &s.tap, Kinds: pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		switch e.Kind {
		case pointer.Press:
			switch {
			case e.Source == pointer.Touch:
				s.pressing, s.pressAt, s.pressPos = true, gtx.Now, e.Position
				kit.RequestFrameAt(gtx, gtx.Now.Add(longPress))
			case e.Buttons.Contain(pointer.ButtonSecondary):
				s.selectAll()
				kit.RequestFrame(gtx)
			}
		case pointer.Drag:
			d := e.Position.Sub(s.pressPos)
			if slop := float32(gtx.Dp(8)); d.X*d.X+d.Y*d.Y > slop*slop {
				s.pressing = false // a scroll or a drag, not a long-press
			}
		case pointer.Release, pointer.Cancel:
			s.pressing = false
		}
	}
	if s.pressing {
		if gtx.Now.Sub(s.pressAt) >= longPress {
			s.pressing = false
			s.selectAll()
		} else {
			kit.RequestFrameAt(gtx, s.pressAt.Add(longPress))
		}
	}
	if s.all {
		HandleDismiss(func() { s.all = false })
		kit.MarkLive()
	}
}

// passFunc returns the input pass (see kit.HandleInput): long-presses
// and right-clicks select it all.
func (s *selectState) passFunc() func(gtx layout.Context) bool {
	if s.pass == nil {
		s.pass = func(gtx layout.Context) bool {
			was := s.all
			s.update(gtx)
			return s.all != was
		}
	}
	return s.pass
}

func (s *selectState) selectAll() {
	s.all = true
}

// layoutCopy registers the press area over the text and, while it's
// all selected, the Copy action above it. text is what Copy copies.
func (s *selectState) layoutCopy(gtx layout.Context, size image.Point, text string) {
	TrackPosition(&s.origin)
	s.copyText = text
	// Pass-through, so a scroll under the text keeps scrolling and the
	// text's own handlers keep working.
	ps := pointer.PassOp{}.Push(gtx.Ops)
	cl := clip.Rect{Max: size}.Push(gtx.Ops)
	event.Op(gtx.Ops, &s.tap)
	cl.Pop()
	ps.Pop()
	if !s.all {
		return
	}
	s.copy.show(image.Rectangle{Min: s.origin, Max: s.origin.Add(size)}, s.copyText, func() { s.all = false })
}

// copyAction is the Copy button shown above selected text on touch
// screens and after a right-click.
type copyAction struct {
	btn     *Pressable
	dismiss widget.Clickable
	now     bool
}

func newCopyAction() *copyAction {
	c := &copyAction{}
	c.btn = NewPressable(func(p PressState) Element {
		bg := "#1f2937"
		if p.Pressed || p.Hovered {
			bg = "#374151"
		}
		return Box(styles.Styles{Background: bg, BorderRadius: 8, Padding: properties.SymmetricInset(14, 8)},
			Text(i18n.T("giode.copy"), styles.Styles{Color: "#f9fafb", FontSize: 14}))
	}).Label(i18n.T("giode.copy")).OnClick(func(ClickEvent) { c.now = true })
	return c
}

// show registers an overlay with the Copy button above (or below) at,
// the selection's bounds in window coordinates, and a tap-anywhere-else
// area. A copy writes text to the clipboard; either ends the selection
// by calling done.
func (c *copyAction) show(at image.Rectangle, text string, done func()) {
	RegisterOverlay(func() Element {
		return Raw(func(gtx layout.Context) layout.Dimensions {
			win := gtx.Constraints.Max
			if c.now {
				c.now = false
				done()
				kit.WriteClipboard(gtx, text)
				kit.RequestFrame(gtx)
				return layout.Dimensions{}
			}
			if c.dismiss.Clicked(gtx) {
				done()
				kit.RequestFrame(gtx)
				return layout.Dimensions{}
			}
			// A tap anywhere else ends the selection.
			c.dismiss.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Dimensions{Size: win}
			})
			track := TrackChild()
			macro := op.Record(gtx.Ops)
			cgtx := gtx
			cgtx.Constraints.Min = image.Point{}
			dims := c.btn.Layout(cgtx)
			call := macro.Stop()
			track.End()
			pos := image.Pt(at.Min.X, at.Min.Y-dims.Size.Y-gtx.Dp(8))
			if pos.Y < gtx.Dp(8) {
				pos.Y = at.Max.Y + gtx.Dp(8)
			}
			pos.X = max(gtx.Dp(8), min(pos.X, win.X-dims.Size.X-gtx.Dp(8)))
			track.Place(pos)
			defer op.Offset(pos).Push(gtx.Ops).Pop()
			call.Add(gtx.Ops)
			return layout.Dimensions{Size: win}
		})
	})
}

// Selectable makes the text selectable. With a mouse, drag to select,
// double-click for a word, and press Ctrl+C (Cmd+C) to copy; a
// right-click selects it all and offers Copy. On a touch screen, a
// long-press selects it all and offers Copy, and dragging over the
// text still scrolls whatever it's in. key identifies the text across
// frames (a message ID, say), since Text itself is rebuilt every frame;
// the selection lives in the keyed store (see Keyed). It returns t for
// chaining.
func (t *TextElement) Selectable(key any) *TextElement {
	t.selKey = key
	return t
}

// layoutSelectable lays the text out with its selection state.
func (t *TextElement) layoutSelectable(gtx layout.Context, st styles.Styles, str string, face font.Font, size unit.Sp, textColor op.CallOp) layout.Dimensions {
	s := Keyed("giode.selectable", t.selKey, newSelectState)
	s.update(gtx)
	kit.HandleInput(s.passFunc())
	if !TouchInput() {
		// widget.Selectable's mouse selection handles its own events.
		kit.OpaqueInput()
	}

	selMacro := op.Record(gtx.Ops)
	paint.ColorOp{Color: properties.CalcColor(selectionColor(st.Color))}.Add(gtx.Ops)
	selColor := selMacro.Stop()

	s.sel.Alignment = st.TextAlign.LayoutAlignment()
	s.sel.MaxLines = int(st.MaxLines)
	s.sel.LineHeight = unit.Sp(st.LineHeight)
	if st.LineHeight != 0 {
		s.sel.LineHeightScale = 1
	}
	if s.sel.Text() != str {
		s.sel.SetText(str)
	}

	var dims layout.Dimensions
	if TouchInput() {
		// widget.Selectable's drag would take touch drags from an
		// enclosing Scroll, so on touch the text is laid out plainly
		// and selection works by long-press.
		label := widget.Label{Alignment: s.sel.Alignment, MaxLines: s.sel.MaxLines, LineHeight: s.sel.LineHeight, LineHeightScale: s.sel.LineHeightScale}
		if s.all {
			macro := op.Record(gtx.Ops)
			dims = label.Layout(gtx, fonts.Shaper(), face, size, str, textColor)
			call := macro.Stop()
			func() {
				defer clip.UniformRRect(image.Rectangle{Max: dims.Size}, gtx.Dp(3)).Push(gtx.Ops).Pop()
				paint.Fill(gtx.Ops, properties.CalcColor(selectionColor(st.Color)))
			}()
			call.Add(gtx.Ops)
		} else {
			dims = label.Layout(gtx, fonts.Shaper(), face, size, str, textColor)
		}
	} else {
		switch {
		case s.all && !s.applied:
			s.sel.SetCaret(0, len([]rune(str)))
			s.applied = true
		case !s.all && s.applied:
			s.sel.ClearSelection()
			s.applied = false
		}
		dims = s.sel.Layout(gtx, fonts.Shaper(), face, size, textColor, selColor)
		if s.applied && s.sel.SelectionLen() == 0 {
			// A click cleared the selection.
			s.all, s.applied = false, false
		}
	}
	s.layoutCopy(gtx, dims.Size, str)
	return dims
}
