//go:build darwin

package screencap

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework Foundation -framework CoreMedia -framework CoreVideo
#cgo ios LDFLAGS: -framework ReplayKit -framework UIKit
#cgo !ios LDFLAGS: -weak_framework ScreenCaptureKit -framework CoreGraphics

#include <stdlib.h>

char *giodeScreenSources(void);
int giodeScreenPermission(void);
char *giodeScreenStart(int ident, unsigned int display, int width, int height, int fps);
void giodeScreenStop(int ident);
*/
import "C"

import (
	"context"
	"errors"
	"fmt"
	"image"
	"strconv"
	"strings"
	"sync"
	"unsafe"

	"github.com/xmhrnx/giode"
)

func sources(ctx context.Context) ([]Source, error) {
	cs := C.giodeScreenSources()
	defer C.free(unsafe.Pointer(cs))
	var out []Source
	for _, line := range strings.Split(C.GoString(cs), "\n") {
		f := strings.Split(line, "\t")
		if len(f) != 7 {
			continue
		}
		n := func(s string) int { v, _ := strconv.Atoi(s); return v }
		x, y, w, h := n(f[2]), n(f[3]), n(f[4]), n(f[5])
		out = append(out, Source{ID: f[0], Name: f[1], Bounds: image.Rect(x, y, x+w, y+h), Primary: f[6] == "1"})
	}
	if len(out) == 0 {
		return nil, errors.New("screencap: no displays")
	}
	return out, nil
}

var captures struct {
	sync.Mutex
	next int
	m    map[int]*Capture
}

func lookup(id C.int) *Capture {
	captures.Lock()
	defer captures.Unlock()
	return captures.m[int(id)]
}

//export giodeScreenBGRA
func giodeScreenBGRA(id C.int, data unsafe.Pointer, w, h, stride C.int) {
	c := lookup(id)
	if c == nil || c.busy.Load() {
		return
	}
	src := unsafe.Slice((*byte)(data), int(stride)*int(h))
	c.deliver(c.fromBGRA(src, int(w), int(h), int(stride)))
}

//export giodeScreenNV12
func giodeScreenNV12(id C.int, y unsafe.Pointer, ys C.int, uv unsafe.Pointer, uvs C.int, w, h C.int, full C.int) {
	c := lookup(id)
	if c == nil || c.busy.Load() {
		return
	}
	Y := unsafe.Slice((*byte)(y), int(ys)*int(h))
	UV := unsafe.Slice((*byte)(uv), int(uvs)*((int(h)+1)/2))
	c.deliver(c.fromNV12(Y, int(ys), UV, int(uvs), int(w), int(h), full != 0))
}

//export giodeScreenStopped
func giodeScreenStopped(id C.int) {
	if c := lookup(id); c != nil {
		c.stopped.Store(true)
	}
}

type platformCapture struct{ id int }

func (p platformCapture) stop() {
	C.giodeScreenStop(C.int(p.id))
	captures.Lock()
	delete(captures.m, p.id)
	captures.Unlock()
}

func (c *Capture) start(ctx context.Context, _ *giode.App) error {
	if C.giodeScreenPermission() == 0 {
		return fmt.Errorf("%w: allow screen recording in System Settings, Privacy & Security", ErrPermissionDenied)
	}
	srcs, err := sources(ctx)
	if err != nil {
		return err
	}
	src := srcs[0]
	for _, s := range srcs {
		if s.ID == c.opts.Source || c.opts.Source == "" && s.Primary {
			src = s
		}
	}
	w, h := c.scaledSize(src.Bounds.Dx(), src.Bounds.Dy())
	display, _ := strconv.ParseUint(src.ID, 10, 32)
	captures.Lock()
	if captures.m == nil {
		captures.m = map[int]*Capture{}
	}
	captures.next++
	id := captures.next
	captures.m[id] = c
	captures.Unlock()
	if msg := C.giodeScreenStart(C.int(id), C.uint(display), C.int(w), C.int(h), C.int(c.opts.FPS)); msg != nil {
		err := errors.New("screencap: " + C.GoString(msg))
		C.free(unsafe.Pointer(msg))
		captures.Lock()
		delete(captures.m, id)
		captures.Unlock()
		return err
	}
	c.plat = platformCapture{id}
	return nil
}
