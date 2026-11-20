//go:build windows

package window

import (
	"unsafe"

	"gioui.org/app"
	"golang.org/x/sys/windows"
)

var (
	user32                    = windows.NewLazySystemDLL("user32.dll")
	dwmapi                    = windows.NewLazySystemDLL("dwmapi.dll")
	procDwmGetWindowAttribute = dwmapi.NewProc("DwmGetWindowAttribute")
	procGetWindowRect         = user32.NewProc("GetWindowRect")
	procSetWindowPos          = user32.NewProc("SetWindowPos")
	procMonitorFromWindow     = user32.NewProc("MonitorFromWindow")
	procGetMonitorInfoW       = user32.NewProc("GetMonitorInfoW")
	procGetDpiForWindow       = user32.NewProc("GetDpiForWindow")
)

const (
	dwmwaExtendedFrameBounds = 9
	monitorDefaultToNearest  = 2
	swpNoSize                = 0x0001
	swpNoZOrder              = 0x0004
	swpNoActivate            = 0x0010
	swpAsyncWindowPos        = 0x4000
)

type winRect struct{ Left, Top, Right, Bottom int32 }

type monitorInfo struct {
	Size          uint32
	Monitor, Work winRect
	Flags         uint32
}

func hwndOf(view app.ViewEvent) uintptr {
	if ev, ok := view.(app.Win32ViewEvent); ok {
		return ev.HWND
	}
	return 0
}

// winScale is the window's pixels per dp: its monitor's DPI over 96,
// as Gio's, else the last frame's.
func winScale(hwnd uintptr, frameScale float32) float32 {
	if procGetDpiForWindow.Find() == nil {
		if dpi, _, _ := procGetDpiForWindow.Call(hwnd); dpi != 0 {
			return float32(dpi) / 96
		}
	}
	return frameScale
}

// visibleFrame is the window's rectangle without the invisible resize
// borders Windows 10 and later put around it, and the offset of
// GetWindowRect's from it.
func visibleFrame(hwnd uintptr) (vis winRect, dx, dy int32) {
	var r winRect
	procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	vis = r
	if procDwmGetWindowAttribute.Find() == nil {
		var e winRect
		if hr, _, _ := procDwmGetWindowAttribute.Call(hwnd, dwmwaExtendedFrameBounds, uintptr(unsafe.Pointer(&e)), unsafe.Sizeof(e)); hr == 0 && e.Right > e.Left {
			vis = e
		}
	}
	return vis, vis.Left - r.Left, vis.Top - r.Top
}

func dpRect(r winRect, s float32) Rect {
	return Rect{X: float32(r.Left) / s, Y: float32(r.Top) / s, Width: float32(r.Right-r.Left) / s, Height: float32(r.Bottom-r.Top) / s}
}

func Frame(view app.ViewEvent, scale float32) (Rect, error) {
	h := hwndOf(view)
	if h == 0 {
		return Rect{}, ErrNoWindowYet
	}
	vis, _, _ := visibleFrame(h)
	return dpRect(vis, winScale(h, scale)), nil
}

func SetPosition(view app.ViewEvent, x, y, scale float32) error {
	h := hwndOf(view)
	if h == 0 {
		return ErrNoWindowYet
	}
	s := winScale(h, scale)
	_, dx, dy := visibleFrame(h)
	// Asynchronous: the window's thread may be waiting for this one.
	procSetWindowPos.Call(h, 0, uintptr(int32(x*s+0.5)-dx), uintptr(int32(y*s+0.5)-dy), 0, 0,
		swpNoSize|swpNoZOrder|swpNoActivate|swpAsyncWindowPos)
	return nil
}

func ScreenOf(view app.ViewEvent, scale float32) (Screen, error) {
	h := hwndOf(view)
	if h == 0 {
		return Screen{}, ErrNoWindowYet
	}
	mon, _, _ := procMonitorFromWindow.Call(h, monitorDefaultToNearest)
	mi := monitorInfo{Size: uint32(unsafe.Sizeof(monitorInfo{}))}
	if ok, _, err := procGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi))); ok == 0 {
		return Screen{}, err
	}
	s := winScale(h, scale)
	return Screen{Bounds: dpRect(mi.Monitor, s), WorkArea: dpRect(mi.Work, s), Scale: s}, nil
}

// SetTopMost leaves always on top to Gio.
func SetTopMost(view app.ViewEvent, on bool) bool { return false }

func Placed(view app.ViewEvent, topMost bool) {}

func ClientOrigin(view app.ViewEvent) (x, y int, ok bool) { return 0, 0, false }
