//go:build windows

package window

import (
	"image"
	"unsafe"

	"gioui.org/app"
	"golang.org/x/sys/windows"
)

var (
	gdi32                  = windows.NewLazySystemDLL("gdi32.dll")
	procCreateIconIndirect = user32.NewProc("CreateIconIndirect")
	procPostMessageW       = user32.NewProc("PostMessageW")
	procCreateDIBSection   = gdi32.NewProc("CreateDIBSection")
	procCreateBitmap       = gdi32.NewProc("CreateBitmap")
	procDeleteObject       = gdi32.NewProc("DeleteObject")
)

const (
	wmSetIcon  = 0x0080
	iconBig    = 1
	iconSmall  = 0
	dibRGB     = 0
	biRGB      = 0
	biBitCount = 32
)

type iconInfo struct {
	fIcon    int32
	xHotspot uint32
	yHotspot uint32
	hbmMask  windows.Handle
	hbmColor windows.Handle
}

type bitmapInfoHeader struct {
	biSize          uint32
	biWidth         int32
	biHeight        int32
	biPlanes        uint16
	biBitCount      uint16
	biCompression   uint32
	biSizeImage     uint32
	biXPelsPerMeter int32
	biYPelsPerMeter int32
	biClrUsed       uint32
	biClrImportant  uint32
}

type bitmapInfo struct {
	header bitmapInfoHeader
	colors [1]uint32
}

// SetIcon converts img to an icon and assigns it to the window with
// WM_SETICON, reporting whether it could.
func SetIcon(view app.ViewEvent, img image.Image) bool {
	wev, ok := view.(app.Win32ViewEvent)
	if !ok || wev.HWND == 0 {
		return false
	}
	setIcon(wev.HWND, img)
	return true
}

func setIcon(hwnd uintptr, img image.Image) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 || w > 256 || h > 256 {
		return
	}
	rgba := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			rgba.Set(x, y, img.At(b.Min.X+x, b.Min.Y+y))
		}
	}

	// BGRA, top-down (negative height).
	bmi := bitmapInfo{}
	bmi.header.biSize = uint32(unsafe.Sizeof(bmi.header))
	bmi.header.biWidth = int32(w)
	bmi.header.biHeight = -int32(h)
	bmi.header.biPlanes = 1
	bmi.header.biBitCount = biBitCount
	bmi.header.biCompression = biRGB

	var bits unsafe.Pointer
	hbmColor, _, _ := procCreateDIBSection.Call(0, uintptr(unsafe.Pointer(&bmi)), dibRGB, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if hbmColor == 0 || bits == nil {
		return
	}
	defer procDeleteObject.Call(hbmColor)

	dst := unsafe.Slice((*byte)(bits), w*h*4)
	i := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := rgba.NRGBAAt(x, y)
			dst[i] = c.B
			dst[i+1] = c.G
			dst[i+2] = c.R
			dst[i+3] = c.A
			i += 4
		}
	}

	// An all-ones mask keeps the color alpha channel authoritative.
	maskPixels := make([]byte, (w*7/8+1)*h)
	for i := range maskPixels {
		maskPixels[i] = 0xff
	}
	hbmMask, _, _ := procCreateBitmap.Call(uintptr(w), uintptr(h), 1, 1, uintptr(unsafe.Pointer(&maskPixels[0])))
	if hbmMask == 0 {
		return
	}
	defer procDeleteObject.Call(hbmMask)

	info := iconInfo{fIcon: 1, hbmMask: windows.Handle(hbmMask), hbmColor: windows.Handle(hbmColor)}
	hIcon, _, _ := procCreateIconIndirect.Call(uintptr(unsafe.Pointer(&info)))
	if hIcon == 0 {
		return
	}
	// The handle stays valid for the lifetime of the window.
	// PostMessage is used instead of SendMessage: a blocking send
	// into a Gio window from its event goroutine deadlocks.
	procPostMessageW.Call(hwnd, wmSetIcon, iconBig, hIcon)
	procPostMessageW.Call(hwnd, wmSetIcon, iconSmall, hIcon)
}

// Local aliases keep the platform-neutral code independent of the
// concrete Gio event types.
type appViewEvent = app.ViewEvent
type win32ViewEvent = app.Win32ViewEvent
