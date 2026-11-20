package notification

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo ios LDFLAGS: -framework UIKit -framework UserNotifications
#cgo !ios LDFLAGS: -framework AppKit -framework UserNotifications

#include <stdlib.h>

int giodeNotificationAvailable(void);
void giodeNotificationInstall(void);
void giodeNotificationCategory(int rid, const char *cid, const char *actionsJSON);
void giodeNotificationShow(int rid, const char *ident, const char *title, const char *body, const char *thread, const char *category, const char *payload, int badge, int silent, int level);
void giodeNotificationRemove(const char *ident);
void giodeNotificationRemoveAll(void);
void giodeNotificationBadge(int n);
*/
import "C"

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"unsafe"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/internal/bridge"
)

var done bridge.Requests[string]

func available() error {
	if C.giodeNotificationAvailable() == 0 {
		return fmt.Errorf("%w: notifications need an app bundle (build the app with giode)", giode.ErrNotSupported)
	}
	return nil
}

func listen(*giode.App) error {
	if err := available(); err != nil {
		return err
	}
	C.giodeNotificationInstall()
	return nil
}

// wait waits for the native side's answer: "" for success, "off" when
// notifications are off, else an error message.
func wait(id int, ch chan string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	msg, err := done.Wait(ctx, id, ch)
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

type actionSpec struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Reply       bool   `json:"reply,omitempty"`
	Placeholder string `json:"placeholder"`
	Send        string `json:"send"`
	Destructive bool   `json:"destructive,omitempty"`
	Foreground  bool   `json:"foreground,omitempty"`
}

func registerCategory(id string, actions []Action) error {
	spec := make([]actionSpec, len(actions))
	for i, a := range actions {
		spec[i] = actionSpec{ID: a.ID, Title: a.Title, Reply: a.Reply, Placeholder: a.Placeholder, Send: sendLabel(a), Destructive: a.Destructive, Foreground: a.Foreground}
	}
	js, _ := json.Marshal(spec)
	cid, cjs := C.CString(id), C.CString(string(js))
	defer C.free(unsafe.Pointer(cid))
	defer C.free(unsafe.Pointer(cjs))
	rid, ch := done.New()
	C.giodeNotificationCategory(C.int(rid), cid, cjs)
	return wait(rid, ch)
}

func defineCategory(id string, actions []Action) error {
	if err := available(); err != nil {
		return err
	}
	return registerCategory(id, actions)
}

// categories holds the categories made for notifications' actions.
var categories = map[string]bool{}

func show(n Notification, ch Channel) error {
	if err := available(); err != nil {
		return err
	}
	category := ""
	if len(n.Actions) > 0 {
		js, _ := json.Marshal(n.Actions)
		sum := sha256.Sum256(js)
		category = "giode." + hex.EncodeToString(sum[:8])
		mu.Lock()
		have := categories[category]
		mu.Unlock()
		if !have {
			if err := registerCategory(category, n.Actions); err != nil {
				return err
			}
			mu.Lock()
			categories[category] = true
			mu.Unlock()
		}
	}
	strs := []*C.char{C.CString(n.ID), C.CString(n.Title), C.CString(n.Body), C.CString(n.Thread), C.CString(category), C.CString(encodePayload(n))}
	defer func() {
		for _, s := range strs {
			C.free(unsafe.Pointer(s))
		}
	}()
	silent := 0
	if n.Silent {
		silent = 1
	}
	rid, wch := done.New()
	C.giodeNotificationShow(C.int(rid), strs[0], strs[1], strs[2], strs[3], strs[4], strs[5], C.int(n.Badge), C.int(silent), C.int(ch.Importance))
	return wait(rid, wch)
}

func remove(id string) error {
	if err := available(); err != nil {
		return err
	}
	cs := C.CString(id)
	defer C.free(unsafe.Pointer(cs))
	C.giodeNotificationRemove(cs)
	return nil
}

func removeAll() error {
	if err := available(); err != nil {
		return err
	}
	C.giodeNotificationRemoveAll()
	return nil
}

func setBadge(n int) error {
	if err := available(); err != nil {
		return err
	}
	C.giodeNotificationBadge(C.int(n))
	return nil
}

func defineChannel(Channel) error { return nil }

//export giodeNotificationDone
func giodeNotificationDone(rid C.int, msg *C.char) {
	s := ""
	if msg != nil {
		s = C.GoString(msg)
	}
	done.Done(int(rid), s)
}

//export giodeNotificationResponse
func giodeNotificationResponse(payloadJSON, action, reply *C.char) {
	p, ok := decodePayload(C.GoString(payloadJSON))
	if !ok {
		return
	}
	respond(Response{ID: p.ID, Action: C.GoString(action), Reply: C.GoString(reply), Data: p.Data})
}
