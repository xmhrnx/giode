package richedit

import (
	"image"
	"image/color"
	"strconv"
	"time"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/internal/textrender"
	"github.com/xmhrnx/giode/richdoc"
	"github.com/xmhrnx/giode/styles/properties"
)

// Layout lays the editor out in the width it's given, as tall as its
// content between MinHeight and the height it's given, scrolling past.
func (e *RichEditor) Layout(gtx layout.Context) layout.Dimensions {
	dims := e.layoutSelf(gtx)
	e.AnchorContext(gtx, dims.Size)
	return dims
}

type taskBox struct {
	r     image.Rectangle
	block int
}

func (e *RichEditor) layoutSelf(gtx layout.Context) layout.Dimensions {
	e.frame++
	if !kit.HandleInput(func(gtx layout.Context) bool { return e.update(gtx) }) {
		kit.OpaqueInput()
	}
	e.update(gtx)
	if e.wantFocus {
		e.wantFocus = false
		gtx.Execute(key.FocusCmd{Tag: e})
	}
	if e.wantBlur {
		e.wantBlur = false
		if gtx.Focused(e) {
			gtx.Execute(key.FocusCmd{Tag: nil})
		}
	}
	if e.copyNext != "" {
		kit.WriteClipboard(gtx, e.copyNext)
		e.copyNext = ""
	}
	elements.TrackPosition(&e.origin)

	doc := e.st.Doc
	n := len(doc.Blocks)
	width := gtx.Constraints.Max.X
	if width >= 1<<20 {
		width = gtx.Dp(480)
	}
	e.width = width
	e.computeMarkers()
	scaleKey := "|" + strconv.Itoa(int(gtx.Metric.PxPerSp*100)) + "|" + strconv.Itoa(int(gtx.Metric.PxPerDp*100))
	if len(e.hs) != n {
		e.hs = resize(e.hs, n)
	}
	oldKeys := e.keys
	e.keys = e.prevKeys[:0]
	e.prevKeys = oldKeys
	for i := range doc.Blocks {
		k := blockKey(&doc.Blocks[i], width, e.markers[i]+scaleKey)
		e.keys = append(e.keys, k)
		if i >= len(oldKeys) || oldKeys[i] != k {
			if v := e.views[k]; v != nil {
				e.hs[i] = v.height
			} else {
				e.hs[i] = e.estimate(gtx, &doc.Blocks[i], width)
			}
		}
	}
	e.layoutTops()

	// The viewport.
	maxH := gtx.Constraints.Max.Y
	content := e.tops[n-1] + e.hs[n-1]
	minH := max(gtx.Dp(36), gtx.Dp(unit.Dp(e.minHeight)), gtx.Constraints.Min.Y)
	h := max(content, minH)
	if maxH < 1<<20 {
		h = min(h, maxH)
	}
	e.size = image.Pt(width, h)

	// Scrolling by wheel and touch.
	scrollMax := max(content-h, 0)
	if dy := e.scroll.Update(gtx.Metric, gtx.Source, gtx.Now, gesture.Vertical,
		pointer.ScrollRange{}, pointer.ScrollRange{Min: -int(e.offY), Max: scrollMax - int(e.offY)}); dy != 0 {
		e.offY += float32(dy)
	}
	if e.scroll.State() != gesture.StateIdle {
		kit.MarkLive()
	}

	// Lay out what's in view and the caret's block, measuring them.
	e.laid = resizeViews(e.laid, n)
	for pass := 0; pass < 2; pass++ {
		if e.reveal {
			e.revealCaret(gtx, h)
		}
		e.offY = min(max(e.offY, 0), float32(max(e.tops[n-1]+e.hs[n-1]-h, 0)))
		clear(e.laid)
		first := e.blockAt(int(e.offY))
		moved := false
		for i := first; i < n && e.tops[i] < int(e.offY)+h; i++ {
			moved = e.measure(gtx, i, width) || moved
		}
		// The caret's and anchor's blocks, and their neighbors, for moving
		// up and down.
		for _, b := range []int{e.st.Sel.Head.Block - 1, e.st.Sel.Head.Block, e.st.Sel.Head.Block + 1, e.st.Sel.Anchor.Block} {
			if b >= 0 && b < n && e.laid[b] == nil {
				moved = e.measure(gtx, b, width) || moved
			}
		}
		if !moved {
			break
		}
		e.layoutTops()
	}
	e.reveal = false
	content = e.tops[n-1] + e.hs[n-1]
	if maxH < 1<<20 {
		h = min(max(content, minH), maxH)
		e.size.Y = h
	}

	defer clip.Rect{Max: e.size}.Push(gtx.Ops).Pop()
	// Input areas: the editor's own under its content, then scrolling.
	event.Op(gtx.Ops, e)
	pointer.CursorText.Add(gtx.Ops)
	e.scroll.Add(gtx.Ops)
	key.InputHintOp{Tag: e, Hint: key.HintText}.Add(gtx.Ops)
	kit.BlockDrag(gtx, e.size)

	pal := e.palette()
	e.taskBoxes = e.taskBoxes[:0]
	sel := e.st.Sel
	start, end := sel.Start(), sel.End()
	off := op.Offset(image.Pt(0, -int(e.offY))).Push(gtx.Ops)
	for i, v := range e.laid {
		if v == nil || e.tops[i]+v.height < int(e.offY) || e.tops[i] > int(e.offY)+h {
			continue
		}
		e.paintBlock(gtx, i, v, pal)
		// The selection over it.
		if !sel.Empty() && i >= start.Block && i <= end.Block {
			b := &doc.Blocks[i]
			if b.Kind.HasText() {
				from, to := 0, b.Len()+1
				if i == start.Block {
					from = start.Offset
				}
				if i == end.Block {
					to = end.Offset
				}
				for _, r := range e.selectionRects(i, from, to) {
					fillRect(gtx, r, pal.sel)
				}
			} else if (i > start.Block || start.Offset == 0) && (i < end.Block || end.Offset > 0) {
				r := image.Rect(0, e.tops[i], e.width, e.tops[i]+v.height-v.g.padBottom)
				if b.Kind == richdoc.Image {
					r.Max.X = v.img.X
				}
				fillRRect(gtx, r, gtx.Dp(4), pal.sel)
			}
		}
	}
	// The caret.
	if e.focused && !e.readOnly && sel.Empty() {
		if r, ok := e.caretRect(sel.Head); ok {
			since := gtx.Now.Sub(e.blink)
			if since < 500*time.Millisecond || (since/(530*time.Millisecond))%2 == 0 || kit.ReducedMotion() {
				w := max(gtx.Dp(1.5), 1)
				r.Min.X, r.Max.X = r.Min.X+1-w/2, r.Min.X+1-w/2+w
				fillRect(gtx, r, pal.caret)
			}
			if !kit.ReducedMotion() {
				kit.RequestFrameAt(gtx, gtx.Now.Add(530*time.Millisecond-since%(530*time.Millisecond)))
			}
		}
	}
	off.Pop()

	if e.Empty() && e.placeholder != "" && e.ime.composing.Start < 0 {
		v := e.laid[0]
		g := e.geomOf(gtx, &doc.Blocks[0])
		ph := []textrender.Span{{Font: fonts.ForText(font.Font{Weight: g.weight}, e.placeholder), Size: g.size, Content: e.placeholder}}
		tgtx := gtx
		tgtx.Constraints = layout.Constraints{Max: image.Pt(max(width-g.indent, 1), 1<<20)}
		l := textrender.LayoutRich(fonts.Shaper(), tgtx, textrender.RichParams{}, ph)
		y := 0
		if v != nil {
			y = v.g.padTop
		}
		st := offset(gtx, image.Pt(g.indent, y-int(e.offY)))
		l.Draw(gtx.Ops, fonts.Shaper(), func(int) color.NRGBA { return pal.muted })
		st.Pop()
	}

	e.updateIME(gtx)
	e.layoutSuggestions(gtx)
	kit.NoteFocusable(e)
	kit.FocusRing(gtx, e, e.size, gtx.Dp(4))
	e.recordSemantics(gtx)
	e.sweep()
	if e.changed {
		e.changed = false
	}
	return layout.Dimensions{Size: e.size}
}

func resize(s []int, n int) []int {
	if cap(s) >= n {
		s = s[:n]
	} else {
		s = append(s[:cap(s)], make([]int, n-cap(s))...)
	}
	return s
}

func resizeViews(s []*blockView, n int) []*blockView {
	if cap(s) >= n {
		return s[:n]
	}
	return make([]*blockView, n)
}

// estimate guesses a block's height before it's laid out.
func (e *RichEditor) estimate(gtx layout.Context, b *richdoc.Block, width int) int {
	g := e.geomOf(gtx, b)
	lineH := gtx.Sp(g.size * 1.2)
	switch b.Kind {
	case richdoc.Table:
		return (lineH + gtx.Dp(16)) * max(len(b.Cells), 1)
	case richdoc.Image:
		return gtx.Dp(200)
	case richdoc.Rule:
		return gtx.Dp(24)
	}
	charW := max(gtx.Sp(g.size*0.5), 1)
	perLine := max((width-g.indent-g.padX)/charW, 1)
	lines := 1
	for _, ln := range splitLines(b.Text()) {
		lines += len([]rune(ln)) / perLine
	}
	return g.padTop + lines*lineH + g.padBottom
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}

// measure lays out block i and reports whether its height changed.
func (e *RichEditor) measure(gtx layout.Context, i, width int) bool {
	v := e.view(gtx, i, width)
	e.laid[i] = v
	if e.hs[i] != v.height {
		e.hs[i] = v.height
		return true
	}
	return false
}

func (e *RichEditor) layoutTops() {
	n := len(e.hs)
	e.tops = resize(e.tops, n)
	y := 0
	for i, h := range e.hs {
		e.tops[i] = y
		y += h
	}
}

// revealCaret scrolls the caret into view.
func (e *RichEditor) revealCaret(gtx layout.Context, h int) {
	p := e.st.Sel.Head
	if p.Block >= len(e.tops) {
		return
	}
	top := e.tops[p.Block]
	bottom := top + e.hs[p.Block]
	if v := e.laid[p.Block]; v != nil {
		if r, ok := e.caretRect(p); ok {
			top, bottom = r.Min.Y, r.Max.Y
		}
	}
	margin := gtx.Dp(8)
	switch {
	case float32(top-margin) < e.offY:
		e.offY = float32(top - margin)
	case float32(bottom+margin) > e.offY+float32(h):
		e.offY = float32(bottom + margin - h)
	}
}

// sweep drops cached views not used for a while.
func (e *RichEditor) sweep() {
	if e.frame%60 != 0 {
		return
	}
	for k, v := range e.views {
		if e.frame-v.used > 120 {
			delete(e.views, k)
		}
	}
}

// paintBlock draws block i at its place.
func (e *RichEditor) paintBlock(gtx layout.Context, i int, v *blockView, pal palette) {
	b := &e.st.Doc.Blocks[i]
	top := e.tops[i]
	st := offset(gtx, image.Pt(0, top))
	defer st.Pop()
	g := v.g
	switch b.Kind {
	case richdoc.Code:
		fillRRect(gtx, image.Rect(0, 0, e.width, v.height-gtx.Sp(properties.ResolveFontSize(e.style.FontSize)*0.5)), gtx.Dp(6), pal.codeBg)
	case richdoc.Quote:
		ind := gtx.Sp(properties.ResolveFontSize(e.style.FontSize) * 1.6)
		x := ind * b.Level
		fillRect(gtx, image.Rect(x, 0, x+gtx.Dp(3), v.height-g.padBottom), pal.border)
	case richdoc.Rule:
		y := v.height / 2
		fillRect(gtx, image.Rect(0, y, e.width, y+max(gtx.Dp(1), 1)), pal.border)
		return
	case richdoc.Image:
		e.paintImage(gtx, b, v, pal)
		return
	case richdoc.Table:
		e.paintTable(gtx, b, v, pal)
		return
	}
	// The list marker or task box, beside the first line.
	if b.Kind.IsList() && len(v.lines) > 0 {
		ln := v.lines[0]
		lineH := ln.Bottom - ln.Top
		mid := g.padTop + ln.Top + lineH/2
		if b.Kind == richdoc.Task {
			s := gtx.Sp(g.size * 0.9)
			x := g.indent - s - gtx.Sp(g.size*0.4)
			r := image.Rect(x, mid-s/2, x+s, mid+s/2)
			e.taskBoxes = append(e.taskBoxes, taskBox{r: r.Add(image.Pt(0, top)), block: i})
			if b.Checked {
				fillRRect(gtx, r, gtx.Dp(3), pal.check)
				tick := textrender.LayoutRich(fonts.Shaper(), unbounded(gtx), textrender.RichParams{},
					[]textrender.Span{{Font: font.Font{Weight: font.Bold}, Size: g.size * 0.75, Content: "✓"}})
				o := offset(gtx, image.Pt(r.Min.X+(s-tick.Dims.Size.X)/2, r.Min.Y+(s-tick.Dims.Size.Y)/2))
				tick.Draw(gtx.Ops, fonts.Shaper(), func(int) color.NRGBA { return pal.checkText })
				o.Pop()
			} else {
				stroke := max(gtx.Dp(1.5), 1)
				paint.FillShape(gtx.Ops, pal.muted, clip.Stroke{Path: clip.UniformRRect(r, gtx.Dp(3)).Path(gtx.Ops), Width: float32(stroke)}.Op())
			}
		} else if v.marker != "" {
			m := textrender.LayoutRich(fonts.Shaper(), unbounded(gtx), textrender.RichParams{},
				[]textrender.Span{{Font: font.Font{}, Size: g.size, Content: v.marker}})
			x := g.indent - m.Dims.Size.X - gtx.Sp(g.size*0.4)
			y := g.padTop + ln.Top + (lineH-m.Dims.Size.Y)/2
			o := offset(gtx, image.Pt(x, y))
			m.Draw(gtx.Ops, fonts.Shaper(), func(int) color.NRGBA { return pal.muted })
			o.Pop()
		}
	}
	t := offset(gtx, image.Pt(g.indent, g.padTop))
	e.paintText(gtx, v.laid, v.spans, v.src, v.chips, g, pal, b.Kind == richdoc.Task && b.Checked)
	t.Pop()
}

// paintText draws laid out spans with their marks; src maps a laid out
// span to its model span.
func (e *RichEditor) paintText(gtx layout.Context, laid *textrender.RichLaid, spans []richdoc.Span, src []int, chips map[int]*textrender.RichLaid, g geom, pal palette, done bool) {
	model := func(i int) (richdoc.Span, int, bool) {
		if i >= len(src) || src[i] >= len(spans) {
			return richdoc.Span{}, 0, false
		}
		return spans[src[i]], src[i], true
	}
	colorOf := func(i int) color.NRGBA {
		sp, _, ok := model(i)
		if !ok {
			return pal.text
		}
		m := sp.Marks
		switch {
		case m.Color != "":
			return properties.CalcColor(m.Color)
		case m.Link != "":
			return pal.link
		case g.muted || done:
			return pal.muted
		}
		return pal.text
	}
	thin := max(gtx.Dp(1), 1)
	// Backgrounds: highlights, code and atoms.
	for _, seg := range laid.Segments {
		sp, _, ok := model(seg.Span)
		if !ok {
			continue
		}
		m := sp.Marks
		r := image.Rectangle{Min: seg.Offset, Max: seg.Offset.Add(seg.Size)}
		switch {
		case sp.Atom != nil && sp.Atom.Kind != "emoji":
			fillRRect(gtx, r, gtx.Dp(4), pal.chip)
		case m.Highlight != "":
			fillRRect(gtx, r, gtx.Dp(2), properties.CalcColor(m.Highlight))
		case m.Code && !g.mono:
			fillRRect(gtx, r.Inset(-gtx.Dp(1)), gtx.Dp(3), pal.codeBg)
		}
	}
	laid.Draw(gtx.Ops, fonts.Shaper(), colorOf)
	for _, seg := range laid.Segments {
		sp, mi, ok := model(seg.Span)
		if !ok {
			continue
		}
		if sp.Atom != nil {
			if l := chips[mi]; l != nil {
				pad := (seg.Size.X - l.Dims.Size.X) / 2
				o := offset(gtx, seg.Offset.Add(image.Pt(pad, 0)))
				c := pal.chipText
				if sp.Atom.Kind == "emoji" {
					c = pal.text
				}
				l.Draw(gtx.Ops, fonts.Shaper(), func(int) color.NRGBA { return c })
				o.Pop()
			}
			continue
		}
		if seg.Box {
			continue // a tab
		}
		c := colorOf(seg.Span)
		x0, x1 := seg.Offset.X, seg.Offset.X+seg.Size.X
		if sp.Marks.Underline || sp.Marks.Link != "" {
			y := seg.Offset.Y + seg.Ascent + max(gtx.Dp(1.5), 1)
			fillRect(gtx, image.Rect(x0, y, x1, y+thin), c)
		}
		if sp.Marks.Strike || done {
			y := seg.Offset.Y + seg.Ascent - seg.Ascent*3/10
			fillRect(gtx, image.Rect(x0, y, x1, y+thin), c)
		}
	}
}

func (e *RichEditor) paintImage(gtx layout.Context, b *richdoc.Block, v *blockView, pal palette) {
	r := image.Rectangle{Max: v.img}
	img := e.image(b.Src)
	if img == nil {
		fillRRect(gtx, r, gtx.Dp(6), pal.codeBg)
		label := b.Alt
		if label == "" {
			label = "🖼"
		}
		l := textrender.LayoutRich(fonts.Shaper(), unbounded(gtx), textrender.RichParams{},
			[]textrender.Span{{Font: fonts.ForText(font.Font{}, label), Size: v.g.size, Content: label}})
		o := offset(gtx, image.Pt((r.Dx()-l.Dims.Size.X)/2, (r.Dy()-l.Dims.Size.Y)/2))
		l.Draw(gtx.Ops, fonts.Shaper(), func(int) color.NRGBA { return pal.muted })
		o.Pop()
		return
	}
	io, ok := e.imageOps[b.Src]
	if !ok {
		io = paint.NewImageOp(img)
		e.imageOps[b.Src] = io
	}
	s := img.Bounds().Size()
	defer clip.UniformRRect(r, gtx.Dp(6)).Push(gtx.Ops).Pop()
	sc := op.Affine(f32.Affine2D{}.Scale(f32.Point{}, f32.Pt(float32(r.Dx())/float32(s.X), float32(r.Dy())/float32(s.Y)))).Push(gtx.Ops)
	io.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	sc.Pop()
}

func (e *RichEditor) paintTable(gtx layout.Context, b *richdoc.Block, v *blockView, pal palette) {
	thin := max(gtx.Dp(1), 1)
	pad := gtx.Dp(8)
	for r, row := range v.cells {
		for c, cell := range row {
			if r == 0 {
				fillRect(gtx, cell.rect, pal.codeBg)
			}
			// Borders: each cell's right and bottom, the table's left and top.
			fillRect(gtx, image.Rect(cell.rect.Max.X-thin, cell.rect.Min.Y, cell.rect.Max.X, cell.rect.Max.Y), pal.border)
			fillRect(gtx, image.Rect(cell.rect.Min.X, cell.rect.Max.Y-thin, cell.rect.Max.X, cell.rect.Max.Y), pal.border)
			if c == 0 {
				fillRect(gtx, image.Rect(cell.rect.Min.X, cell.rect.Min.Y, cell.rect.Min.X+thin, cell.rect.Max.Y), pal.border)
			}
			if r == 0 {
				fillRect(gtx, image.Rect(cell.rect.Min.X, cell.rect.Min.Y, cell.rect.Max.X, cell.rect.Min.Y+thin), pal.border)
			}
			var spans []richdoc.Span
			if c < len(b.Cells[r]) {
				spans = b.Cells[r][c]
			}
			o := offset(gtx, cell.rect.Min.Add(image.Pt(pad, pad)))
			g := v.g
			if r == 0 {
				g.weight = font.Bold
			}
			e.paintText(gtx, cell.laid, spans, cell.src, cell.chips, g, pal, false)
			o.Pop()
		}
	}
}

// updateIME tells the input method the caret block's text, the
// selection and where the caret is.
func (e *RichEditor) updateIME(gtx layout.Context) {
	if !gtx.Focused(e) || e.readOnly {
		return
	}
	p := e.st.Sel.Head
	b := &e.st.Doc.Blocks[p.Block]
	if !b.Kind.HasText() {
		return
	}
	text := blockText(b)
	snip := key.Snippet{Range: key.Range{Start: 0, End: len([]rune(text))}, Text: text}
	if e.ime.block != p.Block || snip != e.ime.snippet {
		e.ime.block = p.Block
		e.ime.snippet = snip
		gtx.Execute(key.SnippetCmd{Tag: e, Snippet: snip})
	}
	rng := key.Range{Start: p.Offset, End: p.Offset}
	if a := e.st.Sel.Anchor; a.Block == p.Block {
		rng.Start = a.Offset
	}
	var caret key.Caret
	if r, ok := e.caretRect(p); ok {
		r = r.Add(image.Pt(0, -int(e.offY)))
		base := r.Min.Y + (r.Dy())*4/5
		caret = key.Caret{Pos: f32.Pt(float32(r.Min.X), float32(base)), Ascent: float32(base - r.Min.Y), Descent: float32(r.Max.Y - base)}
	}
	if rng != e.ime.sel || caret != e.ime.caret {
		e.ime.sel, e.ime.caret = rng, caret
		gtx.Execute(key.SelectionCmd{Tag: e, Range: rng, Caret: caret})
	}
}

// blockText is a block's text as the input method sees it: atoms as
// U+FFFC.
func blockText(b *richdoc.Block) string {
	var out []rune
	for _, s := range b.Spans {
		if s.Atom != nil {
			out = append(out, '￼')
		} else {
			out = append(out, []rune(s.Text)...)
		}
	}
	return string(out)
}

// recordSemantics describes the editor to screen readers: a text field
// with the document's text, caret and selection.
func (e *RichEditor) recordSemantics(gtx layout.Context) {
	text := e.st.Doc.Text()
	global := func(p richdoc.Pos) int {
		n := 0
		for i := 0; i < p.Block && i < len(e.st.Doc.Blocks); i++ {
			n += len([]rune(blockPlain(&e.st.Doc.Blocks[i]))) + 1
		}
		return n + p.Offset
	}
	label := e.placeholder
	sn := elements.SemanticNode{Kind: elements.SemanticInput, Label: label, Value: text, Target: e,
		Focused: e.focused, HasCaret: true, Caret: global(e.st.Sel.Head), Anchor: global(e.st.Sel.Anchor)}
	elements.RecordSemantic(gtx, sn, e.size)
}

func blockPlain(b *richdoc.Block) string {
	d := richdoc.Doc{Blocks: []richdoc.Block{*b}}
	return d.Text()
}
