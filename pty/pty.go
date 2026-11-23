// Package pty runs a program in a pseudo-terminal: the program sees a
// terminal (a shell prompts, colors and full-screen programs work, Ctrl+C
// interrupts) while the app reads what it draws and writes what's
// typed. It's what the Terminal component runs a shell in.
//
// It uses Unix pseudo-terminals on Linux, macOS and Android, and ConPTY
// on Windows 10 1809 and later. Phones and the web can't run programs
// (Android can, in the app's own sandbox); there Start returns
// ErrUnsupported.
//
//	p, err := pty.Start(exec.Command("bash"), pty.Size{Cols: 80, Rows: 24})
//	go io.Copy(screen, p)  // what it draws
//	p.Write([]byte("ls\r")) // what's typed
//	p.Resize(pty.Size{Cols: 120, Rows: 40})
package pty

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// ErrUnsupported is returned where programs can't run in a
// pseudo-terminal.
var ErrUnsupported = errors.New("pty: not supported on this platform")

// Size is a terminal's size in character cells.
type Size struct {
	Cols, Rows int
}

// PTY is a program running in a pseudo-terminal. Read returns what it
// writes to the terminal (io.EOF once it exits and its output is read),
// Write sends it input as typed. Its methods are safe from any
// goroutine; one goroutine should read.
type PTY struct {
	sys
}

// Start runs cmd in a new pseudo-terminal of the given size. cmd's
// Stdin, Stdout and Stderr must be unset: they become the terminal.
// TERM is xterm-256color (and COLORTERM truecolor) unless cmd.Env sets
// it, and with no locale set, a UTF-8 one is (LANG=C.UTF-8 on Linux,
// LC_CTYPE=UTF-8 on macOS). Start returns once the program started.
func Start(cmd *exec.Cmd, size Size) (*PTY, error) {
	if cmd.Stdin != nil || cmd.Stdout != nil || cmd.Stderr != nil {
		return nil, errors.New("pty: cmd's Stdin, Stdout and Stderr must be unset")
	}
	if size.Cols <= 0 {
		size.Cols = 80
	}
	if size.Rows <= 0 {
		size.Rows = 24
	}
	env := cmd.Env
	if env == nil {
		// The app's own TERM describes the terminal it was started
		// from, if any, not this one.
		for _, kv := range os.Environ() {
			if !strings.HasPrefix(kv, "TERM=") && !strings.HasPrefix(kv, "COLORTERM=") {
				env = append(env, kv)
			}
		}
	}
	hasTerm := false
	for _, kv := range env {
		if strings.HasPrefix(kv, "TERM=") {
			hasTerm = true
		}
	}
	if !hasTerm {
		env = append(env[:len(env):len(env)], "TERM=xterm-256color", "COLORTERM=truecolor")
	}
	// An app opened from the desktop often has no locale, which leaves
	// the shell in the C locale: no UTF-8, so non-ASCII text breaks.
	if l := utf8Locale(env); l != "" {
		env = append(env[:len(env):len(env)], l)
	}
	cmd.Env = env
	p := &PTY{}
	if err := p.start(cmd, size); err != nil {
		return nil, err
	}
	return p, nil
}

// utf8Locale returns a locale setting to add to env, which has none;
// "" when it has one.
func utf8Locale(env []string) string {
	for _, kv := range env {
		if strings.HasPrefix(kv, "LANG=") || strings.HasPrefix(kv, "LC_ALL=") || strings.HasPrefix(kv, "LC_CTYPE=") {
			return ""
		}
	}
	switch runtime.GOOS {
	case "darwin":
		return "LC_CTYPE=UTF-8"
	case "linux", "android":
		return "LANG=C.UTF-8"
	}
	return ""
}

// DefaultShell returns the user's shell: $SHELL (or /bin/sh) on Unix,
// PowerShell, or cmd.exe, on Windows.
func DefaultShell() string { return defaultShell() }
