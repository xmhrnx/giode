//go:build !windows && !darwin && !linux && !android && !freebsd && !openbsd && !netbsd

package gl

import "unsafe"

func GetProcAddress(name string) unsafe.Pointer { return nil }

func ReadPixels(fb uint32, w, h int, out []byte) bool { return false }
