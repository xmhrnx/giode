package notification

/*
#include <jni.h>
*/
import "C"

import (
	"encoding/json"
	"errors"
	"time"
	"unsafe"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/android"
)

//go:generate javac --release 8 -classpath $ANDROID_HOME/platforms/android-34/android.jar -d $TEMP/giode_notification/classes notification_android.java NotificationReceiver.java NotificationActivity.java
//go:generate jar cf notification_android.jar -C $TEMP/giode_notification/classes .

const class = "org/giode/notification/notification_android"

func listen(*giode.App) error { return nil }

type actionSpec struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Reply       bool   `json:"reply,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
	Foreground  bool   `json:"foreground,omitempty"`
}

func show(n Notification, ch Channel) error {
	spec := make([]actionSpec, len(n.Actions))
	for i, a := range n.Actions {
		spec[i] = actionSpec{ID: a.ID, Title: a.Title, Reply: a.Reply, Placeholder: a.Placeholder, Foreground: a.Foreground}
	}
	js, _ := json.Marshal(spec)
	msg, err := android.CallString(class, "show",
		"(Landroid/content/Context;Ljava/lang/String;Ljava/lang/String;Ljava/lang/String;Ljava/lang/String;Ljava/lang/String;ILjava/lang/String;Ljava/lang/String;IZ)Ljava/lang/String;",
		android.Context(), n.ID, n.Title, n.Body, n.Thread, n.Channel, int(ch.Importance), string(js), encodePayload(n), n.Badge, n.Silent)
	switch {
	case err != nil:
		return err
	case msg == "off":
		return giode.ErrNotificationsOff
	case msg != "":
		return errors.New("notification: " + msg)
	}
	return nil
}

func remove(id string) error {
	return android.CallVoid(class, "remove", "(Landroid/content/Context;Ljava/lang/String;)V", android.Context(), id)
}

func removeAll() error {
	return android.CallVoid(class, "removeAll", "(Landroid/content/Context;)V", android.Context())
}

func setBadge(n int) error {
	return android.CallVoid(class, "setBadge", "(I)V", n)
}

func defineChannel(c Channel) error {
	return android.CallVoid(class, "defineChannel", "(Landroid/content/Context;Ljava/lang/String;Ljava/lang/String;Ljava/lang/String;I)V",
		android.Context(), c.ID, c.Name, c.Description, int(c.Importance))
}

func defineCategory(string, []Action) error { return nil }

//export Java_org_giode_notification_notification_1android_onResponse
func Java_org_giode_notification_notification_1android_onResponse(env *C.JNIEnv, cls C.jclass, payloadJSON, action, reply C.jstring) C.jboolean {
	e := unsafe.Pointer(env)
	p, ok := decodePayload(android.GoString(e, uintptr(payloadJSON)))
	if !ok {
		return 0
	}
	r := Response{ID: p.ID, Action: android.GoString(e, uintptr(action)), Reply: android.GoString(e, uintptr(reply)), Data: p.Data}
	// Android gives a broadcast about ten seconds.
	if respondWait(r, 8*time.Second) {
		return 1
	}
	return 0
}
