package camera

/*
#include <jni.h>
#include <stdint.h>

static void *giodeDirectAddress(JNIEnv *env, jobject buf) {
	return (*env)->GetDirectBufferAddress(env, buf);
}

static jlong giodeDirectCapacity(JNIEnv *env, jobject buf) {
	return (*env)->GetDirectBufferCapacity(env, buf);
}
*/
import "C"

import (
	"context"
	"errors"
	"unsafe"

	"gioui.org/f32"

	_ "gioui.org/app/permission/camera"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/android"
	"github.com/xmhrnx/giode/internal/bridge"
)

//go:generate javac --release 8 -classpath $ANDROID_HOME/platforms/android-34/android.jar -d $TEMP/giode_camera/classes camera_android.java
//go:generate jar cf camera_android.jar -C $TEMP/giode_camera/classes .

const (
	supported = true
	class     = "org/giode/camera/camera_android"
	// maxPreview caps the preview frames' long side.
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
	ok, err := android.CallBool(class, "hasPermission", "(Landroid/content/Context;)Z", android.Context())
	if err != nil || ok {
		return ok, err
	}
	view := a.NativeHandle()
	if view == 0 {
		return false, errors.New("camera: the window isn't open yet")
	}
	id, ch := permissions.New()
	if err := android.CallVoid(class, "requestPermission", "(Landroid/view/View;I)V", view, id); err != nil {
		permissions.Cancel(id)
		return false, err
	}
	return permissions.Wait(ctx, id, ch)
}

func (c *Camera) open(ctx context.Context) error {
	view := c.app.NativeHandle()
	if view == 0 {
		return errors.New("camera: the window isn't open yet")
	}
	id := register(c)
	c.plat.id = id
	msg, err := android.CallString(class, "open", "(Landroid/view/View;IZI)Ljava/lang/String;", view, id, c.opts.Facing == Front, maxPreview)
	if err != nil {
		unregister(c)
		return err
	}
	if msg != "" {
		unregister(c)
		return errors.New("camera: " + msg)
	}
	return nil
}

func (p platformCamera) close() {
	android.CallVoid(class, "close", "(I)V", p.id)
}

func (p platformCamera) capture(ctx context.Context) ([]byte, error) {
	req, ch := captures.New()
	if err := android.CallVoid(class, "capture", "(II)V", p.id, req); err != nil {
		captures.Cancel(req)
		return nil, err
	}
	r, err := captures.Wait(ctx, req, ch)
	if err != nil {
		return nil, err
	}
	return r.jpeg, r.err
}

func (p platformCamera) hasTorch() bool {
	ok, _ := android.CallBool(class, "hasTorch", "(I)Z", p.id)
	return ok
}

func (p platformCamera) setTorch(on bool) error {
	return android.CallVoid(class, "torch", "(IZ)V", p.id, on)
}

func (p platformCamera) focus(pt f32.Point) error {
	return android.CallVoid(class, "focus", "(IFF)V", p.id, pt.X, pt.Y)
}

// direct copies a direct ByteBuffer's contents.
func direct(env *C.JNIEnv, buf C.jobject, dst []byte) []byte {
	n := int(C.giodeDirectCapacity(env, buf))
	p := C.giodeDirectAddress(env, buf)
	if p == nil || n <= 0 {
		return dst[:0]
	}
	if cap(dst) < n {
		dst = make([]byte, n)
	}
	dst = dst[:n]
	copy(dst, unsafe.Slice((*byte)(p), n))
	return dst
}

//export Java_org_giode_camera_camera_1android_onFrame
func Java_org_giode_camera_camera_1android_onFrame(env *C.JNIEnv, cls C.jclass, id C.jint, y, u, v C.jobject, w, h, yStride, uvStride, uvPixStride, rotation C.jint, mirror C.jboolean) {
	c := lookup(int(id))
	if c == nil || c.closed.Load() {
		return
	}
	// The planes belong to the camera's buffer, which Java releases as
	// soon as this returns: copy them. Each frame gets its own copy, as
	// the converter may still hold the previous one.
	c.deliver(rawFrame{
		y: direct(env, y, nil), u: direct(env, u, nil), v: direct(env, v, nil),
		w: int(w), h: int(h), yStride: int(yStride), uvStride: int(uvStride), uvPixStride: int(uvPixStride),
		rotation: int(rotation), mirror: mirror != 0,
	})
}

//export Java_org_giode_camera_camera_1android_onCapture
func Java_org_giode_camera_camera_1android_onCapture(env *C.JNIEnv, cls C.jclass, req C.jint, jpeg C.jbyteArray, errStr C.jstring) {
	var r captureResult
	if msg := bridge.GoString(unsafe.Pointer(env), uintptr(errStr)); msg != "" {
		r.err = errors.New("camera: " + msg)
	} else {
		r.jpeg = android.GoBytes(unsafe.Pointer(env), uintptr(jpeg))
	}
	captures.Done(int(req), r)
}

//export Java_org_giode_camera_camera_1android_onPermission
func Java_org_giode_camera_camera_1android_onPermission(env *C.JNIEnv, cls C.jclass, req C.jint, granted C.jboolean) {
	permissions.Done(int(req), granted != 0)
}

//export Java_org_giode_camera_camera_1android_onError
func Java_org_giode_camera_camera_1android_onError(env *C.JNIEnv, cls C.jclass, id C.jint, errStr C.jstring) {
	c := lookup(int(id))
	if c == nil {
		return
	}
	c.reportError(errors.New("camera: " + bridge.GoString(unsafe.Pointer(env), uintptr(errStr))))
}
