package terminal

import (
	"image"
	"sync"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/elements"
)

type sink struct {
	mu  sync.Mutex
	got []byte
}

func (s *sink) Write(b []byte) (int, error) {
	s.mu.Lock()
	s.got = append(s.got, b...)
	s.mu.Unlock()
	return len(b), nil
}

func (s *sink) wait(t *testing.T, want string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		got := string(s.got)
		s.mu.Unlock()
		if got == want {
			s.mu.Lock()
			s.got = nil
			s.mu.Unlock()
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	t.Fatalf("sent %q, want %q", s.got, want)
}

type termHarness struct {
	r   input.Router
	t   *Terminal
	out *sink
}

func (h *termHarness) frame() {
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Source: h.r.Source(), Constraints: layout.Exact(image.Pt(400, 300)), Now: time.Now()}
	h.t.Layout(gtx)
	elements.DispatchKeys(gtx)
	h.r.Frame(&ops)
}

func newTermHarness(t *testing.T) *termHarness {
	h := &termHarness{t: New(), out: &sink{}}
	h.t.Attach(h.out)
	h.frame()
	h.t.Focus()
	h.frame()
	h.frame()
	if !h.t.Focused() {
		t.Fatal("not focused")
	}
	return h
}

func TestTerminalSizeAndInput(t *testing.T) {
	h := newTermHarness(t)
	resized := false
	h.t.OnResize(func(cols, rows int) { resized = true })
	cols, rows := h.t.Size()
	if cols < 20 || rows < 5 || cols == 80 && rows == 24 {
		t.Errorf("size %dx%d for 400x300 px", cols, rows)
	}
	_ = resized

	h.r.Queue(key.EditEvent{Text: "ls"})
	h.frame()
	h.out.wait(t, "ls")
	h.r.Queue(key.Event{Name: key.NameReturn, State: key.Press})
	h.frame()
	h.out.wait(t, "\r")
	h.r.Queue(key.Event{Name: "C", Modifiers: key.ModCtrl, State: key.Press})
	h.frame()
	h.out.wait(t, "\x03")
	h.r.Queue(key.Event{Name: key.NameTab, State: key.Press})
	h.frame()
	h.out.wait(t, "\t")
}

func TestTerminalOutputAndSelection(t *testing.T) {
	h := newTermHarness(t)
	h.t.Write([]byte("hello world\r\nsecond"))
	h.frame()
	if got := h.t.Emulator().Snapshot(0).Lines[0].Text(); got != "hello world" {
		t.Fatalf("line 0 = %q", got)
	}
	// A double-click selects a word.
	m := h.t.m
	p := f32.Pt(float32(m.pad+m.x(1)+1), float32(m.pad+m.cellH/2))
	for range 2 {
		h.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: p},
			pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: p})
		h.frame()
	}
	if got := h.t.Selection(); got != "hello" {
		t.Errorf("selection %q, want hello", got)
	}
	h.t.SelectAll()
	if got := h.t.Selection(); got != "hello world\nsecond" {
		t.Errorf("select all %q", got)
	}
}

func TestTerminalReplies(t *testing.T) {
	h := newTermHarness(t)
	h.t.Write([]byte("\x1b[6n"))
	h.out.wait(t, "\x1b[1;1R")
}
