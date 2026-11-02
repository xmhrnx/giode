//go:build windows

package giode

import (
	"bytes"
	"encoding/binary"
	"gioui.org/app"
	"image/png"

	"fyne.io/systray"
)

var (
	procShowWindowAsync     = user32.NewProc("ShowWindowAsync")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procIsIconic            = user32.NewProc("IsIconic")
)

const (
	swHide    = 0
	swShow    = 5
	swRestore = 9

	wmClose      = 0x0010
	wmSysCommand = 0x0112
	scMinimize   = 0xF020
)

// The Async variant posts to the window's thread: a synchronous call
// from the UI goroutine would wait on Gio's window thread, which may
// be waiting on the UI goroutine.

func (a *App) hideNativeWindow() bool {
	hwnd := a.NativeHandle()
	if hwnd == 0 {
		return false
	}
	procShowWindowAsync.Call(hwnd, swHide)
	return true
}

func (a *App) showNativeWindow() {
	hwnd := a.NativeHandle()
	if hwnd == 0 {
		return
	}
	cmd := uintptr(swShow)
	if iconic, _, _ := procIsIconic.Call(hwnd); iconic != 0 {
		cmd = swRestore
	}
	procShowWindowAsync.Call(hwnd, cmd)
	go procSetForegroundWindow.Call(hwnd)
}

// trayWndProc handles the window messages the tray changes: closing and
// minimizing hide the window instead. It reports whether it handled msg.
func trayWndProc(a *App, hwnd, msg, wparam uintptr) bool {
	switch {
	case msg == wmClose && a.closesToTray():
		procShowWindowAsync.Call(hwnd, swHide)
		a.windowHiddenNatively()
		return true
	case msg == wmSysCommand && wparam&0xFFF0 == scMinimize && a.minimizesToTray():
		procShowWindowAsync.Call(hwnd, swHide)
		a.windowHiddenNatively()
		return true
	}
	return false
}

// setTrayIcon sets the tray icon from PNG bytes, wrapped in an ICO
// container: the Windows tray loads .ico files.
func setTrayIcon(icon []byte, _ bool) {
	if ico := pngToICO(icon); ico != nil {
		systray.SetIcon(ico)
	}
}

// pngToICO wraps PNG bytes in a single-image ICO file, which Windows
// Vista and later read.
func pngToICO(p []byte) []byte {
	cfg, err := png.DecodeConfig(bytes.NewReader(p))
	if err != nil || cfg.Width > 256 || cfg.Height > 256 {
		return nil
	}
	dim := func(v int) byte {
		if v == 256 {
			return 0
		}
		return byte(v)
	}
	var b bytes.Buffer
	binary.Write(&b, binary.LittleEndian, [3]uint16{0, 1, 1})
	b.Write([]byte{dim(cfg.Width), dim(cfg.Height), 0, 0})
	binary.Write(&b, binary.LittleEndian, [2]uint16{1, 32})
	binary.Write(&b, binary.LittleEndian, [2]uint32{uint32(len(p)), 22})
	b.Write(p)
	return b.Bytes()
}

func (a *App) warnTrayLimits()        {}
func (a *App) platformInit()          {}
func (a *App) trayView(app.ViewEvent) {}

// applyTrayOptions subclasses the window, if it isn't yet, once a tray
// option is turned on.
func (a *App) applyTrayOptions() {
	if hwnd := a.NativeHandle(); hwnd != 0 {
		a.enableFileDrop(hwnd)
	}
}
