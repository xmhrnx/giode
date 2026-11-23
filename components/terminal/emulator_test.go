package terminal

import (
	"strings"
	"testing"

	"gioui.org/io/key"
)

func screen(e *Emulator) []string {
	s := e.Snapshot(0)
	out := make([]string, len(s.Lines))
	for i, l := range s.Lines {
		out[i] = l.Text()
	}
	return out
}

func feed(e *Emulator, s string) { e.Write([]byte(s)) }

func expect(t *testing.T, e *Emulator, want ...string) {
	t.Helper()
	got := screen(e)
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("screen = %q, want %q", got, want)
	}
}

func TestTextAndWrap(t *testing.T) {
	e := NewEmulator(5, 3, 10)
	feed(e, "hello world\r\nab")
	// "hello" scrolled off the top.
	expect(t, e, " worl", "d", "ab")
	s := e.Snapshot(0)
	if !s.Lines[0].Wrapped || s.Lines[1].Wrapped {
		t.Error("wrap flags wrong")
	}
	if s.CursorX != 2 || s.CursorY != 2 {
		t.Errorf("cursor at %d,%d", s.CursorX, s.CursorY)
	}
}

func TestScrollback(t *testing.T) {
	e := NewEmulator(4, 2, 10)
	feed(e, "1\r\n2\r\n3\r\n4")
	expect(t, e, "3", "4")
	s := e.Snapshot(2)
	if s.Scrollback != 2 || s.Lines[0].Text() != "1" || s.Lines[1].Text() != "2" {
		t.Errorf("scrolled back: %+v", s)
	}
}

func TestCursorMovesAndErase(t *testing.T) {
	e := NewEmulator(6, 3, 0)
	feed(e, "abcdef\r\nghijkl\r\nmnopqr")
	feed(e, "\x1b[2;3H\x1b[K")
	expect(t, e, "abcdef", "gh", "mnopqr")
	feed(e, "\x1b[1;2H\x1b[2P")
	expect(t, e, "adef", "gh", "mnopqr")
	feed(e, "\x1b[2@")
	expect(t, e, "a  def", "gh", "mnopqr")
	feed(e, "\x1b[3;1H\x1b[1K")
	expect(t, e, "a  def", "gh", " nopqr")
	feed(e, "\x1b[2J")
	expect(t, e, "", "", "")
}

func TestSGR(t *testing.T) {
	e := NewEmulator(10, 1, 0)
	feed(e, "\x1b[1;31ma\x1b[38;5;200mb\x1b[48:2::1:2:3mc\x1b[38;2;10;20;30;7md\x1b[0me")
	c := e.Snapshot(0).Lines[0].Cells
	if c[0].Style.Attr != AttrBold || c[0].Style.FG != Indexed(1) {
		t.Errorf("a: %+v", c[0].Style)
	}
	if c[1].Style.FG != Indexed(200) {
		t.Errorf("b: %+v", c[1].Style)
	}
	if c[2].Style.BG != RGB(1, 2, 3) {
		t.Errorf("c: %+v", c[2].Style)
	}
	if c[3].Style.FG != RGB(10, 20, 30) || c[3].Style.Attr&AttrInverse == 0 {
		t.Errorf("d: %+v", c[3].Style)
	}
	if c[4].Style != (Style{}) {
		t.Errorf("e: %+v", c[4].Style)
	}
}

func TestScrollRegionAndLines(t *testing.T) {
	e := NewEmulator(3, 4, 10)
	feed(e, "a\r\nb\r\nc\r\nd")
	feed(e, "\x1b[2;3r\x1b[3;1H\n")
	expect(t, e, "a", "c", "", "d")
	if e.Snapshot(0).Scrollback != 0 {
		t.Error("a scroll region's lines went to the scrollback")
	}
	feed(e, "\x1b[2;1H\x1b[L")
	expect(t, e, "a", "", "c", "d")
	feed(e, "\x1b[M")
	expect(t, e, "a", "c", "", "d")
}

func TestAltScreen(t *testing.T) {
	e := NewEmulator(4, 2, 10)
	feed(e, "main")
	feed(e, "\x1b[?1049h\x1b[Hvim")
	expect(t, e, "vim", "")
	if !e.Modes().AltScreen {
		t.Error("not on the alternate screen")
	}
	feed(e, "\x1b[?1049l")
	expect(t, e, "main", "")
	if s := e.Snapshot(0); s.CursorX != 3 || s.CursorY != 0 {
		t.Errorf("cursor not restored: %d,%d", s.CursorX, s.CursorY)
	}
}

func TestWideAndCombining(t *testing.T) {
	e := NewEmulator(5, 2, 0)
	feed(e, "日本é")
	expect(t, e, "日本é", "")
	if s := e.Snapshot(0); s.CursorX != 5-1 || !strings.HasPrefix(s.Lines[0].Text(), "日") {
		t.Errorf("cursor %d", s.CursorX)
	}
	// A wide character at the last column wraps whole.
	e = NewEmulator(3, 2, 0)
	feed(e, "ab日")
	expect(t, e, "ab", "日")
}

func TestUTF8Split(t *testing.T) {
	e := NewEmulator(4, 1, 0)
	b := []byte("é")
	e.Write(b[:1])
	e.Write(b[1:])
	expect(t, e, "é")
}

func TestLineDrawing(t *testing.T) {
	e := NewEmulator(4, 1, 0)
	feed(e, "\x1b(0lqk\x1b(Bx")
	expect(t, e, "┌─┐x")
}

func TestReplies(t *testing.T) {
	e := NewEmulator(10, 5, 0)
	var got []string
	e.SetReply(func(b []byte) { got = append(got, string(b)) })
	feed(e, "\x1b[3;4H\x1b[6n\x1b[c")
	if len(got) != 2 || got[0] != "\x1b[3;4R" || got[1] != "\x1b[?62;22c" {
		t.Errorf("replies %q", got)
	}
}

func TestTitle(t *testing.T) {
	e := NewEmulator(10, 1, 0)
	var title string
	e.OnTitle(func(s string) { title = s })
	feed(e, "\x1b]0;hello\x07x\x1b]2;world\x1b\\y")
	if title != "world" || e.Title() != "world" {
		t.Errorf("title %q", title)
	}
	expect(t, e, "xy")
}

func TestResize(t *testing.T) {
	e := NewEmulator(5, 3, 10)
	feed(e, "1\r\n2\r\n3")
	e.Resize(5, 2)
	expect(t, e, "2", "3")
	e.Resize(5, 3)
	expect(t, e, "1", "2", "3")
	e.Resize(2, 3)
	feed(e, "\x1b[1;1Hxyz")
	expect(t, e, "xy", "z", "3")
}

func TestModes(t *testing.T) {
	e := NewEmulator(10, 2, 0)
	feed(e, "\x1b[?1h\x1b[?2004h\x1b[?1002h\x1b[?1006h")
	m := e.Modes()
	if !m.AppCursor || !m.BracketedPaste || m.Mouse != MouseDrags || !m.MouseSGR {
		t.Errorf("modes %+v", m)
	}
}

func TestEncodeKey(t *testing.T) {
	for _, tc := range []struct {
		e    key.Event
		m    Modes
		want string
	}{
		{key.Event{Name: key.NameUpArrow}, Modes{}, "\x1b[A"},
		{key.Event{Name: key.NameUpArrow}, Modes{AppCursor: true}, "\x1bOA"},
		{key.Event{Name: key.NameUpArrow, Modifiers: key.ModCtrl}, Modes{}, "\x1b[1;5A"},
		{key.Event{Name: "C", Modifiers: key.ModCtrl}, Modes{}, "\x03"},
		{key.Event{Name: "F", Modifiers: key.ModAlt}, Modes{}, "\x1bf"},
		{key.Event{Name: key.NameDeleteBackward}, Modes{}, "\x7f"},
		{key.Event{Name: key.NameReturn}, Modes{}, "\r"},
		{key.Event{Name: key.NameTab, Modifiers: key.ModShift}, Modes{}, "\x1b[Z"},
		{key.Event{Name: key.NameF5}, Modes{}, "\x1b[15~"},
		{key.Event{Name: key.NameF1}, Modes{}, "\x1bOP"},
		{key.Event{Name: key.NameDeleteForward}, Modes{}, "\x1b[3~"},
	} {
		tc.e.State = key.Press
		if got := string(EncodeKey(tc.e, tc.m)); got != tc.want {
			t.Errorf("%v %v: %q, want %q", tc.e.Modifiers, tc.e.Name, got, tc.want)
		}
	}
}

func TestEncodeMouseAndPaste(t *testing.T) {
	m := Modes{Mouse: MouseClicks, MouseSGR: true}
	if got := string(EncodeMouse(m, MouseLeft, MousePress, 4, 2, 0)); got != "\x1b[<0;5;3M" {
		t.Errorf("SGR press %q", got)
	}
	if got := EncodeMouse(m, MouseNone, MouseMove, 4, 2, 0); got != nil {
		t.Errorf("move reported in click mode: %q", got)
	}
	if got := string(EncodePaste("a\nb\x1b[201~", Modes{BracketedPaste: true})); got != "\x1b[200~a\rb\x1b[201~" {
		t.Errorf("paste %q", got)
	}
}
