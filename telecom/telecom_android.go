package telecom

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

//go:generate javac --release 8 -classpath $ANDROID_HOME/platforms/android-34/android.jar -d $TEMP/giode_telecom/classes telecom_android.java GiodeConnectionService.java
//go:generate jar cf telecom_android.jar -C $TEMP/giode_telecom/classes .

const (
	supported = true
	class     = "org/giode/telecom/telecom_android"
)

func result(msg string, err error) error {
	if err != nil {
		return err
	}
	if msg != "" {
		return errors.New("telecom: " + msg)
	}
	return nil
}

func setup(*giode.App) error {
	return result(android.CallString(class, "setup", "(Landroid/content/Context;Z)Ljava/lang/String;", android.Context(), true))
}

const callSig = "(Ljava/lang/String;Ljava/lang/String;Ljava/lang/String;Z)Ljava/lang/String;"

func reportIncoming(ctx context.Context, c Call) error {
	return result(android.CallString(class, "incoming", callSig, c.ID, c.Handle, c.Name, c.Video))
}

func startOutgoing(c Call) error {
	return result(android.CallString(class, "outgoing", callSig, c.ID, c.Handle, c.Name, c.Video))
}

func reportConnected(id string) error {
	return android.CallVoid(class, "connected", "(Ljava/lang/String;)V", id)
}

func end(id string) error {
	return android.CallVoid(class, "end", "(Ljava/lang/String;I)V", id, -1)
}

func reportEnded(id string, r Reason) error {
	return android.CallVoid(class, "end", "(Ljava/lang/String;I)V", id, int(r))
}

func setMuted(id string, muted bool) error {
	return android.CallVoid(class, "mute", "(Ljava/lang/String;Z)V", id, muted)
}

//export Java_org_giode_telecom_telecom_1android_onEvent
func Java_org_giode_telecom_telecom_1android_onEvent(env *C.JNIEnv, cls C.jclass, kind, a, b C.jstring) {
	e := unsafe.Pointer(env)
	dispatch(bridge.GoString(e, uintptr(kind)), bridge.GoString(e, uintptr(a)), bridge.GoString(e, uintptr(b)))
}
