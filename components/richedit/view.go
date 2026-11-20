package richedit

import (
	"encoding/binary"
	"hash/maphash"
	"image"
	"image/color"
	"math"
	"strconv"
	"strings"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/internal/textrender"
	"github.com/xmhrnx/giode/richdoc"
	"github.com/xmhrnx/giode/styles/properties"
)

// palette is the editor's colors, resolved once per frame.
type palette struct {
	text, muted, link, border, codeBg, chip, chipText, sel, caret, check, checkText color.NRGBA
}

func (e *RichEditor) palette() palette {
	c := func(s string) color.NRGBA { return properties.CalcColor(s) }
	p := palette{
		text: c("theme.text"), muted: c("theme.muted"), link: c("theme.primary"), border: c("theme.border"),
		check: c("theme.primary"), checkText: c("theme.onprimary"),
	}
	if e.style.Color != "" {
		p.text = properties.CalcColor(string(e.style.Color))
	}
	p.caret = p.link
	p.sel = p.link
	p.sel.A = 0x50
	p.codeBg = p.text
	p.codeBg.A = 0x12
	p.chip = p.link
	p.chip.A = 0x24
	p.chipText = p.link
	return p
}

// geom is a block's look: its text size and font, and where its text
// sits in it.
type geom struct {
	size   unit.Sp
	weight font.Weight
	mono   bool
	// indent is the text's start from the block's start; padX the room
	// on the end side; padTop and padBottom the space around the text.
	indent, padX, padTop, padBottom int
	muted                           bool
}

func (e *RichEditor) geomOf(gtx layout.Context, b *richdoc.Block) geom {
	base := properties.ResolveFontSize(e.style.FontSize)
	g := geom{size: base, weight: font.Normal}
	gap := gtx.Sp(base * 0.5)
	g.padBottom = gap
	ind := gtx.Sp(base * 1.6)
	switch b.Kind {
	case richdoc.Heading:
		scale := [...]float32{2, 1.6, 1.35, 1.15, 1, 0.9}[max(0, min(5, b.Level-1))]
		g.size = base * unit.Sp(scale)
		g.weight = font.Bold
		g.padTop = gtx.Sp(base * 0.5)
	case richdoc.Bullet, richdoc.Numbered, richdoc.Task:
		g.indent = ind * (b.Level + 1)
		g.padBottom = gtx.Sp(base * 0.2)
	case richdoc.Quote:
		g.indent = ind*b.Level + gtx.Sp(base)
		g.muted = true
	case richdoc.Code:
		g.mono = true
		g.size = base * 0.9
		pad := gtx.Sp(base * 0.75)
		g.indent, g.padX = pad, pad
		g.padTop, g.padBottom = pad, pad+gap
	}
	return g
}

// blockView is a laid out block, cached by its content and width.
type blockView struct {
	key    uint64
	used   uint64
	height int
	g      geom
	// Text blocks: the laid out text and its caret stops by line, in the
	// block's coordinates; atoms are boxes of one rune, so runes are
	// model positions.
	laid  *textrender.RichLaid
	lines []textrender.StopLine
	spans []richdoc.Span
	empty bool
	// chips are the atoms' labels, by span; src the model span each laid
	// out span is from.
	chips map[int]*textrender.RichLaid
	src   []int
	// Tables: the cells' text and boxes.
	cells   [][]cellView
	cellPad int
	// Images: the picture's size.
	img image.Point
	// The list marker.
	marker string
}

type cellView struct {
	laid  *textrender.RichLaid
	lines []textrender.StopLine
	rect  image.Rectangle
	chips map[int]*textrender.RichLaid
	src   []int
}

var seed = maphash.MakeSeed()

// blockKey hashes what a block's layout depends on.
func blockKey(b *richdoc.Block, width int, extra string) uint64 {
	var h maphash.Hash
	h.SetSeed(seed)
	var buf [8]byte
	num := func(v int) { binary.LittleEndian.PutUint64(buf[:], uint64(v)); h.Write(buf[:]) }
	str := func(s string) { num(len(s)); h.WriteString(s) }
	spans := func(ss []richdoc.Span) {
		num(len(ss))
		for _, s := range ss {
			str(s.Text)
			m := s.Marks
			flags := 0
			for i, on := range []bool{m.Bold, m.Italic, m.Underline, m.Strike, m.Code} {
				if on {
					flags |= 1 << i
				}
			}
			num(flags)
			str(m.Color)
			str(m.Highlight)
			str(m.Link)
			if s.Atom != nil {
				str(s.Atom.Kind)
				str(s.Atom.Label)
			}
		}
	}
	num(int(b.Kind))
	num(b.Level)
	num(width)
	if b.Checked {
		num(1)
	}
	str(b.Lang)
	str(b.Src)
	str(extra)
	spans(b.Spans)
	for _, r := range b.Cells {
		for _, c := range r {
			spans(c)
		}
	}
	return h.Sum64()
}

// trSpans turns a block's spans into spans to lay out, atoms as boxes.
func (e *RichEditor) trSpans(gtx layout.Context, ss []richdoc.Span, g geom, chips map[int]*textrender.RichLaid, src *[]int) []textrender.Span {
	out := make([]textrender.Span, 0, len(ss))
	for i, s := range ss {
		f := font.Font{Weight: g.weight}
		if g.mono || s.Marks.Code {
			f.Typeface = "Go Mono"
		}
		if s.Marks.Bold {
			f.Weight = font.Bold
		}
		if s.Marks.Italic {
			f.Style = font.Italic
		}
		size := g.size
		if s.Marks.Code && !g.mono {
			size *= 0.9
		}
		if s.Atom != nil {
			label := s.Atom.Label
			if label == "" {
				label = s.Atom.Plain()
			}
			if s.Atom.Kind == "emoji" {
				// An emoji is drawn as text, a box all the same.
				label = s.Atom.Plain()
			}
			lf := f
			if s.Atom.Kind != "emoji" {
				lf.Weight = font.SemiBold
			}
			l := textrender.LayoutRich(fonts.Shaper(), unbounded(gtx), textrender.RichParams{},
				[]textrender.Span{{Font: fonts.ForText(lf, label), Size: size, Content: label}})
			chips[i] = l
			pad := gtx.Dp(4)
			if s.Atom.Kind == "emoji" {
				pad = 0
			}
			box := image.Pt(l.Dims.Size.X+2*pad, l.Dims.Size.Y)
			asc := l.Dims.Size.Y
			if len(l.Segments) > 0 {
				asc = l.Segments[0].Offset.Y + l.Segments[0].Ascent
			}
			out = append(out, textrender.Span{Font: f, Size: size, Content: "\ufffc", Box: box, Ascent: asc})
			*src = append(*src, i)
			continue
		}
		// Tabs are boxes four spaces wide, keeping one rune each.
		rest := s.Text
		for {
			k := strings.IndexByte(rest, '\t')
			if k < 0 {
				break
			}
			if k > 0 {
				out = append(out, textrender.Span{Font: fonts.ForText(f, rest[:k]), Size: size, Content: rest[:k]})
				*src = append(*src, i)
			}
			out = append(out, textrender.Span{Font: f, Size: size, Content: "\t", Box: image.Pt(gtx.Sp(size*0.6*4), 1), Ascent: 1})
			*src = append(*src, i)
			rest = rest[k+1:]
		}
		if rest != "" {
			out = append(out, textrender.Span{Font: fonts.ForText(f, rest), Size: size, Content: rest})
			*src = append(*src, i)
		}
	}
	return out
}

func unbounded(gtx layout.Context) layout.Context {
	gtx.Constraints = layout.Constraints{Max: image.Pt(1<<20, 1<<20)}
	return gtx
}

// layoutText lays out spans in width, returning the layout and its stops;
// an empty one has a caret stop at its start.
func (e *RichEditor) layoutText(gtx layout.Context, ss []richdoc.Span, g geom, width int, chips map[int]*textrender.RichLaid, src *[]int) (*textrender.RichLaid, []textrender.StopLine, bool) {
	tgtx := gtx
	tgtx.Constraints = layout.Constraints{Max: image.Pt(max(width, 1), 1<<20)}
	sh := fonts.Shaper()
	*src = (*src)[:0]
	spans := e.trSpans(gtx, ss, g, chips, src)
	n := 0
	starts := make([]int, len(spans))
	for i, s := range spans {
		starts[i] = n
		n += len([]rune(s.Content))
	}
	empty := n == 0
	if empty || len(ss) > 0 && ss[len(ss)-1].Atom == nil && len(ss[len(ss)-1].Text) > 0 && ss[len(ss)-1].Text[len(ss[len(ss)-1].Text)-1] == '\n' {
		// A zero-width space holds an empty line's height.
		f := font.Font{Weight: g.weight}
		if g.mono {
			f.Typeface = "Go Mono"
		}
		spans = append(spans, textrender.Span{Font: f, Size: g.size, Content: "​"})
		starts = append(starts, n)
	}
	laid := textrender.LayoutRich(sh, tgtx, textrender.RichParams{Alignment: text.Start}, spans)
	lines := laid.StopLines(starts)
	if empty {
		for i := range lines {
			lines[i].Stops = lines[i].Stops[:1]
		}
	}
	return laid, lines, empty
}

// view returns block i laid out in width, from cache when unchanged.
func (e *RichEditor) view(gtx layout.Context, i, width int) *blockView {
	b := &e.st.Doc.Blocks[i]
	marker := e.markers[i]
	key := e.keys[i]
	if v := e.views[key]; v != nil {
		v.used = e.frame
		return v
	}
	v := &blockView{key: key, used: e.frame, marker: marker}
	v.g = e.geomOf(gtx, b)
	textW := width - v.g.indent - v.g.padX
	switch b.Kind {
	case richdoc.Table:
		v.layoutTable(gtx, e, b, width)
	case richdoc.Image:
		v.img = e.imageSize(gtx, b, width)
		v.height = v.img.Y + gtx.Dp(12)
	case richdoc.Rule:
		v.height = gtx.Dp(24)
	default:
		v.chips = map[int]*textrender.RichLaid{}
		v.laid, v.lines, v.empty = e.layoutText(gtx, b.Spans, v.g, textW, v.chips, &v.src)
		v.spans = b.Spans
		v.height = v.g.padTop + v.laid.Dims.Size.Y + v.g.padBottom
	}
	e.views[key] = v
	return v
}

// computeMarkers sets each list item's marker: a bullet by depth, or
// its number, counting on through deeper items between.
func (e *RichEditor) computeMarkers() {
	blocks := e.st.Doc.Blocks
	e.markers = e.markers[:0]
	var count [9]int
	for i := range blocks {
		b := &blocks[i]
		m := ""
		if !b.Kind.IsList() {
			count = [9]int{}
		} else {
			l := min(b.Level, 8)
			for k := l + 1; k < 9; k++ {
				count[k] = 0
			}
			switch b.Kind {
			case richdoc.Bullet:
				count[l] = 0
				m = [...]string{"•", "◦", "▪"}[l%3]
			case richdoc.Numbered:
				count[l]++
				n := count[l]
				switch l % 3 {
				case 1:
					m = alpha(n) + "."
				case 2:
					m = roman(n) + "."
				default:
					m = strconv.Itoa(n) + "."
				}
			default:
				count[l] = 0
			}
		}
		e.markers = append(e.markers, m)
	}
}

func alpha(n int) string {
	s := ""
	for n > 0 {
		n--
		s = string(rune('a'+n%26)) + s
		n /= 26
	}
	return s
}

func roman(n int) string {
	vals := []int{1000, 900, 500, 400, 100, 90, 50, 40, 10, 9, 5, 4, 1}
	syms := []string{"m", "cm", "d", "cd", "c", "xc", "l", "xl", "x", "ix", "v", "iv", "i"}
	s := ""
	for i, v := range vals {
		for n >= v {
			s += syms[i]
			n -= v
		}
	}
	return s
}

func (v *blockView) layoutTable(gtx layout.Context, e *RichEditor, b *richdoc.Block, width int) {
	cols := 0
	for _, r := range b.Cells {
		cols = max(cols, len(r))
	}
	if cols == 0 {
		v.height = gtx.Dp(8)
		return
	}
	pad := gtx.Dp(8)
	v.cellPad = pad
	colW := width / cols
	y := 0
	v.cells = make([][]cellView, len(b.Cells))
	for r, row := range b.Cells {
		g := v.g
		if r == 0 {
			g.weight = font.Bold
		}
		h := 0
		v.cells[r] = make([]cellView, cols)
		for c := 0; c < cols; c++ {
			var spans []richdoc.Span
			if c < len(row) {
				spans = row[c]
			}
			cv := cellView{chips: map[int]*textrender.RichLaid{}}
			cv.laid, cv.lines, _ = e.layoutText(gtx, spans, g, colW-2*pad, cv.chips, &cv.src)
			v.cells[r][c] = cv
			laid := cv.laid
			h = max(h, laid.Dims.Size.Y)
		}
		for c := range v.cells[r] {
			x0 := c * colW
			v.cells[r][c].rect = image.Rect(x0, y, x0+colW, y+h+2*pad)
		}
		y += h + 2*pad
	}
	v.height = y + v.g.padBottom
}

func (e *RichEditor) imageSize(gtx layout.Context, b *richdoc.Block, width int) image.Point {
	img := e.image(b.Src)
	maxH := gtx.Dp(420)
	if img == nil {
		return image.Pt(min(width, gtx.Dp(240)), gtx.Dp(120))
	}
	s := img.Bounds().Size()
	if s.X <= 0 || s.Y <= 0 {
		return image.Pt(width, gtx.Dp(120))
	}
	w := float64(min(width, gtx.Dp(unit.Dp(s.X))))
	h := w * float64(s.Y) / float64(s.X)
	if h > float64(maxH) {
		h = float64(maxH)
		w = h * float64(s.X) / float64(s.Y)
	}
	return image.Pt(int(math.Round(w)), int(math.Round(h)))
}

// image returns a block's picture, loaded once.
func (e *RichEditor) image(src string) image.Image {
	if e.loadImage == nil || src == "" {
		return nil
	}
	if img, ok := e.images[src]; ok {
		return img
	}
	img := e.loadImage(src)
	e.images[src] = img
	return img
}

func fillRect(gtx layout.Context, r image.Rectangle, c color.NRGBA) {
	if c.A == 0 || r.Empty() {
		return
	}
	paint.FillShape(gtx.Ops, c, clip.Rect(r).Op())
}

func fillRRect(gtx layout.Context, r image.Rectangle, radius int, c color.NRGBA) {
	if c.A == 0 || r.Empty() {
		return
	}
	paint.FillShape(gtx.Ops, c, clip.UniformRRect(r, radius).Op(gtx.Ops))
}

func offset(gtx layout.Context, p image.Point) op.TransformStack {
	return op.Offset(p).Push(gtx.Ops)
}
