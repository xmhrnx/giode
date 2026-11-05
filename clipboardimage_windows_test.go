//go:build windows

package giode

import (
	"bytes"
	"context"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"runtime"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	procEmptyClipboard   = user32.NewProc("EmptyClipboard")
	procSetClipboardData = user32.NewProc("SetClipboardData")
	procGlobalAllocClip  = kernel32Clip.NewProc("GlobalAlloc")
)

// setClipboard puts data in format f on the clipboard.
func setClipboard(t *testing.T, formats map[uintptr][]byte) {
	t.Helper()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if r, _, err := procOpenClipboard.Call(0); r == 0 {
		t.Skipf("no clipboard: %v", err)
	}
	defer procCloseClipboard.Call()
	procEmptyClipboard.Call()
	for f, data := range formats {
		h, _, _ := procGlobalAllocClip.Call(0x2 /* GMEM_MOVEABLE */, uintptr(len(data)))
		p, _, _ := procGlobalLockClip.Call(h)
		copy(unsafe.Slice((*byte)(*(*unsafe.Pointer)(unsafe.Pointer(&p))), len(data)), data)
		procGlobalUnlockClip.Call(h)
		if r, _, err := procSetClipboardData.Call(f, h); r == 0 {
			// A service session (CI) may have no usable clipboard.
			t.Skipf("SetClipboardData: %v", err)
		}
	}
}

func TestReadClipboardImageWindows(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	src.SetNRGBA(1, 0, color.NRGBA{R: 255, A: 255})
	var buf bytes.Buffer
	png.Encode(&buf, src)
	name, _ := windows.UTF16PtrFromString("PNG")
	pngFormat, _, _ := procRegisterClipboardFormatW.Call(uintptr(unsafe.Pointer(name)))

	setClipboard(t, map[uintptr][]byte{pngFormat: buf.Bytes()})
	img, err := (&App{}).ReadClipboardImage(context.Background())
	if err != nil || img == nil {
		t.Fatalf("PNG: %v, %v", img, err)
	}
	if r, _, _, _ := img.At(1, 0).RGBA(); r>>8 != 255 {
		t.Errorf("PNG pixel %v", img.At(1, 0))
	}

	// Only a bitmap (what Print Screen copies): 1x1 blue, 24-bit.
	dib := make([]byte, 40)
	le := binary.LittleEndian
	le.PutUint32(dib[0:], 40)
	le.PutUint32(dib[4:], 1)
	le.PutUint32(dib[8:], 1)
	le.PutUint16(dib[12:], 1)
	le.PutUint16(dib[14:], 24)
	dib = append(dib, 255, 0, 0, 0)
	setClipboard(t, map[uintptr][]byte{cfDIB: dib})
	img, err = (&App{}).ReadClipboardImage(context.Background())
	if err != nil || img == nil {
		t.Fatalf("DIB: %v, %v", img, err)
	}
	if _, _, b, _ := img.At(0, 0).RGBA(); b>>8 != 255 {
		t.Errorf("DIB pixel %v", img.At(0, 0))
	}

	// Text only: no image.
	setClipboard(t, map[uintptr][]byte{1 /* CF_TEXT */ : []byte("hi\x00")})
	if img, err := (&App{}).ReadClipboardImage(context.Background()); img != nil || err != nil {
		t.Errorf("text: %v, %v", img, err)
	}
}
