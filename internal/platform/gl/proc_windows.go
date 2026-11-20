//go:build windows

package gl

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// OpenGL on Windows is ANGLE: the libEGL.dll and libGLESv2.dll Gio
// loaded, which loading them by name finds again.
var (
	libEGL                = windows.NewLazyDLL("libEGL.dll")
	libGLESv2             = windows.NewLazyDLL("libGLESv2.dll")
	procEGLGetProcAddress = libEGL.NewProc("eglGetProcAddress")
)

// GetProcAddress returns an OpenGL function, or nil.
func GetProcAddress(name string) unsafe.Pointer {
	cname, err := windows.BytePtrFromString(name)
	if err != nil {
		return nil
	}
	if procEGLGetProcAddress.Find() == nil {
		if r, _, _ := procEGLGetProcAddress.Call(uintptr(unsafe.Pointer(cname))); r != 0 {
			return ptr(r)
		}
	}
	if p := libGLESv2.NewProc(name); p.Find() == nil {
		return ptr(p.Addr())
	}
	return nil
}

// ptr converts a C function address.
func ptr(addr uintptr) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&addr))
}

// ReadPixels reads framebuffer fb's pixels, bottom row first, into
// out (w*h*4 bytes of RGBA).
func ReadPixels(fb uint32, w, h int, out []byte) bool {
	getiv, bind, read := GetProcAddress("glGetIntegerv"), GetProcAddress("glBindFramebuffer"), GetProcAddress("glReadPixels")
	if getiv == nil || bind == nil || read == nil {
		return false
	}
	var prev int32
	syscall.SyscallN(uintptr(getiv), glFramebufferBinding, uintptr(unsafe.Pointer(&prev)))
	syscall.SyscallN(uintptr(bind), glFramebuffer, uintptr(fb))
	syscall.SyscallN(uintptr(read), 0, 0, uintptr(w), uintptr(h), glRGBA, glUnsignedByte, uintptr(unsafe.Pointer(&out[0])))
	syscall.SyscallN(uintptr(bind), glFramebuffer, uintptr(prev))
	return true
}
