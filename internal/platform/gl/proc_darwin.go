//go:build darwin

package gl

/*
#include <dlfcn.h>
#include <stdlib.h>
#include <TargetConditionals.h>

static void *giode_gl;

static void giode_gl_open(void) {
#if TARGET_OS_IPHONE
	giode_gl = dlopen("/System/Library/Frameworks/OpenGLES.framework/OpenGLES", RTLD_LAZY | RTLD_GLOBAL);
#else
	giode_gl = dlopen("/System/Library/Frameworks/OpenGL.framework/OpenGL", RTLD_LAZY | RTLD_GLOBAL);
#endif
}

static void *giode_gl_proc(const char *name) {
	return giode_gl ? dlsym(giode_gl, name) : NULL;
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
