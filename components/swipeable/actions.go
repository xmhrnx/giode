// Package swipeable provides list rows that reveal actions when swiped
// sideways, like the delete and pin actions of mobile mail and chat
// apps.

package swipeable

import (
	"image"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/widget"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Action is a button revealed behind a swiped row.
type Action struct {
	// Label is the button text; Icon, when set, shows above it.
	Label string
	Icon  elements.Element
	// Background fills the button (default theme.primary) and Color
	// its label (default theme.onprimary).
	Background properties.Color
	Color      properties.Color
	// OnTap runs when the button is tapped; the row then closes.
	OnTap func()
}

// actionWidth is each action button's width, in dp.
const actionWidth = 80

const animDuration = 180 * time.Millisecond

// layoutActions paints actions side by side from x0, clipped to the
// revealed area; reversed, the first is the rightmost.
func (s *Swipeable) layoutActions(gtx layout.Context, actions []Action, clicks []widget.Clickable, revealed image.Rectangle, x0, aw int, reversed bool) {
	defer clip.Rect(revealed).Push(gtx.Ops).Pop()
	for i, a := range actions {
		bg, fg := a.Background, a.Color
		if bg == "" {
			bg = "theme.primary"
		}
		if fg == "" {
			fg = "theme.onprimary"
		}
		var kids []elements.Element
		if a.Icon != nil {
			kids = append(kids, a.Icon)
		}
		if a.Label != "" {
			kids = append(kids, elements.Text(a.Label, styles.Styles{Color: fg, FontSize: 13}))
		}
		box := elements.Stack(styles.Styles{Width: -1, Height: -1, Background: bg, Color: fg, Justify: properties.JustifyCenter, Align: properties.AlignCenter, Gap: 4}, kids...)
		func() {
			slot := i
			if reversed {
				slot = len(actions) - 1 - i
			}
			pos := image.Pt(x0+slot*aw, 0)
			defer op.Offset(pos).Push(gtx.Ops).Pop()
			pop := elements.PushOffset(pos)
			defer pop()
			agtx := gtx
			agtx.Constraints = layout.Exact(image.Pt(aw, revealed.Dy()))
			clicks[i].Layout(agtx, box.Layout)
		}()
	}
}
