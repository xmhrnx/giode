package terminal

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// The parser follows the VT500 state machine (vt100.net/emu/dec_ansi_parser),
// with UTF-8 text, xterm's private modes and OSC strings.

type pstate uint8

const (
	stGround pstate = iota
	stEscape
	stEscInter
	stCSI
	stCSIIgnore
	stOSC
	stOSCEsc
	stString // DCS, SOS, PM and APC: skipped
	stStringEsc
)

// maxOSC caps an OSC string, so a broken one can't grow without bound.
const maxOSC = 4096

type parser struct {
	state  pstate
	utf8   [4]byte
	nutf8  int
	params [][]int // ';'-separated groups of ':'-separated values; -1 when empty
	cur    []int
	digits bool
	priv   byte // '?', '>', '<', '=' or 0
	inter  []byte
	osc    []byte
}

func (p *parser) feed(e *Emulator, b byte) {
	// A pending UTF-8 sequence takes its continuation bytes.
	if p.nutf8 > 0 {
		if b&0xc0 == 0x80 {
			p.utf8[p.nutf8] = b
			p.nutf8++
			if utf8.FullRune(p.utf8[:p.nutf8]) {
				r, _ := utf8.DecodeRune(p.utf8[:p.nutf8])
				p.nutf8 = 0
				p.text(e, r)
			}
			return
		}
		p.nutf8 = 0
		p.text(e, utf8.RuneError)
	}
	// Controls act in any state but strings.
	switch {
	case b == 0x18 || b == 0x1a: // CAN, SUB
		p.state = stGround
		return
	case b == 0x1b:
		switch p.state {
		case stOSC:
			p.state = stOSCEsc
		case stString:
			p.state = stStringEsc
		default:
			p.state = stEscape
			p.inter = p.inter[:0]
		}
		return
	case b < 0x20 && p.state != stOSC && p.state != stString:
		e.control(b)
		return
	case b == 0x7f && p.state != stOSC:
		return
	}
	switch p.state {
	case stGround:
		p.ground(e, b)
	case stEscape:
		switch {
		case b == '[':
			p.state = stCSI
			p.params, p.cur, p.digits, p.priv = p.params[:0], p.cur[:0], false, 0
			p.inter = p.inter[:0]
		case b == ']':
			p.state = stOSC
			p.osc = p.osc[:0]
		case b == 'P' || b == 'X' || b == '^' || b == '_':
			p.state = stString
		case b >= 0x20 && b <= 0x2f:
			p.inter = append(p.inter, b)
			p.state = stEscInter
		default:
			p.state = stGround
			e.escape(0, b)
		}
	case stEscInter:
		if b >= 0x20 && b <= 0x2f {
			p.inter = append(p.inter, b)
			return
		}
		p.state = stGround
		var in byte
		if len(p.inter) > 0 {
			in = p.inter[0]
		}
		e.escape(in, b)
	case stCSI:
		p.csi(e, b)
	case stCSIIgnore:
		if b >= 0x40 && b <= 0x7e {
			p.state = stGround
		}
	case stOSC:
		if b == 0x07 {
			p.state = stGround
			e.osc(string(p.osc))
			return
		}
		if len(p.osc) < maxOSC {
			p.osc = append(p.osc, b)
		}
	case stOSCEsc:
		p.state = stGround
		e.osc(string(p.osc))
		if b != '\\' {
			p.feed(e, b)
		}
	case stString:
	case stStringEsc:
		p.state = stGround
		if b != '\\' {
			p.state = stString
		}
	}
}

func (p *parser) ground(e *Emulator, b byte) {
	if b < 0x80 {
		e.put(rune(b))
		return
	}
	if b >= 0xc0 && b <= 0xf7 {
		p.utf8[0] = b
		p.nutf8 = 1
		return
	}
	// A stray continuation byte or an invalid lead byte.
	e.put(utf8.RuneError)
}

// text is a decoded UTF-8 character, printed in the ground state.
func (p *parser) text(e *Emulator, r rune) {
	if p.state == stGround {
		e.put(r)
	} else if p.state == stOSC && len(p.osc) < maxOSC {
		p.osc = utf8.AppendRune(p.osc, r)
	}
}

func (p *parser) csi(e *Emulator, b byte) {
	switch {
	case b >= '0' && b <= '9':
		if len(p.cur) == 0 {
			p.cur = append(p.cur, 0)
		}
		v := &p.cur[len(p.cur)-1]
		if *v < 0 {
			*v = 0
		}
		if *v < 100000 {
			*v = *v*10 + int(b-'0')
		}
		p.digits = true
	case b == ':':
		if len(p.cur) == 0 {
			p.cur = append(p.cur, -1)
		}
		p.cur = append(p.cur, -1)
	case b == ';':
		if len(p.cur) == 0 {
			p.cur = append(p.cur, -1)
		}
		p.params = append(p.params, p.cur)
		p.cur = nil
	case b >= '<' && b <= '?':
		if p.priv == 0 && len(p.params) == 0 && len(p.cur) == 0 {
			p.priv = b
		} else {
			p.state = stCSIIgnore
		}
	case b >= 0x20 && b <= 0x2f:
		p.inter = append(p.inter, b)
	case b >= 0x40 && b <= 0x7e:
		if len(p.cur) > 0 || len(p.params) > 0 {
			if len(p.cur) == 0 {
				p.cur = append(p.cur, -1)
			}
			p.params = append(p.params, p.cur)
			p.cur = nil
		}
		p.state = stGround
		var in byte
		if len(p.inter) > 0 {
			in = p.inter[0]
		}
		e.csi(csiParams(p.params), p.priv, in, b)
	default:
		p.state = stCSIIgnore
	}
}

// csiParams are a sequence's parameters.
type csiParams [][]int

// get returns parameter i, or def when it's missing or zero.
func (ps csiParams) get(i, def int) int {
	if i >= len(ps) || len(ps[i]) == 0 || ps[i][0] <= 0 {
		return def
	}
	return ps[i][0]
}

// raw returns parameter i, 0 when missing.
func (ps csiParams) raw(i int) int {
	if i >= len(ps) || len(ps[i]) == 0 || ps[i][0] < 0 {
		return 0
	}
	return ps[i][0]
}

// control runs a C0 control character.
func (e *Emulator) control(b byte) {
	switch b {
	case 0x07:
		if e.onBell != nil {
			e.onBell()
		}
	case 0x08:
		if e.cur.x > 0 {
			e.cur.x--
		}
		e.cur.wrap = false
	case 0x09:
		e.tab(1)
	case 0x0a, 0x0b, 0x0c:
		e.index()
		if e.newline {
			e.cur.x = 0
		}
		e.cur.wrap = false
	case 0x0d:
		e.cur.x, e.cur.wrap = 0, false
	case 0x0e:
		e.gl = 1
	case 0x0f:
		e.gl = 0
	}
}

// escape runs an escape sequence: ESC, an intermediate, a final byte.
func (e *Emulator) escape(inter, final byte) {
	switch inter {
	case '(', ')':
		g := 0
		if inter == ')' {
			g = 1
		}
		if final == '0' {
			e.charsets[g] = '0'
		} else {
			e.charsets[g] = 'B'
		}
		return
	case '#':
		if final == '8' {
			// DECALN: fill the screen with E.
			for y := range e.lines {
				for x := range e.lines[y].Cells {
					e.lines[y].Cells[x] = Cell{Rune: 'E', Width: 1}
				}
			}
		}
		return
	case 0:
	default:
		return
	}
	switch final {
	case '7':
		e.saveCursor()
	case '8':
		e.restoreCursor()
	case 'D':
		e.index()
		e.cur.wrap = false
	case 'E':
		e.index()
		e.cur.x, e.cur.wrap = 0, false
	case 'M':
		e.reverseIndex()
		e.cur.wrap = false
	case 'H':
		e.tabs[e.cur.x] = true
	case 'c':
		title, onTitle, reply, bell := e.title, e.onTitle, e.reply, e.onBell
		e.reset(e.cols, e.rows)
		e.scrollback = nil
		e.title, e.onTitle, e.reply, e.onBell = title, onTitle, reply, bell
	case '=':
		e.appKeypad = true
	case '>':
		e.appKeypad = false
	}
}

// csi runs a control sequence.
func (e *Emulator) csi(ps csiParams, priv, inter, final byte) {
	if priv == '?' {
		switch final {
		case 'h', 'l':
			for i := range ps {
				e.privateMode(ps.raw(i), final == 'h')
			}
		case 'J':
			e.eraseInDisplay(ps.raw(0))
		case 'K':
			e.eraseInLine(ps.raw(0))
		}
		return
	}
	if priv == '>' {
		if final == 'c' {
			// Secondary device attributes: a VT220-class terminal.
			e.send("\x1b[>1;10;0c")
		}
		return
	}
	if priv != 0 {
		return
	}
	switch inter {
	case ' ':
		if final == 'q' {
			// DECSCUSR: 0-2 block, 3-4 underline, 5-6 bar; odd blinks.
			n := ps.raw(0)
			e.cursorBlink = n == 0 || n%2 == 1
			switch {
			case n <= 2:
				e.cursorShape = CursorBlock
			case n <= 4:
				e.cursorShape = CursorUnderline
			default:
				e.cursorShape = CursorBar
			}
		}
		return
	case '!':
		if final == 'p' {
			// DECSTR: soft reset.
			e.cursorHidden, e.insert, e.cur.origin, e.autowrap = false, false, false, true
			e.appCursor, e.appKeypad = false, false
			e.top, e.bottom = 0, e.rows-1
			e.cur.style = Style{}
			e.charsets, e.gl = [2]byte{'B', 'B'}, 0
		}
		return
	case 0:
	default:
		return
	}
	n := ps.get(0, 1)
	switch final {
	case '@':
		e.insertChars(n)
	case 'A':
		e.cur.y = max(e.cur.y-n, e.clampTop())
		e.cur.wrap = false
	case 'B', 'e':
		e.cur.y = min(e.cur.y+n, e.clampBottom())
		e.cur.wrap = false
	case 'C', 'a':
		e.cur.x = min(e.cur.x+n, e.cols-1)
		e.cur.wrap = false
	case 'D':
		e.cur.x = max(e.cur.x-n, 0)
		e.cur.wrap = false
	case 'E':
		e.cur.y = min(e.cur.y+n, e.clampBottom())
		e.cur.x, e.cur.wrap = 0, false
	case 'F':
		e.cur.y = max(e.cur.y-n, e.clampTop())
		e.cur.x, e.cur.wrap = 0, false
	case 'G', '`':
		e.cur.x, e.cur.wrap = min(n-1, e.cols-1), false
	case 'H', 'f':
		e.moveTo(ps.get(1, 1)-1, ps.get(0, 1)-1)
	case 'I':
		e.tab(n)
	case 'J':
		e.eraseInDisplay(ps.raw(0))
	case 'K':
		e.eraseInLine(ps.raw(0))
	case 'L':
		e.insertLines(n)
	case 'M':
		e.deleteLines(n)
	case 'P':
		e.deleteChars(n)
	case 'S':
		e.scrollUp(n)
	case 'T':
		e.scrollDown(n)
	case 'X':
		e.eraseChars(n)
	case 'Z':
		e.backTab(n)
	case 'b':
		// REP: repeat the last character.
		if e.lastRune != 0 {
			for range min(n, 65535) {
				e.put(e.lastRune)
			}
		}
	case 'c':
		// Primary device attributes: a VT220 with ANSI color.
		e.send("\x1b[?62;22c")
	case 'd':
		row := n - 1
		if e.cur.origin {
			row += e.top
		}
		e.cur.y, e.cur.wrap = min(max(row, 0), e.rows-1), false
	case 'g':
		switch ps.raw(0) {
		case 0:
			e.tabs[e.cur.x] = false
		case 3:
			clear(e.tabs)
		}
	case 'h', 'l':
		for i := range ps {
			switch ps.raw(i) {
			case 4:
				e.insert = final == 'h'
			case 20:
				e.newline = final == 'h'
			}
		}
	case 'm':
		e.sgr(ps)
	case 'n':
		switch ps.raw(0) {
		case 5:
			e.send("\x1b[0n")
		case 6:
			y := e.cur.y
			if e.cur.origin {
				y -= e.top
			}
			e.send(fmt.Sprintf("\x1b[%d;%dR", y+1, e.cur.x+1))
		}
	case 'r':
		top, bottom := ps.get(0, 1)-1, ps.get(1, e.rows)-1
		bottom = min(bottom, e.rows-1)
		if top < bottom {
			e.top, e.bottom = top, bottom
			e.moveTo(0, 0)
		}
	case 's':
		e.saveCursor()
	case 'u':
		e.restoreCursor()
	case 't':
		// Window manipulation: report the size in cells.
		if ps.raw(0) == 18 {
			e.send(fmt.Sprintf("\x1b[8;%d;%dt", e.rows, e.cols))
		}
	}
}

func (e *Emulator) clampTop() int {
	if e.cur.y >= e.top {
		return e.top
	}
	return 0
}

func (e *Emulator) clampBottom() int {
	if e.cur.y <= e.bottom {
		return e.bottom
	}
	return e.rows - 1
}

// privateMode sets or resets a DEC private mode.
func (e *Emulator) privateMode(mode int, on bool) {
	switch mode {
	case 1:
		e.appCursor = on
	case 5:
		e.reverse = on
	case 6:
		e.cur.origin = on
		e.moveTo(0, 0)
	case 7:
		e.autowrap = on
	case 12:
		e.cursorBlink = on
	case 25:
		e.cursorHidden = !on
	case 9, 1000:
		e.setMouse(MouseClicks, on)
	case 1002:
		e.setMouse(MouseDrags, on)
	case 1003:
		e.setMouse(MouseMoves, on)
	case 1004:
		e.focusEvents = on
	case 1006:
		e.mouseSGR = on
	case 47, 1047:
		e.setAltScreen(on)
	case 1048:
		if on {
			e.saveCursor()
		} else {
			e.restoreCursor()
		}
	case 1049:
		if on {
			e.saveCursor()
			e.setAltScreen(true)
		} else {
			e.setAltScreen(false)
			e.restoreCursor()
		}
	case 2004:
		e.bracketed = on
	}
}

func (e *Emulator) setMouse(m MouseMode, on bool) {
	if on {
		e.mouse = m
	} else if e.mouse == m {
		e.mouse = MouseOff
	}
}

// sgr sets text attributes and colors.
func (e *Emulator) sgr(ps csiParams) {
	st := &e.cur.style
	if len(ps) == 0 {
		*st = Style{}
		return
	}
	for i := 0; i < len(ps); i++ {
		g := ps[i]
		v := 0
		if len(g) > 0 && g[0] > 0 {
			v = g[0]
		}
		switch {
		case v == 0:
			*st = Style{}
		case v == 1:
			st.Attr |= AttrBold
		case v == 2:
			st.Attr |= AttrDim
		case v == 3:
			st.Attr |= AttrItalic
		case v == 4:
			if len(g) > 1 && g[1] == 0 {
				st.Attr &^= AttrUnderline
			} else {
				st.Attr |= AttrUnderline
			}
		case v == 5 || v == 6:
			st.Attr |= AttrBlink
		case v == 7:
			st.Attr |= AttrInverse
		case v == 8:
			st.Attr |= AttrHidden
		case v == 9:
			st.Attr |= AttrStrike
		case v == 21:
			st.Attr |= AttrUnderline
		case v == 22:
			st.Attr &^= AttrBold | AttrDim
		case v == 23:
			st.Attr &^= AttrItalic
		case v == 24:
			st.Attr &^= AttrUnderline
		case v == 25:
			st.Attr &^= AttrBlink
		case v == 27:
			st.Attr &^= AttrInverse
		case v == 28:
			st.Attr &^= AttrHidden
		case v == 29:
			st.Attr &^= AttrStrike
		case v >= 30 && v <= 37:
			st.FG = Indexed(uint8(v - 30))
		case v == 38 || v == 48:
			var c Color
			var ok bool
			if len(g) > 1 {
				c, ok = extColor(g[1:])
			} else {
				var used int
				c, ok, used = extColorSemi(ps[i+1:])
				i += used
			}
			if ok {
				if v == 38 {
					st.FG = c
				} else {
					st.BG = c
				}
			}
		case v == 39:
			st.FG = DefaultColor
		case v >= 40 && v <= 47:
			st.BG = Indexed(uint8(v - 40))
		case v == 49:
			st.BG = DefaultColor
		case v >= 90 && v <= 97:
			st.FG = Indexed(uint8(v - 90 + 8))
		case v >= 100 && v <= 107:
			st.BG = Indexed(uint8(v - 100 + 8))
		}
	}
}

// extColor reads a colon form color: 5:n or 2:[colorspace:]r:g:b.
func extColor(g []int) (Color, bool) {
	at := func(i int) int {
		if i < len(g) && g[i] > 0 {
			return min(g[i], 255)
		}
		return 0
	}
	switch at(0) {
	case 5:
		return Indexed(uint8(at(1))), len(g) >= 2
	case 2:
		if len(g) >= 5 {
			return RGB(uint8(at(2)), uint8(at(3)), uint8(at(4))), true
		}
		if len(g) == 4 {
			return RGB(uint8(at(1)), uint8(at(2)), uint8(at(3))), true
		}
	}
	return 0, false
}

// extColorSemi reads a semicolon form color: 5;n or 2;r;g;b, returning
// how many parameters it used.
func extColorSemi(ps csiParams) (Color, bool, int) {
	switch ps.raw(0) {
	case 5:
		if len(ps) >= 2 {
			return Indexed(uint8(min(ps.raw(1), 255))), true, 2
		}
	case 2:
		if len(ps) >= 4 {
			c := func(i int) uint8 { return uint8(min(ps.raw(i), 255)) }
			return RGB(c(1), c(2), c(3)), true, 4
		}
	}
	return 0, false, len(ps)
}

// osc runs an operating system command.
func (e *Emulator) osc(s string) {
	num, arg, _ := strings.Cut(s, ";")
	n, err := strconv.Atoi(num)
	if err != nil {
		return
	}
	switch n {
	case 0, 2:
		e.setTitle(arg)
	case 7:
		e.cwd = arg
	}
}
