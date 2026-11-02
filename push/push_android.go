package push

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

//go:generate sh -c "javac --release 8 -nowarn -classpath $ANDROID_HOME/platforms/android-34/android.jar -d $TEMP/giode_push/stubs $(find ../internal/androidstubs -name '*.java')"
//go:generate javac --release 8 -classpath $ANDROID_HOME/platforms/android-34/android.jar:$TEMP/giode_push/stubs -d $TEMP/giode_push/classes push_android.java PushService.java
//go:generate jar cf push_android.jar -C $TEMP/giode_push/classes .

const class = "org/giode/push/push_android"

type tokenResult struct {
	token string
	err   error
}

var (
	tokens      bridge.Requests[tokenResult]
	permissions bridge.Requests[bool]
)

func start(a *giode.App) error {
	if err := bridge.CallVoid(class, "start", "()V"); err != nil {
		return err
	}
	return checkOpened(a)
}

// checkOpened delivers the message whose notification launched the
// app.
func checkOpened(a *giode.App) error {
	view := a.NativeHandle()
	if view == 0 {
		return nil
	}
	js, err := bridge.CallString(class, "openedMessage", "(Landroid/view/View;)Ljava/lang/String;", view)
	if err != nil || js == "" {
		return err
	}
	received(js, true)
	return nil
}

func token(ctx context.Context) (string, error) {
	id, ch := tokens.New()
	if err := bridge.CallVoid(class, "getToken", "(I)V", id); err != nil {
		tokens.Cancel(id)
		return "", err
	}
	r, err := tokens.Wait(ctx, id, ch)
	if err != nil {
		return "", err
	}
	return r.token, r.err
}

func requestPermission(ctx context.Context, a *giode.App) (bool, error) {
	view := a.NativeHandle()
	if view == 0 {
		return false, errors.New("push: the window isn't open yet")
	}
	id, ch := permissions.New()
	if err := bridge.CallVoid(class, "requestPermission", "(Landroid/view/View;I)V", view, id); err != nil {
		permissions.Cancel(id)
		return false, err
	}
	return permissions.Wait(ctx, id, ch)
}

func unregister() error {
	return bridge.CallVoid(class, "deleteToken", "()V")
}

//export Java_org_giode_push_push_1android_onToken
func Java_org_giode_push_push_1android_onToken(env *C.JNIEnv, cls C.jclass, tok C.jstring) {
	deliver(event{kind: evToken, token: bridge.GoString(unsafe.Pointer(env), uintptr(tok))})
}

//export Java_org_giode_push_push_1android_onMessage
func Java_org_giode_push_push_1android_onMessage(env *C.JNIEnv, cls C.jclass, js C.jstring, opened C.jboolean) {
	received(bridge.GoString(unsafe.Pointer(env), uintptr(js)), opened != 0)
}

//export Java_org_giode_push_push_1android_onTokenResult
func Java_org_giode_push_push_1android_onTokenResult(env *C.JNIEnv, cls C.jclass, id C.jint, tok, errStr C.jstring) {
	r := tokenResult{token: bridge.GoString(unsafe.Pointer(env), uintptr(tok))}
	if msg := bridge.GoString(unsafe.Pointer(env), uintptr(errStr)); msg != "" {
		r.err = errors.New("push: " + msg)
	}
	tokens.Done(int(id), r)
}

//export Java_org_giode_push_push_1android_onPermission
func Java_org_giode_push_push_1android_onPermission(env *C.JNIEnv, cls C.jclass, id C.jint, granted C.jboolean) {
	permissions.Done(int(id), granted != 0)
}
