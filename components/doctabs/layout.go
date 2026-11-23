package doctabs

import (
	"image"
	"image/color"
	"time"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Sizes in dp.
const (
	padStart  = 12
	padEnd    = 6
	slotSize  = 20 // the close button / dot
	slotGap   = 4
	newSize   = 32
	dragSlop  = 6
	tipDelay  = 600 * time.Millisecond
	dotRadius = 4
	iconSize  = 16 // a tab's icon
	iconGap   = 6  // between the icon and the title
)

// Layout lays out the strip, as wide as it may be.
func (d *DocTabs) Layout(gtx layout.Context) layout.Dimensions {
	dims := d.layoutSelf(gtx)
	// A right-click or long-press anywhere else on it opens its context
	// menu.
	d.AnchorContext(gtx, dims.Size)
	return dims
}

type colors struct {
	text, muted, border, active, indicator, hover color.NRGBA
	bg                                            color.NRGBA
}

func (d *DocTabs) colors() colors {
	c := func(v properties.Color, def string) color.NRGBA {
		if v == "" {
			v = properties.Color(def)
		}
		return elements.ResolveColor(v)
	}
	return colors{
		text:      c(d.st.Color, "theme.text"),
		muted:     c("", "theme.muted"),
		border:    c(d.st.BorderColor, "theme.border"),
		active:    c("", "theme.background"),
		indicator: c(d.indicator, "theme.primary"),
		hover:     c("", "theme.text/0.07"),
		bg:        c(properties.Color(d.st.Background), "transparent"),
	}
}

func (d *DocTabs) fontSize() properties.FontSize {
	if d.st.FontSize != 0 {
		return d.st.FontSize
	}
	return 13
}

func (d *DocTabs) layoutSelf(gtx layout.Context) layout.Dimensions {
	if d.pass == nil {
		d.pass = func(gtx layout.Context) bool { return d.update(gtx) }
	}
	d.update(gtx)
	scroll0 := d.scrollX
	if !kit.HandleInput(d.pass) {
		kit.OpaqueInput()
	}
	col := d.colors()
	h := gtx.Dp(34)
	if d.st.Height > 0 {
		h = gtx.Dp(unit.Dp(d.st.Height))
	}
	d.height = h
	rtl := kit.RTL(gtx)

	// Natural widths: the title, its padding and the close slot.
	fs := d.fontSize()
	minW, maxW := gtx.Dp(unit.Dp(d.minW)), gtx.Dp(unit.Dp(d.maxW))
	chrome := gtx.Dp(padStart + slotGap + slotSize + padEnd)
	nat := make([]int, len(d.tabs))
	textW := make([]int, len(d.tabs))
	for i, t := range d.tabs {
		mg := gtx
		mg.Constraints = layout.Constraints{Max: image.Pt(1<<20, h)}
		textW[i] = elements.Measure(mg, elements.Text(t.Title, styles.Styles{FontSize: fs, MaxLines: 1})).Size.X
		if t.Icon != nil {
			textW[i] += gtx.Dp(iconSize + iconGap)
		}
		nat[i] = min(max(textW[i]+chrome, minW), maxW)
	}
	newW := 0
	if d.onNew != nil {
		newW = gtx.Dp(newSize)
	}
	width := gtx.Constraints.Max.X
	unbounded := width >= 1<<20
	if unbounded {
		width = sum(nat) + newW
	}
	width = max(width, gtx.Constraints.Min.X)
	avail := max(width-newW, 0)
	widths := fit(nat, avail, minW)
	total := sum(widths)
	d.tabsW, d.total = avail, total

	// Scroll: keep the selected tab in view when asked, and within
	// range.
	maxScroll := max(total-avail, 0)
	if d.reveal {
		d.reveal = false
		x := 0
		for i, t := range d.tabs {
			if t.ID == d.selected {
				if x < d.scrollX {
					d.scrollX = x
				} else if x+widths[i] > d.scrollX+avail {
					d.scrollX = x + widths[i] - avail
				}
				break
			}
			x += widths[i]
		}
	}
	d.scrollX = min(max(d.scrollX, 0), maxScroll)

	// Slots, left to right in the reading direction; mirrored right to
	// left.
	mirror := func(r image.Rectangle) image.Rectangle {
		if !rtl {
			return r
		}
		return image.Rect(width-r.Max.X, r.Min.Y, width-r.Min.X, r.Max.Y)
	}
	slotPx, endPad := gtx.Dp(slotSize), gtx.Dp(padEnd)
	d.slots = d.slots[:0]
	x := -d.scrollX
	for i, t := range d.tabs {
		w := widths[i]
		r := image.Rect(x, 0, x+w, h)
		cx := x + w - endPad - slotPx
		cr := image.Rect(cx, (h-slotPx)/2, cx+slotPx, (h-slotPx)/2+slotPx)
		room := w - chrome
		d.slots = append(d.slots, slot{id: t.ID, r: mirror(r), close: mirror(cr), truncated: textW[i] > room})
		x += w
	}
	d.newRect = image.Rectangle{}
	if d.onNew != nil {
		nx := min(total, avail)
		d.newRect = mirror(image.Rect(nx, 0, nx+newW, h))
	}

	// The strip and the line under it.
	paint.FillShape(gtx.Ops, col.bg, clip.Rect{Max: image.Pt(width, h)}.Op())
	line := max(gtx.Dp(1), 1)
	paint.FillShape(gtx.Ops, col.border, clip.Rect{Min: image.Pt(0, h-line), Max: image.Pt(width, h)}.Op())

	// The input area, under everything the tabs register.
	area := clip.Rect{Max: image.Pt(width, h)}.Push(gtx.Ops)
	event.Op(gtx.Ops, d)
	if d.hover != "" || d.hoverNew {
		pointer.CursorPointer.Add(gtx.Ops)
	}
	area.Pop()

	// Tabs, clipped to their part of the strip; the dragged one last,
	// over the others.
	tabsClip := mirror(image.Rect(0, 0, avail, h))
	cl := clip.Rect(tabsClip).Push(gtx.Ops)
	dragged := -1
	for i := range d.slots {
		if d.press.dragging && d.slots[i].id == d.press.tab {
			dragged = i
			continue
		}
		d.drawTab(gtx, i, d.slots[i].r, col, rtl)
	}
	if dragged >= 0 {
		r := d.slots[dragged].r
		left := int(d.press.dragX - d.press.grab)
		left = min(max(left, tabsClip.Min.X), tabsClip.Max.X-r.Dx())
		d.drawTab(gtx, dragged, image.Rect(left, 0, left+r.Dx(), h), col, rtl)
	}
	cl.Pop()

	if d.onNew != nil {
		d.drawNew(gtx, col)
	}

	// The tabs' menu, anchored to each tab.
	if !d.menuOff {
		for _, s := range d.slots {
			vis := s.r.Intersect(tabsClip)
			if vis.Empty() {
				continue
			}
			off := op.Offset(vis.Min).Push(gtx.Ops)
			pop := elements.PushOffset(vis.Min)
			d.tabCtx.AnchorContext(gtx, vis.Size())
			pop()
			off.Pop()
		}
	}

	// Frameless windows drag by their title bar: tabs and the "+" take
	// their presses instead.
	for _, s := range d.slots {
		if vis := s.r.Intersect(tabsClip); !vis.Empty() {
			off := op.Offset(vis.Min).Push(gtx.Ops)
			kit.BlockDrag(gtx, vis.Size())
			off.Pop()
		}
	}
	if !d.newRect.Empty() {
		off := op.Offset(d.newRect.Min).Push(gtx.Ops)
		kit.BlockDrag(gtx, d.newRect.Size())
		off.Pop()
	}

	d.tooltip(gtx)
	if d.scrollX != scroll0 {
		// The scroll range the pointer filter asked for moved with it.
		d.update(gtx)
	}
	return layout.Dimensions{Size: image.Pt(width, h)}
}

// drawTab draws tab i in r.
func (d *DocTabs) drawTab(gtx layout.Context, i int, r image.Rectangle, col colors, rtl bool) {
	t := d.tabs[i]
	s := d.slots[i]
	sel := t.ID == d.selected
	hovered := t.ID == d.hover
	w, h := r.Dx(), r.Dy()
	off := op.Offset(r.Min).Push(gtx.Ops)
	defer off.Pop()
	pop := elements.PushOffset(r.Min)
	defer pop()
	cl := clip.Rect{Max: image.Pt(w, h)}.Push(gtx.Ops)
	defer cl.Pop()

	line := max(gtx.Dp(1), 1)
	switch {
	case sel:
		paint.FillShape(gtx.Ops, col.active, clip.Rect{Max: image.Pt(w, h)}.Op())
		paint.FillShape(gtx.Ops, col.indicator, clip.Rect{Max: image.Pt(w, gtx.Dp(2))}.Op())
	case hovered || (d.press.dragging && d.press.tab == t.ID):
		paint.FillShape(gtx.Ops, col.hover, clip.Rect{Max: image.Pt(w, h-line)}.Op())
	}
	if !sel {
		// A thin line at the tab's end.
		sx := w - line
		if rtl {
			sx = 0
		}
		inset := h / 4
		paint.FillShape(gtx.Ops, col.border, clip.Rect{Min: image.Pt(sx, inset), Max: image.Pt(sx+line, h-inset)}.Op())
	}

	// The title, cut off with "…".
	padS, padE := gtx.Dp(padStart), gtx.Dp(padEnd)
	slotPx, gap := gtx.Dp(slotSize), gtx.Dp(slotGap)
	room := max(w-padS-gap-slotPx-padE, 0)
	textCol := col.muted
	if sel {
		textCol = col.text
	}
	tx := padS
	if t.Icon != nil {
		// The icon at the start, the title after it.
		isz, igap := gtx.Dp(iconSize), gtx.Dp(iconGap)
		ix := padS
		if rtl {
			ix = w - padS - isz
		}
		igtx := gtx
		igtx.Constraints = layout.Exact(image.Pt(isz, isz))
		ioff := op.Offset(image.Pt(ix, (h-isz)/2)).Push(gtx.Ops)
		ipop := elements.PushOffset(image.Pt(ix, (h-isz)/2))
		t.Icon.Layout(igtx)
		ipop()
		ioff.Pop()
		room = max(room-isz-igap, 0)
		tx = padS + isz + igap
	}
	if rtl {
		tx = w - tx - room
	}
	tgtx := gtx
	tgtx.Constraints = layout.Constraints{Max: image.Pt(room, h)}
	rec := op.Record(gtx.Ops)
	tdims := kit.Unselectable(elements.Text(t.Title, styles.Styles{
		Color: properties.CalcColorReverse(textCol), FontSize: d.fontSize(), MaxLines: 1,
		TextOverflow: properties.TextOverflowEllipsis,
	}).Layout)(tgtx)
	call := rec.Stop()
	toff := op.Offset(image.Pt(tx, (h-tdims.Size.Y)/2)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	toff.Pop()

	// The close button, or the unsaved dot in its place; a pinned
	// tab's pin.
	cr := s.close.Sub(s.r.Min)
	showClose := d.closable && (hovered || (sel && !t.Modified))
	switch {
	case t.Pinned && !(t.Modified && !(hovered && d.hoverClose)):
		if hovered && d.hoverClose {
			rr := clip.UniformRRect(cr, gtx.Dp(4))
			paint.FillShape(gtx.Ops, col.hover, rr.Op(gtx.Ops))
		}
		gc := col.muted
		if sel || (hovered && d.hoverClose) {
			gc = col.text
		}
		drawPin(gtx, cr, gc)
	case t.Pinned:
		// Modified: the dot, the pin when the pointer is over it.
		rad := gtx.Dp(dotRadius)
		c := cr.Min.Add(image.Pt(slotPx/2, slotPx/2))
		paint.FillShape(gtx.Ops, col.text, clip.Ellipse(image.Rect(c.X-rad, c.Y-rad, c.X+rad, c.Y+rad)).Op(gtx.Ops))
	case showClose:
		if hovered && d.hoverClose {
			rr := clip.UniformRRect(cr, gtx.Dp(4))
			paint.FillShape(gtx.Ops, col.hover, rr.Op(gtx.Ops))
		}
		gsize := unit.Dp(10)
		goff := op.Offset(cr.Min.Add(image.Pt((slotPx-gtx.Dp(gsize))/2, (slotPx-gtx.Dp(gsize))/2))).Push(gtx.Ops)
		gc := col.muted
		if sel || d.hoverClose {
			gc = col.text
		}
		drawGlyph(gtx, kit.GlyphClose, gsize, gc)
		goff.Pop()
	case t.Modified:
		rad := gtx.Dp(dotRadius)
		c := cr.Min.Add(image.Pt(slotPx/2, slotPx/2))
		dot := image.Rect(c.X-rad, c.Y-rad, c.X+rad, c.Y+rad)
		dc := col.muted
		if sel {
			dc = col.text
		}
		paint.FillShape(gtx.Ops, dc, clip.Ellipse(dot).Op(gtx.Ops))
	}

	value := ""
	if sel {
		value = "selected"
	}
	if t.Modified {
		value += " modified"
	}
	if t.Pinned {
		value += " pinned"
	}
	elements.RecordSemantic(gtx, elements.SemanticNode{Kind: elements.SemanticTab, Label: t.Title, Value: value, Target: tabActor{d, t.ID}}, image.Pt(w, h))
	if d.closable || t.Pinned {
		label := "Close " + t.Title
		if t.Pinned {
			label = "Unpin " + t.Title
		}
		coff := op.Offset(cr.Min).Push(gtx.Ops)
		cpop := elements.PushOffset(cr.Min)
		elements.RecordSemantic(gtx, elements.SemanticNode{Kind: elements.SemanticButton, Label: label, Target: closeActor{d, t.ID}}, cr.Size())
		cpop()
		coff.Pop()
	}
}

// drawNew draws the "+" button.
func (d *DocTabs) drawNew(gtx layout.Context, col colors) {
	r := d.newRect
	off := op.Offset(r.Min).Push(gtx.Ops)
	defer off.Pop()
	size := gtx.Dp(24)
	b := image.Rect((r.Dx()-size)/2, (r.Dy()-size)/2, (r.Dx()+size)/2, (r.Dy()+size)/2)
	if d.hoverNew {
		paint.FillShape(gtx.Ops, col.hover, clip.UniformRRect(b, gtx.Dp(4)).Op(gtx.Ops))
	}
	gsize := unit.Dp(14)
	g := gtx.Dp(gsize)
	goff := op.Offset(image.Pt((r.Dx()-g)/2, (r.Dy()-g)/2)).Push(gtx.Ops)
	gc := col.muted
	if d.hoverNew {
		gc = col.text
	}
	drawGlyph(gtx, kit.GlyphPlus, gsize, gc)
	goff.Pop()
	pop := elements.PushOffset(r.Min)
	elements.RecordSemantic(gtx, elements.SemanticNode{Kind: elements.SemanticButton, Label: "New Tab", Target: newActor{d}}, r.Size())
	pop()
}

// tooltip shows the hovered tab's tooltip, or its whole title when cut
// off, once the pointer rested on it.
func (d *DocTabs) tooltip(gtx layout.Context) {
	if d.hover == "" || d.hoverClose || d.press.active {
		return
	}
	var s *slot
	for i := range d.slots {
		if d.slots[i].id == d.hover {
			s = &d.slots[i]
		}
	}
	t, ok := d.Tab(d.hover)
	if s == nil || !ok {
		return
	}
	text := t.Tooltip
	if text == "" && s.truncated {
		text = t.Title
	}
	if text == "" {
		return
	}
	if wait := d.hoverAt.Add(tipDelay); gtx.Now.Before(wait) {
		kit.RequestFrameAt(gtx, wait)
		return
	}
	at := elements.AbsolutePosition()
	trigger := s.r.Add(at)
	panel := elements.Box(styles.Styles{
		Background: "#0f172a", Color: "#f8fafc", Padding: properties.SymmetricInset(10, 6),
		BorderRadius: 6, MaxWidth: 480,
	}, elements.Text(text, styles.Styles{Color: "#f8fafc", FontSize: 12}))
	elements.RegisterPortalAround(gtx, trigger, gtx.Dp(4), elements.PlaceAuto, func() elements.Element {
		// The panel hugs the text.
		return elements.Raw(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min = image.Point{}
			return panel.Layout(gtx)
		})
	})
}

// hit returns the tab at p, and whether p is on its close button.
func (d *DocTabs) hit(p image.Point) (id string, onClose bool) {
	if p.Y < 0 || p.Y >= d.height {
		return "", false
	}
	for _, s := range d.slots {
		if p.In(s.r) {
			return s.id, (d.closable || d.pinned(s.id)) && p.In(s.close)
		}
	}
	return "", false
}

// update handles the pointer, and reports whether the strip must be
// laid out again.
func (d *DocTabs) update(gtx layout.Context) bool {
	changed := false
	maxScroll := max(d.total-d.tabsW, 0)
	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target:  d,
			Kinds:   pointer.Press | pointer.Release | pointer.Drag | pointer.Move | pointer.Enter | pointer.Leave | pointer.Cancel | pointer.Scroll,
			ScrollX: pointer.ScrollRange{Min: -d.scrollX, Max: maxScroll - d.scrollX},
			ScrollY: pointer.ScrollRange{Min: -d.scrollX, Max: maxScroll - d.scrollX},
		})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		changed = true
		p := e.Position.Round()
		switch e.Kind {
		case pointer.Move, pointer.Enter:
			d.setHover(gtx, p)
		case pointer.Leave:
			if !d.press.active {
				d.hover, d.hoverClose, d.hoverNew = "", false, false
			}
		case pointer.Scroll:
			s := e.Scroll.X
			if s == 0 {
				s = e.Scroll.Y
			}
			d.scrollX = min(max(d.scrollX+int(s), 0), maxScroll)
			d.setHover(gtx, p)
		case pointer.Press:
			if d.press.active {
				continue
			}
			id, onClose := d.hit(p)
			d.press = pressState{
				id: e.PointerID, active: true, button: e.Buttons, touch: e.Source == pointer.Touch,
				tab: id, close: onClose, newBtn: p.In(d.newRect), at: e.Position, scroll0: d.scrollX,
			}
			if id != "" && !d.menuOff && !d.menu.Opened() {
				d.buildMenu(id)
			}
			switch {
			case e.Buttons.Contain(pointer.ButtonSecondary):
				// The tab's menu opens from its anchor.
				d.press.active = false
			case e.Buttons.Contain(pointer.ButtonTertiary):
			case id != "" && !onClose:
				for _, s := range d.slots {
					if s.id == id {
						d.press.grab = e.Position.X - float32(s.r.Min.X)
					}
				}
				d.selectTab(id)
			}
		case pointer.Drag:
			if !d.press.active || e.PointerID != d.press.id {
				continue
			}
			dx := e.Position.X - d.press.at.X
			if d.press.touch {
				// A finger scrolls the strip.
				if abs(dx) > float32(gtx.Dp(dragSlop)) {
					d.press.moved = true
				}
				if d.press.moved {
					rtlSign := 1
					if kit.RTL(gtx) {
						rtlSign = -1
					}
					d.scrollX = min(max(d.press.scroll0-rtlSign*int(dx), 0), maxScroll)
				}
				continue
			}
			if d.press.tab == "" || d.press.close || !d.press.button.Contain(pointer.ButtonPrimary) {
				continue
			}
			if !d.press.dragging && abs(dx) > float32(gtx.Dp(dragSlop)) {
				d.press.dragging = true
			}
			if d.press.dragging {
				d.press.dragX = e.Position.X
				if d.dragTo(e.Position.X - d.press.grab) {
					d.press.moved = true
				}
			}
		case pointer.Release, pointer.Cancel:
			if !d.press.active || e.PointerID != d.press.id {
				continue
			}
			pr := d.press
			d.press = pressState{}
			if e.Kind == pointer.Cancel {
				continue
			}
			id, onClose := d.hit(p)
			switch {
			case pr.dragging:
				if pr.moved && d.onReorder != nil {
					ids := make([]string, len(d.tabs))
					for i, t := range d.tabs {
						ids[i] = t.ID
					}
					d.onReorder(ids)
				}
			case pr.moved:
			case pr.button.Contain(pointer.ButtonTertiary):
				if d.closable && id != "" && id == pr.tab && !d.pinned(id) {
					d.requestClose([]string{id})
				}
			case pr.close:
				switch {
				case !onClose || id != pr.tab:
				case d.pinned(id):
					d.pin(id, false)
				default:
					d.requestClose([]string{id})
				}
			case pr.newBtn:
				if p.In(d.newRect) && d.onNew != nil {
					d.onNew()
				}
			}
			d.setHover(gtx, p)
		}
	}
	return changed
}

// setHover tracks what the pointer is over.
func (d *DocTabs) setHover(gtx layout.Context, p image.Point) {
	id, onClose := d.hit(p)
	if id != d.hover {
		d.hoverAt = gtx.Now
	}
	d.hover, d.hoverClose, d.hoverNew = id, onClose, p.In(d.newRect)
}

// dragTo moves the dragged tab, its left edge now at left in the
// strip, past the neighbours its middle passed the middle of, and
// reports whether it moved.
func (d *DocTabs) dragTo(left float32) bool {
	i := d.Index(d.press.tab)
	if i < 0 || i >= len(d.slots) {
		return false
	}
	w := float32(d.slots[i].r.Dx())
	moved := false
	// Slots keep last frame's places: compare with the neighbours'
	// middles there.
	rtl := len(d.slots) > 1 && d.slots[0].r.Min.X > d.slots[1].r.Min.X
	for {
		i = d.Index(d.press.tab)
		prev, next := i-1, i+1
		if rtl {
			prev, next = next, prev
		}
		// It passes a neighbour when its middle passes the neighbour's.
		c := left + w/2
		switch {
		// Pinned tabs stay among the pinned, the others among the
		// others.
		case prev >= 0 && prev < len(d.slots) && c < mid(d.slots[prev].r) && d.tabs[prev].Pinned == d.tabs[i].Pinned:
			d.swap(i, prev)
		case next >= 0 && next < len(d.slots) && c > mid(d.slots[next].r) && d.tabs[next].Pinned == d.tabs[i].Pinned:
			d.swap(i, next)
		default:
			return moved
		}
		moved = true
	}
}

// swap exchanges tabs i and j and their slots' places.
func (d *DocTabs) swap(i, j int) {
	d.tabs[i], d.tabs[j] = d.tabs[j], d.tabs[i]
	si, sj := d.slots[i], d.slots[j]
	// The slots move with their tabs; the places stay, sized anew next
	// frame.
	if si.r.Min.X < sj.r.Min.X {
		sj.r = image.Rect(si.r.Min.X, si.r.Min.Y, si.r.Min.X+sj.r.Dx(), si.r.Max.Y)
		si.r = image.Rect(sj.r.Max.X, si.r.Min.Y, sj.r.Max.X+si.r.Dx(), si.r.Max.Y)
	} else {
		si.r = image.Rect(sj.r.Min.X, sj.r.Min.Y, sj.r.Min.X+si.r.Dx(), sj.r.Max.Y)
		sj.r = image.Rect(si.r.Max.X, sj.r.Min.Y, si.r.Max.X+sj.r.Dx(), sj.r.Max.Y)
	}
	d.slots[i], d.slots[j] = sj, si
}

func mid(r image.Rectangle) float32 { return float32(r.Min.X+r.Max.X) / 2 }

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func sum(v []int) int {
	t := 0
	for _, x := range v {
		t += x
	}
	return t
}

// fit shrinks the widest tabs first until they fit avail, none below
// minW: the tabs then scroll.
func fit(nat []int, avail, minW int) []int {
	out := append([]int(nil), nat...)
	if sum(nat) <= avail || len(nat) == 0 {
		return out
	}
	sorted := append([]int(nil), nat...)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j] < sorted[j-1]; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	remaining, cap := avail, 0
	for i, w := range sorted {
		share := remaining / (len(sorted) - i)
		if w > share {
			cap = share
			break
		}
		remaining -= w
	}
	cap = max(cap, minW)
	for i := range out {
		out[i] = min(out[i], cap)
	}
	return out
}

// Automation: a click on a tab selects it, on its close button closes
// it.
type tabActor struct {
	d  *DocTabs
	id string
}

func (a tabActor) Click() { a.d.selectTab(a.id) }

type closeActor struct {
	d  *DocTabs
	id string
}

func (a closeActor) Click() {
	if a.d.pinned(a.id) {
		a.d.pin(a.id, false)
		return
	}
	a.d.requestClose([]string{a.id})
}

type newActor struct{ d *DocTabs }

func (a newActor) Click() {
	if a.d.onNew != nil {
		a.d.onNew()
	}
}

// drawGlyph draws a glyph at its own size, whatever the constraints:
// kit.DrawGlyph fills a minimum it's given.
// pinned reports whether the tab with id is pinned.
func (d *DocTabs) pinned(id string) bool {
	i := d.Index(id)
	return i >= 0 && d.tabs[i].Pinned
}

// drawPin draws a push pin centered in r: a head, a collar and a point.
func drawPin(gtx layout.Context, r image.Rectangle, c color.NRGBA) {
	u := max(gtx.Dp(1), 1)
	cx, cy := (r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2
	head := image.Rect(cx-2*u, cy-5*u, cx+2*u, cy-u)
	collar := image.Rect(cx-4*u, cy-u, cx+4*u, cy+u)
	point := image.Rect(cx-u/2-u%2, cy+u, cx+u/2+1, cy+5*u)
	for _, p := range []image.Rectangle{head, collar, point} {
		paint.FillShape(gtx.Ops, c, clip.Rect(p).Op())
	}
}

func drawGlyph(gtx layout.Context, g kit.Glyph, size unit.Dp, col color.NRGBA) {
	gtx.Constraints = layout.Constraints{Max: image.Pt(1<<16, 1<<16)}
	kit.DrawGlyph(gtx, g, size, col)
}
