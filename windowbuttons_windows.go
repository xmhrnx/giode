package giode

// Window buttons on Windows: the minimize and maximize boxes are window
// styles; close is the system menu's Close item, grayed (WM_CLOSE is
// refused in trayWndProc through vetoClose).

var (
	procGetSystemMenu  = user32.NewProc("GetSystemMenu")
	procEnableMenuItem = user32.NewProc("EnableMenuItem")
)

func (a *App) windowButtonsView() {
	hwnd := a.NativeHandle()
	if hwnd == 0 {
		return
	}
	const (
		gwlStyle      = ^uintptr(15) // -16
		wsMinimizeBox = 0x00020000
		wsMaximizeBox = 0x00010000
	)
	style, _, _ := procGetWindowLongPtrW.Call(hwnd, gwlStyle)
	if a.cfg.NoMinimize {
		style &^= wsMinimizeBox
	}
	if a.cfg.NoMaximize {
		style &^= wsMaximizeBox
	}
	procSetWindowLongPtrW.Call(hwnd, gwlStyle, style)
	if a.cfg.NoClose {
		const scClose, mfGrayed = 0xF060, 0x1
		if menu, _, _ := procGetSystemMenu.Call(hwnd, 0); menu != 0 {
			procEnableMenuItem.Call(menu, scClose, mfGrayed)
		}
	}
	// Redraw the frame: SWP_NOMOVE | SWP_NOSIZE | SWP_NOZORDER | SWP_FRAMECHANGED.
	procSetWindowPos.Call(hwnd, 0, 0, 0, 0, 0, 0x2|0x1|0x4|0x20)
}
