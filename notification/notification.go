// Package notification shows system notifications with more than a
// title and a body: buttons, a reply field, grouping by conversation,
// channels the user can tune, replacing and removing notifications, and
// the app icon's badge count. Listen receives what the user did with
// them, also when that started the app.
//
// For a plain notification, App.Notify is enough.
//
// What each system does:
//
//   - Android: channels (Android 8 and later), buttons, inline reply
//     (Android 7 and later), conversations grouped by Thread, and
//     the badge count as the notification's number, where the launcher
//     shows one. A button or reply starts the app's process when it
//     isn't running, without opening its window, unless the action is
//     Foreground.
//   - iOS and macOS: buttons and a reply field (as notification
//     categories), threads, the badge on the app icon or Dock tile. A
//     channel's importance sets the interruption level.
//   - Windows: toasts with buttons and a reply box, grouped by Thread,
//     and the badge as a number on the taskbar button. A button pressed
//     while the app isn't running starts it (giode build registers the
//     app with Windows; an unpackaged app registers itself the first
//     time it shows one).
//   - Linux: buttons through the freedesktop notification service, a
//     reply field where the service has one (KDE Plasma), the badge on
//     the launcher entry (KDE, Ubuntu's dock) and channels as the
//     notification's urgency and category. Buttons work while the app
//     runs.
//
// On Android 13 and later, iOS and macOS, notifications need the
// user's permission: ask with App.RequestNotificationPermission.
// Show returns giode.ErrNotificationsOff when they're off.
package notification

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/xmhrnx/giode"
)

// Notification is a notification to show.
type Notification struct {
	// ID identifies the notification: showing another with the same ID
	// replaces it, and Remove takes it away. Show makes one up when it's
	// empty.
	ID string
	// Title and Body are its text.
	Title, Body string
	// Thread groups the notifications of one conversation or topic.
	Thread string
	// Channel is the ID of a channel defined with DefineChannel; empty is
	// the default channel.
	Channel string
	// Actions are its buttons, in order. Systems show up to three
	// (Android, Linux) to five (Windows).
	Actions []Action
	// Data comes back in the Response.
	Data map[string]string
	// Badge, when above zero, sets the app's badge count along with the
	// notification (as SetBadge does).
	Badge int
	// Silent shows it without a sound.
	Silent bool
}

// Action is a button on a notification.
type Action struct {
	// ID comes back as Response.Action.
	ID string
	// Title is the button's label.
	Title string
	// Reply makes the button open a text field: the text comes back as
	// Response.Reply.
	Reply bool
	// Placeholder is the reply field's hint.
	Placeholder string
	// Send labels the reply field's send button (iOS, macOS, Windows);
	// the default is "Send" in the user's language.
	Send string
	// Destructive marks an action that deletes something (iOS and macOS
	// show it in red).
	Destructive bool
	// Foreground brings the app to the front; otherwise the action is
	// handled without showing the app where the system can (Android, iOS,
	// macOS, Windows).
	Foreground bool
}

// Response is what the user did with a notification.
type Response struct {
	// ID is the notification's ID.
	ID string
	// Action is the ID of the button pressed, or empty when the user
	// tapped or clicked the notification itself.
	Action string
	// Reply is the text typed into a reply field.
	Reply string
	// Data is the notification's Data.
	Data map[string]string
}

// Importance is how much a channel's notifications interrupt.
type Importance int

const (
	// Default notifications make a sound and show in the list.
	Default Importance = iota
	// Low ones show in the list, without a sound or a banner.
	Low
	// High ones pop up over other apps (heads-up on Android,
	// time-sensitive on iOS where the app has that entitlement, critical
	// urgency on Linux).
	High
)

// Channel is a kind of notification the user can turn off or tune in the
// system's settings (Android). Elsewhere a channel sets its
// notifications' importance.
type Channel struct {
	// ID is used in Notification.Channel.
	ID string
	// Name and Description are shown in the system's settings.
	Name, Description string
	// Importance is how much its notifications interrupt.
	Importance Importance
}

// DefineCategory registers a set of actions under an ID, for remote
// pushes: an APNs payload's "category" names it, and the system shows
// its buttons with the push (iOS and macOS). Responses to them come to
// Listen, with the push's custom string keys as Data. Elsewhere it does
// nothing: Android apps show a data message's notification with Show.
func DefineCategory(id string, actions ...Action) error {
	if id == "" {
		return errors.New("notification: category without an ID")
	}
	return defineCategory(id, actions)
}

// ErrNoID is returned by Remove for an empty ID.
var ErrNoID = errors.New("notification: no ID")

var (
	mu       sync.Mutex
	handler  func(Response)
	queued   []Response
	channels = map[string]Channel{}
	// listening is closed by the first Listen.
	listening = make(chan struct{})
	// shows counts the Show calls for each ID.
	shows = map[string]int{}
)

// Listen calls f with what the user does with the app's notifications,
// including the response that started the app. Call it early, before
// App.Run, where actions can start the app (Windows needs its handler
// registered to take them). f runs on its own goroutine: use App.Post to
// change the UI.
func Listen(a *giode.App, f func(Response)) error {
	mu.Lock()
	handler = f
	q := queued
	queued = nil
	select {
	case <-listening:
	default:
		close(listening)
	}
	mu.Unlock()
	for _, r := range q {
		go f(r)
	}
	return listen(a)
}

// Show shows n, replacing a shown notification with the same ID, and
// returns its ID.
func Show(n Notification) (string, error) {
	if n.ID == "" {
		n.ID = newID()
	}
	mu.Lock()
	ch := channels[n.Channel]
	shows[n.ID]++
	mu.Unlock()
	if err := show(n, ch); err != nil {
		return "", err
	}
	if n.Badge > 0 {
		setBadge(n.Badge)
	}
	return n.ID, nil
}

// Remove takes the notification with the given ID away.
func Remove(id string) error {
	if id == "" {
		return ErrNoID
	}
	return remove(id)
}

// RemoveAll takes all the app's notifications away.
func RemoveAll() error {
	return removeAll()
}

// SetBadge sets the number on the app's icon (iOS, macOS's Dock, the
// Windows taskbar button, Linux launchers that show one); 0 clears it.
// On Android the count goes with the app's next notification.
func SetBadge(n int) error {
	if n < 0 {
		n = 0
	}
	return setBadge(n)
}

// DefineChannel creates or updates a channel. On Android the user's own
// changes to a channel win: its importance can't be changed once
// created, only its name and description.
func DefineChannel(c Channel) error {
	if c.ID == "" {
		return errors.New("notification: channel without an ID")
	}
	if c.Name == "" {
		c.Name = c.ID
	}
	mu.Lock()
	channels[c.ID] = c
	mu.Unlock()
	return defineChannel(c)
}

// respond delivers a response to the handler, or keeps it for Listen.
func respond(r Response) {
	if r.Data == nil {
		r.Data = map[string]string{}
	}
	mu.Lock()
	f := handler
	if f == nil {
		queued = append(queued, r)
	}
	mu.Unlock()
	if f != nil {
		go f(r)
	}
}

// respondWait runs the handler with r and waits for it to return, for
// at most d (waiting for Listen too, when the response started the
// app). It reports whether the handler showed the notification again.
func respondWait(r Response, d time.Duration) bool {
	if r.Data == nil {
		r.Data = map[string]string{}
	}
	deadline := time.NewTimer(d)
	defer deadline.Stop()
	select {
	case <-listening:
	case <-deadline.C:
		respond(r)
		return false
	}
	mu.Lock()
	f := handler
	before := shows[r.ID]
	mu.Unlock()
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		f(r)
	}()
	select {
	case <-finished:
	case <-deadline.C:
	}
	mu.Lock()
	defer mu.Unlock()
	return shows[r.ID] > before
}

func newID() string {
	var b [8]byte
	rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// payload is what goes with a notification to come back with a
// response: its ID and Data, as JSON.
type payload struct {
	ID   string            `json:"i"`
	Data map[string]string `json:"d,omitempty"`
}

func encodePayload(n Notification) string {
	b, _ := json.Marshal(payload{ID: n.ID, Data: n.Data})
	return string(b)
}

func decodePayload(s string) (payload, bool) {
	var p payload
	if json.Unmarshal([]byte(s), &p) != nil || p.ID == "" {
		return payload{}, false
	}
	return p, true
}
