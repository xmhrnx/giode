//go:build (linux || darwin) && !ios

package pty

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"golang.org/x/sys/unix"
)

type sys struct {
	master *os.File
	cmd    *exec.Cmd

	waitOnce sync.Once
	waitErr  error
	code     int
}

func (p *sys) start(cmd *exec.Cmd, size Size) error {
	master, name, err := openMaster()
	if err != nil {
		return err
	}
	tty, err := os.OpenFile(name, os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		master.Close()
		return err
	}
	defer tty.Close()
	p.master = master
	if err := p.Resize(size); err != nil {
		master.Close()
		return err
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = tty, tty, tty
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	// A session of its own, with the terminal as its controlling
	// terminal: job control and Ctrl+C work.
	cmd.SysProcAttr.Setsid = true
	cmd.SysProcAttr.Setctty = true
	cmd.SysProcAttr.Ctty = 0
	if err := cmd.Start(); err != nil {
		master.Close()
		return err
	}
	p.cmd = cmd
	return nil
}

// control runs fn with the master's descriptor, keeping it in the
// runtime's poller (File.Fd would make it blocking).
func control(f *os.File, fn func(fd uintptr) error) error {
	rc, err := f.SyscallConn()
	if err != nil {
		return err
	}
	var ferr error
	if err := rc.Control(func(fd uintptr) { ferr = fn(fd) }); err != nil {
		return err
	}
	return ferr
}

// Read reads what the program wrote; io.EOF once it exited.
func (p *sys) Read(b []byte) (int, error) {
	n, err := p.master.Read(b)
	if err != nil && n == 0 {
		var pe *os.PathError
		if errors.As(err, &pe) && errors.Is(pe.Err, syscall.EIO) || errors.Is(err, os.ErrClosed) {
			// Linux reports EIO once the last program using the
			// terminal exited.
			return 0, io.EOF
		}
	}
	return n, err
}

// Write sends the program input.
func (p *sys) Write(b []byte) (int, error) { return p.master.Write(b) }

// Resize changes the terminal's size; the program gets SIGWINCH.
func (p *sys) Resize(s Size) error {
	ws := &unix.Winsize{Col: uint16(max(1, s.Cols)), Row: uint16(max(1, s.Rows))}
	return control(p.master, func(fd uintptr) error {
		return unix.IoctlSetWinsize(int(fd), unix.TIOCSWINSZ, ws)
	})
}

// Process returns the program's process.
func (p *sys) Process() *os.Process { return p.cmd.Process }

// Wait waits for the program to exit and returns its exit code (-1 if
// a signal ended it) and any error waiting.
func (p *sys) Wait() (int, error) {
	p.waitOnce.Do(func() {
		err := p.cmd.Wait()
		p.code = p.cmd.ProcessState.ExitCode()
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			err = nil
		}
		p.waitErr = err
	})
	return p.code, p.waitErr
}

// Close closes the terminal: the program gets SIGHUP, as when a
// terminal window closes. It doesn't wait for it to exit.
func (p *sys) Close() error {
	return p.master.Close()
}

func defaultShell() string {
	if s := os.Getenv("SHELL"); s != "" {
		return s
	}
	for _, s := range []string{"/bin/bash", "/bin/zsh", "/system/bin/sh"} {
		if _, err := os.Stat(s); err == nil {
			return s
		}
	}
	return "/bin/sh"
}
