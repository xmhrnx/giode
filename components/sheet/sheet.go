// Package sheet provides a modal sheet: a panel that slides up from the
// bottom of the window on phones and, with Adaptive, shows as a
// centered dialog on wide screens.
package sheet

import (
	"image"
	"image/color"
	"time"

	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/components/scroll"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Detent is a height a bottom sheet can rest at.
type Detent int

const (
	// DetentFit sizes the sheet to its content, up to nearly the full
	// window (the default).
	DetentFit Detent = iota
	// DetentMedium is half the window.
	DetentMedium
	// DetentLarge is nearly the full window.
	DetentLarge
)

const animDuration = 220 * time.Millisecond

// adaptiveWidth is the window width, in dp, from which an Adaptive
// sheet shows as a centered dialog.
const adaptiveWidth = 600

// Sheet is a modal panel over a dimmed scrim: a bottom sheet that
// slides up from the bottom edge, or with Adaptive a centered dialog on
// wide windows. It has an optional title, a scrolling body and a
// footer pinned below it. The scrim, Back and Escape close it, as does
// dragging its handle down; with several Detents, dragging moves it
// between them.
//
// Like Drawer it takes no space in the layout and paints over the
// window while open, so it can sit anywhere in the tree. Create it
// once with New, outside the view function.
type Sheet struct {
	open     *bool
	content  func() elements.Element
	footer   func() elements.Element
	title    string
	st       styles.Styles
	detents  []Detent
	detent   int
	adaptive bool
	fixed    bool

	// progress is how far shown the sheet is, 0 to 1.
	progress float32
	last     time.Time
	wasOpen  bool

	scrim widget.Clickable
	panel widget.Clickable
	body  *scroll.Scroll

	drag     gesture.Drag
	dragFrom float32
	dragDY   float32
	dragging bool
	panelH   int
}

// New returns a Sheet bound to open: it shows whenever *open is true.
// content builds the body fresh every frame the sheet is shown. The
// styles argument is optional and styles the panel: Background
// (default theme.surface), Padding (default 16 on the sides), Color.
func New(open *bool, content func() elements.Element, st ...styles.Styles) *Sheet {
	s := &Sheet{open: open, content: content, detents: []Detent{DetentFit}}
	if len(st) > 0 {
		s.st = st[0]
	}
	s.body = scroll.New(func() elements.Element { return s.content() }).FitContent(true)
	return s
}

// Title sets the heading shown above the body. It returns s for
// chaining.
func (s *Sheet) Title(t string) *Sheet {
	s.title = t
	return s
}

// Footer sets content pinned below the scrolling body, such as action
// buttons. It returns s for chaining.
func (s *Sheet) Footer(f func() elements.Element) *Sheet {
	s.footer = f
	return s
}

// Detents sets the heights a bottom sheet rests at, smallest first; it
// opens at the first. Dragging the handle moves between them, and
// below the first closes the sheet. It returns s for chaining.
func (s *Sheet) Detents(d ...Detent) *Sheet {
	if len(d) > 0 {
		s.detents = d
		s.detent = 0
	}
	return s
}

// Adaptive shows the sheet as a centered dialog when the window is at
// least 600dp wide, and as a bottom sheet below that. It returns s for
// chaining.
func (s *Sheet) Adaptive(on bool) *Sheet {
	s.adaptive = on
	return s
}

// Dismissible controls whether the scrim, Back, Escape and dragging
// close the sheet (on by default). It returns s for chaining.
func (s *Sheet) Dismissible(on bool) *Sheet {
	s.fixed = !on
	return s
}

// Styles replaces the panel styles from the next frame. It returns s
// for chaining.
func (s *Sheet) Styles(st styles.Styles) *Sheet {
	s.st = st
	return s
}

// Opened reports whether the sheet is open.
func (s *Sheet) Opened() bool {
	return s.open != nil && *s.open
}

func (s *Sheet) close() {
	if s.open != nil && !s.fixed {
		*s.open = false
	}
}

// Layout updates the sheet and, while it's visible, registers the
// overlay that paints it. It takes no space.
func (s *Sheet) Layout(gtx layout.Context) layout.Dimensions {
	// Handles its input without an input pass; see kit.OpaqueInput.
	kit.OpaqueInput()
	isOpen := s.Opened()
	if isOpen && !s.wasOpen {
		s.detent = 0
	}
	s.wasOpen = isOpen
	if isOpen && !s.fixed {
		elements.HandleDismiss(s.close)
	}
	dt := gtx.Now.Sub(s.last)
	if s.last.IsZero() || dt < 0 || dt > animDuration {
		dt = animDuration / 10
	}
	s.last = gtx.Now
	target := float32(0)
	if isOpen {
		target = 1
	}
	if s.progress != target && !s.dragging {
		step := float32(dt) / float32(animDuration)
		if s.progress < target {
			s.progress = min(target, s.progress+step)
		} else {
			s.progress = max(target, s.progress-step)
		}
	}
	if s.progress != target || s.dragging {
		kit.RequestFrame(gtx)
	}
	if s.progress > 0 {
		elements.RegisterOverlay(func() elements.Element { return elements.Raw(s.layoutOverlay) })
	}
	return layout.Dimensions{}
}

// dialog reports whether the sheet shows as a centered dialog now.
func (s *Sheet) dialog(gtx layout.Context) bool {
	return s.adaptive && gtx.Constraints.Max.X >= gtx.Dp(adaptiveWidth)
}

func (s *Sheet) layoutOverlay(gtx layout.Context) layout.Dimensions {
	win := gtx.Constraints.Max
	if s.scrim.Clicked(gtx) {
		s.close()
		kit.RequestFrame(gtx)
	}
	ease := 1 - (1-s.progress)*(1-s.progress) // ease out
	func() {
		defer clip.Rect{Max: win}.Push(gtx.Ops).Pop()
		s.scrim.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			paint.Fill(gtx.Ops, color.NRGBA{A: uint8(0x66 * ease)})
			return layout.Dimensions{Size: win}
		})
	}()
	if s.dialog(gtx) {
		s.layoutDialog(gtx, win, ease)
	} else {
		s.layoutBottom(gtx, win, ease)
	}
	return layout.Dimensions{Size: win}
}

// height returns the panel height for a detent, or 0 for DetentFit.
func detentHeight(d Detent, win image.Point) int {
	switch d {
	case DetentMedium:
		return win.Y / 2
	case DetentLarge:
		return win.Y * 92 / 100
	}
	return 0
}

// layoutBottom draws the bottom sheet, sliding up by ease.
func (s *Sheet) layoutBottom(gtx layout.Context, win image.Point, ease float32) {
	// Drags on the handle and title move the sheet.
	for {
		e, ok := s.drag.Update(gtx.Metric, gtx.Source, gesture.Vertical)
		if !ok {
			break
		}
		switch e.Kind {
		case pointer.Press:
			s.dragFrom, s.dragDY = e.Position.Y, 0
		case pointer.Drag:
			dy := e.Position.Y - s.dragFrom
			if !s.dragging && abs(dy) < float32(gtx.Dp(8)) {
				continue
			}
			s.dragging = true
			s.dragDY = dy
		case pointer.Release, pointer.Cancel:
			if !s.dragging {
				continue
			}
			s.dragging = false
			s.settle(gtx, win)
			s.dragDY = 0
		}
		kit.RequestFrame(gtx)
	}

	width := min(win.X, gtx.Dp(640))
	maxH := win.Y * 92 / 100
	fixedH := detentHeight(s.detents[min(s.detent, len(s.detents)-1)], win)
	if s.dragging && s.dragDY < 0 && fixedH > 0 {
		// Dragging up stretches a detent sheet toward the next one.
		fixedH = min(maxH, fixedH-int(s.dragDY))
	}
	call, h, track := s.panelOps(gtx, width, maxH, fixedH, true)
	s.panelH = h

	y := win.Y - int(float32(h)*ease)
	if s.dragging && s.dragDY > 0 {
		y += int(s.dragDY)
	}
	pos := image.Pt((win.X-width)/2, y)
	track.Place(pos)
	defer op.Offset(pos).Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
}

// settle picks where a released drag leaves the sheet: the nearest
// detent, or closed when pulled well below the first.
func (s *Sheet) settle(gtx layout.Context, win image.Point) {
	dy := s.dragDY
	threshold := float32(max(gtx.Dp(80), s.panelH/4))
	switch {
	case dy > threshold && s.detent == 0:
		s.close()
	case dy > threshold:
		s.detent--
	case dy < -threshold && s.detent < len(s.detents)-1:
		s.detent++
	}
}

// layoutDialog draws the sheet as a centered dialog, fading in.
func (s *Sheet) layoutDialog(gtx layout.Context, win image.Point, ease float32) {
	width := min(win.X-gtx.Dp(32), gtx.Dp(560))
	call, h, track := s.panelOps(gtx, width, win.Y*85/100, 0, false)
	pos := image.Pt((win.X-width)/2, (win.Y-h)/2+int(float32(gtx.Dp(24))*(1-ease)))
	track.Place(pos)
	defer op.Offset(pos).Push(gtx.Ops).Pop()
	defer paint.PushOpacity(gtx.Ops, ease).Pop()
	call.Add(gtx.Ops)
}

// panelOps records the panel - handle, title, body, footer - at width,
// fixedH tall or, when fixedH is 0, as tall as its content up to maxH,
// and returns it with its height. Semantic nodes and portals inside
// follow it to where it's replayed.
func (s *Sheet) panelOps(gtx layout.Context, width, maxH, fixedH int, handle bool) (op.CallOp, int, elements.ChildTracker) {
	st := styles.Merge(styles.Styles{
		Background: "theme.surface",
		Color:      "theme.text",
		Padding:    properties.Inset{Left: 16, Right: 16},
	}, s.st)
	if handle {
		st.BorderRadii = properties.TopCorners(16)
	} else if st.BorderRadius == 0 {
		st.BorderRadius = 14
	}
	pad := st.Padding

	var header []elements.Element
	if handle {
		header = append(header, elements.Row(styles.Styles{Width: -1, Justify: properties.JustifyCenter, Padding: properties.Inset{Top: 8, Bottom: 8}},
			elements.Box(styles.Styles{Width: 36, Height: 4, BorderRadius: 2, Background: "theme.muted"})))
	} else {
		header = append(header, elements.Box(styles.Styles{Height: 16}))
	}
	if s.title != "" {
		header = append(header, elements.Box(styles.Styles{Padding: properties.Inset{Left: pad.Left, Right: pad.Right, Bottom: 8}},
			elements.Text(s.title, styles.Styles{Color: st.Color, FontSize: 17, FontWeight: properties.FontWeightSemiBold})))
	}
	headerEl := elements.Stack(styles.Styles{Width: -1}, header...)
	var footerEl elements.Element
	if s.footer != nil {
		footerEl = elements.Box(styles.Styles{Width: -1, Padding: properties.Inset{Top: 8, Left: pad.Left, Right: pad.Right, Bottom: 16}}, s.footer())
	} else {
		footerEl = elements.Box(styles.Styles{Height: 16})
	}

	pgtx := gtx
	pgtx.Constraints = layout.Constraints{Min: image.Pt(width, 0), Max: image.Pt(width, maxH)}
	hdrH := elements.Measure(pgtx, headerEl).Size.Y
	ftrH := elements.Measure(pgtx, footerEl).Size.Y

	bodyMax := max(0, maxH-hdrH-ftrH)
	if fixedH > 0 {
		bodyMax = max(0, fixedH-hdrH-ftrH)
	}
	body := elements.Raw(func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.Y = bodyMax
		if fixedH > 0 {
			gtx.Constraints.Min.Y = bodyMax
		}
		return s.body.Layout(gtx)
	})
	bodyEl := elements.Box(styles.Styles{Width: -1, Padding: properties.Inset{Left: pad.Left, Right: pad.Right}}, body)

	panelSt := st
	panelSt.Padding = properties.Inset{}
	panelSt.Width = -1
	panel := elements.Stack(panelSt, headerEl, bodyEl, footerEl)

	track := elements.TrackChild()
	macro := op.Record(gtx.Ops)
	dims := kit.LayoutClickable(pgtx, &s.panel, func(gtx layout.Context) layout.Dimensions {
		d := panel.Layout(gtx)
		if handle && !s.fixed {
			// The handle and title are where a drag moves the sheet.
			defer clip.Rect{Max: image.Pt(d.Size.X, hdrH)}.Push(gtx.Ops).Pop()
			properties.CursorGrab.CursorFor().Add(gtx.Ops)
			s.drag.Add(gtx.Ops)
		}
		return d
	})
	call := macro.Stop()
	track.End()
	return call, dims.Size.Y, track
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// OutOfFlow marks the sheet as taking no space in a Box, so the box's
// Gap leaves it out; see elements.OutOfFlow.
func (s *Sheet) OutOfFlow() {}
