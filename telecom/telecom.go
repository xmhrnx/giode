// Package telecom puts an app's calls in the phone's own call system:
// the full-screen incoming call screen (even when the phone is locked),
// the call in the status bar and recents, and answering from a headset
// or a car. iOS uses CallKit (with PushKit VoIP pushes to ring a
// sleeping app), Android a self-managed ConnectionService. Desktops
// have no system call screen: Supported is false and the app shows its
// own.
//
//	telecom.Setup(app, telecom.Handler{
//		OnAnswer: func(id string) { join(id) },
//		OnEnd:    func(id string) { hangUp(id) },
//		OnMute:   func(id string, m bool) { call.SetMuted(m) },
//	})
//	// When the server says someone's calling:
//	telecom.ReportIncoming(ctx, telecom.Call{ID: id, Name: "Ada", Handle: "ada"})
//
// The call's audio itself goes through rtc or audio; on iOS, start it
// once OnAudio(true) arrives.
package telecom

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"sync"

	"github.com/xmhrnx/giode"
)

// Call describes a call to the system.
type Call struct {
	// ID is a UUID naming the call; NewID makes one.
	ID string
	// Handle is who's calling or being called: a user name, address or
	// number. Name is what's shown; Handle when empty.
	Handle, Name string
	// Video shows it as a video call.
	Video bool
}

// Reason is why the other side or the network ended a call.
type Reason int

const (
	// RemoteEnded: the other side hung up.
	RemoteEnded Reason = iota
	Failed
	Unanswered
	// AnsweredElsewhere: the user answered on another device.
	AnsweredElsewhere
	// DeclinedElsewhere: the user declined on another device.
	DeclinedElsewhere
)

// Handler gets what the user does in the system's call screens, on
// the app's goroutine (as Post does).
type Handler struct {
	// OnAnswer: the user answered an incoming call.
	OnAnswer func(id string)
	// OnStart: the system started an outgoing call: one StartOutgoing
	// asked for, or one the user placed from recents.
	OnStart func(c Call)
	// OnEnd: the user ended or declined a call.
	OnEnd  func(id string)
	OnMute func(id string, muted bool)
	OnHold func(id string, held bool)
	// OnAudio reports the call's audio session: on iOS, start the call's
	// audio when it's true and stop it when false.
	OnAudio func(active bool)
	// OnVoIPToken gets the device's PushKit token (iOS), for the server
	// to send VoIP pushes to.
	OnVoIPToken func(token string)
	// OnVoIPPush turns a VoIP push's payload into the call to ring.
	// iOS ends an app that gets a VoIP push without reporting a call,
	// so the push must carry what's needed; without OnVoIPPush the
	// payload's "id", "handle", "name" and "video" keys are used.
	OnVoIPPush func(payload map[string]any) Call
}

var (
	ErrNotSupported = errors.New("telecom: no system call screen here")
	ErrNotSetUp     = errors.New("telecom: call Setup first")
)

var state struct {
	sync.Mutex
	app   *giode.App
	h     Handler
	ready bool
	sync  bool
}

// Supported reports whether the platform has a system call screen.
func Supported() bool { return supported }

// Setup registers the app with the call system and sets the handler.
func Setup(a *giode.App, h Handler) error {
	state.Lock()
	state.app, state.h = a, h
	state.Unlock()
	if !supported {
		return ErrNotSupported
	}
	if err := setup(a); err != nil {
		return err
	}
	state.Lock()
	state.ready = true
	state.Unlock()
	return nil
}

func ready() error {
	if !supported {
		return ErrNotSupported
	}
	state.Lock()
	defer state.Unlock()
	if !state.ready {
		return ErrNotSetUp
	}
	return nil
}

// NewID makes a call ID.
func NewID() string {
	var b [16]byte
	rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (c *Call) fill() {
	if c.ID == "" {
		c.ID = NewID()
	}
	if c.Name == "" {
		c.Name = c.Handle
	}
}

// ReportIncoming rings: the system shows the incoming call screen.
func ReportIncoming(ctx context.Context, c Call) error {
	if err := ready(); err != nil {
		return err
	}
	c.fill()
	return reportIncoming(ctx, c)
}

// StartOutgoing tells the system the user is calling c; OnStart
// follows. Report ReportConnected when the other side answers.
func StartOutgoing(c Call) (string, error) {
	if err := ready(); err != nil {
		return "", err
	}
	c.fill()
	return c.ID, startOutgoing(c)
}

// ReportConnected tells the system an outgoing call was answered.
func ReportConnected(id string) error {
	if err := ready(); err != nil {
		return err
	}
	return reportConnected(id)
}

// End ends a call from the app: the user hung up in the app's own
// screen.
func End(id string) error {
	if err := ready(); err != nil {
		return err
	}
	return end(id)
}

// ReportEnded tells the system a call ended for another reason: the
// other side hung up, or it failed.
func ReportEnded(id string, r Reason) error {
	if err := ready(); err != nil {
		return err
	}
	return reportEnded(id, r)
}

// SetMuted mutes the call in the system's screen too.
func SetMuted(id string, muted bool) error {
	if err := ready(); err != nil {
		return err
	}
	return setMuted(id, muted)
}

// deliver runs fn with the handler on the app's goroutine.
func deliver(fn func(h Handler)) {
	state.Lock()
	a, h, sync := state.app, state.h, state.sync
	state.Unlock()
	switch {
	case sync:
		fn(h)
	case a == nil:
		go fn(h)
	default:
		a.Post(func() { fn(h) })
	}
}

// dispatchSync dispatches with the handler called at once, for tests.
func dispatchSync(kind, a, b string) {
	state.Lock()
	state.sync = true
	state.Unlock()
	defer func() {
		state.Lock()
		state.sync = false
		state.Unlock()
	}()
	dispatch(kind, a, b)
}

// callFromPayload is a VoIP push's call, from the handler or the
// payload's keys.
func callFromPayload(p map[string]any) Call {
	state.Lock()
	h := state.h
	state.Unlock()
	if h.OnVoIPPush != nil {
		c := h.OnVoIPPush(p)
		c.fill()
		return c
	}
	str := func(k string) string { s, _ := p[k].(string); return s }
	video, _ := p["video"].(bool)
	c := Call{ID: str("id"), Handle: str("handle"), Name: str("name"), Video: video}
	c.fill()
	return c
}
