//go:build linux || darwin

package pty

import (
	"io"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func readAll(t *testing.T, p *PTY) string {
	t.Helper()
	done := make(chan string)
	go func() {
		b, _ := io.ReadAll(p)
		done <- string(b)
	}()
	select {
	case s := <-done:
		return s
	case <-time.After(10 * time.Second):
		t.Fatal("no EOF from the terminal")
		return ""
	}
}

func TestRun(t *testing.T) {
	p, err := Start(exec.Command("/bin/sh", "-c", `stty size; echo "$TERM"; test -t 0 && echo tty; exit 3`), Size{Cols: 91, Rows: 17})
	if err != nil {
		t.Fatal(err)
	}
	out := readAll(t, p)
	for _, want := range []string{"17 91", "xterm-256color", "tty"} {
		if !strings.Contains(out, want) {
			t.Errorf("output %q lacks %q", out, want)
		}
	}
	if code, err := p.Wait(); code != 3 || err != nil {
		t.Errorf("Wait() = %d, %v; want 3", code, err)
	}
	p.Close()
}

func TestInputAndResize(t *testing.T) {
	p, err := Start(exec.Command("/bin/sh", "-c", `read line; stty size; echo "got $line"`), Size{Cols: 80, Rows: 24})
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Resize(Size{Cols: 100, Rows: 30}); err != nil {
		t.Fatal(err)
	}
	p.Write([]byte("hello\r"))
	out := readAll(t, p)
	if !strings.Contains(out, "30 100") || !strings.Contains(out, "got hello") {
		t.Errorf("output %q", out)
	}
	p.Wait()
	p.Close()
}
