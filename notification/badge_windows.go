package notification

import (
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/xmhrnx/giode/internal/winrt"
)

// The badge on Windows is an overlay icon on the app's taskbar buttons
// (ITaskbarList3), drawn here: a number on a red disc.

var (
	user32                  = windows.NewLazySystemDLL("user32.dll")
	gdi32                   = windows.NewLazySystemDLL("gdi32.dll")
	procEnumWindows         = user32.NewProc("EnumWindows")
	procGetWindowThreadPID  = user32.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible     = user32.NewProc("IsWindowVisible")
	procGetWindow           = user32.NewProc("GetWindow")
	procGetSystemMetrics    = user32.NewProc("GetSystemMetrics")
	procCreateIconIndirect  = user32.NewProc("CreateIconIndirect")
	procDestroyIcon         = user32.NewProc("DestroyIcon")
	procCreateDIBSection    = gdi32.NewProc("CreateDIBSection")
	procCreateBitmap        = gdi32.NewProc("CreateBitmap")
	procDeleteObject        = gdi32.NewProc("DeleteObject")
	clsidTaskbarList        = winrt.ParseGUID("56fdf344-fd6d-11d0-958a-006097c9a090")
	iidTaskbarList3         = winrt.ParseGUID("ea1afb91-9e28-4b86-90e9-9e9f8a5eefaf")
	enumWindowsCallback     uintptr
	enumWindowsCallbackOnce = make(chan struct{}, 1)
	found                   []uintptr
)

// appWindows returns the process's visible top-level windows.
func appWindows() []uintptr {
	select {
	case enumWindowsCallbackOnce <- struct{}{}:
		enumWindowsCallback = windows.NewCallback(func(hwnd, _ uintptr) uintptr {
			var pid uint32
			procGetWindowThreadPID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
			if pid != uint32(windows.GetCurrentProcessId()) {
				return 1
			}
			if v, _, _ := procIsWindowVisible.Call(hwnd); v == 0 {
				return 1
			}
			if owner, _, _ := procGetWindow.Call(hwnd, 4 /* GW_OWNER */); owner != 0 {
				return 1
			}
			found = append(found, hwnd)
			return 1
		})
	default:
	}
	found = nil
	procEnumWindows.Call(enumWindowsCallback, 0)
	return found
}

type bitmapInfoHeader struct {
	Size                         uint32
	Width, Height                int32
	Planes, BitCount             uint16
	Compression, SizeImage       uint32
	XPelsPerMeter, YPelsPerMeter int32
	ClrUsed, ClrImportant        uint32
}

type iconInfo struct {
	Icon               int32
	XHotspot, YHotspot uint32
	Mask, Color        uintptr
}

// badgeIcon makes an HICON of n; destroy it with DestroyIcon.
func badgeIcon(n int) uintptr {
	size, _, _ := procGetSystemMetrics.Call(49) // SM_CXSMICON
	if size == 0 {
		size = 16
	}
	img := badgeImage(n, int(size))
	hdr := bitmapInfoHeader{Size: uint32(unsafe.Sizeof(bitmapInfoHeader{})), Width: int32(size), Height: -int32(size), Planes: 1, BitCount: 32}
	var bits unsafe.Pointer
	color, _, _ := procCreateDIBSection.Call(0, uintptr(unsafe.Pointer(&hdr)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if color == 0 || bits == nil {
		return 0
	}
	defer procDeleteObject.Call(color)
	dst := unsafe.Slice((*byte)(bits), len(img.Pix))
	for i := 0; i < len(img.Pix); i += 4 {
		dst[i], dst[i+1], dst[i+2], dst[i+3] = img.Pix[i+2], img.Pix[i+1], img.Pix[i], img.Pix[i+3]
	}
	maskBits := make([]byte, ((size+15)/16*2)*size)
	mask, _, _ := procCreateBitmap.Call(size, size, 1, 1, uintptr(unsafe.Pointer(&maskBits[0])))
	if mask == 0 {
		return 0
	}
	defer procDeleteObject.Call(mask)
	info := iconInfo{Icon: 1, Mask: mask, Color: color}
	icon, _, _ := procCreateIconIndirect.Call(uintptr(unsafe.Pointer(&info)))
	return icon
}

func setBadge(n int) error {
	hwnds := appWindows()
	if len(hwnds) == 0 {
		return nil
	}
	done := make(chan error, 1)
	// The taskbar list wants a single-threaded apartment: a thread of its
	// own, for this call.
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		switch err := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED); err {
		case nil, syscall.Errno(1):
			defer windows.CoUninitialize()
		case syscall.Errno(0x80010106):
		default:
			done <- err
			return
		}
		tb, err := winrt.CreateInstance(clsidTaskbarList, iidTaskbarList3)
		if err != nil {
			done <- err
			return
		}
		defer tb.Release()
		if err := tb.Call(3); err != nil { // HrInit
			done <- err
			return
		}
		var icon uintptr
		if n > 0 {
			icon = badgeIcon(n)
			defer procDestroyIcon.Call(icon)
		}
		desc, _ := windows.UTF16PtrFromString(itoa(n))
		for _, h := range hwnds {
			err = tb.Call(18, h, icon, uintptr(unsafe.Pointer(desc))) // SetOverlayIcon
		}
		done <- err
	}()
	return <-done
}
