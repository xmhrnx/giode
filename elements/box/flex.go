package box

import (
	"gioui.org/layout"

	"github.com/xmhrnx/giode/elements/internal/base"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// layoutFlex lays children along an axis with align, justify and gap
// distribution, wrapping onto new lines when Wrap is active and the
// children overflow the main axis. st may differ from b.st when hover
// styles are active.
// layoutFlex lays children out along the flex axis. With zs set, it
// records each child's drawing into *zs instead of painting it, for
// painting in z order.
func (b *BoxElement) layoutFlex(gtx layout.Context, st styles.Styles, children []base.Element, zs *[]zPaint) layout.Dimensions {
	axis := st.FlexDirection.Axis()
	mainMax := mainConstraint(gtx.Constraints, axis)

	// Neutralize the main-axis minimum when the container hugs that
	// axis: Gio passes the parent's cross minimum through to rigid
	// children, which would otherwise inflate hug-sized containers
	// via the spacing distribution.
	// A growing box (FlexGrow) fills the space its parent hands it,
	// like CSS flex-grow, rather than shrinking back to its content.
	hugMain := func() bool {
		if st.FlexGrow > 0 {
			return false
		}
		if axis == layout.Horizontal {
			return st.Width == 0 && st.WidthPct == 0
		}
		return st.Height == 0 && st.HeightPct == 0
	}
	if hugMain() {
		if axis == layout.Horizontal {
			gtx.Constraints.Min.X = 0
		} else {
			gtx.Constraints.Min.Y = 0
		}
	}

	if st.Wrap.Resolved(axis) && !base.IsUnbounded(mainMax) {
		return b.layoutFlexWrap(gtx, st, children, axis, mainMax, zs)
	}

	// Gio passes the container's cross-axis minimum on to every child,
	// stretching it. That matches CSS's default (align-items: stretch)
	// for start alignment, but centered or end-aligned children must
	// keep their own size or there is nothing to align.
	unstretch := st.Align == properties.AlignCenter || st.Align == properties.AlignEnd || st.Align == properties.AlignBaseline
	if st.Align == properties.AlignStretch {
		gtx.Constraints = stretchCross(gtx, children, axis, mainMax)
	}
	// Children are laid out before Gio's flex places them; the tracker
	// moves what each one registered (portals, semantic nodes, tracked
	// positions) to where it lands.
	track := &flexTracker{spans: make([]flexSpan, len(children)), record: zs != nil}
	kids := make([]layout.FlexChild, 0, len(children))
	for i, child := range children {
		w := child.Layout
		if pct := base.FlexPercentOf(child, axis); pct > 0 && !base.IsUnbounded(mainMax) {
			w = (&mainSized{el: child, axis: axis, size: mainMax * pct / 100}).Layout
		}
		if unstretch {
			w = unstretchCross(w, axis)
		}
		if st.Align == properties.AlignStretch {
			w = stretchChild(child, w, axis)
		}
		w = track.wrap(i, w)
		wt := base.FlexWeightOf(child, axis)
		if wt > 0 && base.IsUnbounded(mainMax) {
			// Nothing to share out: in a Scroll, or a column with no
			// height to fill, growing would take infinite space and
			// push everything after the child out of reach. Like CSS,
			// the child keeps its natural size instead.
			base.WarnOnce("giode: FlexGrow on %s has no effect: its container has unlimited %s (inside a Scroll, or with no fixed size), so it keeps its natural size", base.Describe(child), axisName(axis))
			wt = 0
		}
		if wt > 0 && base.FlexPercentOf(child, axis) == 0 {
			kids = append(kids, layout.Flexed(float32(wt), w))
		} else {
			kids = append(kids, layout.Rigid(w))
		}
	}
	flex := layout.Flex{
		Axis:      axis,
		Spacing:   st.Justify.FlexSpacing(),
		Alignment: st.Align.FlexAlignment(),
		Gap:       int(st.Gap),
	}
	dims := flex.Layout(gtx, kids...)
	track.place(gtx.Constraints, flex)
	if zs != nil {
		for i, sp := range track.spans {
			*zs = append(*zs, zPaint{call: sp.call, off: sp.off, size: sp.dims.Size, z: base.ZIndexOf(children[i]), track: sp.track})
		}
	}
	return dims
}
