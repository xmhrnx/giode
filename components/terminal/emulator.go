package terminal

import (
	"strings"
	"sync"
)

// MouseMode is which mouse events the program in the terminal asked
// for.
type MouseMode uint8

const (
	MouseOff MouseMode = iota
	// MouseClicks reports presses and releases (mode 1000).
	MouseClicks
	// MouseDrags also reports moves with a button held (mode 1002).
	MouseDrags
	// MouseMoves reports every move (mode 1003).
	MouseMoves
)

// CursorShape is the cursor's shape a program asked for (DECSCUSR).
type CursorShape uint8

const (
	CursorBlock CursorShape = iota
	CursorUnderline
	CursorBar
)

// Emulator is a VT100/xterm terminal emulator: it reads what a program
// writes - text and escape sequences - and keeps the screen it draws:
// cells with colors and attributes, the cursor, the scrollback, the
// alternate screen full-screen programs use, and the modes they set.
// It answers the program's queries (cursor position, device
// attributes) through the reply function. The Terminal element draws
// one; an app can also drive one on its own.
//
// Its methods are safe from any goroutine.
type Emulator struct {
	mu sync.Mutex

	cols, rows int
	lines      []Line // the active screen
	main, alt  []Line
	altActive  bool
	scrollback []Line
	maxBack    int

	cur struct {
		x, y   int
		style  Style
		wrap   bool // the next character goes on the next line
		origin bool
	}
	saved, savedAlt savedCursor
	top, bottom     int // the scroll region, inclusive

	autowrap, insert, newline, reverse bool
	appCursor, appKeypad, bracketed    bool
	cursorHidden                       bool
	cursorShape                        CursorShape
	cursorBlink                        bool
	mouse                              MouseMode
	mouseSGR, focusEvents              bool

	tabs     []bool
	charsets [2]byte // 'B' ASCII or '0' DEC line drawing, for G0 and G1
	gl       int

	title, cwd string
	lastRune   rune

	p parser

	reply   func([]byte)
	onTitle func(string)
	onBell  func()
	// changed counts changes, for the view to know when to repaint.
	changed uint64
}

type savedCursor struct {
	x, y     int
	style    Style
	origin   bool
	charsets [2]byte
	gl       int
	valid    bool
}

// NewEmulator returns an emulator of cols by rows cells, keeping
// scrollback lines of history (0 for none).
func NewEmulator(cols, rows, scrollback int) *Emulator {
	e := &Emulator{maxBack: scrollback}
	e.reset(max(cols, 1), max(rows, 1))
	return e
}

func (e *Emulator) reset(cols, rows int) {
	e.cols, e.rows = cols, rows
	e.main = make([]Line, rows)
	e.alt = make([]Line, rows)
	for i := range e.main {
		e.main[i] = blankLine(cols, Style{})
		e.alt[i] = blankLine(cols, Style{})
	}
	e.lines, e.altActive = e.main, false
	e.cur.x, e.cur.y, e.cur.style, e.cur.wrap, e.cur.origin = 0, 0, Style{}, false, false
	e.saved, e.savedAlt = savedCursor{}, savedCursor{}
	e.top, e.bottom = 0, rows-1
	e.autowrap, e.insert, e.newline, e.reverse = true, false, false, false
	e.appCursor, e.appKeypad, e.bracketed = false, false, false
	e.cursorHidden, e.cursorShape, e.cursorBlink = false, CursorBlock, false
	e.mouse, e.mouseSGR, e.focusEvents = MouseOff, false, false
	e.charsets, e.gl = [2]byte{'B', 'B'}, 0
	e.resetTabs()
	e.changed++
}

func (e *Emulator) resetTabs() {
	e.tabs = make([]bool, e.cols)
	for i := 8; i < e.cols; i += 8 {
		e.tabs[i] = true
	}
}

// SetReply sets fn to receive the emulator's answers to the program's
// queries, to write back to it.
func (e *Emulator) SetReply(fn func([]byte)) {
	e.mu.Lock()
	e.reply = fn
	e.mu.Unlock()
}

// OnTitle has fn run when the program sets the window title. It runs
// on the goroutine calling Write.
func (e *Emulator) OnTitle(fn func(title string)) {
	e.mu.Lock()
	e.onTitle = fn
	e.mu.Unlock()
}

// OnBell has fn run on a bell character.
func (e *Emulator) OnBell(fn func()) {
	e.mu.Lock()
	e.onBell = fn
	e.mu.Unlock()
}

// Write processes output of the program: text and escape sequences.
// It never fails.
func (e *Emulator) Write(p []byte) (int, error) {
	e.mu.Lock()
	var replies [][]byte
	reply := e.reply
	e.reply = func(b []byte) { replies = append(replies, b) }
	var titles []string
	onTitle := e.onTitle
	e.onTitle = func(t string) { titles = append(titles, t) }
	bells := 0
	onBell := e.onBell
	e.onBell = func() { bells++ }
	for _, b := range p {
		e.p.feed(e, b)
	}
	e.reply, e.onTitle, e.onBell = reply, onTitle, onBell
	e.changed++
	e.mu.Unlock()
	// Callbacks run unlocked: they may read the emulator.
	if reply != nil {
		for _, r := range replies {
			reply(r)
		}
	}
	if onTitle != nil {
		for _, t := range titles {
			onTitle(t)
		}
	}
	if onBell != nil {
		for range bells {
			onBell()
		}
	}
	return len(p), nil
}

// Size returns its size in cells.
func (e *Emulator) Size() (cols, rows int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.cols, e.rows
}

// Title returns the title the program set last.
func (e *Emulator) Title() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.title
}

// WorkingDir returns the directory the shell reported it's in (OSC 7),
// as the URL it sent ("file://host/path"); "" if it didn't.
func (e *Emulator) WorkingDir() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.cwd
}

// Snapshot is the state the view draws, copied under the lock.
type Snapshot struct {
	// Lines are the visible lines, top to bottom.
	Lines []Line
	// CursorX and CursorY are the cursor's cell; CursorVisible whether
	// to draw it (false while scrolled back).
	CursorX, CursorY int
	CursorVisible    bool
	CursorShape      CursorShape
	CursorBlink      bool
	// Reverse is set when the program swapped the default colors.
	Reverse bool
	// Scrollback is how many lines of history there are.
	Scrollback int
	// Top is the history line (see Pos) of the first visible line.
	Top int
}

// Snapshot returns the screen, scrolled back by offset lines of
// history (0 shows the live screen).
func (e *Emulator) Snapshot(offset int) Snapshot {
	e.mu.Lock()
	defer e.mu.Unlock()
	back := e.scrollback
	if e.altActive {
		back = nil
	}
	offset = max(0, min(offset, len(back)))
	s := Snapshot{
		Lines: make([]Line, 0, e.rows), CursorX: e.cur.x, CursorY: e.cur.y + offset,
		CursorVisible: !e.cursorHidden && offset < e.rows, CursorShape: e.cursorShape,
		CursorBlink: e.cursorBlink, Reverse: e.reverse, Scrollback: len(back),
		Top: len(back) - offset,
	}
	for i := len(back) - offset; i < len(back) && len(s.Lines) < e.rows; i++ {
		s.Lines = append(s.Lines, copyLine(back[i], e.cols))
	}
	for i := 0; len(s.Lines) < e.rows; i++ {
		s.Lines = append(s.Lines, copyLine(e.lines[i], e.cols))
	}
	return s
}

func copyLine(l Line, cols int) Line {
	c := Line{Cells: make([]Cell, cols), Wrapped: l.Wrapped}
	n := copy(c.Cells, l.Cells)
	fillBlank(c.Cells[n:], Style{})
	return c
}

// Modes are the input modes the program set, which decide how keys and
// the mouse are sent to it.
type Modes struct {
	AppCursor, AppKeypad, BracketedPaste bool
	Mouse                                MouseMode
	MouseSGR, FocusEvents                bool
	AltScreen                            bool
}

// Modes returns the input modes in effect.
func (e *Emulator) Modes() Modes {
	e.mu.Lock()
	defer e.mu.Unlock()
	return Modes{
		AppCursor: e.appCursor, AppKeypad: e.appKeypad, BracketedPaste: e.bracketed,
		Mouse: e.mouse, MouseSGR: e.mouseSGR, FocusEvents: e.focusEvents, AltScreen: e.altActive,
	}
}

// Changed returns a counter that increments whenever the screen may
// have changed.
func (e *Emulator) Changed() uint64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.changed
}

// ClearScrollback drops the history.
func (e *Emulator) ClearScrollback() {
	e.mu.Lock()
	e.scrollback = nil
	e.changed++
	e.mu.Unlock()
}

// Resize changes its size. Lines are cut or padded, not rewrapped;
// shrinking moves the top lines into the scrollback to keep the cursor
// on screen, and growing brings them back.
func (e *Emulator) Resize(cols, rows int) {
	cols, rows = max(cols, 1), max(rows, 1)
	e.mu.Lock()
	defer e.mu.Unlock()
	if cols == e.cols && rows == e.rows {
		return
	}
	resizeLines := func(ls []Line) []Line {
		for i := range ls {
			ls[i] = fitLine(ls[i], cols)
		}
		return ls
	}
	// Rows: the main screen trades lines with the scrollback, keeping
	// its cursor on screen.
	main := resizeLines(e.main)
	cy := e.cur.y
	if e.altActive {
		cy = e.saved.y
	}
	for len(main) > rows {
		if last := len(main) - 1; last > cy && isBlank(main[last]) {
			main = main[:last]
			continue
		}
		e.pushBack(main[0])
		main = main[1:]
		cy--
	}
	for len(main) < rows {
		if n := len(e.scrollback); n > 0 {
			main = append([]Line{fitLine(e.scrollback[n-1], cols)}, main...)
			e.scrollback = e.scrollback[:n-1]
			cy++
			continue
		}
		main = append(main, blankLine(cols, Style{}))
	}
	if e.altActive {
		e.saved.y = cy
	} else {
		e.cur.y = cy
	}
	alt := resizeLines(e.alt)
	if len(alt) > rows {
		alt = alt[:rows]
	}
	for len(alt) < rows {
		alt = append(alt, blankLine(cols, Style{}))
	}
	e.main, e.alt = main, alt
	if e.altActive {
		e.lines = e.alt
	} else {
		e.lines = e.main
	}
	e.cols, e.rows = cols, rows
	e.top, e.bottom = 0, rows-1
	e.cur.x = min(max(e.cur.x, 0), cols-1)
	e.cur.y = min(max(e.cur.y, 0), rows-1)
	e.saved.x, e.saved.y = min(e.saved.x, cols-1), min(max(e.saved.y, 0), rows-1)
	e.cur.wrap = false
	e.resetTabs()
	e.changed++
}

func isBlank(l Line) bool {
	for _, c := range l.Cells {
		if c.Rune != 0 || c.Style.BG != DefaultColor {
			return false
		}
	}
	return true
}

func fitLine(l Line, cols int) Line {
	if len(l.Cells) == cols {
		return l
	}
	if len(l.Cells) > cols {
		l.Cells = l.Cells[:cols:cols]
		if c := &l.Cells[cols-1]; c.Width == 2 {
			// Half a wide character doesn't fit.
			*c = Cell{Width: 1, Style: Style{BG: c.Style.BG}}
		}
		l.Wrapped = false
		return l
	}
	cells := make([]Cell, cols)
	n := copy(cells, l.Cells)
	fillBlank(cells[n:], Style{})
	l.Cells = cells
	return l
}

func (e *Emulator) pushBack(l Line) {
	if e.maxBack <= 0 {
		return
	}
	if len(e.scrollback) >= e.maxBack {
		// Drop the oldest in chunks, so a long run of output doesn't
		// copy the history line by line.
		drop := max(1, e.maxBack/8)
		e.scrollback = append(e.scrollback[:0], e.scrollback[drop:]...)
	}
	e.scrollback = append(e.scrollback, l)
}

// --- Screen operations (called with mu held) -------------------------------

func (e *Emulator) blank() Line { return blankLine(e.cols, e.cur.style) }

// put writes one printable character at the cursor.
func (e *Emulator) put(r rune) {
	if e.charsets[e.gl] == '0' && r >= 0x5f && r <= 0x7e {
		r = decGraphics[r-0x5f]
	}
	w := runeWidth(r)
	if w == 0 {
		e.combine(r)
		return
	}
	if e.cur.wrap && e.autowrap {
		e.lines[e.cur.y].Wrapped = true
		e.cur.x = 0
		e.index()
	}
	e.cur.wrap = false
	if w == 2 && e.cur.x == e.cols-1 {
		if !e.autowrap || e.cols < 2 {
			return
		}
		// A wide character doesn't fit in the last column: it goes on
		// the next line.
		e.lines[e.cur.y].Cells[e.cur.x] = Cell{Width: 1, Style: Style{BG: e.cur.style.BG}}
		e.lines[e.cur.y].Wrapped = true
		e.cur.x = 0
		e.index()
	}
	row := e.lines[e.cur.y].Cells
	if e.insert {
		copy(row[e.cur.x+w:], row[e.cur.x:])
	}
	e.clearWide(row, e.cur.x)
	if w == 2 {
		e.clearWide(row, e.cur.x+1)
	}
	row[e.cur.x] = Cell{Rune: r, Width: uint8(w), Style: e.cur.style}
	if w == 2 {
		row[e.cur.x+1] = Cell{Width: 0, Style: e.cur.style}
	}
	e.lastRune = r
	e.cur.x += w
	if e.cur.x >= e.cols {
		e.cur.x = e.cols - 1
		e.cur.wrap = true
	}
}

// clearWide blanks the other half of a wide character that cell x is
// part of, before x is overwritten.
func (e *Emulator) clearWide(row []Cell, x int) {
	if x < 0 || x >= len(row) {
		return
	}
	switch row[x].Width {
	case 0:
		if x > 0 && row[x-1].Width == 2 {
			row[x-1] = Cell{Width: 1, Style: Style{BG: row[x-1].Style.BG}}
		}
	case 2:
		if x+1 < len(row) {
			row[x+1] = Cell{Width: 1, Style: Style{BG: row[x].Style.BG}}
		}
	}
}

// combine joins a combining mark to the character before the cursor.
func (e *Emulator) combine(mark rune) {
	x, y := e.cur.x-1, e.cur.y
	if e.cur.wrap {
		x = e.cur.x
	}
	if x < 0 {
		return
	}
	row := e.lines[y].Cells
	if row[x].Width == 0 && x > 0 {
		x--
	}
	if r, ok := compose(row[x].Rune, mark); ok && row[x].Rune != 0 {
		row[x].Rune = r
	}
}

// index moves the cursor down, scrolling at the bottom of the region.
func (e *Emulator) index() {
	if e.cur.y == e.bottom {
		e.scrollUp(1)
	} else if e.cur.y < e.rows-1 {
		e.cur.y++
	}
}

// reverseIndex moves the cursor up, scrolling at the top of the region.
func (e *Emulator) reverseIndex() {
	if e.cur.y == e.top {
		e.scrollDown(1)
	} else if e.cur.y > 0 {
		e.cur.y--
	}
}

// scrollUp moves the region's lines up by n, blank lines entering at
// the bottom. Lines leaving the top of the main screen go to the
// scrollback.
func (e *Emulator) scrollUp(n int) {
	n = min(n, e.bottom-e.top+1)
	for range n {
		if e.top == 0 && !e.altActive {
			e.pushBack(e.lines[0])
		}
		copy(e.lines[e.top:e.bottom], e.lines[e.top+1:e.bottom+1])
		e.lines[e.bottom] = e.blank()
	}
}

// scrollDown moves the region's lines down by n.
func (e *Emulator) scrollDown(n int) {
	n = min(n, e.bottom-e.top+1)
	for range n {
		copy(e.lines[e.top+1:e.bottom+1], e.lines[e.top:e.bottom])
		e.lines[e.top] = e.blank()
	}
}

// moveTo places the cursor, relative to the scroll region in origin
// mode.
func (e *Emulator) moveTo(x, y int) {
	minY, maxY := 0, e.rows-1
	if e.cur.origin {
		y += e.top
		minY, maxY = e.top, e.bottom
	}
	e.cur.x = min(max(x, 0), e.cols-1)
	e.cur.y = min(max(y, minY), maxY)
	e.cur.wrap = false
}

func (e *Emulator) eraseInLine(mode int) {
	row := e.lines[e.cur.y].Cells
	switch mode {
	case 0:
		e.clearWide(row, e.cur.x)
		fillBlank(row[e.cur.x:], e.cur.style)
		e.lines[e.cur.y].Wrapped = false
	case 1:
		e.clearWide(row, e.cur.x)
		fillBlank(row[:e.cur.x+1], e.cur.style)
	case 2:
		fillBlank(row, e.cur.style)
		e.lines[e.cur.y].Wrapped = false
	}
	e.cur.wrap = false
}

func (e *Emulator) eraseInDisplay(mode int) {
	switch mode {
	case 0:
		e.eraseInLine(0)
		for y := e.cur.y + 1; y < e.rows; y++ {
			e.lines[y] = e.blank()
		}
	case 1:
		e.eraseInLine(1)
		for y := 0; y < e.cur.y; y++ {
			e.lines[y] = e.blank()
		}
	case 2:
		for y := range e.lines {
			e.lines[y] = e.blank()
		}
	case 3:
		e.scrollback = nil
	}
}

func (e *Emulator) insertLines(n int) {
	if e.cur.y < e.top || e.cur.y > e.bottom {
		return
	}
	top := e.top
	e.top = e.cur.y
	e.scrollDown(n)
	e.top = top
	e.cur.x, e.cur.wrap = 0, false
}

func (e *Emulator) deleteLines(n int) {
	if e.cur.y < e.top || e.cur.y > e.bottom {
		return
	}
	top := e.top
	e.top = e.cur.y
	n = min(n, e.bottom-e.top+1)
	for range n {
		copy(e.lines[e.top:e.bottom], e.lines[e.top+1:e.bottom+1])
		e.lines[e.bottom] = e.blank()
	}
	e.top = top
	e.cur.x, e.cur.wrap = 0, false
}

func (e *Emulator) insertChars(n int) {
	row := e.lines[e.cur.y].Cells
	n = min(n, e.cols-e.cur.x)
	e.clearWide(row, e.cur.x)
	copy(row[e.cur.x+n:], row[e.cur.x:])
	fillBlank(row[e.cur.x:e.cur.x+n], e.cur.style)
	e.cur.wrap = false
}

func (e *Emulator) deleteChars(n int) {
	row := e.lines[e.cur.y].Cells
	n = min(n, e.cols-e.cur.x)
	e.clearWide(row, e.cur.x)
	e.clearWide(row, e.cur.x+n-1)
	copy(row[e.cur.x:], row[e.cur.x+n:])
	fillBlank(row[e.cols-n:], e.cur.style)
	e.cur.wrap = false
}

func (e *Emulator) eraseChars(n int) {
	row := e.lines[e.cur.y].Cells
	n = min(n, e.cols-e.cur.x)
	e.clearWide(row, e.cur.x)
	e.clearWide(row, e.cur.x+n-1)
	fillBlank(row[e.cur.x:e.cur.x+n], e.cur.style)
	e.cur.wrap = false
}

func (e *Emulator) tab(n int) {
	for ; n > 0 && e.cur.x < e.cols-1; n-- {
		e.cur.x++
		for e.cur.x < e.cols-1 && !e.tabs[e.cur.x] {
			e.cur.x++
		}
	}
	e.cur.wrap = false
}

func (e *Emulator) backTab(n int) {
	for ; n > 0 && e.cur.x > 0; n-- {
		e.cur.x--
		for e.cur.x > 0 && !e.tabs[e.cur.x] {
			e.cur.x--
		}
	}
	e.cur.wrap = false
}

func (e *Emulator) saveCursor() {
	s := savedCursor{x: e.cur.x, y: e.cur.y, style: e.cur.style, origin: e.cur.origin, charsets: e.charsets, gl: e.gl, valid: true}
	if e.altActive {
		e.savedAlt = s
	} else {
		e.saved = s
	}
}

func (e *Emulator) restoreCursor() {
	s := e.saved
	if e.altActive {
		s = e.savedAlt
	}
	if !s.valid {
		s = savedCursor{charsets: [2]byte{'B', 'B'}}
	}
	e.cur.x, e.cur.y = min(s.x, e.cols-1), min(s.y, e.rows-1)
	e.cur.style, e.cur.origin, e.charsets, e.gl = s.style, s.origin, s.charsets, s.gl
	e.cur.wrap = false
}

// setAltScreen switches to the alternate screen, cleared, or back.
func (e *Emulator) setAltScreen(on bool) {
	if on == e.altActive {
		return
	}
	e.altActive = on
	if on {
		for i := range e.alt {
			e.alt[i] = blankLine(e.cols, Style{})
		}
		e.lines = e.alt
	} else {
		e.lines = e.main
	}
	e.top, e.bottom = 0, e.rows-1
	e.cur.wrap = false
}

func (e *Emulator) setTitle(t string) {
	e.title = t
	if e.onTitle != nil {
		e.onTitle(t)
	}
}

func (e *Emulator) send(s string) {
	if e.reply != nil {
		e.reply([]byte(s))
	}
}

// decGraphics is the DEC Special Graphics character set, for 0x5f-0x7e:
// the line drawing characters of full-screen programs.
var decGraphics = [...]rune{
	' ', '◆', '▒', '␉', '␌', '␍', '␊', '°', '±', '␤', '␋', '┘', '┐', '┌', '└', '┼',
	'⎺', '⎻', '─', '⎼', '⎽', '├', '┤', '┴', '┬', '│', '≤', '≥', 'π', '≠', '£', '·',
}

// Pos is a cell in the history: Line counts from the oldest scrollback
// line (Snapshot.Top is the first visible line), Col from the left.
type Pos struct{ Line, Col int }

// Before reports whether p comes before q.
func (p Pos) Before(q Pos) bool {
	return p.Line < q.Line || p.Line == q.Line && p.Col < q.Col
}

// line returns history line i, scrollback then screen.
func (e *Emulator) line(i int) (Line, bool) {
	back := e.scrollback
	if e.altActive {
		back = nil
	}
	switch {
	case i < 0:
		return Line{}, false
	case i < len(back):
		return back[i], true
	case i-len(back) < len(e.lines):
		return e.lines[i-len(back)], true
	}
	return Line{}, false
}

// Text returns the text from cell from up to, not including, cell to,
// in history positions. Lines that wrapped join without a line break;
// trailing blanks of each line, and blank lines at the end, are left
// out.
func (e *Emulator) Text(from, to Pos) string {
	if to.Before(from) {
		from, to = to, from
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	var b []rune
	for i := from.Line; i <= to.Line; i++ {
		l, ok := e.line(i)
		if !ok {
			break
		}
		start, end := 0, len(l.Cells)
		if i == from.Line {
			start = min(max(from.Col, 0), end)
		}
		if i == to.Line {
			end = min(max(to.Col, 0), end)
		}
		seg := Line{Cells: l.Cells[start:end]}
		b = append(b, []rune(seg.Text())...)
		if i < to.Line && !l.Wrapped {
			b = append(b, '\n')
		}
	}
	// Blank lines at the end (the empty screen under a prompt) aren't
	// text.
	return strings.TrimRight(string(b), "\n")
}

// WordAt returns the word around p - a run of letters, digits and path
// characters - as the positions of its first cell and past its last.
func (e *Emulator) WordAt(p Pos) (from, to Pos) {
	e.mu.Lock()
	defer e.mu.Unlock()
	l, ok := e.line(p.Line)
	if !ok || p.Col < 0 || p.Col >= len(l.Cells) {
		return p, p
	}
	word := func(c Cell) bool {
		r := c.Rune
		return c.Width == 0 || r > 0x7f || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r != 0 && strings.ContainsRune("_-./~:@%+#", r)
	}
	if !word(l.Cells[p.Col]) {
		return p, Pos{p.Line, p.Col + 1}
	}
	s, t := p.Col, p.Col
	for s > 0 && word(l.Cells[s-1]) {
		s--
	}
	for t < len(l.Cells) && word(l.Cells[t]) {
		t++
	}
	return Pos{p.Line, s}, Pos{p.Line, t}
}

// History returns how many lines there are: the scrollback and the
// screen.
func (e *Emulator) History() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.altActive {
		return e.rows
	}
	return len(e.scrollback) + e.rows
}
