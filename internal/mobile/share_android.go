package mobile

/*
#include <jni.h>

static void giodeIntRegion(JNIEnv *env, jintArray a, jsize n, jint *buf) {
	(*env)->GetIntArrayRegion(env, a, 0, n, buf);
}

static void giodeLongRegion(JNIEnv *env, jlongArray a, jsize n, jlong *buf) {
	(*env)->GetLongArrayRegion(env, a, 0, n, buf);
}

static jsize giodeShareArrayLength(JNIEnv *env, jarray a) {
	return (*env)->GetArrayLength(env, a);
}
*/
import "C"

import (
	"os"
	"sync"
	"unsafe"

	"git.wow.st/gmp/jni"
)

const shareClass = "org/giode/mobile/ShareActivity"

var (
	shareMu      sync.Mutex
	shareHandler func(Shared)
)

// ListenShares hands what other apps share with this one (through
// ShareActivity, declared by giode build's app.share_target) to handler:
// the shares received before the call, then each as it arrives. handler
// runs on the Android main thread and owns the files.
func ListenShares(vm, ctx uintptr, handler func(Shared)) error {
	shareMu.Lock()
	shareHandler = handler
	shareMu.Unlock()
	return callClass(vm, ctx, shareClass, func(env jni.Env, cls jni.Class) error {
		m := jni.GetStaticMethodID(env, cls, "listen", "()V")
		return jni.CallStaticVoidMethod(env, cls, m)
	})
}

//export Java_org_giode_mobile_ShareActivity_onShared
func Java_org_giode_mobile_ShareActivity_onShared(cenv *C.JNIEnv, cls C.jclass, text, subject C.jstring, fds C.jintArray, names, mimes C.jobjectArray, sizes C.jlongArray) {
	env := jni.EnvFor(uintptr(unsafe.Pointer(cenv)))
	s := Shared{Text: goString(env, text), Subject: goString(env, subject)}
	n := int(C.giodeShareArrayLength(cenv, C.jarray(fds)))
	if n > 0 {
		fdBuf := make([]C.jint, n)
		sizeBuf := make([]C.jlong, n)
		C.giodeIntRegion(cenv, fds, C.jsize(n), &fdBuf[0])
		C.giodeLongRegion(cenv, sizes, C.jsize(n), &sizeBuf[0])
		for i := 0; i < n; i++ {
			no, _ := jni.GetObjectArrayElement(env, jni.ObjectArray(names), jni.Size(i))
			mo, _ := jni.GetObjectArrayElement(env, jni.ObjectArray(mimes), jni.Size(i))
			name := jni.GoString(env, jni.String(no))
			s.Files = append(s.Files, SharedFile{
				File: os.NewFile(uintptr(fdBuf[i]), name),
				Name: name,
				MIME: jni.GoString(env, jni.String(mo)),
				Size: int64(sizeBuf[i]),
			})
			jni.DeleteLocalRef(env, no)
			jni.DeleteLocalRef(env, mo)
		}
	}
	shareMu.Lock()
	h := shareHandler
	shareMu.Unlock()
	if h == nil {
		for _, f := range s.Files {
			f.File.Close()
		}
		return
	}
	h(s)
}
