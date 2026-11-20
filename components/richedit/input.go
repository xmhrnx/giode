package richedit

import (
	"image"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"

	"gioui.org/f32"
	"gioui.org/io/clipboard"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/transfer"
	"gioui.org/layout"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/richdoc"
)

// pointerState is a press being tracked.
type pointerState struct {
	dragging  bool
	touch     bool
	id        pointer.ID
	start     f32.Point
	moved     bool
	lastClick time.Time
	lastPos   image.Point
	clicks    int
	// unit is what a drag extends by: 0 characters, 1 words, 2 blocks.
	unit     int
	anchor   richdoc.Selection
	cellDrag bool
}

// lastCopy is the text last copied from an editor and its rich content,
// for pasting it back with its formatting.
var lastCopy struct {
	sync.Mutex
	text string
	doc  *richdoc.Doc
}

var editKeys = []key.Name{
	key.NameLeftArrow, key.NameRightArrow, key.NameUpArrow, key.NameDownArrow,
	key.NameHome, key.NameEnd, key.NamePageUp, key.NamePageDown,
	key.NameDeleteBackward, key.NameDeleteForward, key.NameReturn, key.NameEnter,
}

// update handles input since the last frame and reports whether the
// editor changed.
func (e *RichEditor) update(gtx layout.Context) bool {
	filters := []event.Filter{
		key.FocusFilter{Target: e},
		pointer.Filter{Target: e, Kinds: pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel},
		transfer.TargetFilter{Target: e, Type: "application/text"},
	}
	mods := key.ModShift | key.ModShortcut | key.ModAlt | key.ModCtrl
	for _, n := range editKeys {
		filters = append(filters, key.Filter{Focus: e, Name: n, Optional: mods})
	}
	for _, n := range []key.Name{"A", "C", "X", "V", "Z", "Y", "B", "I", "U", "K", "E", "1", "2", "3", "4", "5", "6", "0", "7", "8", "9"} {
		filters = append(filters, key.Filter{Focus: e, Name: n, Required: key.ModShortcut, Optional: key.ModShift | key.ModAlt})
	}
	if e.sugg.open || e.tabbable() || e.cell.on {
		filters = append(filters, key.Filter{Focus: e, Name: key.NameTab, Optional: key.ModShift})
	}
	if e.sugg.open || e.cell.on {
		filters = append(filters, key.Filter{Focus: e, Name: key.NameEscape})
	}
	// Shift+F10 opens the context menu at the caret.
	filters = append(filters, key.Filter{Focus: e, Name: key.NameF10, Required: key.ModShift})
	changed := false
	for {
		ev, ok := gtx.Event(filters...)
		if !ok {
			break
		}
		changed = true
		e.blink = gtx.Now
		switch ev := ev.(type) {
		case key.FocusEvent:
			e.focused = ev.Focus
			e.ime = imeState{block: -1, composing: key.Range{Start: -1, End: -1}}
			if ev.Focus && !e.readOnly {
				gtx.Execute(key.SoftKeyboardCmd{Show: true})
			}
			if !ev.Focus {
				e.sugg.open = false
			}
		case key.Event:
			if ev.State == key.Press && gtx.Focused(e) {
				e.onKey(gtx, ev)
			}
		case key.EditEvent:
			e.onEdit(gtx, ev)
		case key.CompositionEvent:
			e.ime.composing = key.Range(ev)
		case key.SnippetEvent:
			// The whole block is always the snippet.
		case key.SelectionEvent:
			if e.cell.on && e.ime.block == -2 {
				if spans, ok := e.cellSpans(); ok {
					n := richdoc.SpansLen(spans)
					e.cell.anchor, e.cell.head = min(max(ev.Start, 0), n), min(max(ev.End, 0), n)
				}
			} else if b := e.ime.block; b >= 0 && b < len(e.st.Doc.Blocks) {
				n := e.st.Doc.BlockLen(b)
				e.st.Select(richdoc.Selection{
					Anchor: richdoc.Pos{Block: b, Offset: min(max(ev.Start, 0), n)},
					Head:   richdoc.Pos{Block: b, Offset: min(max(ev.End, 0), n)},
				})
				e.desiredX = -1
			}
		case transfer.DataEvent:
			r := ev.Open()
			b, err := io.ReadAll(io.LimitReader(r, 8<<20))
			r.Close()
			if err == nil && !e.readOnly {
				if e.cell.on {
					e.cellText(string(b))
				} else {
					e.paste(string(b))
				}
			}
		case pointer.Event:
			e.onPointer(gtx, ev)
		}
	}
	e.updateQuery()
	return changed
}

// tabbable reports whether Tab indents here: in a list or code block.
// Elsewhere it moves focus on.
func (e *RichEditor) tabbable() bool {
	if e.readOnly || e.st == nil {
		return false
	}
	b := &e.st.Doc.Blocks[e.st.Sel.Head.Block]
	return b.Kind.IsList() || b.Kind == richdoc.Code
}

// onEdit applies text the keyboard or input method typed: Range is in
// the caret block's text.
func (e *RichEditor) onEdit(gtx layout.Context, ev key.EditEvent) {
	if e.readOnly {
		return
	}
	text := ev.Text
	if e.cell.on {
		if text == "\n" {
			e.cellKey(gtx, key.Event{Name: key.NameReturn, State: key.Press})
		} else {
			e.cellEdit(ev)
		}
		return
	}
	if text == "\n" {
		e.enter(gtx, false, false)
		return
	}
	b := e.ime.block
	sel := e.st.Sel
	if b >= 0 && b < len(e.st.Doc.Blocks) && key.Range(ev.Range) != e.ime.sel {
		n := e.st.Doc.BlockLen(b)
		e.st.Select(richdoc.Selection{
			Anchor: richdoc.Pos{Block: b, Offset: min(max(ev.Range.Start, 0), n)},
			Head:   richdoc.Pos{Block: b, Offset: min(max(ev.Range.End, 0), n)},
		})
		if sel.Start().Block == sel.End().Block {
			e.st.Stored = nil
		}
	}
	rules := e.st.Rules
	if e.ime.composing.Start >= 0 {
		e.st.Rules = false
	}
	if text == "" {
		e.run(e.st.Delete(true, richdoc.Char))
	} else {
		e.run(e.st.InsertText(text))
	}
	e.st.Rules = rules
}

func (e *RichEditor) onKey(gtx layout.Context, ev key.Event) {
	shift := ev.Modifiers.Contain(key.ModShift)
	short := ev.Modifiers.Contain(key.ModShortcut)
	word := ev.Modifiers.Contain(key.ModShortcutAlt)
	if e.sugg.open && e.suggestKey(ev) {
		return
	}
	if e.cell.on && ev.Name != key.NameF10 && e.cellKey(gtx, ev) {
		return
	}
	if e.cell.on {
		// Undo and the like act on the document; the cell's caret goes.
		e.cell.on = false
	}
	ro := e.readOnly
	if ev.Name == key.NameF10 {
		e.openMenuAtCaret()
		return
	}
	switch ev.Name {
	case key.NameReturn, key.NameEnter:
		if !ro {
			e.enter(gtx, shift, short)
		}
	case key.NameDeleteBackward, key.NameDeleteForward:
		if ro {
			return
		}
		unit := richdoc.Char
		if word {
			unit = richdoc.Word
		}
		if short && ev.Modifiers.Contain(key.ModShift) {
			unit = richdoc.LineEdge
		}
		e.run(e.st.Delete(ev.Name == key.NameDeleteBackward, unit))
	case key.NameTab:
		if !ro {
			b := &e.st.Doc.Blocks[e.st.Sel.Head.Block]
			if b.Kind == richdoc.Code && !shift {
				e.run(e.st.InsertText("\t"))
			} else {
				e.run(e.st.Indent(shift))
			}
		}
	case key.NameLeftArrow, key.NameRightArrow:
		forward := ev.Name == key.NameRightArrow
		if kit.RTL(gtx) {
			forward = !forward
		}
		e.moveHorizontal(forward, word, shift)
	case key.NameUpArrow, key.NameDownArrow:
		dir := 1
		if ev.Name == key.NameUpArrow {
			dir = -1
		}
		if short {
			if dir < 0 {
				e.moveTo(richdoc.Pos{}, shift)
			} else {
				e.moveTo(e.st.Doc.End(), shift)
			}
			return
		}
		e.moveVertical(dir, shift, 1)
	case key.NamePageUp, key.NamePageDown:
		dir := 1
		if ev.Name == key.NamePageUp {
			dir = -1
		}
		e.moveVertical(dir, shift, max(e.size.Y/gtx.Dp(20)-1, 1))
	case key.NameHome, key.NameEnd:
		end := ev.Name == key.NameEnd
		switch {
		case short && end:
			e.moveTo(e.st.Doc.End(), shift)
		case short:
			e.moveTo(richdoc.Pos{}, shift)
		default:
			e.moveTo(e.lineEdge(e.st.Sel.Head, end), shift)
		}
	case key.NameEscape:
		e.sugg.open = false
		e.sugg.dismissed = e.sugg.at
	default:
		if !short {
			return
		}
		e.shortcut(gtx, ev, shift)
	}
}

func (e *RichEditor) shortcut(gtx layout.Context, ev key.Event, shift bool) {
	alt := ev.Modifiers.Contain(key.ModAlt)
	switch ev.Name {
	case "A":
		e.st.SelectAll()
	case "C", "X":
		if ev.Name == "X" && shift {
			break // strike, below
		}
		if e.st.Sel.Empty() {
			return
		}
		frag := e.st.Fragment()
		text := strings.TrimSuffix(frag.Text(), "\n")
		lastCopy.Lock()
		lastCopy.text, lastCopy.doc = text, frag
		lastCopy.Unlock()
		kit.WriteClipboard(gtx, text)
		if ev.Name == "X" && !e.readOnly {
			e.run(e.st.Delete(true, richdoc.Char))
		}
	case "V":
		if !e.readOnly {
			gtx.Execute(clipboard.ReadCmd{Tag: e})
		}
	case "Z":
		if shift {
			e.Redo()
		} else {
			e.Undo()
		}
	case "Y":
		e.Redo()
	case "B":
		e.ToggleMark(richdoc.Bold)
	case "I":
		e.ToggleMark(richdoc.Italic)
	case "U":
		e.ToggleMark(richdoc.Underline)
	case "E":
		e.ToggleMark(richdoc.InlineCode)
	case "K":
		e.editLink()
	case "1", "2", "3", "4", "5", "6":
		if alt {
			e.SetKind(richdoc.Heading, int(ev.Name[0]-'0'))
		}
	case "0":
		if alt {
			e.SetKind(richdoc.Paragraph, 0)
		}
	case "7", "8", "9":
		if shift {
			e.SetKind([]richdoc.Kind{richdoc.Numbered, richdoc.Bullet, richdoc.Task}[ev.Name[0]-'7'], 0)
		}
	}
	if ev.Name == "X" && shift {
		e.ToggleMark(richdoc.Strike)
	}
	e.blink = time.Now()
}

// editLink asks the app for a URL for the selection (Ctrl+K).
func (e *RichEditor) editLink() {
	if e.onLinkEdit == nil || e.readOnly {
		return
	}
	if e.st.Sel.Empty() {
		// The word at the caret.
		e.selectWord(e.st.Sel.Head)
	}
	cur := e.st.Marks().Link
	e.onLinkEdit(cur, e.SetLink)
}

// enter handles Enter: submitting, a line break or a new block.
func (e *RichEditor) enter(gtx layout.Context, shift, short bool) {
	if e.sugg.open && e.acceptSuggestion() {
		return
	}
	switch e.submitOn {
	case SubmitEnter:
		if !shift && !short {
			e.submit()
			return
		}
		if shift {
			e.run(e.st.Enter())
			return
		}
	case SubmitShortcut:
		if short {
			e.submit()
			return
		}
	}
	if shift {
		// A line break within the block.
		e.run(e.st.InsertLineBreak())
		return
	}
	e.run(e.st.Enter())
}

func (e *RichEditor) submit() {
	if e.onSubmit != nil {
		e.onSubmit()
	}
}

// moveTo moves the caret to p, extending the selection when extend.
func (e *RichEditor) moveTo(p richdoc.Pos, extend bool) {
	sel := e.st.Sel
	if extend {
		sel.Head = p
	} else {
		sel = richdoc.Caret(p)
	}
	e.st.Select(sel)
	e.desiredX = -1
	e.reveal = true
}

func (e *RichEditor) moveHorizontal(forward, word, extend bool) {
	sel := e.st.Sel
	if !sel.Empty() && !extend {
		if forward {
			e.moveTo(sel.End(), false)
		} else {
			e.moveTo(sel.Start(), false)
		}
		return
	}
	p := sel.Head
	d := e.st.Doc
	text := []rune(blockText(&d.Blocks[p.Block]))
	n := d.BlockLen(p.Block)
	switch {
	case forward && p.Offset >= n:
		if p.Block+1 < len(d.Blocks) {
			p = richdoc.Pos{Block: p.Block + 1}
		}
	case !forward && p.Offset == 0:
		if p.Block > 0 {
			p = richdoc.Pos{Block: p.Block - 1, Offset: d.BlockLen(p.Block - 1)}
		}
	case !d.Blocks[p.Block].Kind.HasText():
		p.Offset = 1 - p.Offset
	case word && forward:
		p.Offset = richdoc.WordEnd(text, p.Offset)
	case word:
		p.Offset = richdoc.WordStart(text, p.Offset)
	case forward:
		p.Offset = richdoc.NextStop(text, p.Offset)
	default:
		p.Offset = richdoc.PrevStop(text, p.Offset)
	}
	e.moveTo(p, extend)
}

func (e *RichEditor) moveVertical(dir int, extend bool, lines int) {
	p := e.st.Sel.Head
	if !extend && !e.st.Sel.Empty() {
		if dir < 0 {
			p = e.st.Sel.Start()
		} else {
			p = e.st.Sel.End()
		}
	}
	x := e.desiredX
	if x < 0 {
		if r, ok := e.caretRect(p); ok {
			x = r.Min.X + 1
		}
	}
	for i := 0; i < lines; i++ {
		np, ok := e.vertical(p, dir, x)
		if !ok || np == p {
			break
		}
		p = np
	}
	e.moveTo(p, extend)
	e.desiredX = x
}

// selectWord selects the word at p.
func (e *RichEditor) selectWord(p richdoc.Pos) {
	b := &e.st.Doc.Blocks[p.Block]
	if !b.Kind.HasText() {
		e.st.Select(richdoc.Selection{Anchor: richdoc.Pos{Block: p.Block}, Head: richdoc.Pos{Block: p.Block, Offset: 1}})
		return
	}
	text := []rune(blockText(b))
	from, to := p.Offset, p.Offset
	for from > 0 && richdoc.IsWordRune(text[from-1]) {
		from--
	}
	for to < len(text) && richdoc.IsWordRune(text[to]) {
		to++
	}
	e.st.Select(richdoc.Selection{Anchor: richdoc.Pos{Block: p.Block, Offset: from}, Head: richdoc.Pos{Block: p.Block, Offset: to}})
}

func (e *RichEditor) onPointer(gtx layout.Context, ev pointer.Event) {
	if len(e.laid) != len(e.st.Doc.Blocks) {
		return
	}
	pt := ev.Position.Round()
	content := image.Pt(pt.X, pt.Y+int(e.offY))
	switch ev.Kind {
	case pointer.Press:
		if ev.Source == pointer.Mouse && !ev.Buttons.Contain(pointer.ButtonPrimary) {
			return
		}
		kit.NotePointerPress()
		e.wantFocus = true
		e.ptr.touch = ev.Source == pointer.Touch
		e.ptr.id, e.ptr.start, e.ptr.moved = ev.PointerID, ev.Position, false
		// A task's box toggles it.
		for _, tb := range e.taskBoxes {
			if content.In(tb.r.Inset(-gtx.Dp(4))) {
				if !e.readOnly {
					e.run(e.st.ToggleTask(tb.block))
				}
				return
			}
		}
		if e.ptr.touch {
			return // a tap places the caret on release, so swipes scroll
		}
		if b, r, c, o, ok := e.cellAt(content); ok {
			if e.cell.on && e.cell.block == b && e.cell.row == r && e.cell.col == c && ev.Modifiers.Contain(key.ModShift) {
				e.cell.head = o
			} else {
				e.enterCell(b, r, c, o)
			}
			e.ptr.dragging = false
			e.ptr.cellDrag = true
			return
		}
		e.cell.on = false
		p := e.hit(content)
		// Links open with the shortcut key, or any click when read-only.
		if url := e.linkAt(p); url != "" && (e.readOnly || ev.Modifiers.Contain(key.ModShortcut)) {
			if e.onLinkClick != nil {
				e.onLinkClick(url)
			}
			return
		}
		now := gtx.Now
		if now.Sub(e.ptr.lastClick) < 400*time.Millisecond &&
			abs(pt.X-e.ptr.lastPos.X) < gtx.Dp(6) && abs(pt.Y-e.ptr.lastPos.Y) < gtx.Dp(6) {
			e.ptr.clicks++
		} else {
			e.ptr.clicks = 1
		}
		e.ptr.lastClick, e.ptr.lastPos = now, pt
		switch {
		case ev.Modifiers.Contain(key.ModShift):
			e.moveTo(p, true)
			e.ptr.unit = 0
		case e.ptr.clicks == 2:
			e.selectWord(p)
			e.ptr.unit = 1
		case e.ptr.clicks >= 3:
			e.st.Select(richdoc.Selection{Anchor: richdoc.Pos{Block: p.Block}, Head: richdoc.Pos{Block: p.Block, Offset: e.st.Doc.BlockLen(p.Block)}})
			e.ptr.unit = 2
		default:
			e.moveTo(p, false)
			e.ptr.unit = 0
		}
		e.ptr.anchor = e.st.Sel
		e.ptr.dragging = true
		e.desiredX = -1
	case pointer.Drag:
		if ev.PointerID != e.ptr.id {
			return
		}
		if ev.Position.Sub(e.ptr.start).X*ev.Position.Sub(e.ptr.start).X+ev.Position.Sub(e.ptr.start).Y*ev.Position.Sub(e.ptr.start).Y > float32(gtx.Dp(6)*gtx.Dp(6)) {
			e.ptr.moved = true
		}
		if e.ptr.cellDrag && e.cell.on {
			if b, r, c, o, ok := e.cellAt(content); ok && b == e.cell.block && r == e.cell.row && c == e.cell.col {
				e.cell.head = o
			}
			return
		}
		if !e.ptr.dragging {
			return
		}
		// Past the edges, scroll.
		if pt.Y < 0 {
			e.offY += float32(pt.Y) / 4
			kit.RequestFrame(gtx)
		} else if pt.Y > e.size.Y {
			e.offY += float32(pt.Y-e.size.Y) / 4
			kit.RequestFrame(gtx)
		}
		p := e.hit(content)
		a := e.ptr.anchor
		switch e.ptr.unit {
		case 1, 2:
			e.extendByUnit(a, p)
		default:
			e.st.Select(richdoc.Selection{Anchor: a.Anchor, Head: p})
		}
	case pointer.Release, pointer.Cancel:
		if e.ptr.touch && ev.Kind == pointer.Release && !e.ptr.moved {
			if b, r, c, o, ok := e.cellAt(content); ok {
				e.enterCell(b, r, c, o)
				gtx.Execute(key.SoftKeyboardCmd{Show: true})
				e.ptr.dragging = false
				return
			}
			e.cell.on = false
			p := e.hit(content)
			now := gtx.Now
			if url := e.linkAt(p); url != "" && e.readOnly && e.onLinkClick != nil {
				e.onLinkClick(url)
			} else if now.Sub(e.ptr.lastClick) < 400*time.Millisecond {
				e.selectWord(p)
			} else {
				e.moveTo(p, false)
				gtx.Execute(key.SoftKeyboardCmd{Show: true})
			}
			e.ptr.lastClick = now
		}
		e.ptr.dragging = false
		e.ptr.cellDrag = false
	}
}

// extendByUnit extends a word or block selection started at a to p.
func (e *RichEditor) extendByUnit(a richdoc.Selection, p richdoc.Pos) {
	start, end := a.Start(), a.End()
	save := e.st.Sel
	if e.ptr.unit == 1 {
		e.selectWord(p)
	} else {
		e.st.Select(richdoc.Selection{Anchor: richdoc.Pos{Block: p.Block}, Head: richdoc.Pos{Block: p.Block, Offset: e.st.Doc.BlockLen(p.Block)}})
	}
	w := e.st.Sel
	_ = save
	if p.Less(start) {
		e.st.Select(richdoc.Selection{Anchor: end, Head: w.Start()})
	} else {
		e.st.Select(richdoc.Selection{Anchor: start, Head: w.End()})
	}
}

// linkAt returns the link at p, if any.
func (e *RichEditor) linkAt(p richdoc.Pos) string {
	b := &e.st.Doc.Blocks[p.Block]
	if !b.Kind.HasText() {
		return ""
	}
	m := richdoc.MarksAt(b.Spans, p.Offset+1)
	if p.Offset >= b.Len() {
		return ""
	}
	return m.Link
}

var markdownish = regexp.MustCompile(`(?m)^(#{1,6} |[-*+] |\d+[.)] |> |\x60\x60\x60|- \[[ xX]\] |\|.*\|$)|\*\*[^*\n]+\*\*|\[[^\]\n]+\]\([^)\s]+\)`)

// paste inserts pasted text: as copied from an editor with its
// formatting, as Markdown when it looks like it, or as plain text.
func (e *RichEditor) paste(text string) {
	lastCopy.Lock()
	same := lastCopy.doc != nil && lastCopy.text == text
	frag := lastCopy.doc
	lastCopy.Unlock()
	inCode := e.st.Doc.Blocks[e.st.Sel.Head.Block].Kind == richdoc.Code
	switch {
	case same && !inCode:
		e.run(e.st.InsertDoc(frag))
	case e.pasteMD && !inCode && markdownish.MatchString(text):
		e.run(e.st.InsertDoc(richdoc.ParseMarkdown(text, nil)))
	default:
		e.run(e.st.InsertText(text))
	}
	e.hist.Break()
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// openMenuAtCaret opens the context menu below the caret, for keyboard
// users.
func (e *RichEditor) openMenuAtCaret() {
	m := e.ContextMenu()
	if m == nil {
		return
	}
	e.updateMenu()
	r, ok := e.cellCaretRect()
	if !ok {
		r, ok = e.caretRect(e.st.Sel.Head)
	}
	p := image.Pt(0, 0)
	if ok {
		p = image.Pt(r.Min.X, r.Max.Y-int(e.offY))
	}
	m.OpenAtWindow(e.origin.Add(p))
}
