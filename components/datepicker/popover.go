package datepicker

import (
	"image"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// popover is a panel opened under a field: a press outside it, Escape
// or Back closes it.
type popover struct {
	open    bool
	at      image.Point
	outside int
	onClose func()
}

// layout queues the panel beside the trigger, of size triggerSize at the
// current position, while it's open.
func (p *popover) layout(gtx layout.Context, triggerSize image.Point, content func() elements.Element) {
	if !p.open {
		return
	}
	elements.HandleDismiss(p.close)
	elements.RegisterOverlay(func() elements.Element { return elements.Raw(p.layoutOutside) })
	elements.TrackPosition(&p.at)
	trigger := image.Rectangle{Min: p.at, Max: p.at.Add(triggerSize)}
	elements.RegisterPortalAround(gtx, trigger, gtx.Dp(4), elements.PlaceAuto, func() elements.Element {
		return elements.Box(styles.Styles{
			Background: "theme.background", BorderRadius: 10, BorderWidth: 1, BorderColor: "theme.border",
			Padding:   properties.UniformInset(8),
			BoxShadow: properties.BoxShadow{Y: 6, Blur: 20, Color: "#00000033"},
		}, elements.Raw(func(gtx layout.Context) layout.Dimensions {
			kit.OpaqueInput()
			return content().Layout(gtx)
		}))
	})
}

func (p *popover) close() {
	if p.open {
		p.open = false
		if p.onClose != nil {
			p.onClose()
		}
	}
}

// layoutOutside covers the window under the panel and closes it on a
// press there.
func (p *popover) layoutOutside(gtx layout.Context) layout.Dimensions {
	for {
		ev, ok := gtx.Source.Event(pointer.Filter{Target: &p.outside, Kinds: pointer.Press})
		if !ok {
			break
		}
		if _, ok := ev.(pointer.Event); ok && p.open {
			p.close()
			kit.RequestFrame(gtx)
		}
	}
	defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, &p.outside)
	return layout.Dimensions{}
}
