package ocr

/*
#include <jni.h>
*/
import "C"

import (
	"context"
	"errors"
	"unsafe"

	"github.com/xmhrnx/giode/android"
	"github.com/xmhrnx/giode/internal/bridge"
)

//go:generate sh -c "javac --release 8 -nowarn -classpath $ANDROID_HOME/platforms/android-34/android.jar -d $TEMP/giode_ocr/stubs $(find ../internal/androidstubs -name '*.java')"
//go:generate javac --release 8 -classpath $ANDROID_HOME/platforms/android-34/android.jar:$TEMP/giode_ocr/stubs -d $TEMP/giode_ocr/classes ocr_android.java
//go:generate jar cf ocr_android.jar -C $TEMP/giode_ocr/classes .

const (
	supported = true
	class     = "org/giode/ocr/ocr_android"
)

type answer struct {
	json []byte
	err  error
}

var requests bridge.Requests[answer]

func recognize(ctx context.Context, data []byte, opts Options) ([]byte, error) {
	id, ch := requests.New()
	if err := android.CallVoid(class, "recognize", "(I[B)V", id, data); err != nil {
		requests.Cancel(id)
		return nil, err
	}
	a, err := requests.Wait(ctx, id, ch)
	if err != nil {
		return nil, err
	}
	return a.json, a.err
}

//export Java_org_giode_ocr_ocr_1android_onResult
func Java_org_giode_ocr_ocr_1android_onResult(env *C.JNIEnv, cls C.jclass, id C.jint, js, errStr C.jstring) {
	var a answer
	if msg := bridge.GoString(unsafe.Pointer(env), uintptr(errStr)); msg != "" {
		a.err = errors.New("ocr: " + msg)
	} else {
		a.json = []byte(bridge.GoString(unsafe.Pointer(env), uintptr(js)))
	}
	requests.Done(int(id), a)
}
