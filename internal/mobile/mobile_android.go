package mobile

/*
#include <jni.h>

static jsize giodeArrayLength(JNIEnv *env, jarray a) {
	return (*env)->GetArrayLength(env, a);
}

static void giodeIntArray(JNIEnv *env, jintArray a, jint *out, jsize n) {
	(*env)->GetIntArrayRegion(env, a, 0, n, out);
}

static void giodeLongArray(JNIEnv *env, jlongArray a, jlong *out, jsize n) {
	(*env)->GetLongArrayRegion(env, a, 0, n, out);
}
*/
import "C"

import (
	"errors"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"git.wow.st/gmp/jni"
)

//go:generate javac --release 8 -classpath $ANDROID_HOME/platforms/android-34/android.jar -d $TEMP/giode_mobile/classes mobile_android.java services_android.java picker_android.java KeepAliveService.java ShareActivity.java
//go:generate jar cf mobile_android.jar -C $TEMP/giode_mobile/classes .

const (
	helperClass    = "org/giode/mobile/mobile_android"
	servicesClass  = "org/giode/mobile/services_android"
	pickerClass    = "org/giode/mobile/picker_android"
	keepAliveClass = "org/giode/mobile/KeepAliveService"
)

// call loads the helper class through the app's class loader and runs
// f with it on the current thread, attached to the JVM.
func call(vm, ctx uintptr, f func(env jni.Env, cls jni.Class) error) error {
	return callClass(vm, ctx, helperClass, f)
}

// callClass is call for one of the helper's classes.
func callClass(vm, ctx uintptr, class string, f func(env jni.Env, cls jni.Class) error) error {
	return jni.Do(jni.JVMFor(vm), func(env jni.Env) error {
		cls, err := jni.LoadClass(env, jni.ClassLoaderFor(env, jni.Object(ctx)), class)
		if err != nil {
			return err
		}
		return f(env, cls)
	})
}

// WatchLifecycle starts reporting background and foreground changes to
// the handlers set with SetLifecycleHandlers. ctx is the Android
// application context. Calling it again does nothing.
func WatchLifecycle(vm, ctx uintptr) error {
	return call(vm, ctx, func(env jni.Env, cls jni.Class) error {
		m := jni.GetStaticMethodID(env, cls, "watchLifecycle", "(Landroid/content/Context;)V")
		return jni.CallStaticVoidMethod(env, cls, m, jni.Value(ctx))
	})
}

// Publish copies the file at path into the shared Downloads collection
// and returns its content URI, readable by the app it's shared with.
// It copies the whole file, so call it off the main thread. Android 10
// or later.
func Publish(vm, ctx uintptr, path, mime string) (string, error) {
	var uri string
	err := call(vm, ctx, func(env jni.Env, cls jni.Class) error {
		m := jni.GetStaticMethodID(env, cls, "publish", "(Landroid/content/Context;Ljava/lang/String;Ljava/lang/String;)Ljava/lang/String;")
		obj, err := jni.CallStaticObjectMethod(env, cls, m,
			jni.Value(ctx),
			jni.Value(jni.JavaString(env, path)),
			jni.Value(jni.JavaString(env, mime)),
		)
		if err != nil {
			return err
		}
		uri = jni.GoString(env, jni.String(obj))
		return nil
	})
	return uri, err
}

// ShareText opens the share sheet for text from view, the GioView.
// Call it on the main thread.
func ShareText(vm, ctx, view uintptr, text, subject, title string) error {
	return call(vm, ctx, func(env jni.Env, cls jni.Class) error {
		m := jni.GetStaticMethodID(env, cls, "shareText", "(Landroid/view/View;Ljava/lang/String;Ljava/lang/String;Ljava/lang/String;)V")
		return jni.CallStaticVoidMethod(env, cls, m,
			jni.Value(view),
			jni.Value(jni.JavaString(env, text)),
			jni.Value(jni.JavaString(env, subject)),
			jni.Value(jni.JavaString(env, title)),
		)
	})
}

// ShareURI opens the share sheet for a URI returned by Publish. Call
// it on the main thread.
func ShareURI(vm, ctx, view uintptr, uri, mime, title string) error {
	return call(vm, ctx, func(env jni.Env, cls jni.Class) error {
		m := jni.GetStaticMethodID(env, cls, "shareURI", "(Landroid/view/View;Ljava/lang/String;Ljava/lang/String;Ljava/lang/String;)V")
		return jni.CallStaticVoidMethod(env, cls, m,
			jni.Value(view),
			jni.Value(jni.JavaString(env, uri)),
			jni.Value(jni.JavaString(env, mime)),
			jni.Value(jni.JavaString(env, title)),
		)
	})
}

// ShareURIs opens the share sheet for several items published with
// Publish. Call it on the main thread.
func ShareURIs(vm, ctx, view uintptr, uris []string, mime, title string) error {
	return call(vm, ctx, func(env jni.Env, cls jni.Class) error {
		m := jni.GetStaticMethodID(env, cls, "shareURIs", "(Landroid/view/View;Ljava/lang/String;Ljava/lang/String;Ljava/lang/String;)V")
		return jni.CallStaticVoidMethod(env, cls, m,
			jni.Value(view),
			jni.Value(jni.JavaString(env, strings.Join(uris, "\n"))),
			jni.Value(jni.JavaString(env, mime)),
			jni.Value(jni.JavaString(env, title)),
		)
	})
}

// SaveToDownloads copies the file at path into the shared Downloads
// collection, in folder when it isn't empty, and returns its content
// URI. Off the main thread; Android 10 or later.
func SaveToDownloads(vm, ctx uintptr, path, mime, folder string) (string, error) {
	var uri string
	err := call(vm, ctx, func(env jni.Env, cls jni.Class) error {
		m := jni.GetStaticMethodID(env, cls, "saveToDownloads", "(Landroid/content/Context;Ljava/lang/String;Ljava/lang/String;Ljava/lang/String;)Ljava/lang/String;")
		obj, err := jni.CallStaticObjectMethod(env, cls, m,
			jni.Value(ctx),
			jni.Value(jni.JavaString(env, path)),
			jni.Value(jni.JavaString(env, mime)),
			jni.Value(jni.JavaString(env, folder)),
		)
		if err != nil {
			return err
		}
		uri = jni.GoString(env, jni.String(obj))
		return nil
	})
	return uri, err
}

// OpenDownloads shows the Downloads app. Call it on the main thread.
func OpenDownloads(vm, ctx, view uintptr) error {
	return call(vm, ctx, func(env jni.Env, cls jni.Class) error {
		m := jni.GetStaticMethodID(env, cls, "openDownloads", "(Landroid/view/View;)V")
		return jni.CallStaticVoidMethod(env, cls, m, jni.Value(view))
	})
}

// SaveToGallery copies the picture or video at path into the shared
// Pictures (or Movies) collection, in album's folder. It copies the
// whole file, so call it off the main thread. Android 10 or later.
func SaveToGallery(vm, ctx uintptr, path, mime, album string) error {
	return call(vm, ctx, func(env jni.Env, cls jni.Class) error {
		m := jni.GetStaticMethodID(env, cls, "saveToGallery", "(Landroid/content/Context;Ljava/lang/String;Ljava/lang/String;Ljava/lang/String;)Ljava/lang/String;")
		_, err := jni.CallStaticObjectMethod(env, cls, m,
			jni.Value(ctx),
			jni.Value(jni.JavaString(env, path)),
			jni.Value(jni.JavaString(env, mime)),
			jni.Value(jni.JavaString(env, album)),
		)
		return err
	})
}

//export Java_org_giode_mobile_mobile_1android_onBackground
func Java_org_giode_mobile_mobile_1android_onBackground(env *C.JNIEnv, cls C.jclass) {
	notify(&onBackground)
}

//export Java_org_giode_mobile_mobile_1android_onForeground
func Java_org_giode_mobile_mobile_1android_onForeground(env *C.JNIEnv, cls C.jclass) {
	notify(&onForeground)
}

// OpenURL opens url in the app that handles it. Call it on the main
// thread.
func OpenURL(vm, ctx, view uintptr, url string) error {
	return callClass(vm, ctx, servicesClass, func(env jni.Env, cls jni.Class) error {
		m := jni.GetStaticMethodID(env, cls, "openURL", "(Landroid/view/View;Ljava/lang/String;)V")
		return jni.CallStaticVoidMethod(env, cls, m, jni.Value(view), jni.Value(jni.JavaString(env, url)))
	})
}

// Haptic performs the android.view.HapticFeedbackConstants feedback c
// on view. Call it on the main thread.
func Haptic(vm, ctx, view uintptr, c int) error {
	return callClass(vm, ctx, servicesClass, func(env jni.Env, cls jni.Class) error {
		m := jni.GetStaticMethodID(env, cls, "haptic", "(Landroid/view/View;I)Z")
		_, err := jni.CallStaticBooleanMethod(env, cls, m, jni.Value(view), jni.Value(c))
		return err
	})
}

// KeepScreenOn keeps the screen on while the app is shown. Call it on
// the main thread.
func KeepScreenOn(vm, ctx, view uintptr, on bool) error {
	return callClass(vm, ctx, servicesClass, func(env jni.Env, cls jni.Class) error {
		m := jni.GetStaticMethodID(env, cls, "keepScreenOn", "(Landroid/view/View;Z)V")
		v := 0
		if on {
			v = 1
		}
		return jni.CallStaticVoidMethod(env, cls, m, jni.Value(view), jni.Value(v))
	})
}

// KeepAliveDeclared reports whether the app's manifest declares
// giode's foreground service, which StartKeepAlive needs.
func KeepAliveDeclared(vm, ctx uintptr) (bool, error) {
	var ok bool
	err := callClass(vm, ctx, keepAliveClass, func(env jni.Env, cls jni.Class) error {
		m := jni.GetStaticMethodID(env, cls, "declared", "(Landroid/content/Context;)Z")
		var err error
		ok, err = jni.CallStaticBooleanMethod(env, cls, m, jni.Value(ctx))
		return err
	})
	return ok, err
}

// StartKeepAlive starts giode's foreground service with reason in its
// notification, or updates the notification when it's running. Call
// it while the app is in the foreground.
func StartKeepAlive(vm, ctx uintptr, reason string) error {
	return callClass(vm, ctx, keepAliveClass, func(env jni.Env, cls jni.Class) error {
		m := jni.GetStaticMethodID(env, cls, "start", "(Landroid/content/Context;Ljava/lang/String;)V")
		return jni.CallStaticVoidMethod(env, cls, m, jni.Value(ctx), jni.Value(jni.JavaString(env, reason)))
	})
}

// StopKeepAlive stops the foreground service.
func StopKeepAlive(vm, ctx uintptr) error {
	return callClass(vm, ctx, keepAliveClass, func(env jni.Env, cls jni.Class) error {
		m := jni.GetStaticMethodID(env, cls, "stop", "(Landroid/content/Context;)V")
		return jni.CallStaticVoidMethod(env, cls, m, jni.Value(ctx))
	})
}

// PlaySound plays 16-bit PCM samples once, as a notification sound.
// Any thread.
func PlaySound(vm, ctx uintptr, pcm []byte, rate, channels int) error {
	return callClass(vm, ctx, servicesClass, func(env jni.Env, cls jni.Class) error {
		m := jni.GetStaticMethodID(env, cls, "playSound", "(Landroid/content/Context;[BII)V")
		return jni.CallStaticVoidMethod(env, cls, m, jni.Value(ctx), jni.Value(jni.NewByteArray(env, pcm)), jni.Value(rate), jni.Value(channels))
	})
}

// SecureSet stores value under name, encrypted with a key in the
// Android Keystore.
func SecureSet(vm, ctx uintptr, name string, value []byte) error {
	return callClass(vm, ctx, servicesClass, func(env jni.Env, cls jni.Class) error {
		m := jni.GetStaticMethodID(env, cls, "secureSet", "(Landroid/content/Context;Ljava/lang/String;[B)V")
		return jni.CallStaticVoidMethod(env, cls, m, jni.Value(ctx), jni.Value(jni.JavaString(env, name)), jni.Value(jni.NewByteArray(env, value)))
	})
}

// SecureGet returns the value stored under name, and whether there was
// one.
func SecureGet(vm, ctx uintptr, name string) ([]byte, bool, error) {
	var out []byte
	found := false
	err := callClass(vm, ctx, servicesClass, func(env jni.Env, cls jni.Class) error {
		m := jni.GetStaticMethodID(env, cls, "secureGet", "(Landroid/content/Context;Ljava/lang/String;)[B")
		obj, err := jni.CallStaticObjectMethod(env, cls, m, jni.Value(ctx), jni.Value(jni.JavaString(env, name)))
		if err != nil || obj == 0 {
			return err
		}
		found = true
		out = append([]byte(nil), jni.GetByteArrayElements(env, jni.ByteArray(obj))...)
		return nil
	})
	return out, found, err
}

// SecureDelete removes the value stored under name.
func SecureDelete(vm, ctx uintptr, name string) error {
	return callClass(vm, ctx, servicesClass, func(env jni.Env, cls jni.Class) error {
		m := jni.GetStaticMethodID(env, cls, "secureDelete", "(Landroid/content/Context;Ljava/lang/String;)V")
		return jni.CallStaticVoidMethod(env, cls, m, jni.Value(ctx), jni.Value(jni.JavaString(env, name)))
	})
}

// requests tracks pending picker and dictation requests by id.
var (
	reqMu     sync.Mutex
	nextReq   = 0x4000
	photos    = map[int]chan Photo{}
	photoSets = map[int]chan Photos{}
	speech    = map[int]chan Speech{}
	docSets   = map[int]chan Documents{}
	notifyReq = map[int]chan bool{}
)

func newID() int {
	reqMu.Lock()
	defer reqMu.Unlock()
	nextReq = (nextReq + 1) & 0xffff // request codes must fit 16 bits
	if nextReq < 0x4000 {
		nextReq = 0x4000
	}
	return nextReq
}

// PickPhoto opens the photo picker (or the camera) and returns a
// channel receiving the result. Call it on the main thread.
func PickPhoto(vm, ctx, view uintptr, maxSide int, camera bool) (<-chan Photo, error) {
	id := newID()
	ch := make(chan Photo, 1)
	reqMu.Lock()
	photos[id] = ch
	reqMu.Unlock()
	err := callClass(vm, ctx, pickerClass, func(env jni.Env, cls jni.Class) error {
		m := jni.GetStaticMethodID(env, cls, "pickPhoto", "(Landroid/view/View;IIZ)V")
		c := 0
		if camera {
			c = 1
		}
		return jni.CallStaticVoidMethod(env, cls, m, jni.Value(view), jni.Value(id), jni.Value(maxSide), jni.Value(c))
	})
	if err != nil {
		reqMu.Lock()
		delete(photos, id)
		reqMu.Unlock()
		return nil, err
	}
	return ch, nil
}

// PickPhotos opens the photo picker for up to limit photos (0 for no
// limit) and returns a channel receiving them. Call it on the main
// thread.
func PickPhotos(vm, ctx, view uintptr, maxSide, limit int) (<-chan Photos, error) {
	id := newID()
	ch := make(chan Photos, 1)
	reqMu.Lock()
	photoSets[id] = ch
	reqMu.Unlock()
	err := callClass(vm, ctx, pickerClass, func(env jni.Env, cls jni.Class) error {
		m := jni.GetStaticMethodID(env, cls, "pickPhotos", "(Landroid/view/View;III)V")
		return jni.CallStaticVoidMethod(env, cls, m, jni.Value(view), jni.Value(id), jni.Value(maxSide), jni.Value(limit))
	})
	if err != nil {
		reqMu.Lock()
		delete(photoSets, id)
		reqMu.Unlock()
		return nil, err
	}
	return ch, nil
}

// PickDocuments shows the document picker (with media, the system
// Photo Picker where Android has it) for files of the MIME types mimes
// (any when empty), several with multiple, up to limit (0 for no
// limit), and returns a channel receiving them, unchanged, as file
// descriptors. Call it on the main thread.
func PickDocuments(vm, ctx, view uintptr, mimes []string, multiple bool, limit int, media bool) (<-chan Documents, error) {
	id := newID()
	ch := make(chan Documents, 1)
	reqMu.Lock()
	docSets[id] = ch
	reqMu.Unlock()
	err := callClass(vm, ctx, pickerClass, func(env jni.Env, cls jni.Class) error {
		m := jni.GetStaticMethodID(env, cls, "pickDocuments", "(Landroid/view/View;ILjava/lang/String;ZIZ)V")
		return jni.CallStaticVoidMethod(env, cls, m, jni.Value(view), jni.Value(id),
			jni.Value(jni.JavaString(env, strings.Join(mimes, "\n"))), boolValue(multiple), jni.Value(limit), boolValue(media))
	})
	if err != nil {
		reqMu.Lock()
		delete(docSets, id)
		reqMu.Unlock()
		return nil, err
	}
	return ch, nil
}

// closeFD closes a descriptor PickDocuments returned.
func closeFD(fd int) { syscall.Close(fd) }

func boolValue(b bool) jni.Value {
	if b {
		return 1
	}
	return 0
}

//export Java_org_giode_mobile_picker_1android_onDocuments
func Java_org_giode_mobile_picker_1android_onDocuments(cenv *C.JNIEnv, cls C.jclass, id C.jint, fds C.jintArray, names C.jobjectArray, sizes C.jlongArray, errStr C.jstring) {
	env := jni.EnvFor(uintptr(unsafe.Pointer(cenv)))
	reqMu.Lock()
	ch := docSets[int(id)]
	delete(docSets, int(id))
	reqMu.Unlock()
	var ds Documents
	switch msg := goString(env, errStr); {
	case msg != "":
		ds.Err = errors.New(msg)
	case fds == 0:
		ds.Err = ErrCanceled
	default:
		n := C.giodeArrayLength(cenv, C.jarray(fds))
		if n > 0 {
			fdv := make([]C.jint, n)
			sizev := make([]C.jlong, n)
			C.giodeIntArray(cenv, fds, &fdv[0], n)
			C.giodeLongArray(cenv, sizes, &sizev[0], n)
			for i := 0; i < int(n); i++ {
				no, _ := jni.GetObjectArrayElement(env, jni.ObjectArray(names), jni.Size(i))
				ds.Docs = append(ds.Docs, Document{FD: int(fdv[i]), Name: jni.GoString(env, jni.String(no)), Size: int64(sizev[i])})
				jni.DeleteLocalRef(env, no)
			}
		}
	}
	if ch == nil {
		// Nobody waits any more: don't leak the descriptors.
		for _, d := range ds.Docs {
			closeFD(d.FD)
		}
		return
	}
	ch <- ds
}

// NotificationsEnabled reports whether the app may post notifications.
func NotificationsEnabled(vm, ctx uintptr) (bool, error) {
	var ok bool
	err := callClass(vm, ctx, pickerClass, func(env jni.Env, cls jni.Class) error {
		m := jni.GetStaticMethodID(env, cls, "notificationsEnabled", "(Landroid/content/Context;)Z")
		var err error
		ok, err = jni.CallStaticBooleanMethod(env, cls, m, jni.Value(ctx))
		return err
	})
	return ok, err
}

// RequestNotifications asks for the notification permission (Android
// 13 and later) and returns a channel receiving whether notifications
// are enabled then. Call it on the main thread.
func RequestNotifications(vm, ctx, view uintptr) (<-chan bool, error) {
	id := newID()
	ch := make(chan bool, 1)
	reqMu.Lock()
	notifyReq[id] = ch
	reqMu.Unlock()
	err := callClass(vm, ctx, pickerClass, func(env jni.Env, cls jni.Class) error {
		m := jni.GetStaticMethodID(env, cls, "requestNotifications", "(Landroid/view/View;I)V")
		return jni.CallStaticVoidMethod(env, cls, m, jni.Value(view), jni.Value(id))
	})
	if err != nil {
		reqMu.Lock()
		delete(notifyReq, id)
		reqMu.Unlock()
		return nil, err
	}
	return ch, nil
}

//export Java_org_giode_mobile_picker_1android_onNotifyPermission
func Java_org_giode_mobile_picker_1android_onNotifyPermission(cenv *C.JNIEnv, cls C.jclass, id C.jint, enabled C.jboolean) {
	reqMu.Lock()
	ch := notifyReq[int(id)]
	delete(notifyReq, int(id))
	reqMu.Unlock()
	if ch != nil {
		ch <- enabled != 0
	}
}

// Dictate starts speech recognition and returns its id and a channel
// receiving partial and final results. Call it on the main thread.
func Dictate(vm, ctx, view uintptr, lang string) (int, <-chan Speech, error) {
	id := newID()
	ch := make(chan Speech, 16)
	reqMu.Lock()
	speech[id] = ch
	reqMu.Unlock()
	err := callClass(vm, ctx, pickerClass, func(env jni.Env, cls jni.Class) error {
		m := jni.GetStaticMethodID(env, cls, "dictate", "(Landroid/view/View;ILjava/lang/String;)V")
		return jni.CallStaticVoidMethod(env, cls, m, jni.Value(view), jni.Value(id), jni.Value(jni.JavaString(env, lang)))
	})
	if err != nil {
		reqMu.Lock()
		delete(speech, id)
		reqMu.Unlock()
		return 0, nil, err
	}
	return id, ch, nil
}

// StopDictation ends recognition early; the final result follows.
// Call it on the main thread.
func StopDictation(vm, ctx uintptr, id int) error {
	return callClass(vm, ctx, pickerClass, func(env jni.Env, cls jni.Class) error {
		m := jni.GetStaticMethodID(env, cls, "stopDictation", "(I)V")
		return jni.CallStaticVoidMethod(env, cls, m, jni.Value(id))
	})
}

// goString converts a Java string; cgo maps JNI references to
// uintptr.
func goString(env jni.Env, s C.jstring) string {
	return jni.GoString(env, jni.String(s))
}

//export Java_org_giode_mobile_picker_1android_onPhoto
func Java_org_giode_mobile_picker_1android_onPhoto(cenv *C.JNIEnv, cls C.jclass, id C.jint, jpeg C.jbyteArray, name, errStr C.jstring) {
	env := jni.EnvFor(uintptr(unsafe.Pointer(cenv)))
	reqMu.Lock()
	ch := photos[int(id)]
	delete(photos, int(id))
	reqMu.Unlock()
	if ch == nil {
		return
	}
	var p Photo
	switch msg := goString(env, errStr); {
	case msg != "":
		p.Err = errors.New(msg)
	case jpeg == 0:
		p.Err = ErrCanceled
	default:
		p.JPEG = append([]byte(nil), jni.GetByteArrayElements(env, jni.ByteArray(jpeg))...)
		p.Name = goString(env, name)
	}
	ch <- p
}

//export Java_org_giode_mobile_picker_1android_onPhotos
func Java_org_giode_mobile_picker_1android_onPhotos(cenv *C.JNIEnv, cls C.jclass, id C.jint, jpegs, names C.jobjectArray, errStr C.jstring) {
	env := jni.EnvFor(uintptr(unsafe.Pointer(cenv)))
	reqMu.Lock()
	ch := photoSets[int(id)]
	delete(photoSets, int(id))
	reqMu.Unlock()
	if ch == nil {
		return
	}
	var ps Photos
	switch msg := goString(env, errStr); {
	case msg != "":
		ps.Err = errors.New(msg)
	case jpegs == 0:
		ps.Err = ErrCanceled
	default:
		n := int(C.giodeArrayLength(cenv, C.jarray(jpegs)))
		for i := 0; i < n; i++ {
			jo, err := jni.GetObjectArrayElement(env, jni.ObjectArray(jpegs), jni.Size(i))
			if err != nil {
				ps.Err = err
				break
			}
			no, _ := jni.GetObjectArrayElement(env, jni.ObjectArray(names), jni.Size(i))
			ps.Photos = append(ps.Photos, Photo{
				JPEG: append([]byte(nil), jni.GetByteArrayElements(env, jni.ByteArray(jo))...),
				Name: jni.GoString(env, jni.String(no)),
			})
			jni.DeleteLocalRef(env, jo)
			jni.DeleteLocalRef(env, no)
		}
	}
	ch <- ps
}

//export Java_org_giode_mobile_picker_1android_onSpeech
func Java_org_giode_mobile_picker_1android_onSpeech(cenv *C.JNIEnv, cls C.jclass, id C.jint, text C.jstring, done C.jboolean, errStr C.jstring) {
	env := jni.EnvFor(uintptr(unsafe.Pointer(cenv)))
	reqMu.Lock()
	ch := speech[int(id)]
	if done != 0 {
		delete(speech, int(id))
	}
	reqMu.Unlock()
	if ch == nil {
		return
	}
	s := Speech{Text: goString(env, text), Done: done != 0}
	if msg := goString(env, errStr); msg != "" {
		s.Err = errors.New(msg)
	}
	select {
	case ch <- s:
	default: // a slow reader misses partial results, never the final one
		if s.Done {
			go func() { ch <- s }()
		}
	}
}

// ClipboardImageURI returns the content URI of the clipboard's first
// image, or "" when it holds none. Call it on the main thread.
func ClipboardImageURI(vm, ctx uintptr) (string, error) {
	var uri string
	err := call(vm, ctx, func(env jni.Env, cls jni.Class) error {
		m := jni.GetStaticMethodID(env, cls, "clipboardImageURI", "(Landroid/content/Context;)Ljava/lang/String;")
		obj, err := jni.CallStaticObjectMethod(env, cls, m, jni.Value(ctx))
		if err != nil {
			return err
		}
		uri = jni.GoString(env, jni.String(obj))
		return nil
	})
	return uri, err
}

// ReadURI returns up to max bytes of a content URI. Any thread.
func ReadURI(vm, ctx uintptr, uri string, max int) ([]byte, error) {
	var data []byte
	err := call(vm, ctx, func(env jni.Env, cls jni.Class) error {
		m := jni.GetStaticMethodID(env, cls, "readURI", "(Landroid/content/Context;Ljava/lang/String;I)[B")
		obj, err := jni.CallStaticObjectMethod(env, cls, m, jni.Value(ctx), jni.Value(jni.JavaString(env, uri)), jni.Value(max))
		if err != nil {
			return err
		}
		if obj != 0 {
			data = append([]byte(nil), jni.GetByteArrayElements(env, jni.ByteArray(obj))...)
		}
		return nil
	})
	return data, err
}
