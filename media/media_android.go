package media

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
	"image"
	"sync"
	"time"
	"unsafe"

	"github.com/xmhrnx/giode/android"
	"github.com/xmhrnx/giode/internal/bridge"
	"github.com/xmhrnx/giode/video"
)

//go:generate javac --release 8 -classpath $ANDROID_HOME/platforms/android-34/android.jar -d $TEMP/giode_media/classes media_android.java
//go:generate jar cf media_android.jar -C $TEMP/giode_media/classes .

const class = "org/giode/media/media_android"

var droidPlayers struct {
	sync.Mutex
	next int
	m    map[int]*Player
}

func droidLookup(id C.jint) *Player {
	droidPlayers.Lock()
	defer droidPlayers.Unlock()
	return droidPlayers.m[int(id)]
}

type droidEngine struct{ id int }

func newEngine(ctx context.Context, p *Player, src string) (engine, error) {
	droidPlayers.Lock()
	if droidPlayers.m == nil {
		droidPlayers.m = map[int]*Player{}
	}
	droidPlayers.next++
	id := droidPlayers.next
	droidPlayers.m[id] = p
	droidPlayers.Unlock()
	vol := float32(p.opts.Volume)
	if p.opts.Muted {
		vol = 0
	}
	msg, err := android.CallString(class, "open", "(Landroid/content/Context;ILjava/lang/String;F)Ljava/lang/String;",
		android.Context(), id, src, vol)
	if err == nil && msg != "" {
		err = errors.New("media: " + msg)
	}
	if err != nil {
		droidPlayers.Lock()
		delete(droidPlayers.m, id)
		droidPlayers.Unlock()
		return nil, err
	}
	return droidEngine{id}, nil
}

func (e droidEngine) play()  { android.CallVoid(class, "play", "(I)V", e.id) }
func (e droidEngine) pause() { android.CallVoid(class, "pause", "(I)V", e.id) }
func (e droidEngine) seek(d time.Duration) {
	android.CallVoid(class, "seek", "(II)V", e.id, int(d/time.Millisecond))
}
func (e droidEngine) setRate(r float64) { android.CallVoid(class, "speed", "(IF)V", e.id, float32(r)) }
func (e droidEngine) setVolume(v float64) {
	android.CallVoid(class, "volume", "(IF)V", e.id, float32(v))
}

func (e droidEngine) position() time.Duration {
	ms, _ := android.CallInt(class, "position", "(I)I", e.id)
	return time.Duration(ms) * time.Millisecond
}

func (e droidEngine) close() {
	android.CallVoid(class, "close", "(I)V", e.id)
	droidPlayers.Lock()
	delete(droidPlayers.m, e.id)
	droidPlayers.Unlock()
}

func direct(env *C.JNIEnv, buf C.jobject) []byte {
	n := int(C.giodeDirectCapacity(env, buf))
	p := C.giodeDirectAddress(env, buf)
	if p == nil || n <= 0 {
		return nil
	}
	return unsafe.Slice((*byte)(p), n)
}

//export Java_org_giode_media_media_1android_onReady
func Java_org_giode_media_media_1android_onReady(env *C.JNIEnv, cls C.jclass, id, ms, w, h C.jint) {
	if p := droidLookup(id); p != nil {
		go p.ready(time.Duration(ms)*time.Millisecond, image.Pt(int(w), int(h)))
	}
}

//export Java_org_giode_media_media_1android_onFrame
func Java_org_giode_media_media_1android_onFrame(env *C.JNIEnv, cls C.jclass, id C.jint, y, u, v C.jobject, w, h, yStride, uvStride, uvPixStride C.jint) {
	p := droidLookup(id)
	if p == nil {
		return
	}
	// The planes are only valid during the call; PushI420 copies.
	p.surface.PushI420(video.I420{
		Y: direct(env, y), U: direct(env, u), V: direct(env, v),
		StrideY: int(yStride), StrideU: int(uvStride), StrideV: int(uvStride), PixelStrideUV: int(uvPixStride),
		Width: int(w), Height: int(h),
	})
}

//export Java_org_giode_media_media_1android_onEnded
func Java_org_giode_media_media_1android_onEnded(env *C.JNIEnv, cls C.jclass, id C.jint) {
	if p := droidLookup(id); p != nil {
		go p.ended()
	}
}

//export Java_org_giode_media_media_1android_onError
func Java_org_giode_media_media_1android_onError(env *C.JNIEnv, cls C.jclass, id C.jint, msg C.jstring) {
	if p := droidLookup(id); p != nil {
		err := errors.New("media: " + bridge.GoString(unsafe.Pointer(env), uintptr(msg)))
		go p.failed(err)
	}
}

// silentForTests is unused on Android.
var silentForTests bool
