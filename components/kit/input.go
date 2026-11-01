package kit

import (
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget"
)

// Input passes.
//
// An elements.Memo replays its subtree's recorded drawing on frames
// nothing in it changed - including frames with input aimed at widgets
// inside, like a hover or a tap. A widget can't process its events
// without being laid out, and Gio stops routing events to a handler
// that skips a frame, so each interactive widget registers an input
// pass with HandleInput while it's laid out: a function that processes
// its pending events and reports whether its appearance changed. On a
// replayed frame the memo runs the passes instead of laying the
// subtree out, and lays it out again only when one reports a change.
//
// Every built-in interactive component registers a pass. A custom
// component handling input inside a memo either registers one too, or
// calls OpaqueInput, which makes the memo lay its subtree out on every
// frame with pointer activity over it.

// InputScope collects the input passes of a subtree being laid out.
type InputScope struct {
	// Passes are the registered input passes, in layout order.
	Passes []func(gtx layout.Context) bool
	// Opaque is set when the subtree has input handling without a pass.
	Opaque bool
}

var inputStack []*InputScope

// PushInputScope collects the passes registered until pop is called
// into s. Scopes nest; a pass goes to the innermost one. Elements that
// cache output (elements.Memo) use it; components don't need to.
func PushInputScope(s *InputScope) (pop func()) {
	inputStack = append(inputStack, s)
	n := len(inputStack)
	return func() {
		inputStack = inputStack[:n-1]
	}
}

// HandleInput registers pass as the input pass of the widget being laid
// out; see the package comment. pass must process the same events the
// widget's Layout processes (asking for them with the same filters),
// and return true when the widget must be laid out again to show a
// change. It reports whether a cache will run pass: when it does, the
// widget doesn't need MarkLive just to keep receiving input.
func HandleInput(pass func(gtx layout.Context) bool) bool {
	if len(inputStack) == 0 {
		return false
	}
	s := inputStack[len(inputStack)-1]
	s.Passes = append(s.Passes, pass)
	return true
}

// OpaqueInput reports that the widget being laid out handles input
// without an input pass, so enclosing caches must lay it out whenever
// the pointer is active over them.
func OpaqueInput() {
	for _, s := range inputStack {
		s.Opaque = true
	}
}

// RunPasses runs passes, without marking enclosing caches live (the
// caller decides from the results), and reports whether any changed.
// Every pass runs, so each widget keeps receiving its events.
func RunPasses(gtx layout.Context, passes []func(gtx layout.Context) bool) bool {
	saved := liveStack
	liveStack = nil
	changed := false
	for _, p := range passes {
		if p(gtx) {
			changed = true
		}
	}
	liveStack = saved
	return changed
}

// ClickablePass returns an input pass for a widget built on a
// widget.Clickable. update must process the clickable's clicks exactly
// as the widget's Layout does (calling its handler) and report whether
// it handled one; the pass reports that, or a change in hover or press
// state.
func ClickablePass(c *widget.Clickable, update func(gtx layout.Context) bool) func(gtx layout.Context) bool {
	return func(gtx layout.Context) bool {
		h, p := c.Hovered(), c.Pressed()
		fired := update(gtx)
		return fired || h != c.Hovered() || p != c.Pressed()
	}
}

// noSelect counts the Unselectable widgets being laid out.
var noSelect int

// Unselectable wraps w so text laid out in it isn't part of an
// enclosing SelectionArea's selection: a button's label, a menu row.
// Interactive components wrap their content with it.
func Unselectable(w layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		noSelect++
		defer func() { noSelect-- }()
		return w(gtx)
	}
}

// TextSelectable reports whether text being laid out may join a
// SelectionArea's selection: it's not inside an Unselectable widget.
func TextSelectable() bool {
	return noSelect == 0
}

// LayoutClickable lays w out with click's input area under it, like
// click.Layout(gtx, w) but without clipping what w paints to its size:
// a box shadow, which falls outside the box, stays whole instead of
// showing only where it's under the box's rounded corners. w's own
// interactive children stay on top of the click area.
func LayoutClickable(gtx layout.Context, click *widget.Clickable, w layout.Widget) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := w(gtx)
	call := macro.Stop()
	cgtx := gtx
	cgtx.Constraints = layout.Exact(dims.Size)
	click.Layout(cgtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: dims.Size, Baseline: dims.Baseline}
	})
	call.Add(gtx.Ops)
	return dims
}
