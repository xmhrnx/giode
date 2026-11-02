package printing

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

//go:generate javac --release 8 -classpath $ANDROID_HOME/platforms/android-34/android.jar -d $TEMP/giode_printing/classes printing_android.java
//go:generate jar cf printing_android.jar -C $TEMP/giode_printing/classes .

const (
	supported = true
	class     = "org/giode/printing/printing_android"
)

var requests bridge.Requests[error]

func printFile(ctx context.Context, a *giode.App, path string, opts Options) error {
	if !isPDF(path) {
		return errors.New("printing: Android prints PDFs only")
	}
	view := a.NativeHandle()
	if view == 0 {
		return errors.New("printing: the window isn't open yet")
	}
	id, ch := requests.New()
	if err := android.CallVoid(class, "print", "(Landroid/view/View;ILjava/lang/String;Ljava/lang/String;)V", view, id, path, opts.JobName); err != nil {
		requests.Cancel(id)
		return err
	}
	err, werr := requests.Wait(ctx, id, ch)
	if werr != nil {
		return werr
	}
	return err
}

//export Java_org_giode_printing_printing_1android_onDone
func Java_org_giode_printing_printing_1android_onDone(env *C.JNIEnv, cls C.jclass, id C.jint, msg C.jstring) {
	var err error
	if m := bridge.GoString(unsafe.Pointer(env), uintptr(msg)); m != "" {
		err = errors.New("printing: " + m)
	}
	requests.Done(int(id), err)
}
