package review

/*
#include <jni.h>
*/
import "C"

import (
	"context"
	"errors"
	"unsafe"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/internal/bridge"
)

//go:generate sh -c "javac --release 8 -nowarn -classpath $ANDROID_HOME/platforms/android-34/android.jar -d $TEMP/giode_review/stubs $(find ../internal/androidstubs -name '*.java')"
//go:generate javac --release 8 -classpath $ANDROID_HOME/platforms/android-34/android.jar:$TEMP/giode_review/stubs -d $TEMP/giode_review/classes review_android.java
//go:generate jar cf review_android.jar -C $TEMP/giode_review/classes .

var requests bridge.Requests[error]

func request(ctx context.Context, a *giode.App) error {
	view := a.NativeHandle()
	if view == 0 {
		return errors.New("review: the window isn't open yet")
	}
	id, ch := requests.New()
	if err := bridge.CallVoid("org/giode/review/review_android", "request", "(Landroid/view/View;I)V", view, id); err != nil {
		requests.Cancel(id)
		return err
	}
	err, werr := requests.Wait(ctx, id, ch)
	if werr != nil {
		return werr
	}
	return err
}

//export Java_org_giode_review_review_1android_onDone
func Java_org_giode_review_review_1android_onDone(env *C.JNIEnv, cls C.jclass, id C.jint, errStr C.jstring) {
	var err error
	if msg := bridge.GoString(unsafe.Pointer(env), uintptr(errStr)); msg != "" {
		err = errors.New("review: " + msg)
	}
	requests.Done(int(id), err)
}
