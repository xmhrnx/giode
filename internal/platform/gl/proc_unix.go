//go:build linux || android || freebsd || openbsd || netbsd

package gl

/*
#cgo linux LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdlib.h>

typedef void *(*giode_get_proc)(const char *);

static void *giode_egl, *giode_gles;
static giode_get_proc giode_egl_get_proc;

// giode_gl_open loads EGL and GLES, which Gio already has: dlopen finds
// the loaded copies.
static void giode_gl_open(void) {
	const char *egl[] = {"libEGL.so.1", "libEGL.so", NULL};
	const char *gles[] = {"libGLESv2.so.2", "libGLESv2.so", NULL};
	for (int i = 0; egl[i] && !giode_egl; i++) giode_egl = dlopen(egl[i], RTLD_LAZY | RTLD_GLOBAL);
	for (int i = 0; gles[i] && !giode_gles; i++) giode_gles = dlopen(gles[i], RTLD_LAZY | RTLD_GLOBAL);
	if (giode_egl) giode_egl_get_proc = (giode_get_proc)dlsym(giode_egl, "eglGetProcAddress");
}

static void *giode_gl_proc(const char *name) {
	void *p = giode_egl_get_proc ? giode_egl_get_proc(name) : NULL;
	if (!p && giode_gles) p = dlsym(giode_gles, name);
	return p;
}
*/
import "C"

import (
	"sync"
	"unsafe"
)

var glOpen sync.Once

// GetProcAddress returns an OpenGL function, or nil.
func GetProcAddress(name string) unsafe.Pointer {
	glOpen.Do(func() { C.giode_gl_open() })
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	return C.giode_gl_proc(cname)
}
