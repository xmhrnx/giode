package base

import (
	"github.com/xmhrnx/giode/components/kit"
)

// breakpointStack holds the names of the breakpoints whose views are
// being laid out, innermost last. Like the absolute-position tracker,
// it relies on Layout being single-threaded and synchronous.
var breakpointStack []string

// PushBreakpoint marks name as the active breakpoint until
// PopBreakpoint runs. The Responsive element calls it around laying
// out the matching view.
func PushBreakpoint(name string) {
	breakpointStack = append(breakpointStack, name)
}

// PopBreakpoint ends the innermost PushBreakpoint.
func PopBreakpoint() {
	breakpointStack = breakpointStack[:len(breakpointStack)-1]
}

// ActiveBreakpoint returns the Name of the breakpoint the innermost
// enclosing Responsive matched, or "" outside any Responsive (or when
// the matched breakpoint has no name). Only meaningful while being
// laid out as part of the current frame.
func ActiveBreakpoint() string {
	if len(breakpointStack) == 0 {
		return ""
	}
	return breakpointStack[len(breakpointStack)-1]
}

// Compact reports whether the innermost enclosing Responsive matched a
// breakpoint named "compact" - the conventional name for a phone
// layout. Outside any Responsive, it reports whether the window is
// narrower than CompactWidth.
func Compact() bool {
	if len(breakpointStack) == 0 {
		return windowSize.X > 0 && PxToDp(windowSize.X) < CompactWidth
	}
	return ActiveBreakpoint() == "compact"
}

// CompactWidth is the window width, in dp, under which Compact reports
// true outside any Responsive.
const CompactWidth = 600

func init() {
	kit.Screen = func() (int, string) { return windowSize.X, ActiveBreakpoint() }
}
