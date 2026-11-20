package background

/*
#include <jni.h>
*/
import "C"

import (
	"context"
	"errors"
	"sync"
	"unsafe"

	"github.com/xmhrnx/giode/android"
)

//go:generate javac --release 8 -classpath $ANDROID_HOME/platforms/android-34/android.jar -d $TEMP/giode_background/classes background_android.java BackgroundJobService.java
//go:generate jar cf background_android.jar -C $TEMP/giode_background/classes .

const (
	class    = "org/giode/background/background_android"
	jobClass = "org/giode/background/BackgroundJobService"
)

var runs = struct {
	sync.Mutex
	cancel map[int]context.CancelFunc
}{cancel: map[int]context.CancelFunc{}}

func schedule(name string, o Options) error {
	msg, err := android.CallString(class, "schedule", "(Landroid/content/Context;Ljava/lang/String;JJZZ)Ljava/lang/String;",
		android.Context(), name, o.Interval.Milliseconds(), o.Delay.Milliseconds(), o.Network, o.Charging)
	if err != nil {
		return err
	}
	if msg != "" {
		return errors.New("background: " + msg)
	}
	return nil
}

func cancel(name string) error {
	return android.CallVoid(class, "cancel", "(Landroid/content/Context;Ljava/lang/String;)V", android.Context(), name)
}

//export Java_org_giode_background_BackgroundJobService_run
func Java_org_giode_background_BackgroundJobService_run(env *C.JNIEnv, cls C.jclass, rid C.jint, name C.jstring) {
	n := android.GoString(unsafe.Pointer(env), uintptr(name))
	id := int(rid)
	ctx, stop := context.WithCancel(context.Background())
	runs.Lock()
	runs.cancel[id] = stop
	runs.Unlock()
	go func() {
		err := run(ctx, n)
		runs.Lock()
		_, live := runs.cancel[id]
		delete(runs.cancel, id)
		runs.Unlock()
		stop()
		if live {
			android.CallVoid(jobClass, "done", "(IZ)V", id, err != nil && !errors.Is(err, ErrUnknown))
		}
	}()
}

//export Java_org_giode_background_BackgroundJobService_stop
func Java_org_giode_background_BackgroundJobService_stop(env *C.JNIEnv, cls C.jclass, rid C.jint) {
	runs.Lock()
	stop := runs.cancel[int(rid)]
	delete(runs.cancel, int(rid))
	runs.Unlock()
	if stop != nil {
		stop()
	}
}
