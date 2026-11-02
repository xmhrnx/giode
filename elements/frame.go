package elements

import (
	"image/color"
	"math"
	"sync"
	"sync/atomic"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/styles/properties"
)

// Threading.
//
// The view function, every Layout, scroll content closures and event
// handlers (OnClick, OnChange, ...) all run on one goroutine: the UI
// goroutine. Data changed only there needs no locks. Code running
// elsewhere - a network callback, a timer, an MCP tool - hands work to
// the UI goroutine with App.Post (or App.RunOnUI to wait for it), or
// uses State, which is safe from any goroutine.

var (
	afterMu     sync.Mutex
	afterLayout []func()
)

// AfterLayout runs fn once the current frame's layout is finished, on
// the UI goroutine, then requests another frame. A handler that must
// change the model the rest of this frame is still reading - removing
// the list item being laid out, say - defers the change with it.
// Called outside a frame, fn runs after the next one.
func AfterLayout(fn func()) {
	afterMu.Lock()
	afterLayout = append(afterLayout, fn)
	afterMu.Unlock()
}

// RunAfterLayout runs and clears the AfterLayout queue, reporting
// whether anything ran. The app calls it after laying out each frame;
// component authors don't need to.
func RunAfterLayout() bool {
	afterMu.Lock()
	fns := afterLayout
	afterLayout = nil
	afterMu.Unlock()
	for _, fn := range fns {
		fn()
	}
	if len(fns) > 0 {
		InvalidateCaches()
	}
	return len(fns) > 0
}

// displayScale is the last frame's pixels per dp, stored as float32
// bits so it can be read from any goroutine.
var displayScale atomic.Uint32

func init() {
	displayScale.Store(math.Float32bits(1))
}

// SetMetric records the frame's display metric. The app calls it every
// frame; component authors don't need to.
func SetMetric(m unit.Metric) {
	s := m.PxPerDp
	if s <= 0 {
		s = 1
	}
	displayScale.Store(math.Float32bits(s))
}

// DisplayScale returns the display's pixels per dp as of the last
// frame (1 on a standard display, 2 on a typical HiDPI one). It is
// safe to call from anywhere, including outside Layout, to convert
// sizes for code that doesn't have a layout context.
func DisplayScale() float32 {
	return math.Float32frombits(displayScale.Load())
}

// DpToPx converts dp to pixels with the last frame's display scale.
func DpToPx(v float32) int {
	return int(math.Round(float64(v * DisplayScale())))
}

// PxToDp converts pixels to dp with the last frame's display scale.
func PxToDp(px int) float32 {
	return float32(px) / DisplayScale()
}

// Inherited color.
//
// Like CSS's currentColor, a Box with Color set passes it down: Text,
// icons and glyphs inside that set no Color of their own use it, and
// the value "currentColor" refers to it explicitly. A button's icon
// therefore follows the button's label color without being told.

var colorStack []string

// PushColor makes c the inherited color until the returned function
// runs. Empty colors are ignored. Custom containers that set a text
// color for their content call it around laying the content out.
func PushColor(c string) (pop func()) {
	if c == "" || c == "currentColor" {
		return func() {}
	}
	colorStack = append(colorStack, c)
	n := len(colorStack)
	return func() { colorStack = colorStack[:n-1] }
}

// CurrentColor returns the inherited color, or "" outside any colored
// container.
func CurrentColor() string {
	if len(colorStack) == 0 {
		return ""
	}
	return colorStack[len(colorStack)-1]
}

// ResolveColor resolves an element's own color: c when set, otherwise
// the inherited color, otherwise black.
func ResolveColor(c string) color.NRGBA {
	if c == "" || c == "currentColor" {
		c = CurrentColor()
	}
	return properties.ResolveColor(c)
}

// Measure lays el out without painting, taking input, registering
// portals or Back handlers, or recording semantics, and returns its size - for
// components that must know a child's size before placing it.
func Measure(gtx layout.Context, el Element) layout.Dimensions {
	scratch := gtx
	scratch.Ops = new(op.Ops)
	scratch.Source = gtx.Source.Disabled()
	saved := semantics
	semantics = nil
	portals, backs, nprobes := len(portalQueue), len(backStack), len(probes)
	dims := el.Layout(scratch)
	portalQueue = portalQueue[:portals]
	backStack = backStack[:backs]
	probes = probes[:nprobes]
	semantics = saved
	return dims
}
