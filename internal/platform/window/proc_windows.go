package window

import (
	"sync"

	"gioui.org/app"
	"github.com/xmhrnx/giode/internal/platform"
	"github.com/xmhrnx/giode/internal/platform/filedrag"
	"github.com/xmhrnx/giode/internal/platform/shell"
	"golang.org/x/sys/windows"
)

// File drop on Windows. Gio has no drag-and-drop events, so the app
// subclasses its own window: DragAcceptFiles makes Explorer send
// WM_DROPFILES, which a replacement window procedure turns into paths
// and hands to the UI goroutine. The same procedure hides the window
// to the tray on close or minimize (see trayWndProc). Every other
// message goes on to Gio's original procedure.

var (
	procDefWindowProcW              = user32.NewProc("DefWindowProcW")
	procCallWindowProcW             = user32.NewProc("CallWindowProcW")
	procChangeWindowMessageFilterEx = user32.NewProc("ChangeWindowMessageFilterEx")
)

const (
	wmDropFiles  = 0x0233
	wmCopyData   = 0x004A
	wmCopyGlobal = 0x0049
	wmNCDestroy  = 0x0082
	gwlpWndProc  = ^uintptr(3) // -4
	msgfltAllow  = 1
)

// dropWindows maps subclassed windows to their host and original
// procedure. The window procedure is one callback for all windows:
// Windows limits how many callbacks a process may create.
var (
	dropMu      sync.Mutex
	dropWindows = map[uintptr]*dropTarget{}
	dropProc    = windows.NewCallback(dropWndProc)
)

type dropTarget struct {
	h       platform.Host
	oldProc uintptr
}

// SubclassOptions says which drops the window takes: WM_DROPFILES, and
// drag enter, over and leave through OLE.
type SubclassOptions struct {
	AcceptFiles, DragEvents bool
}

// Subclass replaces the window's procedure, once, to receive dropped
// files, close and minimize to the tray, drag out, the share sheet and
// UI Automation.
//
// It runs on its own goroutine: the view event that delivers hwnd is
// handled while Gio's window thread waits for the event goroutine, and
// DragAcceptFiles sends (not posts) a style-change message to that
// thread, which would deadlock if sent from the event goroutine.
func Subclass(h platform.Host, view app.ViewEvent, opts SubclassOptions) {
	w, ok := view.(app.Win32ViewEvent)
	if !ok || w.HWND == 0 {
		return
	}
	hwnd := w.HWND
	dropMu.Lock()
	if _, ok := dropWindows[hwnd]; ok {
		dropMu.Unlock()
		return
	}
	// Register the target before swapping the procedure, so messages
	// arriving in between already find Gio's procedure to forward to.
	old, _, _ := procGetWindowLongPtrW.Call(hwnd, gwlpWndProc)
	if old == 0 {
		dropMu.Unlock()
		h.Logf(platform.Warning, "giode: file drop unavailable: no window procedure")
		return
	}
	dropWindows[hwnd] = &dropTarget{h: h, oldProc: old}
	dropMu.Unlock()

	go func() {
		// Let drops through from an unelevated Explorer when the app
		// runs elevated (as administrator); without this, Windows'
		// message filtering (UIPI) silently blocks them.
		for _, m := range []uintptr{wmDropFiles, wmCopyData, wmCopyGlobal} {
			procChangeWindowMessageFilterEx.Call(hwnd, m, msgfltAllow, 0)
		}
		if prev, _, _ := procSetWindowLongPtrW.Call(hwnd, gwlpWndProc, dropProc); prev == 0 {
			dropMu.Lock()
			delete(dropWindows, hwnd)
			dropMu.Unlock()
			h.Logf(platform.Warning, "giode: file drop unavailable: couldn't subclass the window")
			return
		}
		if opts.AcceptFiles {
			filedrag.AcceptFiles(hwnd)
			h.Logf(platform.Debug, "giode: file drop enabled")
		}
		if opts.DragEvents {
			// Drag enter, over and leave come through OLE, which takes
			// over drops from apps at the same integrity level; from an
			// unelevated Explorer to an elevated app WM_DROPFILES still
			// delivers the drop.
			procPostMessageW.Call(hwnd, filedrag.WMRegisterOLE, 0, 0)
		}
	}()
}

func dropWndProc(hwnd, msg, wparam, lparam uintptr) uintptr {
	dropMu.Lock()
	t := dropWindows[hwnd]
	if msg == wmNCDestroy {
		// The window is going away: restore Gio's procedure so the
		// final messages reach it, and forget the window.
		delete(dropWindows, hwnd)
		if t != nil {
			procSetWindowLongPtrW.Call(hwnd, gwlpWndProc, t.oldProc)
			filedrag.RevokeOLE(hwnd)
			if Hooks.WindowGone != nil {
				Hooks.WindowGone(hwnd)
			}
		}
	}
	dropMu.Unlock()
	if t == nil {
		r, _, _ := procDefWindowProcW.Call(hwnd, msg, wparam, lparam)
		return r
	}
	if trayWndProc(hwnd, msg, wparam) {
		return 0
	}
	switch msg {
	case wmDropFiles:
		filedrag.DropFiles(wparam)
		return 0
	case filedrag.WMRegisterOLE:
		h := t.h
		filedrag.RegisterOLE(hwnd, func(format string, args ...any) { h.Logf(platform.Warning, format, args...) })
		return 0
	case filedrag.WMStartDrag:
		filedrag.RunExport(t.h, hwnd)
		return 0
	case shell.WMShare:
		shell.RunShare(hwnd)
		return 0
	case wmGetObject:
		// UI Automation asking for the window's provider (lParam
		// UiaRootObjectId, sign-extended or not).
		if int32(uint32(lparam)) == uiaRootObjectID && Hooks.GetObject != nil {
			if r, ok := Hooks.GetObject(hwnd, wparam, lparam); ok {
				return r
			}
		}
	}
	r, _, _ := procCallWindowProcW.Call(t.oldProc, hwnd, msg, wparam, lparam)
	return r
}

// trayWndProc handles the window messages the tray changes: closing and
// minimizing hide the window instead. It reports whether it handled msg.
func trayWndProc(hwnd, msg, wparam uintptr) bool {
	ask := func(f func() bool) bool { return f != nil && f() }
	hidden := func() {
		HideHWND(hwnd)
		if Hooks.HiddenOnClose != nil {
			Hooks.HiddenOnClose()
		}
	}
	switch {
	case msg == wmClose && ask(Hooks.VetoClose):
		return true
	case msg == wmClose && ask(Hooks.HideOnClose):
		hidden()
		return true
	case msg == wmSysCommand && wparam&0xFFF0 == scMinimize && ask(Hooks.MinimizeToTray):
		hidden()
		return true
	}
	return false
}

const (
	wmClose         = 0x0010
	wmSysCommand    = 0x0112
	scMinimize      = 0xF020
	wmGetObject     = 0x003D
	uiaRootObjectID = -25
)

// Subclassed reports whether Subclass replaced hwnd's procedure.
func Subclassed(hwnd uintptr) bool {
	dropMu.Lock()
	defer dropMu.Unlock()
	_, ok := dropWindows[hwnd]
	return ok
}

func init() {
	filedrag.Subclassed = Subclassed
}
