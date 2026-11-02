package dropdown

import (
	"image"
	"image/color"

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

// SheetMode selects when a Select shows its options as a bottom sheet
// rather than a list below the field.
type SheetMode int

const (
	// SheetNever always opens the list below the field (the default).
	SheetNever SheetMode = iota
	// SheetCompact opens a bottom sheet on compact screens: inside a
	// Responsive breakpoint named "compact" (see elements.Compact), or
	// in a window narrower than 600dp. This is the usual phone pattern
	// for pickers.
	SheetCompact
	// SheetAlways always opens a bottom sheet.
	SheetAlways
)

// sheetState is what a Select needs while showing a sheet.
type sheetState struct {
	scrim, panel widget.Clickable
	list         *scroll.Scroll
}

// Sheet selects when the options open as a bottom sheet: a panel along
// the bottom edge of the window over a dimmed scrim, which closes when
// the scrim is tapped or Back is pressed. It returns s for chaining.
func (s *Select) Sheet(m SheetMode) *Select {
	s.sheetMode = m
	return s
}

// useSheet reports whether opening now shows a sheet. Called during
// Layout, so Compact sees the enclosing breakpoint.
func (s *Select) useSheet() bool {
	switch s.sheetMode {
	case SheetAlways:
		return true
	case SheetCompact:
		if elements.Compact() {
			return true
		}
		w := elements.WindowSize().X
		return w > 0 && elements.PxToDp(w) < elements.CompactWidth
	}
	return false
}

// layoutSheet lays out the scrim and the bottom panel over the whole window.
func (s *Select) layoutSheet(gtx layout.Context, st styles.Styles) layout.Dimensions {
	win := gtx.Constraints.Max
	if s.sheet.scrim.Clicked(gtx) {
		s.open = false
		kit.RequestFrame(gtx)
		return layout.Dimensions{}
	}
	s.sheet.scrim.Layout(gtx, kit.Unselectable(func(gtx layout.Context) layout.Dimensions {
		defer clip.Rect{Max: win}.Push(gtx.Ops).Pop()
		paint.Fill(gtx.Ops, color.NRGBA{A: 0x66})
		return layout.Dimensions{Size: win}
	}))

	options := func() elements.Element {
		rows := make([]elements.Element, 0, len(s.options))
		for i := range s.options {
			rows = append(rows, s.sheetRow(i, st))
		}
		return elements.Box(styles.Styles{FlexDirection: properties.FlexDirectionColumn, Width: -1}, rows...)
	}
	if s.sheet.list == nil {
		s.sheet.list = scroll.New(options)
	}

	// The panel spans the width and at most 60% of the height,
	// scrolling when the options don't fit.
	panelSt := styles.Styles{
		Background:  st.Background,
		BorderRadii: properties.TopCorners(12),
		Padding:     properties.Inset{Top: 8, Bottom: 8},
		Width:       -1,
	}
	maxH := win.Y * 6 / 10
	pgtx := gtx
	pgtx.Constraints = layout.Constraints{Min: image.Pt(win.X, 0), Max: image.Pt(win.X, maxH)}
	macro := op.Record(gtx.Ops)
	content := elements.Raw(func(gtx layout.Context) layout.Dimensions {
		// Hug the options when they fit; scroll when they don't.
		if elements.Measure(gtx, options()).Size.Y < gtx.Constraints.Max.Y {
			return options().Layout(gtx)
		}
		return s.sheet.list.Layout(gtx)
	})
	dims := kit.LayoutClickable(pgtx, &s.sheet.panel, elements.Box(panelSt, content).Layout)
	call := macro.Stop()

	pos := image.Pt(0, win.Y-dims.Size.Y)
	off := op.Offset(pos).Push(gtx.Ops)
	pop := elements.PushOffset(pos)
	call.Add(gtx.Ops)
	pop()
	off.Pop()
	return layout.Dimensions{Size: win}
}

// sheetRow renders one option of the sheet, tall enough to tap, with
// the selected one highlighted.
func (s *Select) sheetRow(i int, st styles.Styles) elements.Element {
	click := s.itemClick[i]
	return elements.Raw(func(gtx layout.Context) layout.Dimensions {
		if click.Clicked(gtx) {
			s.Choose(i)
			kit.RequestFrame(gtx)
		}
		dims := click.Layout(gtx, kit.Unselectable(func(gtx layout.Context) layout.Dimensions {
			rowSt := styles.Styles{Padding: properties.SymmetricInset(20, 14), Width: -1}
			switch {
			case click.Pressed() || click.Hovered():
				rowSt.Background = s.hover
				if rowSt.Background == "" {
					rowSt.Background = properties.CalcColorReverse(defaultHover)
				}
			case i == s.selected:
				rowSt.Background = kit.Alpha(st.Color, 0.12)
			}
			return elements.Box(rowSt, elements.Text(s.options[i], styles.Styles{Color: st.Color, FontSize: 16})).Layout(gtx)
		}))
		kit.BlockDrag(gtx, dims.Size)
		return dims
	})
}
