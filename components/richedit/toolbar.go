package richedit

import (
	"image"
	"image/color"

	"gioui.org/font"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/internal/textrender"
	"github.com/xmhrnx/giode/richdoc"
)

// toolButton is a toolbar button.
type toolButton struct {
	label, name string
	f           font.Font
	under, strk bool
	active      func() bool
	action      func()
	tag         int
	sep         bool // a separator before it
}

// Toolbar returns a row of formatting buttons for the editor: marks,
// headings, lists, quote, code, link (with OnLinkEdit), undo and redo.
// Pressing them leaves the keyboard focus in the editor. Apps wanting
// another set build their own from the editor's commands.
func (e *RichEditor) Toolbar() elements.Element {
	if e.tools == nil {
		e.tools = e.buildTools()
	}
	return elements.Raw(e.layoutToolbar)
}

func (e *RichEditor) buildTools() []*toolButton {
	kindIs := func(k richdoc.Kind, level int) func() bool {
		return func() bool { b, l := e.Kind(); return b == k && (k != richdoc.Heading || l == level) }
	}
	mark := func(k richdoc.MarkKind) func() bool { return func() bool { return e.HasMark(k) } }
	toggle := func(k richdoc.MarkKind) func() {
		return func() {
			if e.cell.on {
				e.cellMark(k)
			} else {
				e.ToggleMark(k)
			}
		}
	}
	setKind := func(k richdoc.Kind, level int) func() { return func() { e.SetKind(k, level) } }
	bt := []*toolButton{
		{label: "B", name: "Bold", f: font.Font{Weight: font.Bold}, active: mark(richdoc.Bold), action: toggle(richdoc.Bold)},
		{label: "I", name: "Italic", f: font.Font{Style: font.Italic}, active: mark(richdoc.Italic), action: toggle(richdoc.Italic)},
		{label: "U", name: "Underline", under: true, active: mark(richdoc.Underline), action: toggle(richdoc.Underline)},
		{label: "S", name: "Strikethrough", strk: true, active: mark(richdoc.Strike), action: toggle(richdoc.Strike)},
		{label: "</>", name: "Code", f: font.Font{Typeface: "Go Mono"}, active: mark(richdoc.InlineCode), action: toggle(richdoc.InlineCode)},
		{label: "H1", name: "Heading 1", f: font.Font{Weight: font.Bold}, sep: true, active: kindIs(richdoc.Heading, 1), action: setKind(richdoc.Heading, 1)},
		{label: "H2", name: "Heading 2", f: font.Font{Weight: font.Bold}, active: kindIs(richdoc.Heading, 2), action: setKind(richdoc.Heading, 2)},
		{label: "•", name: "Bulleted list", sep: true, active: kindIs(richdoc.Bullet, 0), action: setKind(richdoc.Bullet, 0)},
		{label: "1.", name: "Numbered list", active: kindIs(richdoc.Numbered, 0), action: setKind(richdoc.Numbered, 0)},
		{label: "☐", name: "Checklist", active: kindIs(richdoc.Task, 0), action: setKind(richdoc.Task, 0)},
		{label: "“", name: "Quote", f: font.Font{Weight: font.Bold}, active: kindIs(richdoc.Quote, 0), action: setKind(richdoc.Quote, 0)},
		{label: "{ }", name: "Code block", f: font.Font{Typeface: "Go Mono"}, active: kindIs(richdoc.Code, 0), action: setKind(richdoc.Code, 0)},
	}
	if e.onLinkEdit != nil {
		bt = append(bt, &toolButton{label: "Link", name: "Link", sep: true, action: e.editLink,
			active: func() bool { return e.st.Marks().Link != "" }})
	}
	bt = append(bt,
		&toolButton{label: "Undo", name: "Undo", sep: true, action: e.Undo},
		&toolButton{label: "Redo", name: "Redo", action: e.Redo},
	)
	return bt
}

func (e *RichEditor) layoutToolbar(gtx layout.Context) layout.Dimensions {
	pal := e.palette()
	size := 15.0
	h := gtx.Dp(32)
	gap := gtx.Dp(2)
	sepW := gtx.Dp(9)
	maxW := gtx.Constraints.Max.X
	x, y := 0, 0
	type placed struct {
		b *toolButton
		r image.Rectangle
		l *textrender.RichLaid
	}
	var out []placed
	for _, b := range e.tools {
		f := b.f
		l := textrender.LayoutRich(fonts.Shaper(), unbounded(gtx), textrender.RichParams{},
			[]textrender.Span{{Font: fonts.ForText(f, b.label), Size: unitSp(size), Content: b.label}})
		w := max(h, l.Dims.Size.X+gtx.Dp(16))
		if b.sep && x > 0 {
			x += sepW
		}
		if x > 0 && x+w > maxW {
			x, y = 0, y+h+gap
		}
		out = append(out, placed{b, image.Rect(x, y, x+w, y+h), l})
		x += w + gap
	}
	// Clicks.
	for _, p := range out {
		b := p.b
		for {
			ev, ok := gtx.Event(pointer.Filter{Target: &b.tag, Kinds: pointer.Press | pointer.Release | pointer.Cancel})
			if !ok {
				break
			}
			if pe, ok := ev.(pointer.Event); ok && pe.Kind == pointer.Release && !e.readOnly {
				b.action()
				e.wantFocus = true
				kit.RequestFrame(gtx)
			}
		}
	}
	dims := image.Pt(maxW, y+h)
	for i, p := range out {
		b := p.b
		r := p.r
		if b.sep && i > 0 {
			mid := r.Min.X - sepW/2 - gap
			fillRect(gtx, image.Rect(mid, r.Min.Y+gtx.Dp(6), mid+max(gtx.Dp(1), 1), r.Max.Y-gtx.Dp(6)), pal.border)
		}
		on := b.active != nil && b.active()
		disabled := e.readOnly || b.name == "Undo" && !e.CanUndo() || b.name == "Redo" && !e.CanRedo()
		if on {
			fillRRect(gtx, r, gtx.Dp(6), pal.chip)
		}
		c := pal.text
		if on {
			c = pal.link
		}
		if disabled {
			c = pal.muted
			c.A /= 2
		}
		o := offset(gtx, image.Pt(r.Min.X+(r.Dx()-p.l.Dims.Size.X)/2, r.Min.Y+(r.Dy()-p.l.Dims.Size.Y)/2))
		p.l.Draw(gtx.Ops, fonts.Shaper(), func(int) color.NRGBA { return c })
		lw, lh := p.l.Dims.Size.X, p.l.Dims.Size.Y
		if b.under {
			fillRect(gtx, image.Rect(0, lh-gtx.Dp(3), lw, lh-gtx.Dp(3)+max(gtx.Dp(1), 1)), c)
		}
		if b.strk {
			fillRect(gtx, image.Rect(0, lh/2, lw, lh/2+max(gtx.Dp(1), 1)), c)
		}
		o.Pop()
		area := clip.Rect(r).Push(gtx.Ops)
		event.Op(gtx.Ops, &b.tag)
		pointer.CursorPointer.Add(gtx.Ops)
		semantic.Button.Add(gtx.Ops)
		semantic.LabelOp(b.name).Add(gtx.Ops)
		elements.RecordSemantic(gtx, elements.SemanticNode{Kind: elements.SemanticButton, Label: b.name,
			Checked: checkState(b.active != nil, on), Disabled: disabled}, r.Size())
		area.Pop()
	}
	kit.BlockDrag(gtx, dims)
	return layout.Dimensions{Size: dims}
}

func checkState(checkable, on bool) elements.CheckState {
	switch {
	case !checkable:
		return elements.NotCheckable
	case on:
		return elements.Checked
	}
	return elements.Unchecked
}

func unitSp(v float64) unit.Sp { return unit.Sp(v) }
