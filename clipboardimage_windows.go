//go:build windows

package giode

import (
	"context"
	"errors"
	"image"
	"runtime"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	procOpenClipboard              = user32.NewProc("OpenClipboard")
	procCloseClipboard             = user32.NewProc("CloseClipboard")
	procGetClipboardData           = user32.NewProc("GetClipboardData")
	procIsClipboardFormatAvailable = user32.NewProc("IsClipboardFormatAvailable")
	procRegisterClipboardFormatW   = user32.NewProc("RegisterClipboardFormatW")
	kernel32Clip                   = windows.NewLazySystemDLL("kernel32.dll")
	procGlobalLockClip             = kernel32Clip.NewProc("GlobalLock")
	procGlobalUnlockClip           = kernel32Clip.NewProc("GlobalUnlock")
	procGlobalSizeClip             = kernel32Clip.NewProc("GlobalSize")
)

const (
	cfDIB   = 8
	cfDIBV5 = 17
)

// readClipboardImage reads the "PNG" format browsers and editors put on
// the clipboard, else the device-independent bitmap every image copy
// (and Print Screen) provides.
func (a *App) readClipboardImage(ctx context.Context) (image.Image, error) {
	// The clipboard belongs to the thread that opened it.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	// Another app may hold it open for a moment.
	for i := 0; ; i++ {
		if r, _, _ := procOpenClipboard.Call(0); r != 0 {
			break
		}
		if i == 20 {
			return nil, errors.New("giode: the clipboard is in use by another app")
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(25 * time.Millisecond):
		}
	}
	defer procCloseClipboard.Call()

	name, _ := windows.UTF16PtrFromString("PNG")
	if png, _, _ := procRegisterClipboardFormatW.Call(uintptr(unsafe.Pointer(name))); png != 0 {
		if data := clipboardBytes(png); data != nil {
			return decodeClipboardImage(data)
		}
	}
	for _, f := range []uintptr{cfDIBV5, cfDIB} {
		if data := clipboardBytes(f); data != nil {
			return parseDIB(data)
		}
	}
	return nil, nil
}

// clipboardBytes copies the clipboard's data in format f, or returns
// nil when it has none.
func clipboardBytes(f uintptr) []byte {
	if ok, _, _ := procIsClipboardFormatAvailable.Call(f); ok == 0 {
		return nil
	}
	h, _, _ := procGetClipboardData.Call(f)
	if h == 0 {
		return nil
	}
	size, _, _ := procGlobalSizeClip.Call(h)
	if size == 0 || size > maxClipboardImage {
		return nil
	}
	p, _, _ := procGlobalLockClip.Call(h)
	if p == 0 {
		return nil
	}
	defer procGlobalUnlockClip.Call(h)
	src := unsafe.Slice((*byte)(*(*unsafe.Pointer)(unsafe.Pointer(&p))), int(size))
	return append([]byte(nil), src...)
}
