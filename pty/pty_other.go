//go:build !(linux || darwin || windows) || ios

package pty

import (
	"os"
	"os/exec"
)

type sys struct{}

func (p *sys) start(cmd *exec.Cmd, size Size) error { return ErrUnsupported }

// Read reads what the program wrote.
func (p *sys) Read(b []byte) (int, error) { return 0, ErrUnsupported }

// Write sends the program input.
func (p *sys) Write(b []byte) (int, error) { return 0, ErrUnsupported }

// Resize changes the terminal's size.
func (p *sys) Resize(s Size) error { return ErrUnsupported }

// Process returns the program's process.
func (p *sys) Process() *os.Process { return nil }

// Wait waits for the program to exit.
func (p *sys) Wait() (int, error) { return -1, ErrUnsupported }

// Close closes the terminal.
func (p *sys) Close() error { return nil }

func defaultShell() string { return "" }
