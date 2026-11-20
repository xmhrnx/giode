package screencap

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
	"fmt"
	"image"
	"sync"
	"unsafe"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/android"
	"github.com/xmhrnx/giode/internal/bridge"
)

//go:generate javac --release 8 -classpath $ANDROID_HOME/platforms/android-34/android.jar -d $TEMP/giode_screencap/classes screencap_android.java ScreenCaptureService.java
//go:generate jar cf screencap_android.jar -C $TEMP/giode_screencap/classes .

const class = "org/giode/screencap/screencap_android"

var (
	starts bridge.Requests[string]
	active struct {
		sync.Mutex
		next int
		c    *Capture
		id   int
	}
)

func sources(context.Context) ([]Source, error) {
	s, err := android.CallString(class, "screenSize", "(Landroid/content/Context;)Ljava/lang/String;", android.Context())
	if err != nil {
		return nil, err
	}
	var w, h int
	if _, err := fmt.Sscan(s, &w, &h); err != nil {
		return nil, fmt.Errorf("screencap: screen size %q", s)
	}
	return []Source{{ID: "screen", Name: "Screen", Bounds: image.Rect(0, 0, w, h), Primary: true}}, nil
}

type platformCapture struct{ id int }

func (p platformCapture) stop() {
	active.Lock()
	if active.id == p.id {
		active.c = nil
	}
	active.Unlock()
	android.CallVoid(class, "stop", "(Landroid/content/Context;)V", android.Context())
}

func (c *Capture) start(ctx context.Context, a *giode.App) error {
	if a == nil {
		return errors.New("screencap: asking to capture the screen needs the app")
	}
	view := a.NativeHandle()
	if view == 0 {
		return errors.New("screencap: the window isn't open yet")
	}
	srcs, err := sources(ctx)
	if err != nil {
		return err
	}
	w, h := c.scaledSize(srcs[0].Bounds.Dx(), srcs[0].Bounds.Dy())
	active.Lock()
	if active.c != nil {
		active.Unlock()
		return errors.New("screencap: a capture is already running")
	}
	active.next++
	id := active.next
	active.c, active.id = c, id
	active.Unlock()
	fail := func(err error) error {
		active.Lock()
		if active.id == id {
			active.c = nil
		}
		active.Unlock()
		return err
	}
	req, ch := starts.New()
	if err := android.CallVoid(class, "start", "(Landroid/view/View;IIIII)V", view, req, id, w, h, c.opts.FPS); err != nil {
		starts.Cancel(req)
		return fail(err)
	}
	msg, err := starts.Wait(ctx, req, ch)
	if err != nil {
		android.CallVoid(class, "stop", "(Landroid/content/Context;)V", android.Context())
		return fail(err)
	}
	switch msg {
	case "":
	case "denied":
		return fail(ErrPermissionDenied)
	default:
		return fail(errors.New("screencap: " + msg))
	}
	c.plat = platformCapture{id}
	return nil
}

func current(id C.jint) *Capture {
	active.Lock()
	defer active.Unlock()
	if active.id != int(id) {
		return nil
	}
	return active.c
}

//export Java_org_giode_screencap_screencap_1android_onStarted
func Java_org_giode_screencap_screencap_1android_onStarted(env *C.JNIEnv, cls C.jclass, req C.jint, errStr C.jstring) {
	starts.Done(int(req), bridge.GoString(unsafe.Pointer(env), uintptr(errStr)))
}

//export Java_org_giode_screencap_screencap_1android_onFrame
func Java_org_giode_screencap_screencap_1android_onFrame(env *C.JNIEnv, cls C.jclass, id C.jint, buf C.jobject, w, h, stride C.jint) {
	c := current(id)
	if c == nil || c.busy.Load() {
		return
	}
	p := C.giodeDirectAddress(env, buf)
	n := int(C.giodeDirectCapacity(env, buf))
	if p == nil || n < int(stride)*(int(h)-1)+int(w)*4 {
		return
	}
	c.deliver(fromRGBA(unsafe.Slice((*byte)(p), n), int(w), int(h), int(stride)))
}

//export Java_org_giode_screencap_screencap_1android_onStopped
func Java_org_giode_screencap_screencap_1android_onStopped(env *C.JNIEnv, cls C.jclass, id C.jint) {
	if c := current(id); c != nil {
		c.stopped.Store(true)
	}
}
