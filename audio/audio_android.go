package audio

/*
#include <jni.h>
#include <stdint.h>

static jsize giodeArrayLen(JNIEnv *env, jarray a) {
	return (*env)->GetArrayLength(env, a);
}

static void giodeGetShorts(JNIEnv *env, jshortArray a, jsize n, int16_t *dst) {
	(*env)->GetShortArrayRegion(env, a, 0, n, (jshort *)dst);
}

static void giodeSetShorts(JNIEnv *env, jshortArray a, jsize n, const int16_t *src) {
	(*env)->SetShortArrayRegion(env, a, 0, n, (const jshort *)src);
}
*/
import "C"

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"unsafe"

	_ "gioui.org/app/permission/microphone"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/android"
	"github.com/xmhrnx/giode/internal/bridge"
)

//go:generate javac --release 8 -classpath $ANDROID_HOME/platforms/android-34/android.jar -d $TEMP/giode_audio/classes audio_android.java
//go:generate jar cf audio_android.jar -C $TEMP/giode_audio/classes .

const class = "org/giode/audio/audio_android"

var permissions bridge.Requests[bool]

func requestPermission(ctx context.Context, a *giode.App) (bool, error) {
	ok, err := android.CallBool(class, "hasPermission", "(Landroid/content/Context;)Z", android.Context())
	if err != nil || ok {
		return ok, err
	}
	if a == nil {
		return false, errors.New("audio: asking for the microphone needs the app")
	}
	view := a.NativeHandle()
	if view == 0 {
		return false, errors.New("audio: the window isn't open yet")
	}
	id, ch := permissions.New()
	if err := android.CallVoid(class, "requestPermission", "(Landroid/view/View;I)V", view, id); err != nil {
		permissions.Cancel(id)
		return false, err
	}
	return permissions.Wait(ctx, id, ch)
}

//export Java_org_giode_audio_audio_1android_onPermission
func Java_org_giode_audio_audio_1android_onPermission(env *C.JNIEnv, cls C.jclass, req C.jint, granted C.jboolean) {
	permissions.Done(int(req), granted != 0)
}

func devices(ctx context.Context, input bool) ([]Device, error) {
	s, err := android.CallString(class, "devices", "(Landroid/content/Context;Z)Ljava/lang/String;", android.Context(), input)
	if err != nil {
		return nil, err
	}
	var out []Device
	for _, line := range strings.Split(s, "\n") {
		f := strings.Split(line, "\t")
		if len(f) == 3 {
			out = append(out, Device{ID: f[0], Name: f[1], Default: f[2] == "1"})
		}
	}
	return out, nil
}

// The open streams, by id, for the Java callbacks.
var streams struct {
	sync.Mutex
	next    int
	inputs  map[int]*Input
	outputs map[int]*Output
}

func register(in *Input, out *Output) int {
	streams.Lock()
	defer streams.Unlock()
	if streams.inputs == nil {
		streams.inputs, streams.outputs = map[int]*Input{}, map[int]*Output{}
	}
	streams.next++
	id := streams.next
	if in != nil {
		streams.inputs[id] = in
	}
	if out != nil {
		streams.outputs[id] = out
	}
	return id
}

func unregister(id int) {
	streams.Lock()
	delete(streams.inputs, id)
	delete(streams.outputs, id)
	streams.Unlock()
}

//export Java_org_giode_audio_audio_1android_onAudio
func Java_org_giode_audio_audio_1android_onAudio(env *C.JNIEnv, cls C.jclass, id C.jint, data C.jshortArray, n C.jint) {
	streams.Lock()
	in := streams.inputs[int(id)]
	streams.Unlock()
	if in == nil || n <= 0 {
		return
	}
	buf := make([]int16, int(n))
	C.giodeGetShorts(env, data, n, (*C.int16_t)(unsafe.Pointer(&buf[0])))
	in.deliver(buf)
}

//export Java_org_giode_audio_audio_1android_onPull
func Java_org_giode_audio_audio_1android_onPull(env *C.JNIEnv, cls C.jclass, id C.jint, buf C.jshortArray) C.jint {
	streams.Lock()
	out := streams.outputs[int(id)]
	streams.Unlock()
	if out == nil {
		return 0
	}
	n := int(C.giodeArrayLen(env, C.jarray(buf)))
	s := make([]int16, n)
	out.pull(s)
	C.giodeSetShorts(env, buf, C.jsize(n), (*C.int16_t)(unsafe.Pointer(&s[0])))
	return C.jint(n)
}

func deviceID(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

type platformInput struct{ id int }

func (in *Input) open(ctx context.Context, _ *giode.App) error {
	id := register(in, nil)
	msg, err := android.CallString(class, "openInput", "(Landroid/content/Context;IIIZII)Ljava/lang/String;",
		android.Context(), id, in.opts.SampleRate, in.opts.Channels, in.opts.Voice, deviceID(in.opts.Device), in.opts.frameSamples())
	if err == nil && msg != "" {
		err = errors.New("audio: " + msg)
	}
	if err != nil {
		unregister(id)
		return err
	}
	in.plat = platformInput{id}
	return nil
}

func (p platformInput) close() error {
	err := android.CallVoid(class, "close", "(Landroid/content/Context;I)V", android.Context(), p.id)
	unregister(p.id)
	return err
}

type platformOutput struct{ id int }

func (o *Output) open(ctx context.Context, _ *giode.App) error {
	id := register(nil, o)
	msg, err := android.CallString(class, "openOutput", "(Landroid/content/Context;IIIZZI)Ljava/lang/String;",
		android.Context(), id, o.opts.SampleRate, o.opts.Channels, o.opts.Voice, o.opts.SpeakerPhone, deviceID(o.opts.Device))
	if err == nil && msg != "" {
		err = errors.New("audio: " + msg)
	}
	if err != nil {
		unregister(id)
		return err
	}
	o.plat = platformOutput{id}
	return nil
}

func (p platformOutput) close() error {
	err := android.CallVoid(class, "close", "(Landroid/content/Context;I)V", android.Context(), p.id)
	unregister(p.id)
	return err
}
