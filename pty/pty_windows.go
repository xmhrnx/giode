package pty

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

type sys struct {
	hpc     windows.Handle
	in, out *os.File
	proc    windows.Handle
	process *os.Process

	mu        sync.Mutex
	consoleUp bool

	waitOnce sync.Once
	waitErr  error
	code     int
}

func (p *sys) start(cmd *exec.Cmd, size Size) (err error) {
	if cmd.Err != nil {
		return cmd.Err
	}
	var inR, inW, outR, outW windows.Handle
	if err := windows.CreatePipe(&inR, &inW, nil, 0); err != nil {
		return err
	}
	if err := windows.CreatePipe(&outR, &outW, nil, 0); err != nil {
		windows.CloseHandle(inR)
		windows.CloseHandle(inW)
		return err
	}
	// The pseudo console keeps its own copies of its ends.
	defer windows.CloseHandle(inR)
	defer windows.CloseHandle(outW)
	p.in = os.NewFile(uintptr(inW), "pty-in")
	p.out = os.NewFile(uintptr(outR), "pty-out")
	defer func() {
		if err != nil {
			p.in.Close()
			p.out.Close()
		}
	}()
	if err := windows.CreatePseudoConsole(coord(size), inR, outW, 0, &p.hpc); err != nil {
		return err
	}
	p.consoleUp = true
	defer func() {
		if err != nil {
			windows.ClosePseudoConsole(p.hpc)
		}
	}()

	attrs, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		return err
	}
	defer attrs.Delete()
	// The attribute's value is the console handle itself, passed where a
	// pointer goes.
	hpc := *(*unsafe.Pointer)(unsafe.Pointer(&p.hpc))
	if err := attrs.Update(windows.PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE, hpc, unsafe.Sizeof(p.hpc)); err != nil {
		return err
	}
	si := &windows.StartupInfoEx{ProcThreadAttributeList: attrs.List()}
	si.Cb = uint32(unsafe.Sizeof(*si))
	// No standard handles: the program uses the pseudo console's, not
	// the app's.
	si.Flags = windows.STARTF_USESTDHANDLES

	path := cmd.Path
	if !filepath.IsAbs(path) && cmd.Dir != "" {
		if abs, err := filepath.Abs(path); err == nil {
			path = abs
		}
	}
	args := append([]string{path}, cmd.Args[1:]...)
	cmdLine, err := windows.UTF16PtrFromString(windows.ComposeCommandLine(args))
	if err != nil {
		return err
	}
	var dir *uint16
	if cmd.Dir != "" {
		if dir, err = windows.UTF16PtrFromString(cmd.Dir); err != nil {
			return err
		}
	}
	env := envBlock(cmd.Env)
	var pi windows.ProcessInformation
	flags := uint32(windows.EXTENDED_STARTUPINFO_PRESENT | windows.CREATE_UNICODE_ENVIRONMENT)
	if err := windows.CreateProcess(nil, cmdLine, nil, nil, false, flags, &env[0], dir, &si.StartupInfo, &pi); err != nil {
		return err
	}
	windows.CloseHandle(pi.Thread)
	p.proc = pi.Process
	p.process, _ = os.FindProcess(int(pi.ProcessId))
	cmd.Process = p.process
	// The pseudo console's output ends only when it's closed: close it
	// once the program exits, so Read returns io.EOF.
	go func() {
		p.Wait()
		p.closeConsole()
	}()
	return nil
}

func coord(s Size) windows.Coord {
	return windows.Coord{X: int16(min(max(s.Cols, 1), 32767)), Y: int16(min(max(s.Rows, 1), 32767))}
}

// envBlock is env as CreateProcess takes it: NUL-separated, ending in
// two NULs.
func envBlock(env []string) []uint16 {
	var b []uint16
	for _, kv := range env {
		b = append(b, utf16.Encode([]rune(kv))...)
		b = append(b, 0)
	}
	if len(b) == 0 {
		b = append(b, 0)
	}
	return append(b, 0)
}

func (p *sys) closeConsole() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.consoleUp {
		p.consoleUp = false
		windows.ClosePseudoConsole(p.hpc)
	}
}

// Read reads what the program wrote; io.EOF once it exited.
func (p *sys) Read(b []byte) (int, error) {
	n, err := p.out.Read(b)
	if errors.Is(err, os.ErrClosed) {
		return n, errEOF
	}
	return n, err
}

// Write sends the program input.
func (p *sys) Write(b []byte) (int, error) { return p.in.Write(b) }

// Resize changes the terminal's size.
func (p *sys) Resize(s Size) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.consoleUp {
		return nil
	}
	return windows.ResizePseudoConsole(p.hpc, coord(s))
}

// Process returns the program's process.
func (p *sys) Process() *os.Process { return p.process }

// Wait waits for the program to exit and returns its exit code and any
// error waiting.
func (p *sys) Wait() (int, error) {
	p.waitOnce.Do(func() {
		if _, err := windows.WaitForSingleObject(p.proc, windows.INFINITE); err != nil {
			p.code, p.waitErr = -1, err
			return
		}
		var code uint32
		if err := windows.GetExitCodeProcess(p.proc, &code); err != nil {
			p.code, p.waitErr = -1, err
		} else {
			p.code = int(code)
		}
		windows.CloseHandle(p.proc)
	})
	return p.code, p.waitErr
}

// Close closes the terminal, which ends the programs attached to it.
func (p *sys) Close() error {
	p.in.Close()
	p.closeConsole()
	return p.out.Close()
}

func defaultShell() string {
	if s, err := exec.LookPath("pwsh.exe"); err == nil {
		return s
	}
	if s, err := exec.LookPath("powershell.exe"); err == nil {
		return s
	}
	if s := os.Getenv("ComSpec"); s != "" {
		return s
	}
	return "cmd.exe"
}

var errEOF = io.EOF
