package responsive

import (
	"gioui.org/layout"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/elements/internal/base"
)

// Breakpoint is a simplified media query: it matches when the
// available width is at most MaxWidth dp. A breakpoint with neither
// MaxWidth nor Width set always matches and can serve as the fallback
// when listed last.
type Breakpoint struct {
	// MaxWidth is the widest available width, in dp, that matches.
	MaxWidth int
	// Width is the widest available width in physical pixels.
	//
	// Deprecated: pixels depend on the display density, so a layout
	// tuned on a desktop switches at a different size on a phone. Use
	// MaxWidth, which is in dp. Width applies only when MaxWidth is 0.
	Width int
	// Name identifies the breakpoint to the content of its view
	// through ActiveBreakpoint while the view is laid out. Optional.
	Name string
	View func() base.Element
}

type responsiveEl struct {
	bps []Breakpoint
}

// matches reports whether bp applies to an available width of w
// pixels.
func (bp Breakpoint) matches(gtx layout.Context, w int) bool {
	switch {
	case bp.MaxWidth > 0:
		return w <= gtx.Dp(unit.Dp(bp.MaxWidth))
	case bp.Width > 0:
		return w <= bp.Width
	}
	return true
}

// Responsive renders different views depending on the available
// width. Breakpoints are evaluated in order and the first match wins,
// so list the narrowest first and a fallback without MaxWidth last:
//
//	giode.Responsive(
//		giode.Breakpoint{MaxWidth: 600, Name: "compact", View: phoneView},
//		giode.Breakpoint{MaxWidth: 1024, Name: "medium", View: tabletView},
//		giode.Breakpoint{View: desktopView},
//	)
//
// Only the matching view is built each frame. MaxWidth is in dp and
// is compared with the width the parent container makes available,
// not the window's. Components inside the chosen view can ask which
// breakpoint matched with ActiveBreakpoint or Compact instead of
// having a flag passed down to them.
func Responsive(bps ...Breakpoint) base.Element {
	return &responsiveEl{bps: bps}
}

func (r *responsiveEl) Layout(gtx layout.Context) layout.Dimensions {
	w := gtx.Constraints.Max.X
	for _, bp := range r.bps {
		if bp.matches(gtx, w) {
			base.PushBreakpoint(bp.Name)
			dims := bp.View().Layout(gtx)
			base.PopBreakpoint()
			return dims
		}
	}
	return layout.Dimensions{}
}
