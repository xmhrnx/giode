// Package richedit is Giode's rich text editor: paragraphs, headings,
// lists, checklists, quotes, code, tables, images and rules, with inline
// formatting, mentions and other atoms, Markdown shortcuts, slash
// commands, undo, and Markdown, JSON and HTML in and out. Its document
// is a richdoc.Doc; its edits stream as richdoc operations for syncing.
package richedit

import (
	"image"
	"strings"
	"time"

	"gioui.org/gesture"
	"gioui.org/io/key"
	"gioui.org/op/paint"

	"github.com/xmhrnx/giode/components/menu"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/richdoc"
	"github.com/xmhrnx/giode/styles"
)

// SubmitMode is what Enter does.
type SubmitMode int

const (
	// SubmitNone: Enter starts a new paragraph.
	SubmitNone SubmitMode = iota
	// SubmitEnter: Enter submits and Shift+Enter starts a new paragraph,
	// as chat composers do.
	SubmitEnter
	// SubmitShortcut: Ctrl+Enter (Cmd+Enter on Apple) submits.
	SubmitShortcut
)

// Suggestion is an entry of a suggestion list, shown while text follows
// a trigger character ("@ada", "#45", "/head", ":smi").
type Suggestion struct {
	// Label is shown; Detail beside it, dimmed.
	Label, Detail string
	// Picking it inserts Atom, or Text, in place of the trigger and
	// query; or runs Command after removing them.
	Atom    *richdoc.Atom
	Text    string
	Command func(e *RichEditor)
}

// Suggester returns the suggestions for what's been typed after its
// trigger. It runs on the UI goroutine every time the query changes:
// filter a local list, or return what a search started earlier found.
type Suggester func(query string) []Suggestion

// RichEditor is a stateful rich text editor. Create it once with New
// and lay it out every frame.
type RichEditor struct {
	elements.Contextual

	st          *richdoc.State
	hist        richdoc.History
	style       styles.Styles
	placeholder string
	readOnly    bool
	submitOn    SubmitMode
	minHeight   int // dp
	pasteMD     bool

	onChange    func()
	onOps       func(tx *richdoc.Tx)
	onSubmit    func()
	onLinkClick func(url string)
	onLinkEdit  func(current string, set func(url string))
	loadImage   func(src string) image.Image
	suggesters  map[rune]Suggester
	slash       bool

	// The view.
	views    map[uint64]*blockView
	frame    uint64
	tops     []int // each block's top, from the content's top
	hs       []int // the blocks' last known heights
	keys     []uint64
	prevKeys []uint64
	imageOps map[string]paint.ImageOp
	markers  []string
	// laid are the blocks laid out in the last frame, by index; nil
	// for those out of view.
	laid   []*blockView
	width  int
	offY   float32
	scroll gesture.Scroll
	size   image.Point
	origin image.Point
	images map[string]image.Image

	focused, wantFocus, wantBlur bool
	blink                        time.Time
	// desiredX keeps the caret's column moving up and down; -1 when
	// unset.
	desiredX    int
	reveal      bool
	ime         imeState
	ptr         pointerState
	copyNext    string
	sugg        suggestState
	taskBoxes   []taskBox
	pendingRead bool
	changed     bool
	cell        cellCaret
	menu        *menu.Menu
	menuBuilt   bool
	menuFor     menuKey
	tools       []*toolButton
}

// New returns an empty editor with a placeholder. The styles set its
// text color, font size and box.
func New(placeholder string, st ...styles.Styles) *RichEditor {
	e := &RichEditor{
		st:          richdoc.NewState(nil),
		placeholder: placeholder,
		views:       map[uint64]*blockView{},
		images:      map[string]image.Image{},
		imageOps:    map[string]paint.ImageOp{},
		desiredX:    -1,
		slash:       true,
		pasteMD:     true,
		suggesters:  map[rune]Suggester{},
	}
	if len(st) > 0 {
		e.style = st[0]
	}
	e.ime.block = -1
	return e
}

// State is the editing state: the document, selection and commands.
// Edits made through it directly aren't recorded for undo or streamed;
// use the editor's methods.
func (e *RichEditor) State() *richdoc.State { return e.st }

// Doc is the document being edited. Don't change it directly.
func (e *RichEditor) Doc() *richdoc.Doc { return e.st.Doc }

// SetDoc replaces the document and clears the undo history.
func (e *RichEditor) SetDoc(d *richdoc.Doc) *RichEditor {
	if d == nil || len(d.Blocks) == 0 {
		d = richdoc.New()
	}
	e.st.Doc = d
	e.st.Select(richdoc.Caret(d.End()))
	e.reveal = true
	e.hist.Clear()
	e.ime.block = -1
	return e
}

// SetMarkdown replaces the document with Markdown's.
func (e *RichEditor) SetMarkdown(md string) *RichEditor {
	return e.SetDoc(richdoc.ParseMarkdown(md, nil))
}

// SetText replaces the document with plain text, a paragraph a line.
func (e *RichEditor) SetText(s string) *RichEditor { return e.SetDoc(richdoc.FromText(s)) }

// Markdown returns the document as Markdown.
func (e *RichEditor) Markdown() string { return e.st.Doc.Markdown() }

// Text returns the document's plain text.
func (e *RichEditor) Text() string { return e.st.Doc.Text() }

// HTML returns the document as HTML.
func (e *RichEditor) HTML() string { return e.st.Doc.HTML() }

// Empty reports whether the document has no text, tables or images.
func (e *RichEditor) Empty() bool {
	d := e.st.Doc
	return len(d.Blocks) == 1 && d.Blocks[0].Kind.HasText() && d.Blocks[0].Len() == 0
}

// Clear empties the document, as sending a chat message does; it can be
// undone.
func (e *RichEditor) Clear() *RichEditor {
	e.run(e.st.SetDoc(richdoc.New()))
	return e
}

// OnChange sets the handler called after every edit.
func (e *RichEditor) OnChange(fn func()) *RichEditor { e.onChange = fn; return e }

// OnOps sets the handler called with every transaction applied - typed,
// pasted, undone - for syncing the document with collaborators.
// Transactions marshal to JSON.
func (e *RichEditor) OnOps(fn func(tx *richdoc.Tx)) *RichEditor { e.onOps = fn; return e }

// Apply applies a collaborator's operations, keeping the caret and
// selection in place around them. It clears the undo history, whose
// steps no longer fit.
func (e *RichEditor) Apply(ops []richdoc.Op) error {
	err := e.st.Apply(ops)
	e.hist.Clear()
	e.ime.block = -1
	e.changed = true
	return err
}

// SubmitOn sets what Enter does; OnSubmit what submitting does.
func (e *RichEditor) SubmitOn(m SubmitMode) *RichEditor { e.submitOn = m; return e }

// OnSubmit sets the handler of Enter or Ctrl+Enter (SubmitOn).
func (e *RichEditor) OnSubmit(fn func()) *RichEditor { e.onSubmit = fn; return e }

// ReadOnly shows the document without editing it; links open on a
// click.
func (e *RichEditor) ReadOnly(on bool) *RichEditor { e.readOnly = on; return e }

// Placeholder sets the text shown while the document is empty.
func (e *RichEditor) Placeholder(s string) *RichEditor { e.placeholder = s; return e }

// MinHeight sets the editor's least height; it grows with its content
// up to the height it's given, then scrolls.
func (e *RichEditor) MinHeight(dp int) *RichEditor { e.minHeight = dp; return e }

// MarkdownShortcuts turns the Markdown typed at the start of paragraphs
// and around text into formatting ("- ", "# ", "**bold**"); on by
// default.
func (e *RichEditor) MarkdownShortcuts(on bool) *RichEditor { e.st.Rules = on; return e }

// PasteMarkdown reads pasted text that looks like Markdown as Markdown;
// on by default.
func (e *RichEditor) PasteMarkdown(on bool) *RichEditor { e.pasteMD = on; return e }

// SlashCommands offers block types, a table, a divider and the like when
// "/" is typed at the start of a word; on by default.
func (e *RichEditor) SlashCommands(on bool) *RichEditor { e.slash = on; return e }

// Suggest shows the suggestions fn returns while text follows trigger:
// '@' for mentions, '#' for tickets or deals, ':' for emoji.
func (e *RichEditor) Suggest(trigger rune, fn Suggester) *RichEditor {
	if fn == nil {
		delete(e.suggesters, trigger)
	} else {
		e.suggesters[trigger] = fn
	}
	return e
}

// LoadImage sets how image blocks get their pictures, by URL; without
// it they show as placeholders.
func (e *RichEditor) LoadImage(fn func(src string) image.Image) *RichEditor {
	e.loadImage = fn
	clear(e.images)
	clear(e.imageOps)
	return e
}

// OnLinkClick sets what opening a link does: Ctrl or Cmd and click, or
// any click when read-only. Unset, nothing opens.
func (e *RichEditor) OnLinkClick(fn func(url string)) *RichEditor { e.onLinkClick = fn; return e }

// OnLinkEdit sets what Ctrl+K (Cmd+K) does: ask for a URL, starting
// from the selection's link, and call set with it ("" unlinks).
func (e *RichEditor) OnLinkEdit(fn func(current string, set func(url string))) *RichEditor {
	e.onLinkEdit = fn
	return e
}

// Focus gives the editor keyboard focus at its next layout.
func (e *RichEditor) Focus() { e.wantFocus = true }

// Blur takes keyboard focus from it.
func (e *RichEditor) Blur() { e.wantBlur = true }

// Focused reports whether it has keyboard focus.
func (e *RichEditor) Focused() bool { return e.focused }

// Selection returns the selection.
func (e *RichEditor) Selection() richdoc.Selection { return e.st.Sel }

// Select sets the selection.
func (e *RichEditor) Select(s richdoc.Selection) {
	e.st.Select(s)
	e.desiredX = -1
	e.reveal = true
}

// run records and reports a command's transaction.
func (e *RichEditor) run(tx *richdoc.Tx) {
	e.blink = time.Now()
	e.reveal = true
	if tx == nil {
		return
	}
	e.desiredX = -1
	if len(tx.Ops) == 0 {
		return
	}
	e.hist.Record(tx)
	e.changed = true
	if e.onOps != nil {
		e.onOps(tx)
	}
	if e.onChange != nil {
		e.onChange()
	}
}

// Commands for toolbars and menus. Each applies to the selection, is
// undoable, and does nothing while read-only.

// ToggleMark turns bold, italic, underline, strike or code on or off.
func (e *RichEditor) ToggleMark(k richdoc.MarkKind) {
	e.do(func() *richdoc.Tx { return e.st.ToggleMark(k) })
}

// HasMark reports whether the selection (or what's typed next) has it.
func (e *RichEditor) HasMark(k richdoc.MarkKind) bool { return e.st.HasMark(k) }

// SetKind makes the selected blocks headings (with level), list items,
// quotes or code; or paragraphs when they already are.
func (e *RichEditor) SetKind(k richdoc.Kind, level int) {
	e.do(func() *richdoc.Tx { return e.st.SetKind(k, level) })
}

// Kind returns the kind and level of the block at the caret.
func (e *RichEditor) Kind() (richdoc.Kind, int) {
	b := &e.st.Doc.Blocks[e.st.Sel.Head.Block]
	return b.Kind, b.Level
}

// Indent nests the selected list items deeper, or shallower.
func (e *RichEditor) Indent(out bool) { e.do(func() *richdoc.Tx { return e.st.Indent(out) }) }

// SetLink links the selection; "" unlinks it.
func (e *RichEditor) SetLink(url string) {
	e.do(func() *richdoc.Tx { return e.st.SetLink(richdoc.SafeURL(url)) })
}

// SetColor colors the selection's text, or highlights it; "" clears.
func (e *RichEditor) SetColor(color string, highlight bool) {
	e.do(func() *richdoc.Tx { return e.st.SetColor(color, highlight) })
}

// ClearFormatting removes the selection's marks.
func (e *RichEditor) ClearFormatting() { e.do(e.st.ClearMarks) }

// InsertAtom inserts a mention, reference or other atom at the caret.
func (e *RichEditor) InsertAtom(a richdoc.Atom) {
	e.do(func() *richdoc.Tx { return e.st.InsertAtom(a, nil, " ") })
}

// InsertText inserts text at the caret, replacing the selection.
func (e *RichEditor) InsertText(s string) { e.do(func() *richdoc.Tx { return e.st.InsertText(s) }) }

// InsertMarkdown inserts Markdown's content at the caret.
func (e *RichEditor) InsertMarkdown(md string) {
	e.do(func() *richdoc.Tx { return e.st.InsertDoc(richdoc.ParseMarkdown(md, nil)) })
}

// InsertHTML inserts HTML's content, sanitized, at the caret.
func (e *RichEditor) InsertHTML(html string) {
	e.do(func() *richdoc.Tx { return e.st.InsertDoc(richdoc.ParseHTML(html)) })
}

// InsertImage inserts an image block.
func (e *RichEditor) InsertImage(src, alt string) {
	e.do(func() *richdoc.Tx { return e.st.InsertBlock(richdoc.Block{Kind: richdoc.Image, Src: src, Alt: alt}) })
}

// InsertTable inserts a table of rows by cols empty cells, the first row
// its header.
func (e *RichEditor) InsertTable(rows, cols int) {
	rows, cols = max(rows, 1), max(cols, 1)
	cells := make([][][]richdoc.Span, rows)
	for i := range cells {
		cells[i] = make([][]richdoc.Span, cols)
	}
	e.do(func() *richdoc.Tx { return e.st.InsertBlock(richdoc.Block{Kind: richdoc.Table, Cells: cells}) })
}

// InsertRule inserts a horizontal rule.
func (e *RichEditor) InsertRule() {
	e.do(func() *richdoc.Tx { return e.st.InsertBlock(richdoc.Block{Kind: richdoc.Rule}) })
}

// Undo and Redo step through the history.
func (e *RichEditor) Undo() {
	if !e.readOnly {
		e.applied(e.hist.Undo(e.st))
	}
}

func (e *RichEditor) Redo() {
	if !e.readOnly {
		e.applied(e.hist.Redo(e.st))
	}
}

// applied reports a transaction history applied, without recording it.
func (e *RichEditor) applied(tx *richdoc.Tx) {
	e.blink = time.Now()
	e.reveal = true
	e.desiredX = -1
	if tx == nil || len(tx.Ops) == 0 {
		return
	}
	e.changed = true
	if e.onOps != nil {
		e.onOps(tx)
	}
	if e.onChange != nil {
		e.onChange()
	}
}

// CanUndo and CanRedo report whether there's a step to undo or redo.
func (e *RichEditor) CanUndo() bool { return e.hist.CanUndo() }
func (e *RichEditor) CanRedo() bool { return e.hist.CanRedo() }

// SelectAll selects the whole document.
func (e *RichEditor) SelectAll() { e.st.SelectAll() }

func (e *RichEditor) do(cmd func() *richdoc.Tx) {
	if e.readOnly {
		return
	}
	e.run(cmd())
	e.wantFocus = true
}

// selectedText returns the selection as plain text.
func (e *RichEditor) selectedText() string {
	if e.st.Sel.Empty() {
		return ""
	}
	return strings.TrimSuffix(e.st.Fragment().Text(), "\n")
}

// imeState is what the input method was last told.
type imeState struct {
	block     int
	snippet   key.Snippet
	sel       key.Range
	caret     key.Caret
	composing key.Range
}
