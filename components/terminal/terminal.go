// Package terminal is a terminal: a VT100/xterm emulator (Emulator), the
// key and mouse encodings programs expect, and Terminal, the element
// that draws one and takes the keyboard for it, running a shell or any
// program in a pseudo-terminal (package pty) or fed by the app (an SSH
// session, a serial port).
package terminal

import (
	"io"
	"os/exec"
	"sync"
	"time"

	"gioui.org/font"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/components/input"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/pty"
	"github.com/xmhrnx/giode/state"
	"github.com/xmhrnx/giode/styles/properties"
)

// Terminal draws an Emulator's screen in a monospace font and sends it
// what's typed: a shell with Start, or any program the app connects with
// Attach and Write. It fills the space it's given, and resizes the
// program's terminal to match. Create it once with New and lay it out
// every frame.
//
// It takes the keyboard while focused (a click focuses it): typed text
// in any layout or input method, and the keys programs expect - arrows,
// Ctrl+C, Tab, Escape, function keys. Copy is Ctrl+Shift+C (Cmd+C on
// Apple platforms), paste Ctrl+Shift+V (Cmd+V); dragging selects text
// and a double-click a word. Shift+Page Up and Page Down, or the wheel,
// scroll through the history. Programs that ask for the mouse (htop,
// vim with mouse=a) get it; Shift+drag still selects.
type Terminal struct {
	emu *Emulator
	kb  *input.KeyboardInput

	fontSize unit.Sp
	family   font.Typeface
	padding  unit.Dp
	palette  *Palette

	mu      sync.Mutex
	out     io.Writer
	sendCh  chan []byte
	pty     *pty.PTY
	pending []func()

	onResize func(cols, rows int)
	onTitle  func(string)
	onExit   func(code int, err error)
	onBell   func()

	// UI goroutine state.
	focused    bool
	scroll     int
	scrollAcc  float32
	sel        selection
	m          metrics
	cols, rows int
	resized    bool
	rowCache   []rowCache
	pal        *resolved
	palKey     paletteKey
	wantCopy   bool
	wantPaste  bool
	lastPress  time.Time
	clicks     int
	mouseBtn   MouseButton
	reporting  bool // the pointer press went to the program
}

type selection struct {
	active, dragging bool
	anchor, head     Pos
}

// New returns a Terminal of 80x24 cells, until it's laid out, keeping
// 10000 lines of history.
func New() *Terminal {
	t := &Terminal{
		emu:      NewEmulator(80, 24, 10000),
		fontSize: 13,
		family:   "Go Mono",
		padding:  6,
		cols:     80,
		rows:     24,
	}
	t.emu.SetReply(t.Send)
	t.emu.OnTitle(func(title string) {
		t.post(func() {
			if t.onTitle != nil {
				t.onTitle(title)
			}
		})
	})
	t.emu.OnBell(func() {
		t.post(func() {
			if t.onBell != nil {
				t.onBell()
			}
		})
	})
	t.kb = input.NewKeyboardInput(&gridEl{t: t}).
		OnText(t.typed).
		OnKey(t.key).
		OnFocus(func() { t.focused = true; t.Send(EncodeFocus(t.emu.Modes(), true)) }).
		OnBlur(func() { t.focused = false; t.Send(EncodeFocus(t.emu.Modes(), false)) }).
		Keyboard(input.KeyboardText).
		Label("Terminal")
	return t
}

// Emulator returns its emulator, for reading the screen or feeding it
// directly.
func (t *Terminal) Emulator() *Emulator { return t.emu }

// Start runs cmd in a pseudo-terminal of the terminal's size, connected
// to it: what it writes shows, what's typed goes to it. A nil cmd runs
// the user's shell (pty.DefaultShell). OnExit reports when it ends.
// Where programs can't run (phones, the web), it returns
// pty.ErrUnsupported.
func (t *Terminal) Start(cmd *exec.Cmd) error {
	if cmd == nil {
		sh := pty.DefaultShell()
		if sh == "" {
			return pty.ErrUnsupported
		}
		cmd = exec.Command(sh)
	}
	cols, rows := t.emu.Size()
	p, err := pty.Start(cmd, pty.Size{Cols: cols, Rows: rows})
	if err != nil {
		return err
	}
	t.mu.Lock()
	t.pty = p
	t.mu.Unlock()
	t.Attach(p)
	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := p.Read(buf)
			if n > 0 {
				t.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
		code, err := p.Wait()
		t.post(func() {
			if t.onExit != nil {
				t.onExit(code, err)
			}
		})
	}()
	return nil
}

// Attach sends what's typed (and the emulator's replies to the
// program's queries) to w, which may block: writes happen on a
// goroutine of their own, in order. Show the program's output with
// Write.
func (t *Terminal) Attach(w io.Writer) *Terminal {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.out = w
	if t.sendCh == nil {
		ch := make(chan []byte, 256)
		t.sendCh = ch
		go func() {
			for b := range ch {
				t.mu.Lock()
				w := t.out
				t.mu.Unlock()
				if w != nil {
					w.Write(b)
				}
			}
		}()
	}
	return t
}

// Write shows output of the program: text and escape sequences. Safe
// from any goroutine.
func (t *Terminal) Write(p []byte) (int, error) {
	t.emu.Write(p)
	state.RequestRedraw()
	return len(p), nil
}

// Send sends b to the program as if typed. Safe from any goroutine.
func (t *Terminal) Send(b []byte) {
	if len(b) == 0 {
		return
	}
	t.mu.Lock()
	ch := t.sendCh
	t.mu.Unlock()
	if ch != nil {
		ch <- append([]byte(nil), b...)
	}
}

// Paste sends text as pasted: bracketed when the program asked for
// that, so a shell doesn't run pasted lines one by one.
func (t *Terminal) Paste(text string) {
	t.scroll = 0
	t.Send(EncodePaste(text, t.emu.Modes()))
}

// Close closes its pseudo-terminal, if Start opened one: the program
// gets a hang-up, as when a terminal window closes.
func (t *Terminal) Close() error {
	t.mu.Lock()
	p := t.pty
	t.mu.Unlock()
	if p == nil {
		return nil
	}
	return p.Close()
}

// post runs fn on the UI goroutine, in the next frame.
func (t *Terminal) post(fn func()) {
	t.mu.Lock()
	t.pending = append(t.pending, fn)
	t.mu.Unlock()
	state.RequestRedraw()
}

func (t *Terminal) runPending() {
	t.mu.Lock()
	fns := t.pending
	t.pending = nil
	t.mu.Unlock()
	for _, fn := range fns {
		fn()
	}
}

// OnResize has fn run when the terminal's size in cells changes, as its
// space changes: an attached program (an SSH session) is told the new
// size there. Start's program is resized already. It returns t for
// chaining.
func (t *Terminal) OnResize(fn func(cols, rows int)) *Terminal {
	t.onResize = fn
	return t
}

// OnTitle has fn run, on the UI goroutine, when the program sets the
// window title. It returns t for chaining.
func (t *Terminal) OnTitle(fn func(title string)) *Terminal {
	t.onTitle = fn
	return t
}

// OnExit has fn run, on the UI goroutine, when Start's program exits,
// with its exit code (-1 when a signal ended it). It returns t for
// chaining.
func (t *Terminal) OnExit(fn func(code int, err error)) *Terminal {
	t.onExit = fn
	return t
}

// OnBell has fn run, on the UI goroutine, when the program rings the
// bell. It returns t for chaining.
func (t *Terminal) OnBell(fn func()) *Terminal {
	t.onBell = fn
	return t
}

// FontSize sets the text size; 13sp by default. It returns t for
// chaining.
func (t *Terminal) FontSize(sp float32) *Terminal {
	t.fontSize = unit.Sp(sp)
	return t
}

// FontFamily sets the font, which should be monospace; Go Mono by
// default. Characters it lacks come from the fallback fonts. It returns
// t for chaining.
func (t *Terminal) FontFamily(family string) *Terminal {
	t.family = font.Typeface(family)
	return t
}

// Padding sets the space around the cells; 6dp by default. It returns t
// for chaining.
func (t *Terminal) Padding(dp float32) *Terminal {
	t.padding = unit.Dp(dp)
	return t
}

// Palette sets its colors; by default DefaultDarkPalette or
// DefaultLightPalette, following the app's mode. It returns t for
// chaining.
func (t *Terminal) Palette(p Palette) *Terminal {
	t.palette = &p
	return t
}

// Label sets the name screen readers announce; "Terminal" by default.
// It returns t for chaining.
func (t *Terminal) Label(label string) *Terminal {
	t.kb.Label(label)
	return t
}

// Focus gives it keyboard focus in the next frame.
func (t *Terminal) Focus() { t.kb.Focus() }

// Blur takes keyboard focus away from it.
func (t *Terminal) Blur() { t.kb.Blur() }

// Focused reports whether it had keyboard focus in the last frame.
func (t *Terminal) Focused() bool { return t.kb.Focused() }

// Size returns its size in cells.
func (t *Terminal) Size() (cols, rows int) { return t.emu.Size() }

// Selection returns the selected text; "" when nothing is selected.
func (t *Terminal) Selection() string {
	if !t.sel.active || t.sel.anchor == t.sel.head {
		return ""
	}
	return t.emu.Text(t.sel.anchor, t.sel.head)
}

// SelectAll selects the whole history.
func (t *Terminal) SelectAll() {
	n := t.emu.History()
	cols, _ := t.emu.Size()
	t.sel = selection{active: true, anchor: Pos{0, 0}, head: Pos{n - 1, cols}}
	state.RequestRedraw()
}

// ClearSelection drops the selection.
func (t *Terminal) ClearSelection() {
	t.sel = selection{}
	state.RequestRedraw()
}

// Clear clears the screen and the history, as Cmd+K does in a Mac
// terminal; the program isn't told.
func (t *Terminal) Clear() {
	t.emu.Write([]byte("\x1b[H\x1b[2J"))
	t.emu.ClearScrollback()
	t.scroll = 0
	t.sel = selection{}
	state.RequestRedraw()
}

// ScrollBy scrolls the view through the history by lines, back for a
// positive count; ScrollToBottom shows the live screen again.
func (t *Terminal) ScrollBy(lines int) {
	back := t.emu.History() - t.rows
	t.scroll = min(max(t.scroll+lines, 0), max(back, 0))
	state.RequestRedraw()
}

// ScrollToBottom shows the live screen.
func (t *Terminal) ScrollToBottom() {
	t.scroll = 0
	state.RequestRedraw()
}

// Layout lays it out, filling the space it's given.
func (t *Terminal) Layout(gtx layout.Context) layout.Dimensions {
	t.runPending()
	t.measure(gtx)
	return t.kb.Layout(gtx)
}

// typed sends typed text.
func (t *Terminal) typed(s string) {
	t.scroll = 0
	t.sel = selection{}
	t.Send([]byte(s))
}

// key sends a key press, or acts on the terminal's own shortcuts.
func (t *Terminal) key(e key.Event) bool {
	if e.State != key.Press {
		return true
	}
	mods := e.Modifiers
	shortcut := mods.Contain(key.ModShortcut)
	// Copy and paste: Cmd+C and Cmd+V on Apple platforms, where Ctrl+C
	// stays an interrupt; Ctrl+Shift+C and V elsewhere.
	clip := shortcut && (key.ModShortcut != key.ModCtrl || mods.Contain(key.ModShift))
	switch {
	case clip && e.Name == "C":
		if t.Selection() != "" {
			t.wantCopy = true
		}
		return true
	case clip && e.Name == "V":
		t.wantPaste = true
		return true
	case mods == key.ModShift && e.Name == key.NamePageUp:
		t.ScrollBy(t.rows - 1)
		return true
	case mods == key.ModShift && e.Name == key.NamePageDown:
		t.ScrollBy(-(t.rows - 1))
		return true
	case mods == key.ModShift && e.Name == key.NameHome:
		t.ScrollBy(t.emu.History())
		return true
	case mods == key.ModShift && e.Name == key.NameEnd:
		t.ScrollToBottom()
		return true
	}
	b := EncodeKey(e, t.emu.Modes())
	if b == nil {
		return false
	}
	t.scroll = 0
	t.sel = selection{}
	t.Send(b)
	return true
}

// defaultPalette is the palette for the app's mode.
func defaultPalette() Palette {
	if properties.CurrentMode() == properties.ModeDark {
		return DefaultDarkPalette
	}
	return DefaultLightPalette
}

var _ elements.Element = (*Terminal)(nil)
