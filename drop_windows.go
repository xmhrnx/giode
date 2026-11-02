//go:build windows

package giode

import (
	"sync"
	"unsafe"

	"gioui.org/f32"
	"golang.org/x/sys/windows"
)

// File drop on Windows. Gio has no drag-and-drop events, so the app
// subclasses its own window: DragAcceptFiles makes Explorer send
// WM_DROPFILES, which a replacement window procedure turns into paths
// and hands to the UI goroutine. The same procedure hides the window
// to the tray on close or minimize (see trayWndProc). Every other
// message goes on to Gio's original procedure.

var (
	shell32                         = windows.NewLazySystemDLL("shell32.dll")
	procDragAcceptFiles             = shell32.NewProc("DragAcceptFiles")
	procDragQueryFileW              = shell32.NewProc("DragQueryFileW")
	procDragFinish                  = shell32.NewProc("DragFinish")
	procDragQueryPoint              = shell32.NewProc("DragQueryPoint")
	procSetWindowLongPtrW           = user32.NewProc("SetWindowLongPtrW")
	procGetWindowLongPtrW           = user32.NewProc("GetWindowLongPtrW")
	procDefWindowProcW              = user32.NewProc("DefWindowProcW")
	procCallWindowProcW             = user32.NewProc("CallWindowProcW")
	procChangeWindowMessageFilterEx = user32.NewProc("ChangeWindowMessageFilterEx")
)

const (
	wmDropFiles   = 0x0233
	wmCopyData    = 0x004A
	wmCopyGlobal  = 0x0049
	wmNCDestroy   = 0x0082
	wmRegisterOLE = 0x8000 + 0x4744 // WM_APP + "GD": register the IDropTarget on the window's thread
	gwlpWndProc   = ^uintptr(3)     // -4
	msgfltAllow   = 1
	maxDropFiles  = 10000
	maxPathLength = 32768
)

// dropWindows maps subclassed windows to their app and original
// procedure. The window procedure is one callback for all windows:
// Windows limits how many callbacks a process may create.
var (
	dropMu      sync.Mutex
	dropWindows = map[uintptr]*dropTarget{}
	dropProc    = windows.NewCallback(dropWndProc)
)

type dropTarget struct {
	app     *App
	oldProc uintptr
}

// enableFileDrop subclasses hwnd to receive dropped files, and for
// MinimizeToTray and CloseToTray, once.
//
// It runs on its own goroutine: the view event that delivers hwnd is
// handled while Gio's window thread waits for the event goroutine, and
// DragAcceptFiles sends (not posts) a style-change message to that
// thread, which would deadlock if sent from the event goroutine.
func (a *App) enableFileDrop(hwnd uintptr) {
	if (!a.fileDragWanted() && !a.closeToTray.Load() && !a.minimizeToTray.Load()) || hwnd == 0 {
		return
	}
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
		a.logf(LogLevelWarning, "giode: file drop unavailable: no window procedure")
		return
	}
	dropWindows[hwnd] = &dropTarget{app: a, oldProc: old}
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
			a.logf(LogLevelWarning, "giode: file drop unavailable: couldn't subclass the window")
			return
		}
		if a.fileDragWanted() {
			procDragAcceptFiles.Call(hwnd, 1)
			a.logf(LogLevelDebug, "giode: file drop enabled")
		}
		if a.cfg.OnFileDrag != nil {
			// Drag enter, over and leave come through OLE, which takes
			// over drops from apps at the same integrity level; from an
			// unelevated Explorer to an elevated app WM_DROPFILES still
			// delivers the drop.
			procPostMessageW.Call(hwnd, wmRegisterOLE, 0, 0)
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
			revokeOLEDrop(hwnd)
		}
	}
	dropMu.Unlock()
	if t == nil {
		r, _, _ := procDefWindowProcW.Call(hwnd, msg, wparam, lparam)
		return r
	}
	if trayWndProc(t.app, hwnd, msg, wparam) {
		return 0
	}
	switch msg {
	case wmDropFiles:
		var pt [2]int32
		procDragQueryPoint.Call(wparam, uintptr(unsafe.Pointer(&pt)))
		paths := readDrop(wparam)
		procDragFinish.Call(wparam)
		t.app.deliverFileDrop(paths, f32.Pt(float32(pt[0]), float32(pt[1])))
		return 0
	case wmRegisterOLE:
		registerOLEDrop(t.app, hwnd)
		return 0
	}
	r, _, _ := procCallWindowProcW.Call(t.oldProc, hwnd, msg, wparam, lparam)
	return r
}

// readDrop reads the paths of a drop handle (an HDROP).
func readDrop(hdrop uintptr) []string {
	n, _, _ := procDragQueryFileW.Call(hdrop, 0xFFFFFFFF, 0, 0)
	n = min(n, maxDropFiles)
	paths := make([]string, 0, n)
	for i := uintptr(0); i < n; i++ {
		l, _, _ := procDragQueryFileW.Call(hdrop, i, 0, 0)
		if l == 0 || l > maxPathLength {
			continue
		}
		buf := make([]uint16, l+1)
		procDragQueryFileW.Call(hdrop, i, uintptr(unsafe.Pointer(&buf[0])), l+1)
		paths = append(paths, windows.UTF16ToString(buf))
	}
	return paths
}

// installFileDrag has nothing to do before the window exists on
// Windows: enableFileDrop sets drops up with the window handle.
func installFileDrag(a *App) {}
