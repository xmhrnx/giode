package camera

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo ios LDFLAGS: -framework UIKit -framework AVFoundation -framework CoreMedia -framework CoreVideo -framework Accelerate
#cgo !ios LDFLAGS: -framework AppKit -framework AVFoundation -framework CoreMedia -framework CoreVideo -framework Accelerate

#include <stdint.h>
#include <stdlib.h>

int giodeCameraAuthorized(void);
void giodeCameraRequestAccess(int req);
const char *giodeCameraOpen(int id, int front, int maxSide);
void giodeCameraClose(int id);
void giodeCameraCapture(int id, int req);
int giodeCameraHasTorch(int id);
const char *giodeCameraTorch(int id, int on);
const char *giodeCameraFocus(int id, float x, float y);
*/
import "C"

import (
	"context"
	"errors"
	"unsafe"

	"gioui.org/f32"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/internal/bridge"
)

const (
	supported  = true
	maxPreview = 1280
)

var (
	permissions bridge.Requests[bool]
	captures    bridge.Requests[captureResult]
)

type captureResult struct {
	jpeg []byte
	err  error
}

type platformCamera struct {
	id int
}

func requestPermission(ctx context.Context, a *giode.App) (bool, error) {
	switch C.giodeCameraAuthorized() {
	case 1:
		return true, nil
	case 0:
		return false, nil
	}
	id, ch := permissions.New()
	C.giodeCameraRequestAccess(C.int(id))
	return permissions.Wait(ctx, id, ch)
}

// cerr turns a C error string (nil for none, freed here) into an error.
func cerr(msg *C.char) error {
	if msg == nil {
		return nil
	}
	defer C.free(unsafe.Pointer(msg))
	return errors.New("camera: " + C.GoString(msg))
}

func (c *Camera) open(ctx context.Context) error {
	id := register(c)
	c.plat.id = id
	front := 0
	if c.opts.Facing == Front {
		front = 1
	}
	if err := cerr(C.giodeCameraOpen(C.int(id), C.int(front), maxPreview)); err != nil {
		unregister(c)
		return err
	}
	return nil
}

func (p platformCamera) close() { C.giodeCameraClose(C.int(p.id)) }

func (p platformCamera) capture(ctx context.Context) ([]byte, error) {
	req, ch := captures.New()
	C.giodeCameraCapture(C.int(p.id), C.int(req))
	r, err := captures.Wait(ctx, req, ch)
	if err != nil {
		return nil, err
	}
	return r.jpeg, r.err
}

func (p platformCamera) hasTorch() bool { return C.giodeCameraHasTorch(C.int(p.id)) != 0 }

func (p platformCamera) setTorch(on bool) error {
	v := 0
	if on {
		v = 1
	}
	return cerr(C.giodeCameraTorch(C.int(p.id), C.int(v)))
}

func (p platformCamera) focus(pt f32.Point) error {
	return cerr(C.giodeCameraFocus(C.int(p.id), C.float(pt.X), C.float(pt.Y)))
}

//export giodeCameraOnFrame
func giodeCameraOnFrame(id C.int, bgra unsafe.Pointer, w, h, stride C.int) {
	c := lookup(int(id))
	if c == nil || c.closed.Load() {
		return
	}
	n := int(stride) * int(h)
	buf := make([]byte, n)
	copy(buf, unsafe.Slice((*byte)(bgra), n))
	c.deliver(rawFrame{bgra: buf, w: int(w), h: int(h), stride: int(stride)})
}

//export giodeCameraOnCapture
func giodeCameraOnCapture(req C.int, jpeg unsafe.Pointer, n C.int, msg *C.char) {
	var r captureResult
	if msg != nil {
		r.err = errors.New("camera: " + C.GoString(msg))
	} else {
		r.jpeg = C.GoBytes(jpeg, n)
	}
	captures.Done(int(req), r)
}

//export giodeCameraOnPermission
func giodeCameraOnPermission(req C.int, granted C.int) {
	permissions.Done(int(req), granted != 0)
}

//export giodeCameraOnError
func giodeCameraOnError(id C.int, msg *C.char) {
	if c := lookup(int(id)); c != nil {
		c.reportError(errors.New("camera: " + C.GoString(msg)))
	}
}
