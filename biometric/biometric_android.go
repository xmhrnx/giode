package biometric

/*
#include <jni.h>
*/
import "C"

import (
	"context"
	"errors"
	"unsafe"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/android"
	"github.com/xmhrnx/giode/internal/bridge"
)

//go:generate javac --release 8 -classpath $ANDROID_HOME/platforms/android-34/android.jar -d $TEMP/giode_biometric/classes biometric_android.java
//go:generate jar cf biometric_android.jar -C $TEMP/giode_biometric/classes .

const (
	supported = true
	class     = "org/giode/biometric/biometric_android"
)

var requests bridge.Requests[error]

func available() Kind {
	n, err := android.CallInt(class, "available", "(Landroid/content/Context;)I", android.Context())
	if err != nil {
		return None
	}
	return Kind(n)
}

func authenticate(ctx context.Context, a *giode.App, opts Options) error {
	view := a.NativeHandle()
	if view == 0 {
		return errors.New("biometric: the window isn't open yet")
	}
	id, ch := requests.New()
	if err := android.CallVoid(class, "authenticate", "(Landroid/view/View;ILjava/lang/String;Ljava/lang/String;Ljava/lang/String;Z)V",
		view, id, opts.Title, opts.Reason, opts.Cancel, opts.AllowPasscode); err != nil {
		requests.Cancel(id)
		return err
	}
	err, werr := requests.Wait(ctx, id, ch)
	if werr != nil {
		return werr
	}
	return err
}

//export Java_org_giode_biometric_biometric_1android_onResult
func Java_org_giode_biometric_biometric_1android_onResult(env *C.JNIEnv, cls C.jclass, id, code C.jint, msg C.jstring) {
	requests.Done(int(id), resultError(int(code), bridge.GoString(unsafe.Pointer(env), uintptr(msg))))
}
