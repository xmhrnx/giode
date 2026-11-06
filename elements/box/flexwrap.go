package box

import (
	"sync"

	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/elements/internal/base"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

var (
	wrapItems = sync.Pool{New: func() any { return new([]wrapItem) }}
	wrapLines = sync.Pool{New: func() any { return new([]wrapLine) }}
)

// layoutFlexWrap measures every child once (each is laid out exactly
// once, so stateful children behave correctly), partitions them into
// lines that fit mainMax, and stacks the lines on the cross axis.
//
// Percentage-sized children resolve to an exact main-axis size before
// measuring. FlexGrow has no effect here: a wrapped line's leftover
// space cannot be redistributed after a child has already been laid
// out once, so Justify only spaces items apart (as it already does
// for rigid children without wrap), it does not grow them.
func (b *BoxElement) layoutFlexWrap(gtx layout.Context, st styles.Styles, children []base.Element, axis layout.Axis, mainMax int, zs *[]zPaint) layout.Dimensions {
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

	// The items and lines come from pools: a box's are done with when
	// it returns, and every frame lays out many boxes.
	itemsBuf := wrapItems.Get().(*[]wrapItem)
	linesBuf := wrapLines.Get().(*[]wrapLine)
	defer func() {
		clear(*itemsBuf)
		clear(*linesBuf)
		wrapItems.Put(itemsBuf)
		wrapLines.Put(linesBuf)
	}()
	items := (*itemsBuf)[:0]
	for ci, child := range children {
		el := child
		if pct := base.FlexPercentOf(child, axis); pct > 0 {
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
		track := base.TrackChild()
		macro := op.Record(gtx.Ops)
		dims := el.Layout(cgtx)
		ops := macro.Stop()
		track.End()
		it := wrapItem{ops: ops, main: axisMain(dims.Size, axis), cross: axisCross(dims.Size, axis), track: track, z: base.ZIndexOf(child)}
		items = append(items, it)
	}
	*itemsBuf = items

	// Each line is a run of the items, sliced from them.
	lines := (*linesBuf)[:0]
	var cur wrapLine
	start := 0
	for i, it := range items {
		add := it.main
		if i > start {
			add += gap
		}
		if i > start && cur.used+add > mainMax {
			cur.items = items[start:i]
			lines = append(lines, cur)
			cur = wrapLine{}
			start = i
			add = it.main
		}
		cur.used += add
		if it.cross > cur.thickness {
			cur.thickness = it.cross
		}
	}
	if start < len(items) {
		cur.items = items[start:]
		lines = append(lines, cur)
	}
	*linesBuf = lines

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
