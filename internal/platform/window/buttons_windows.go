package window

import "github.com/xmhrnx/giode/internal/platform"

// Window buttons on Windows: the minimize and maximize boxes are window
// styles; close is the system menu's Close item, grayed (WM_CLOSE is
// refused in trayWndProc through vetoClose).

var (
	procGetSystemMenu     = user32.NewProc("GetSystemMenu")
	procGetWindowLongPtrW = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtrW = user32.NewProc("SetWindowLongPtrW")
	procEnableMenuItem    = user32.NewProc("EnableMenuItem")
)

// Buttons removes the title bar buttons asked for.
func Buttons(h platform.Host, noMin, noMax, noClose bool) {
	hwnd := h.NativeHandle()
	if hwnd == 0 {
		return
	}
	const (
		gwlStyle      = ^uintptr(15) // -16
		wsMinimizeBox = 0x00020000
		wsMaximizeBox = 0x00010000
	)
	style, _, _ := procGetWindowLongPtrW.Call(hwnd, gwlStyle)
	if noMin {
		style &^= wsMinimizeBox
	}
	if noMax {
		style &^= wsMaximizeBox
	}
	procSetWindowLongPtrW.Call(hwnd, gwlStyle, style)
	if noClose {
		const scClose, mfGrayed = 0xF060, 0x1
		if menu, _, _ := procGetSystemMenu.Call(hwnd, 0); menu != 0 {
			procEnableMenuItem.Call(menu, scClose, mfGrayed)
		}
	}
	// Redraw the frame: SWP_NOMOVE | SWP_NOSIZE | SWP_NOZORDER | SWP_FRAMECHANGED.
	procSetWindowPos.Call(hwnd, 0, 0, 0, 0, 0, 0x2|0x1|0x4|0x20)
}
