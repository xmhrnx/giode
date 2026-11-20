//go:build linux || android || freebsd || openbsd || netbsd || darwin

package gl

/*
typedef void (*giode_getiv)(unsigned int, int *);
typedef void (*giode_bind)(unsigned int, unsigned int);
typedef void (*giode_read)(int, int, int, int, unsigned int, unsigned int, void *);

static void giode_read_pixels(void *getiv, void *bind, void *read, unsigned int fb, int w, int h, void *out) {
	int prev = 0;
	((giode_getiv)getiv)(0x8CA6, &prev);
	((giode_bind)bind)(0x8D40, fb);
	((giode_read)read)(0, 0, w, h, 0x1908, 0x1401, out);
	((giode_bind)bind)(0x8D40, (unsigned int)prev);
}
*/
import "C"

import "unsafe"

// ReadPixels reads framebuffer fb's pixels, bottom row first, into
// out (w*h*4 bytes of RGBA).
func ReadPixels(fb uint32, w, h int, out []byte) bool {
	getiv, bind, read := GetProcAddress("glGetIntegerv"), GetProcAddress("glBindFramebuffer"), GetProcAddress("glReadPixels")
	if getiv == nil || bind == nil || read == nil {
		return false
	}
	C.giode_read_pixels(getiv, bind, read, C.uint(fb), C.int(w), C.int(h), unsafe.Pointer(&out[0]))
	return true
}
