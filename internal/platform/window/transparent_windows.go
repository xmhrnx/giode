package window

import (
	"unsafe"

	"github.com/xmhrnx/giode/internal/platform"
	"golang.org/x/sys/windows"
)

// A transparent window on Windows: DWM blends the window with what's
// behind it by its pixels' alpha once blur-behind is on for an empty
// region (as GLFW does for transparent framebuffers).

var (
	procDwmEnableBlurBehindWindow = windows.NewLazySystemDLL("dwmapi.dll").NewProc("DwmEnableBlurBehindWindow")
	procCreateRectRgn             = windows.NewLazySystemDLL("gdi32.dll").NewProc("CreateRectRgn")
	procDeleteObjectRgn           = windows.NewLazySystemDLL("gdi32.dll").NewProc("DeleteObject")
)

type dwmBlurBehind struct {
	flags                 uint32
	enable                int32
	region                uintptr
	transitionOnMaximized int32
}

// MakeTransparent lets the window show through where the UI is
// transparent.
func MakeTransparent(h platform.Host) {
	hwnd := h.NativeHandle()
	if hwnd == 0 || procDwmEnableBlurBehindWindow.Find() != nil {
		return
	}
	rgn, _, _ := procCreateRectRgn.Call(0, 0, ^uintptr(0), ^uintptr(0)) // (0,0,-1,-1): empty
	defer procDeleteObjectRgn.Call(rgn)
	bb := dwmBlurBehind{flags: 1 | 2, enable: 1, region: rgn} // DWM_BB_ENABLE | DWM_BB_BLURREGION
	procDwmEnableBlurBehindWindow.Call(hwnd, uintptr(unsafe.Pointer(&bb)))
}
