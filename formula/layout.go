package formula

import (
	"image/color"

	"gioui.org/f32"
	giofont "gioui.org/font"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"golang.org/x/image/math/fixed"

	"github.com/xmhrnx/giode/fonts"
)

// box is a laid out piece: its width, height above the baseline and
// depth below, in pixels, and how to draw it with its baseline's left
// end at (x, y).
type box struct {
	w, asc, desc float32
	// italic is how far the ink leans past the width (italic letters),
	// for placing a superscript.
	italic float32
	draw   func(d *drawer, x, y float32)
}

// drawer paints boxes.
type drawer struct {
	ops    *op.Ops
	col    color.NRGBA
	shaper *text.Shaper
}

// mstyle is TeX's math style: display, text, script or scriptscript
// (0 to 3), with the size of an em in pixels at the formula's base size.
type mstyle struct {
	level int
	base  float32
}

// em is the font size at the style's level.
func (s mstyle) em() float32 {
	switch s.level {
	case 2:
		return s.base * 0.7
	case 3:
		return s.base * 0.5
	}
	return s.base
}

func (s mstyle) script() mstyle {
	n := s
	if n.level < 2 {
		n.level = 2
	} else {
		n.level = 3
	}
	return n
}

// fracParts is the style of a fraction's numerator and denominator.
func (s mstyle) fracParts() mstyle {
	n := s
	switch s.level {
	case 0:
		n.level = 1
	case 1:
		n.level = 2
	default:
		n.level = 3
	}
	return n
}

func (s mstyle) display() bool { return s.level == 0 }

// Font metrics of STIX Two Math, in em.
const (
	axisHeight = 0.258
	ruleThick  = 0.066
	xHeight    = 0.47
)

// layouter lays a formula out.
type layouter struct {
	shaper *text.Shaper
	math   giofont.Font
}

func newLayouter() *layouter {
	return &layouter{
		shaper: fonts.Shaper(),
		math:   fonts.Resolve(giofont.Font{Typeface: Family}),
	}
}

// glyphs shapes s at size px in face f, as a box drawn from its glyphs.
func (l *layouter) glyphs(s string, f giofont.Font, px float32) box {
	if s == "" || px <= 0 {
		return box{}
	}
	l.shaper.LayoutString(text.Parameters{
		Font:     f,
		PxPerEm:  fixed.Int26_6(px * 64),
		MaxWidth: 1 << 24,
		MaxLines: 1,
	}, s)
	var gs []text.Glyph
	for {
		g, ok := l.shaper.NextGlyph()
		if !ok {
			break
		}
		gs = append(gs, g)
	}
	if len(gs) == 0 {
		return box{}
	}
	first := gs[0]
	var b box
	asc, desc := float32(0), float32(0)
	right := float32(0)
	for _, g := range gs {
		x := fix(g.X - first.X)
		if t := -fix(g.Bounds.Min.Y); t > asc {
			asc = t
		}
		if t := fix(g.Bounds.Max.Y); t > desc {
			desc = t
		}
		if r := x + fix(g.Bounds.Max.X); r > right {
			right = r
		}
		b.w = x + fix(g.Advance)
	}
	b.asc, b.desc = asc, desc
	b.italic = max(0, right-b.w)
	// The glyphs' paths are relative to the first one's baseline point.
	b.draw = func(d *drawer, x, y float32) {
		path := d.shaper.Shape(gs)
		tr := op.Affine(f32.AffineId().Offset(f32.Pt(x, y))).Push(d.ops)
		paint.FillShape(d.ops, d.col, clip.Outline{Path: path}.Op())
		tr.Pop()
	}
	return b
}

func fix(v fixed.Int26_6) float32 { return float32(v) / 64 }

// rule is a filled rectangle, w by h, its top t above the baseline.
func rule(w, top, h float32) box {
	return box{w: w, asc: max(top, 0), desc: max(h-top, 0), draw: func(d *drawer, x, y float32) {
		fillRect(d, x, y-top, w, h)
	}}
}

func fillRect(d *drawer, x, y, w, h float32) {
	var p clip.Path
	p.Begin(d.ops)
	p.MoveTo(f32.Pt(x, y))
	p.LineTo(f32.Pt(x+w, y))
	p.LineTo(f32.Pt(x+w, y+h))
	p.LineTo(f32.Pt(x, y+h))
	p.Close()
	paint.FillShape(d.ops, d.col, clip.Outline{Path: p.End()}.Op())
}

// fillDot fills a circle of radius r at (cx, cy).
func fillDot(d *drawer, cx, cy, r float32) {
	const k = 0.5523 // a quarter circle as a cubic
	var p clip.Path
	p.Begin(d.ops)
	p.MoveTo(f32.Pt(cx+r, cy))
	p.CubeTo(f32.Pt(cx+r, cy+k*r), f32.Pt(cx+k*r, cy+r), f32.Pt(cx, cy+r))
	p.CubeTo(f32.Pt(cx-k*r, cy+r), f32.Pt(cx-r, cy+k*r), f32.Pt(cx-r, cy))
	p.CubeTo(f32.Pt(cx-r, cy-k*r), f32.Pt(cx-k*r, cy-r), f32.Pt(cx, cy-r))
	p.CubeTo(f32.Pt(cx+k*r, cy-r), f32.Pt(cx+r, cy-k*r), f32.Pt(cx+r, cy))
	p.Close()
	paint.FillShape(d.ops, d.col, clip.Outline{Path: p.End()}.Op())
}

// hspace is empty width.
func hspace(w float32) box { return box{w: w} }

// hbox sets boxes side by side.
func hbox(bs ...box) box {
	var out box
	xs := make([]float32, len(bs))
	for i, b := range bs {
		xs[i] = out.w
		out.w += b.w
		out.asc = max(out.asc, b.asc)
		out.desc = max(out.desc, b.desc)
	}
	if n := len(bs); n > 0 {
		out.italic = bs[n-1].italic
	}
	out.draw = func(d *drawer, x, y float32) {
		for i, b := range bs {
			if b.draw != nil {
				b.draw(d, x+xs[i], y)
			}
		}
	}
	return out
}

// shift moves b's baseline up by dy.
func shift(b box, dy float32) box {
	inner := b
	return box{w: b.w, asc: b.asc + dy, desc: b.desc - dy, italic: b.italic, draw: func(d *drawer, x, y float32) {
		if inner.draw != nil {
			inner.draw(d, x, y-dy)
		}
	}}
}

// layoutItem is a laid out atom with its spacing class.
type layoutItem struct {
	b   box
	cls class
}

// list lays a list of nodes out side by side, spaced as TeX spaces math.
// cellStart marks an alignment cell after the first, which spaces a
// leading relation as if something came before it.
func (l *layouter) list(ns []node, st mstyle, cellStart bool) box {
	var items []layoutItem
	for _, n := range ns {
		b, cls, ok := l.node(n, st)
		if !ok {
			continue
		}
		items = append(items, layoutItem{b, cls})
	}
	// A binary operator with nothing to operate on is ordinary: a unary
	// minus.
	for i := range items {
		if items[i].cls != clsBin {
			continue
		}
		prevBin := i == 0 && !cellStart
		if i > 0 {
			switch items[i-1].cls {
			case clsBin, clsOp, clsRel, clsOpen, clsPunct:
				prevBin = true
			}
		}
		next := i == len(items)-1
		if !next {
			switch items[i+1].cls {
			case clsRel, clsClose, clsPunct:
				next = true
			}
		}
		if prevBin || next {
			items[i].cls = clsOrd
		}
	}
	mu := st.em() / 18
	var bs []box
	for i, it := range items {
		if i > 0 {
			if sp := spacing(items[i-1].cls, it.cls, st.level >= 2); sp > 0 {
				bs = append(bs, hspace(sp*mu))
			}
		} else if cellStart {
			if sp := spacing(clsOrd, it.cls, st.level >= 2); sp > 0 {
				bs = append(bs, hspace(sp*mu))
			}
		}
		bs = append(bs, it.b)
	}
	return hbox(bs...)
}

// spacing is the space between atoms of classes a and b, in mu (an
// eighteenth of an em), as TeX's table has it; script styles keep only
// the thin spaces.
func spacing(a, b class, script bool) float32 {
	const thin, med, thick = 3, 4, 5
	var sp float32
	switch a {
	case clsOrd, clsClose:
		switch b {
		case clsOp:
			sp = thin
		case clsBin:
			sp = med
		case clsRel:
			sp = thick
		}
	case clsOp:
		switch b {
		case clsOrd, clsOp:
			sp = thin
		case clsRel:
			sp = thick
		}
	case clsBin:
		switch b {
		case clsOrd, clsOp, clsOpen:
			sp = med
		}
	case clsRel:
		switch b {
		case clsOrd, clsOp, clsOpen:
			sp = thick
		}
	case clsPunct:
		switch b {
		case clsOrd, clsOp, clsRel, clsOpen, clsPunct, clsClose:
			sp = thin
		}
	}
	if script && sp != thin {
		return 0
	}
	return sp
}

// node lays one node out.
func (l *layouter) node(n node, st mstyle) (box, class, bool) {
	em := st.em()
	switch n := n.(type) {
	case atom:
		return l.glyphs(n.s, l.math, em), n.cls, true
	case textRun:
		// Upright, in the formula's font, as LaTeX sets \text in its own.
		return l.glyphs(n.s, l.math, em), clsOrd, true
	case group:
		return l.list(n.body, st, false), clsOrd, true
	case space:
		return hspace(n.em * em), clsOrd, true
	case frac:
		return l.frac(n, st), clsOrd, true
	case radical:
		return l.radical(n, st), clsOrd, true
	case scripts:
		return l.scripts(n, st)
	case bigOp:
		return l.bigOp(n, st, nil, nil), clsOp, true
	case function:
		return l.glyphs(n.name, l.math, em), clsOp, true
	case accent:
		return l.accent(n, st), clsOrd, true
	case delimits:
		return l.delimits(n, st), clsOrd, true
	case sizedDelim:
		if n.s == "." {
			return hspace(em * 0.12), n.cls, true
		}
		return l.delim(n.s, n.scale*em, st), n.cls, true
	}
	return box{}, clsOrd, false
}

// frac sets a fraction: numerator over a rule over the denominator,
// centered on the math axis.
func (l *layouter) frac(n frac, st mstyle) box {
	inner := st
	switch n.style {
	case 1:
		inner.level = 0
	case 2:
		inner.level = 1
	}
	parts := inner.fracParts()
	em := inner.em()
	num := l.list(n.num, parts, false)
	den := l.list(n.den, parts, false)
	t := max(ruleThick*em, 1)
	if n.style == -1 {
		t = 0 // \binom: no rule
	}
	axis := axisHeight * em
	gap, u, v := t, 0.394*em, 0.345*em
	if inner.display() {
		gap, u, v = 3*t, 0.677*em, 0.686*em
	}
	if n.style == -1 {
		gap = max(gap, 0.15*em)
	}
	// Clear the rule by gap above and below.
	u = max(u, axis+t/2+gap+num.desc)
	v = max(v, den.asc+gap+t/2-axis)
	pad := 0.12 * em
	w := max(num.w, den.w) + 2*pad
	return box{w: w, asc: u + num.asc, desc: v + den.desc, draw: func(d *drawer, x, y float32) {
		num.draw(d, x+(w-num.w)/2, y-u)
		den.draw(d, x+(w-den.w)/2, y+v)
		if t > 0 {
			fillRect(d, x+pad*0.4, y-axis-t/2, w-pad*0.8, t)
		}
	}}
}

// radical sets a root: the radical sign drawn to the body's height, a
// rule over the body, and the index, if any, in the crook.
func (l *layouter) radical(n radical, st mstyle) box {
	em := st.em()
	body := l.list(n.body, st, false)
	t := max(ruleThick*em, 1)
	clear := t + 0.12*em
	if st.display() {
		clear = t + 0.2*em
	}
	body.asc = max(body.asc, xHeight*em)
	top := -(body.asc + clear + t)        // the rule's top, from the baseline
	bottom := max(body.desc, 0) + 0.06*em // the sign's foot
	h := bottom - top
	sw := 0.45*em + 0.06*h // the sign's width
	var index box
	ix := float32(0)
	if len(n.index) > 0 {
		index = l.list(n.index, mstyle{level: 3, base: st.base}, false)
		// The index's right end over the sign's crook, raised to 60%.
		ix = max(0, index.w-0.55*sw)
	}
	gap := 0.08 * em
	w := ix + sw + body.w + gap
	asc := -top
	if len(n.index) > 0 {
		asc = max(asc, -(bottom-0.6*h)+index.asc)
	}
	return box{w: w, asc: asc, desc: bottom, draw: func(d *drawer, x, y float32) {
		x0 := x + ix
		// The sign: a short tick up, a thick stroke down, the long one
		// up to the rule.
		var p clip.Path
		p.Begin(d.ops)
		p.MoveTo(f32.Pt(x0, y+top+0.62*h))
		p.LineTo(f32.Pt(x0+0.2*sw, y+top+0.54*h))
		p.LineTo(f32.Pt(x0+0.5*sw, y+bottom))
		p.LineTo(f32.Pt(x0+sw, y+top+t/2))
		p.LineTo(f32.Pt(x0+sw+body.w+gap, y+top+t/2))
		paint.FillShape(d.ops, d.col, clip.Stroke{Path: p.End(), Width: t}.Op())
		// The down stroke, heavier.
		var q clip.Path
		q.Begin(d.ops)
		q.MoveTo(f32.Pt(x0+0.2*sw, y+top+0.54*h))
		q.LineTo(f32.Pt(x0+0.5*sw, y+bottom))
		paint.FillShape(d.ops, d.col, clip.Stroke{Path: q.End(), Width: t * 1.7}.Op())
		body.draw(d, x0+sw+gap/2, y)
		if len(n.index) > 0 {
			index.draw(d, x0+0.55*sw-index.w, y+bottom-0.6*h)
		}
	}}
}

// scripts sets a base with a superscript, a subscript or both.
func (l *layouter) scripts(n scripts, st mstyle) (box, class, bool) {
	if op, ok := n.base.(bigOp); ok {
		return l.bigOp(op, st, n.sup, n.sub), clsOp, true
	}
	if fn, ok := n.base.(function); ok && fn.limits && st.display() {
		base := l.glyphs(fn.name, l.math, st.em())
		return l.limits(base, n.sup, n.sub, st), clsOp, true
	}
	base, cls, _ := l.node(n.base, st)
	em := st.em()
	sst := st.script()
	var sup, sub box
	if n.sup != nil {
		sup = l.list(n.sup, sst, false)
	}
	if n.sub != nil {
		sub = l.list(n.sub, sst, false)
	}
	// TeX's rules, simplified: a superscript clears the base's top, a
	// subscript its bottom, and the two keep apart.
	supEm := sst.em()
	u := max(base.asc-0.39*supEm, 0.413*em)
	if !st.display() {
		u = max(base.asc-0.39*supEm, 0.363*em)
	}
	u = max(u, sup.desc+0.25*xHeight*em)
	v := max(base.desc+0.15*em, 0.15*em)
	if n.sup != nil && n.sub != nil {
		v = max(v, 0.247*em)
		if gap := (u - sup.desc) - (sub.asc - v); gap < 4*ruleThick*em {
			v += 4*ruleThick*em - gap
		}
	}
	v = max(v, sub.asc-0.8*xHeight*em)
	supX := base.w + base.italic
	subX := base.w
	w := base.w
	if n.sup != nil {
		w = max(w, supX+sup.w)
	}
	if n.sub != nil {
		w = max(w, subX+sub.w)
	}
	w += 0.05 * em
	asc, desc := base.asc, base.desc
	if n.sup != nil {
		asc = max(asc, u+sup.asc)
	}
	if n.sub != nil {
		desc = max(desc, v+sub.desc)
	}
	return box{w: w, asc: asc, desc: desc, draw: func(d *drawer, x, y float32) {
		base.draw(d, x, y)
		if n.sup != nil {
			sup.draw(d, x+supX, y-u)
		}
		if n.sub != nil {
			sub.draw(d, x+subX, y+v)
		}
	}}, cls, true
}

// bigOp sets a large operator, centered on the axis, larger in display
// style, with its limits above and below or beside it.
func (l *layouter) bigOp(n bigOp, st mstyle, sup, sub []node) box {
	em := st.em()
	scale := float32(1)
	isInt := !n.limits
	if st.display() {
		scale = 1.45
		if isInt {
			scale = 2.0
		}
	} else if st.level == 1 && isInt {
		scale = 1.25
	}
	g := l.glyphs(n.s, l.math, em*scale)
	// Center the glyph on the math axis.
	g = shift(g, axisHeight*em-(g.asc-g.desc)/2)
	if sup == nil && sub == nil {
		return g
	}
	if n.limits && st.display() {
		return l.limits(g, sup, sub, st)
	}
	sst := st.script()
	var sb, bb box
	if sup != nil {
		sb = l.list(sup, sst, false)
	}
	if sub != nil {
		bb = l.list(sub, sst, false)
	}
	supX := g.w + g.italic
	subX := g.w
	if isInt {
		// An integral's slant: the upper limit further right.
		subX = g.w - 0.3*g.w
		supX = g.w + 0.08*em
	}
	u := g.asc - 0.35*sb.asc
	v := g.desc - 0.15*bb.asc
	if !isInt {
		u = max(g.asc-0.39*sst.em(), 0.363*em)
		v = max(g.desc+0.15*em, 0.247*em)
	}
	w := g.w
	if sup != nil {
		w = max(w, supX+sb.w)
	}
	if sub != nil {
		w = max(w, subX+bb.w)
	}
	w += 0.08 * em
	asc, desc := g.asc, g.desc
	if sup != nil {
		asc = max(asc, u+sb.asc)
	}
	if sub != nil {
		desc = max(desc, v+bb.desc)
	}
	return box{w: w, asc: asc, desc: desc, draw: func(d *drawer, x, y float32) {
		g.draw(d, x, y)
		if sup != nil {
			sb.draw(d, x+supX, y-u)
		}
		if sub != nil {
			bb.draw(d, x+subX, y+v)
		}
	}}
}

// limits stacks sup above base and sub below it, centered.
func (l *layouter) limits(base box, sup, sub []node, st mstyle) box {
	em := st.em()
	sst := st.script()
	var sb, bb box
	if sup != nil {
		sb = l.list(sup, sst, false)
	}
	if sub != nil {
		bb = l.list(sub, sst, false)
	}
	gap := 0.14 * em
	w := max(base.w, sb.w, bb.w)
	asc, desc := base.asc, base.desc
	if sup != nil {
		asc += gap + sb.desc + sb.asc
	}
	if sub != nil {
		desc += gap + bb.asc + bb.desc
	}
	return box{w: w, asc: asc, desc: desc, draw: func(d *drawer, x, y float32) {
		base.draw(d, x+(w-base.w)/2, y)
		if sup != nil {
			sb.draw(d, x+(w-sb.w)/2, y-base.asc-gap-sb.desc)
		}
		if sub != nil {
			bb.draw(d, x+(w-bb.w)/2, y+base.desc+gap+bb.asc)
		}
	}}
}

// accent sets a mark over the body: an arrow, a hat, a bar, dots or a
// tilde.
func (l *layouter) accent(n accent, st mstyle) box {
	em := st.em()
	body := l.list(n.body, st, false)
	if n.kind == "underline" {
		t := max(ruleThick*em, 1)
		gap := 0.1 * em
		return box{w: body.w, asc: body.asc, desc: body.desc + gap + t, italic: body.italic, draw: func(d *drawer, x, y float32) {
			body.draw(d, x, y)
			fillRect(d, x, y+body.desc+gap, body.w, t)
		}}
	}
	t := max(ruleThick*em, 1)
	top := max(body.asc, xHeight*em) + 0.1*em
	// Over a single italic letter the mark sits over its slanted middle.
	cx := body.w/2 + body.italic/2
	var h float32
	var drawMark func(d *drawer, x, y float32)
	switch n.kind {
	case "bar", "overline":
		h = t
		x0, x1 := float32(0), body.w
		if n.kind == "bar" && body.w < 0.8*em {
			x0, x1 = cx-0.28*em, cx+0.28*em
		}
		drawMark = func(d *drawer, x, y float32) { fillRect(d, x+x0, y-top-t, x1-x0, t) }
	case "vec", "overrightarrow":
		h = 0.22 * em
		half := max(0.28*em, body.w/2)
		if n.kind == "overrightarrow" {
			half = body.w / 2
			cx = body.w / 2
		}
		drawMark = func(d *drawer, x, y float32) {
			mid := y - top - h/2
			var p clip.Path
			p.Begin(d.ops)
			p.MoveTo(f32.Pt(x+cx-half, mid))
			p.LineTo(f32.Pt(x+cx+half, mid))
			p.MoveTo(f32.Pt(x+cx+half-0.16*em, mid-h/2))
			p.LineTo(f32.Pt(x+cx+half, mid))
			p.LineTo(f32.Pt(x+cx+half-0.16*em, mid+h/2))
			paint.FillShape(d.ops, d.col, clip.Stroke{Path: p.End(), Width: t}.Op())
		}
	case "hat", "widehat", "check":
		h = 0.2 * em
		half := max(0.22*em, min(body.w/2, em))
		if n.kind == "hat" {
			half = 0.22 * em
		}
		drawMark = func(d *drawer, x, y float32) {
			base := y - top
			tip := base - h
			if n.kind == "check" {
				base, tip = tip, base
			}
			var p clip.Path
			p.Begin(d.ops)
			p.MoveTo(f32.Pt(x+cx-half, base))
			p.LineTo(f32.Pt(x+cx, tip))
			p.LineTo(f32.Pt(x+cx+half, base))
			paint.FillShape(d.ops, d.col, clip.Stroke{Path: p.End(), Width: t}.Op())
		}
	case "dot", "ddot":
		r := 0.06 * em
		h = 2 * r
		xs := []float32{cx}
		if n.kind == "ddot" {
			xs = []float32{cx - 0.13*em, cx + 0.13*em}
		}
		drawMark = func(d *drawer, x, y float32) {
			for _, dx := range xs {
				fillDot(d, x+dx, y-top-r, r)
			}
		}
	default: // tilde, breve, acute, grave: their glyphs
		mark := map[string]string{"tilde": "~", "widetilde": "~", "breve": "˘", "acute": "´", "grave": "`"}[n.kind]
		g := l.glyphs(mark, l.math, em*0.9)
		h = g.asc + g.desc
		drawMark = func(d *drawer, x, y float32) {
			g.draw(d, x+cx-g.w/2, y-top-g.desc)
		}
	}
	return box{w: body.w, asc: top + h, desc: body.desc, italic: body.italic, draw: func(d *drawer, x, y float32) {
		body.draw(d, x, y)
		drawMark(d, x, y)
	}}
}

// delimits sets \left ... \right: the delimiters sized to the body,
// centered on the axis.
func (l *layouter) delimits(n delimits, st mstyle) box {
	em := st.em()
	body := l.list(n.body, st, false)
	axis := axisHeight * em
	// Cover the body around the axis, as TeX does (90%, less 0.5em).
	half := max(body.asc-axis, body.desc+axis)
	size := max(2*half*0.901, 2*half-0.5*em, em*1.0)
	left := l.delim(n.left, size, st)
	right := l.delim(n.right, size, st)
	return hbox(left, body, right)
}

// delim draws a delimiter about size tall, centered on the axis; "." is
// none.
func (l *layouter) delim(s string, size float32, st mstyle) box {
	em := st.em()
	if s == "." || s == "" {
		return hspace(0.12 * em)
	}
	g := l.glyphs(s, l.math, em)
	if h := g.asc + g.desc; h > 0 && size > h*1.05 {
		// Larger: the glyph at a larger size, which keeps its shape.
		g = l.glyphs(s, l.math, em*size/h)
	}
	return shift(g, axisHeight*em-(g.asc-g.desc)/2)
}

// rows splits a formula at \\ into lines, each split at & into cells.
func rows(ns []node) [][][]node {
	var out [][][]node
	row := [][]node{nil}
	for _, n := range ns {
		switch n.(type) {
		case lineBreak:
			out = append(out, row)
			row = [][]node{nil}
		case alignMark:
			row = append(row, nil)
		default:
			row[len(row)-1] = append(row[len(row)-1], n)
		}
	}
	return append(out, row)
}

// layoutFormula sets a whole formula: one line, or several aligned at
// their & marks - the cells alternately right and left aligned, as in
// LaTeX's align.
func (l *layouter) layoutFormula(ns []node, st mstyle) box {
	rs := rows(ns)
	if len(rs) == 1 && len(rs[0]) == 1 {
		return l.list(rs[0][0], st, false)
	}
	em := st.em()
	cells := make([][]box, len(rs))
	var cols []float32
	for i, r := range rs {
		for j, c := range r {
			b := l.list(c, st, j > 0)
			cells[i] = append(cells[i], b)
			if j >= len(cols) {
				cols = append(cols, 0)
			}
			cols[j] = max(cols[j], b.w)
		}
	}
	// Pairs of columns (right, left) are a gap apart.
	colX := make([]float32, len(cols))
	var w float32
	for j := range cols {
		if j > 0 && j%2 == 0 {
			w += 2 * em
		}
		colX[j] = w
		w += cols[j]
	}
	gap := 0.35 * em
	type line struct {
		y        float32
		asc, des float32
	}
	lines := make([]line, len(cells))
	var y float32
	for i, r := range cells {
		var asc, desc float32
		for _, b := range r {
			asc = max(asc, b.asc)
			desc = max(desc, b.desc)
		}
		asc = max(asc, 0.75*em)
		desc = max(desc, 0.25*em)
		if i > 0 {
			y += gap
		}
		y += asc
		lines[i] = line{y: y, asc: asc, des: desc}
		y += desc
	}
	total := y
	asc := lines[0].asc
	return box{w: w, asc: asc, desc: total - asc, draw: func(d *drawer, x, y0 float32) {
		top := y0 - asc
		for i, r := range cells {
			for j, b := range r {
				bx := x + colX[j]
				if j%2 == 0 {
					bx += cols[j] - b.w // right-aligned
				}
				b.draw(d, bx, top+lines[i].y)
			}
		}
	}}
}
