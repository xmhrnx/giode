// Package sheet provides a modal sheet: a panel that slides up from the
// bottom of the window on phones and, with Adaptive, shows as a
// centered dialog on wide screens.

package sheet

import (
	"image"
	"image/color"

	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
)

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
