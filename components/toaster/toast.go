// Package toaster shows small notifications that pop up in a corner of
// the window and disappear on their own.

package toaster

import (
	"image"
	"time"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/components/transition"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

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

// The accent bar's width and its distance from the toast's start edge.
const (
	stripeWidth = 3
	stripeInset = 8
)

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
	var children []elements.Element
	if !t.noStripe {
		// Room for the accent bar.
		children = append(children, elements.Box(styles.Styles{Width: stripeInset + stripeWidth}))
	}
	children = append(children,
		elements.Box(styles.Styles{FlexGrow: 1, Padding: properties.SymmetricInset(12, 10)},
			elements.Text(ts.msg, styles.Styles{FontSize: st.FontSize})),
	)
	if ts.act != nil {
		children = append(children, ts.act)
	}
	children = append(children, elements.Box(styles.Styles{Padding: properties.Inset{End: 8, Start: 4}}, ts.close))
	row := elements.Row(box, children...)
	if t.noStripe {
		return ts.body.Content(func(elements.PressState) elements.Element { return row })
	}
	// The kind's accent bar: a short rounded bar inset from the start
	// edge (clear of the rounded corners), at the toast's own height,
	// painted once that height is known.
	face := elements.RawFunc(func(gtx layout.Context) layout.Dimensions {
		dims := row.Layout(gtx)
		w, x := gtx.Dp(stripeWidth), gtx.Dp(stripeInset)
		if kit.RTL(gtx) {
			x = dims.Size.X - x - w
		}
		inset := min(gtx.Dp(10), dims.Size.Y/4)
		bar := image.Rect(x, inset, x+w, dims.Size.Y-inset)
		paint.FillShape(gtx.Ops, properties.CalcColor(ts.kind.color()), clip.UniformRRect(bar, w/2).Op(gtx.Ops))
		return dims
	})
	return ts.body.Content(func(elements.PressState) elements.Element { return face })
}
