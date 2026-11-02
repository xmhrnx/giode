package elements

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op"

	"github.com/mehran9675/giode/styles"
	"github.com/mehran9675/giode/styles/properties"
)

// layoutFlex lays children along an axis with align, justify and gap
// distribution, wrapping onto new lines when Wrap is active and the
// children overflow the main axis. st may differ from b.st when hover
// styles are active.
// layoutFlex lays children out along the flex axis. With zs set, it
// records each child's drawing into *zs instead of painting it, for
// painting in z order.
func (b *BoxElement) layoutFlex(gtx layout.Context, st styles.Styles, children []Element, zs *[]zPaint) layout.Dimensions {
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

	if st.Wrap.Resolved(axis) && !isUnbounded(mainMax) {
		return b.layoutFlexWrap(gtx, st, children, axis, mainMax, zs)
	}

	// Gio passes the container's cross-axis minimum on to every child,
	// stretching it. That matches CSS's default (align-items: stretch)
	// for start alignment, but centered or end-aligned children must
	// keep their own size or there is nothing to align.
	unstretch := st.Align == properties.AlignCenter || st.Align == properties.AlignEnd
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
		if pct := flexPercentOf(child, axis); pct > 0 && !isUnbounded(mainMax) {
			w = (&mainSized{el: child, axis: axis, size: mainMax * pct / 100}).Layout
		}
		if unstretch {
			w = unstretchCross(w, axis)
		}
		if st.Align == properties.AlignStretch {
			w = stretchChild(child, w, axis)
		}
		w = track.wrap(i, w)
		wt := flexWeightOf(child, axis)
		if wt > 0 && isUnbounded(mainMax) {
			// Nothing to share out: in a Scroll, or a column with no
			// height to fill, growing would take infinite space and
			// push everything after the child out of reach. Like CSS,
			// the child keeps its natural size instead.
			warnOnce("giode: FlexGrow on %s has no effect: its container has unlimited %s (inside a Scroll, or with no fixed size), so it keeps its natural size", describe(child), axisName(axis))
			wt = 0
		}
		if wt > 0 && flexPercentOf(child, axis) == 0 {
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
			*zs = append(*zs, zPaint{call: sp.call, off: sp.off, size: sp.dims.Size, z: zIndexOf(children[i]), track: sp.track})
		}
	}
	return dims
}

// unstretchCross lets a child size itself on the cross axis.
func unstretchCross(w layout.Widget, axis layout.Axis) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		if axis == layout.Horizontal {
			gtx.Constraints.Min.Y = 0
		} else {
			gtx.Constraints.Min.X = 0
		}
		return w(gtx)
	}
}

// flexSpan is what one flex child registered, and its measured size.
type flexSpan struct {
	track ChildTracker
	dims  layout.Dimensions
	// call and off are the recorded drawing and offset of a child
	// painted in z order.
	call op.CallOp
	off  image.Point
}

// flexTracker shifts what flex children registered by their final
// offsets. A child lays out before the flex container knows where it
// goes, so its portals, semantic nodes and tracked positions are
// recorded at the container's origin; once the container is done, each
// child's are shifted by the offset Gio's flex gave it.
type flexTracker struct {
	spans []flexSpan
	// record keeps each child's drawing out of the flex, in its span.
	record bool
}

func (t *flexTracker) wrap(i int, w layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		t.spans[i].track = TrackChild()
		var macro op.MacroOp
		if t.record {
			macro = op.Record(gtx.Ops)
		}
		dims := w(gtx)
		if t.record {
			t.spans[i].call = macro.Stop()
		}
		t.spans[i].track.End()
		t.spans[i].dims = dims
		return dims
	}
}

// place replicates the placement step of layout.Flex.Layout.
func (t *flexTracker) place(cs layout.Constraints, f layout.Flex) {
	n := len(t.spans)
	mainMin := axisMain(cs.Min, f.Axis)
	crossMin := axisCross(cs.Min, f.Axis)
	size, maxCross, maxBaseline := 0, crossMin, 0
	for _, s := range t.spans {
		size += axisMain(s.dims.Size, f.Axis)
		maxCross = max(maxCross, axisCross(s.dims.Size, f.Axis))
		maxBaseline = max(maxBaseline, s.dims.Size.Y-s.dims.Baseline)
	}
	if n > 1 && f.Gap > 0 {
		size += f.Gap * (n - 1)
	}
	space := max(0, mainMin-size)
	main := 0
	switch f.Spacing {
	case layout.SpaceSides:
		main += space / 2
	case layout.SpaceStart:
		main += space
	case layout.SpaceEvenly:
		main += space / (1 + n)
	case layout.SpaceAround:
		if n > 0 {
			main += space / (n * 2)
		}
	}
	for i, s := range t.spans {
		cross := 0
		switch f.Alignment {
		case layout.End:
			cross = maxCross - axisCross(s.dims.Size, f.Axis)
		case layout.Middle:
			cross = (maxCross - axisCross(s.dims.Size, f.Axis)) / 2
		case layout.Baseline:
			if f.Axis == layout.Horizontal {
				cross = maxBaseline - (s.dims.Size.Y - s.dims.Baseline)
			}
		}
		t.spans[i].off = axisPoint(f.Axis, main, cross)
		s.track.Place(t.spans[i].off)
		main += axisMain(s.dims.Size, f.Axis)
		if i < n-1 {
			main += f.Gap
			switch f.Spacing {
			case layout.SpaceEvenly:
				main += space / (1 + n)
			case layout.SpaceAround:
				main += space / n
			case layout.SpaceBetween:
				if n > 1 {
					main += space / (n - 1)
				}
			}
		}
	}
}

// mainSized wraps an element to force its size on the flex main axis,
// used for percentage-sized flex children.
type mainSized struct {
	el   Element
	axis layout.Axis
	size int
}

func (s *mainSized) Layout(gtx layout.Context) layout.Dimensions {
	if s.axis == layout.Horizontal {
		gtx.Constraints.Min.X = s.size
		gtx.Constraints.Max.X = s.size
	} else {
		gtx.Constraints.Min.Y = s.size
		gtx.Constraints.Max.Y = s.size
	}
	return s.el.Layout(gtx)
}

// wrapItem is one measured child: its recorded paint ops (so it is
// laid out exactly once) plus its size resolved into main/cross axis
// terms.
type wrapItem struct {
	ops         op.CallOp
	main, cross int
	// track holds what the item registered, shifted once its position
	// is known.
	track ChildTracker
	z     int
}

// wrapLine is one run of items that fit within the main axis.
type wrapLine struct {
	items     []wrapItem
	used      int // sum of item main sizes plus internal gaps
	thickness int // the line's extent on the cross axis
}

// layoutFlexWrap measures every child once (each is laid out exactly
// once, so stateful children behave correctly), partitions them into
// lines that fit mainMax, and stacks the lines on the cross axis.
//
// Percentage-sized children resolve to an exact main-axis size before
// measuring. FlexGrow has no effect here: a wrapped line's leftover
// space cannot be redistributed after a child has already been laid
// out once, so Justify only spaces items apart (as it already does
// for rigid children without wrap), it does not grow them.
func (b *BoxElement) layoutFlexWrap(gtx layout.Context, st styles.Styles, children []Element, axis layout.Axis, mainMax int, zs *[]zPaint) layout.Dimensions {
	gap := int(st.Gap)

	// A container that hugs the main axis has no leftover space to
	// distribute: mainMax is only the space it may grow into, while
	// the container (and any clip around it, e.g. a Clickable's) ends
	// up as wide as its content. Justify must not push the content
	// across that unused space.
	hugMain := false
	switch {
	case st.FlexGrow > 0:
		// Growing boxes fill what they're given; see layoutFlex.
	case axis == layout.Horizontal:
		hugMain = st.Width == 0 && st.WidthPct == 0
	default:
		hugMain = st.Height == 0 && st.HeightPct == 0
	}

	// Stretched items are measured first, so each can be laid out once
	// at its line's cross size.
	var stretchTo []int
	if st.Align == properties.AlignStretch {
		stretchTo = wrapStretch(gtx, children, axis, mainMax, gap)
	}

	items := make([]wrapItem, 0, len(children))
	for ci, child := range children {
		el := child
		if pct := flexPercentOf(child, axis); pct > 0 {
			el = &mainSized{el: child, axis: axis, size: mainMax * pct / 100}
		}
		cgtx := gtx
		if axis == layout.Horizontal {
			cgtx.Constraints.Min.X = 0
			cgtx.Constraints.Max.X = mainMax
		} else {
			cgtx.Constraints.Min.Y = 0
			cgtx.Constraints.Max.Y = mainMax
		}
		if stretchTo != nil {
			if axis == layout.Horizontal {
				cgtx.Constraints.Min.Y = stretchTo[ci]
			} else {
				cgtx.Constraints.Min.X = stretchTo[ci]
			}
			el = stretchedElement{child: el, box: child, axis: axis}
		}
		if st.Align == properties.AlignCenter || st.Align == properties.AlignEnd {
			// See layoutFlex: aligned items keep their own cross size.
			if axis == layout.Horizontal {
				cgtx.Constraints.Min.Y = 0
			} else {
				cgtx.Constraints.Min.X = 0
			}
		}
		track := TrackChild()
		macro := op.Record(gtx.Ops)
		dims := el.Layout(cgtx)
		ops := macro.Stop()
		track.End()
		it := wrapItem{ops: ops, main: axisMain(dims.Size, axis), cross: axisCross(dims.Size, axis), track: track, z: zIndexOf(child)}
		items = append(items, it)
	}

	var lines []wrapLine
	var cur wrapLine
	for _, it := range items {
		add := it.main
		if len(cur.items) > 0 {
			add += gap
		}
		if len(cur.items) > 0 && cur.used+add > mainMax {
			lines = append(lines, cur)
			cur = wrapLine{}
			add = it.main
		}
		cur.items = append(cur.items, it)
		cur.used += add
		if it.cross > cur.thickness {
			cur.thickness = it.cross
		}
	}
	if len(cur.items) > 0 {
		lines = append(lines, cur)
	}

	// Lines share any spare cross-axis space of a container with a
	// minimum cross size (a fixed Height on a Row, say), like CSS's
	// default align-content: stretch, so aligned items align within
	// the container rather than within their own line.
	if len(lines) > 0 {
		total := gap * (len(lines) - 1)
		for _, ln := range lines {
			total += ln.thickness
		}
		if extra := axisCross(gtx.Constraints.Min, axis) - total; extra > 0 {
			for i := range lines {
				lines[i].thickness += extra / len(lines)
			}
		}
	}

	crossOffset := 0
	contentMain := 0
	for li, ln := range lines {
		if ln.used > contentMain {
			contentMain = ln.used
		}
		leftover := mainMax - ln.used
		if hugMain {
			leftover = 0
		}
		lead, between := justifySpacing(st.Justify, leftover, len(ln.items))
		mainOffset := lead
		for i, it := range ln.items {
			crossPos := alignOffset(st.Align, ln.thickness, it.cross)
			pos := axisPoint(axis, mainOffset, crossOffset+crossPos)
			it.track.Place(pos)
			if zs != nil {
				*zs = append(*zs, zPaint{call: it.ops, off: pos, size: axisPoint(axis, it.main, it.cross), z: it.z, track: it.track})
			} else {
				off := op.Offset(pos).Push(gtx.Ops)
				it.ops.Add(gtx.Ops)
				off.Pop()
			}
			mainOffset += it.main
			if i < len(ln.items)-1 {
				mainOffset += gap + between
			}
		}
		crossOffset += ln.thickness
		if li < len(lines)-1 {
			crossOffset += gap
		}
	}

	// Report the widest line's actual extent, not the full mainMax: a
	// hugging container (Min < mainMax, e.g. a button sized to its
	// label) must still hug its wrapped content instead of claiming
	// all the space it was merely allowed to wrap within. A container
	// required to fill or that's exactly sized (Min == mainMax) still
	// reports mainMax, since contentMain is clamped up to it.
	mainSize := contentMain
	if minMain := minConstraint(gtx.Constraints, axis); mainSize < minMain {
		mainSize = minMain
	}
	if mainSize > mainMax {
		mainSize = mainMax
	}

	// The cross extent is clamped the same way, so a container with a
	// forced cross size (e.g. Height: -1) still fills it even when its
	// wrapped content is shorter.
	crossSize := crossOffset
	if minCross := axisCross(gtx.Constraints.Min, axis); crossSize < minCross {
		crossSize = minCross
	}
	if maxCross := axisCross(gtx.Constraints.Max, axis); crossSize > maxCross {
		crossSize = maxCross
	}

	return layout.Dimensions{Size: axisPoint(axis, mainSize, crossSize)}
}

// justifySpacing splits leftover main-axis space into a leading
// offset and extra per-gap spacing, mirroring Gio's Spacing modes.
func justifySpacing(justify properties.Justify, leftover, n int) (lead, between int) {
	if leftover <= 0 || n <= 0 {
		return 0, 0
	}
	switch justify {
	case properties.JustifyEnd:
		return leftover, 0
	case properties.JustifyCenter:
		return leftover / 2, 0
	case properties.JustifySpaceBetween:
		if n <= 1 {
			return 0, 0
		}
		return 0, leftover / (n - 1)
	case properties.JustifySpaceAround:
		g := leftover / n
		return g / 2, g
	case properties.JustifySpaceEvenly:
		g := leftover / (n + 1)
		return g, g
	default: // JustifyStart
		return 0, 0
	}
}

// alignOffset positions an item of the given cross-axis extent within
// a line of the given thickness.
func alignOffset(align properties.Align, thickness, cross int) int {
	switch align {
	case properties.AlignCenter:
		return (thickness - cross) / 2
	case properties.AlignEnd:
		return thickness - cross
	default:
		return 0
	}
}

// axisName names the length along axis, for diagnostics.
func axisName(axis layout.Axis) string {
	if axis == layout.Horizontal {
		return "width"
	}
	return "height"
}

// axisMain and axisCross read a size along and across axis.
func axisMain(p image.Point, axis layout.Axis) int {
	if axis == layout.Horizontal {
		return p.X
	}
	return p.Y
}

func axisCross(p image.Point, axis layout.Axis) int {
	if axis == layout.Horizontal {
		return p.Y
	}
	return p.X
}

// axisPoint builds a point from a main/cross pair for axis.
func axisPoint(axis layout.Axis, main, cross int) image.Point {
	if axis == layout.Horizontal {
		return image.Pt(main, cross)
	}
	return image.Pt(cross, main)
}

// stretchCross raises the cross-axis minimum to the largest child's
// cross size, measured first, for AlignStretch: Gio's flex passes that
// minimum to every child, so all of them grow to the largest.
func stretchCross(gtx layout.Context, children []Element, axis layout.Axis, mainMax int) layout.Constraints {
	cs := gtx.Constraints
	mgtx := gtx
	if axis == layout.Horizontal {
		mgtx.Constraints = layout.Constraints{Max: image.Pt(mainMax, cs.Max.Y)}
	} else {
		mgtx.Constraints = layout.Constraints{Max: image.Pt(cs.Max.X, mainMax)}
	}
	cross := 0
	for _, c := range children {
		d := Measure(mgtx, c)
		if axis == layout.Horizontal {
			cross = max(cross, d.Size.Y)
		} else {
			cross = max(cross, d.Size.X)
		}
	}
	if axis == layout.Horizontal {
		cs.Min.Y = max(cs.Min.Y, min(cross, cs.Max.Y))
	} else {
		cs.Min.X = max(cs.Min.X, min(cross, cs.Max.X))
	}
	return cs
}

// stretchChild lays a child of an AlignStretch container out at exactly
// the cross size stretchCross settled on: a Box without its own cross
// size fills it, as an element sized by its content wouldn't.
func stretchChild(child Element, w layout.Widget, axis layout.Axis) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		box, _ := child.(*BoxElement)
		if axis == layout.Horizontal {
			if gtx.Constraints.Min.Y > 0 {
				gtx.Constraints.Max.Y = gtx.Constraints.Min.Y
				if box != nil {
					box.stretchY = true
					defer func() { box.stretchY = false }()
				}
			}
		} else if gtx.Constraints.Min.X > 0 {
			gtx.Constraints.Max.X = gtx.Constraints.Min.X
			if box != nil {
				box.stretchX = true
				defer func() { box.stretchX = false }()
			}
		}
		return w(gtx)
	}
}

// stretchedElement lays an element out with stretchChild.
type stretchedElement struct {
	child Element
	box   Element
	axis  layout.Axis
}

func (s stretchedElement) Layout(gtx layout.Context) layout.Dimensions {
	return stretchChild(s.box, s.child.Layout, s.axis)(gtx)
}

// wrapStretch measures a wrapping container's children, breaks them
// into lines as the layout will, and returns each child's cross size:
// its line's thickest.
func wrapStretch(gtx layout.Context, children []Element, axis layout.Axis, mainMax, gap int) []int {
	mgtx := gtx
	if axis == layout.Horizontal {
		mgtx.Constraints = layout.Constraints{Max: image.Pt(mainMax, gtx.Constraints.Max.Y)}
	} else {
		mgtx.Constraints = layout.Constraints{Max: image.Pt(gtx.Constraints.Max.X, mainMax)}
	}
	out := make([]int, len(children))
	start, used, thick := 0, 0, 0
	end := func(i int) {
		for j := start; j < i; j++ {
			out[j] = thick
		}
	}
	for i, c := range children {
		el := c
		if pct := flexPercentOf(c, axis); pct > 0 {
			el = &mainSized{el: c, axis: axis, size: mainMax * pct / 100}
		}
		d := Measure(mgtx, el)
		m, cr := axisMain(d.Size, axis), axisCross(d.Size, axis)
		add := m
		if i > start {
			add += gap
		}
		if i > start && used+add > mainMax {
			end(i)
			start, used, thick, add = i, 0, 0, m
		}
		used += add
		thick = max(thick, cr)
	}
	end(len(children))
	return out
}
