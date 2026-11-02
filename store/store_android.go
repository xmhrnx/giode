package store

/*
#include <jni.h>
*/
import "C"

import (
	"context"
	"encoding/json"
	"errors"
	"unsafe"

	"gioui.org/app"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/internal/bridge"
)

//go:generate sh -c "javac --release 8 -nowarn -classpath $ANDROID_HOME/platforms/android-34/android.jar -d $TEMP/giode_store/stubs $(find ../internal/androidstubs -name '*.java')"
//go:generate javac --release 8 -classpath $ANDROID_HOME/platforms/android-34/android.jar:$TEMP/giode_store/stubs -d $TEMP/giode_store/classes store_android.java
//go:generate jar cf store_android.jar -C $TEMP/giode_store/classes .

const class = "org/giode/store/store_android"

var requests bridge.Requests[result]

// call runs a Java request taking (context or view, id, args...) and
// waits for its answer.
func call(ctx context.Context, method, sig string, target uintptr, args ...any) (string, error) {
	id, ch := requests.New()
	if err := bridge.CallVoid(class, method, sig, append([]any{target, id}, args...)...); err != nil {
		requests.Cancel(id)
		return "", err
	}
	r, err := requests.Wait(ctx, id, ch)
	if err != nil {
		return "", err
	}
	return r.json, r.err
}

func products(ctx context.Context, ids []string) ([]Product, error) {
	b, _ := json.Marshal(ids)
	js, err := call(ctx, "products", "(Landroid/content/Context;ILjava/lang/String;)V", app.AppContext(), string(b))
	if err != nil {
		return nil, err
	}
	return parseProducts(js)
}

func purchase(ctx context.Context, a *giode.App, p Product) (Purchase, error) {
	view := a.NativeHandle()
	if view == 0 {
		return Purchase{}, errors.New("store: the window isn't open yet")
	}
	js, err := call(ctx, "purchase", "(Landroid/view/View;ILjava/lang/String;)V", view, p.ID)
	if err != nil {
		return Purchase{}, err
	}
	return parsePurchase(js)
}

func finish(ctx context.Context, p Purchase, consume bool) error {
	_, err := call(ctx, "finish", "(Landroid/content/Context;ILjava/lang/String;Z)V", app.AppContext(), p.Token, consume)
	return err
}

func owned(ctx context.Context) ([]Purchase, error) {
	js, err := call(ctx, "owned", "(Landroid/content/Context;I)V", app.AppContext())
	if err != nil {
		return nil, err
	}
	return parsePurchases(js)
}

// startUpdates does nothing: Play Billing reports updates to the
// client's listener, set up with the first request.
func startUpdates() {}

//export Java_org_giode_store_store_1android_onResult
func Java_org_giode_store_store_1android_onResult(env *C.JNIEnv, cls C.jclass, id C.jint, js, errStr C.jstring) {
	requests.Done(int(id), result{
		json: bridge.GoString(unsafe.Pointer(env), uintptr(js)),
		err:  nativeError(bridge.GoString(unsafe.Pointer(env), uintptr(errStr))),
	})
}

//export Java_org_giode_store_store_1android_onUpdate
func Java_org_giode_store_store_1android_onUpdate(env *C.JNIEnv, cls C.jclass, js C.jstring) {
	if p, err := parsePurchase(bridge.GoString(unsafe.Pointer(env), uintptr(js))); err == nil {
		update(p)
	}
}
