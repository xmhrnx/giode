// Package push receives remote push notifications: Firebase Cloud
// Messaging (FCM) on Android, the Apple Push Notification service
// (APNs) on iOS and macOS. Elsewhere its functions return
// giode.ErrNotSupported.
//
// The app hands Token's result to its server, which sends messages to
// that device through FCM or APNs. A notification message arriving
// while the app is in the background is shown by the system; tapping it
// opens the app and calls Handlers.Open. Messages arriving while the
// app runs (and data-only messages) go to Handlers.Message.
//
// Setup, beyond importing the package:
//
//   - Android: a Firebase project with the app added, and its
//     google-services.json set as android.google_services in
//     giode.toml. The giode CLI adds Firebase Messaging and the
//     package's service to the app.
//   - iOS and macOS: push enabled for the app id in the Apple developer
//     account, and ios.push (or macos.push) set to "development" or
//     "production" in giode.toml, which adds the aps-environment
//     entitlement. The server sends to APNs with the app's key.
package push

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/xmhrnx/giode"
)

// Message is a received push message.
type Message struct {
	// ID is the message id FCM or APNs gave it, when there is one.
	ID string
	// Title and Body are the notification's text; empty for a data-only
	// message.
	Title, Body string
	// Data is the message's custom key-value payload.
	Data map[string]string
}

// Handlers receive push events. They run on their own goroutine: use
// App.Post to change the UI.
type Handlers struct {
	// Token is called when the device's token changes (FCM rotates
	// them); send the new one to the server.
	Token func(token string)
	// Message is called for a message received while the app runs.
	Message func(Message)
	// Open is called when the user opened the app by tapping a
	// notification.
	Open func(Message)
}

var (
	mu       sync.Mutex
	handlers Handlers
	started  bool
	// queued holds events from before Listen.
	queued []event
)

type event struct {
	kind  int
	token string
	msg   Message
}

const (
	evToken = iota
	evMessage
	evOpen
)

// Listen starts delivering push events to h, including the ones that
// arrived before it (a notification that launched the app). Call it
// early, before App.Run; a later call replaces the handlers.
func Listen(a *giode.App, h Handlers) error {
	mu.Lock()
	handlers = h
	q := queued
	queued = nil
	first := !started
	started = true
	mu.Unlock()
	for _, e := range q {
		dispatch(e)
	}
	if first {
		return start(a)
	}
	return nil
}

// Token returns the device's push token: the FCM registration token on
// Android, the APNs device token (hex) on iOS and macOS. It registers
// the device with the push service the first time, which needs a
// network connection.
func Token(ctx context.Context) (string, error) {
	return token(ctx)
}

// RequestPermission asks the user to allow notifications, which iOS,
// macOS and Android 13 and later need before showing them; elsewhere it
// reports true. It reports whether they're allowed.
func RequestPermission(ctx context.Context, a *giode.App) (bool, error) {
	return requestPermission(ctx, a)
}

// Unregister deletes the device's token, so the server's messages no
// longer arrive, until Token registers again.
func Unregister() error {
	return unregister()
}

// deliver hands an event to its handler, or queues it until Listen.
func deliver(e event) {
	mu.Lock()
	if !started {
		queued = append(queued, e)
		mu.Unlock()
		return
	}
	mu.Unlock()
	go dispatch(e)
}

func dispatch(e event) {
	mu.Lock()
	h := handlers
	mu.Unlock()
	switch {
	case e.kind == evToken && h.Token != nil:
		h.Token(e.token)
	case e.kind == evMessage && h.Message != nil:
		h.Message(e.msg)
	case e.kind == evOpen && h.Open != nil:
		h.Open(e.msg)
	}
}

// parseMessage reads the JSON the native side sends: id, title, body
// and data, whose values may be any JSON (APNs payloads); non-strings
// are kept as JSON text.
func parseMessage(s string) (Message, error) {
	var raw struct {
		ID    string                     `json:"id"`
		Title string                     `json:"title"`
		Body  string                     `json:"body"`
		Data  map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal([]byte(s), &raw); err != nil {
		return Message{}, err
	}
	m := Message{ID: raw.ID, Title: raw.Title, Body: raw.Body, Data: map[string]string{}}
	for k, v := range raw.Data {
		var str string
		if json.Unmarshal(v, &str) == nil {
			m.Data[k] = str
		} else {
			m.Data[k] = string(v)
		}
	}
	return m, nil
}

// received delivers a message from native code as JSON.
func received(js string, opened bool) {
	m, err := parseMessage(js)
	if err != nil {
		return
	}
	kind := evMessage
	if opened {
		kind = evOpen
	}
	deliver(event{kind: kind, msg: m})
}
