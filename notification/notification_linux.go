//go:build linux && !android

package notification

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"gioui.org/app"
	"github.com/godbus/dbus/v5"

	"github.com/xmhrnx/giode"
)

// Linux notifications go to the freedesktop notification service
// (org.freedesktop.Notifications) over the session bus. Responses come
// as its ActionInvoked and NotificationReplied signals, which reach the
// connection that showed the notification: they work while the app
// runs.

const (
	fdName  = "org.freedesktop.Notifications"
	fdPath  = "/org/freedesktop/Notifications"
	fdIface = "org.freedesktop.Notifications"
	// replyAction is KDE's inline reply action key.
	replyAction = "inline-reply"
)

type shown struct {
	id      string
	payload payload
	// reply is the ID of the action with the reply field.
	reply string
	// buttons maps the action keys sent to the service to action IDs.
	buttons map[string]string
}

var lin struct {
	sync.Mutex
	conn  *dbus.Conn
	err   error
	caps  map[string]bool
	byNum map[uint32]*shown
	byID  map[string]uint32
}

// connect opens the app's own session bus connection, once, and
// subscribes to the service's signals.
func connect() (*dbus.Conn, error) {
	lin.Lock()
	defer lin.Unlock()
	if lin.conn != nil || lin.err != nil {
		return lin.conn, lin.err
	}
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		lin.err = err
		return nil, err
	}
	if err := conn.AddMatchSignal(dbus.WithMatchInterface(fdIface), dbus.WithMatchObjectPath(fdPath)); err != nil {
		conn.Close()
		lin.err = err
		return nil, err
	}
	var caps []string
	conn.Object(fdName, fdPath).Call(fdIface+".GetCapabilities", 0).Store(&caps)
	lin.caps = map[string]bool{}
	for _, c := range caps {
		lin.caps[c] = true
	}
	lin.byNum = map[uint32]*shown{}
	lin.byID = map[string]uint32{}
	lin.conn = conn
	signals := make(chan *dbus.Signal, 16)
	conn.Signal(signals)
	go func() {
		for s := range signals {
			handleSignal(s)
		}
	}()
	return conn, nil
}

func handleSignal(s *dbus.Signal) {
	if len(s.Body) < 2 {
		return
	}
	num, ok := s.Body[0].(uint32)
	if !ok {
		return
	}
	lin.Lock()
	n := lin.byNum[num]
	if n != nil && s.Name != fdIface+".NotificationClosed" {
		// Most services close a notification when it's used.
		delete(lin.byNum, num)
		delete(lin.byID, n.id)
	}
	lin.Unlock()
	if n == nil {
		return
	}
	switch s.Name {
	case fdIface + ".ActionInvoked":
		key, _ := s.Body[1].(string)
		r := Response{ID: n.id, Data: n.payload.Data}
		switch {
		case key == "default":
		case key == replyAction:
			r.Action = n.reply
		default:
			id, ok := n.buttons[key]
			if !ok {
				return
			}
			r.Action = id
		}
		respond(r)
	case fdIface + ".NotificationReplied":
		text, _ := s.Body[1].(string)
		respond(Response{ID: n.id, Action: n.reply, Reply: text, Data: n.payload.Data})
	case fdIface + ".NotificationClosed":
		lin.Lock()
		if lin.byNum[num] == n {
			delete(lin.byNum, num)
			delete(lin.byID, n.id)
		}
		lin.Unlock()
	}
}

func listen(*giode.App) error {
	_, err := connect()
	return err
}

// desktopID is the app's desktop entry name: its ID (giode build sets
// it), else the executable's name.
func desktopID() string {
	if app.ID != "" {
		return app.ID
	}
	exe, _ := os.Executable()
	return strings.TrimSuffix(filepath.Base(exe), filepath.Ext(exe))
}

func show(n Notification, ch Channel) error {
	conn, err := connect()
	if err != nil {
		return err
	}
	s := &shown{id: n.ID, payload: payload{ID: n.ID, Data: n.Data}, buttons: map[string]string{}}
	actions := []string{"default", ""}
	lin.Lock()
	inline := lin.caps["inline-reply"]
	hasActions := lin.caps["actions"]
	lin.Unlock()
	if hasActions {
		for i, a := range n.Actions {
			if a.Reply && inline && s.reply == "" {
				s.reply = a.ID
				actions = append(actions, replyAction, a.Title)
				continue
			}
			key := strconv.Itoa(i) + ":" + a.ID
			s.buttons[key] = a.ID
			actions = append(actions, key, a.Title)
		}
	}
	hints := map[string]dbus.Variant{
		"desktop-entry": dbus.MakeVariant(desktopID()),
	}
	switch ch.Importance {
	case Low:
		hints["urgency"] = dbus.MakeVariant(byte(0))
	case High:
		hints["urgency"] = dbus.MakeVariant(byte(2))
	default:
		hints["urgency"] = dbus.MakeVariant(byte(1))
	}
	if ch.ID != "" {
		hints["category"] = dbus.MakeVariant(ch.ID)
	}
	if n.Silent || ch.Importance == Low {
		hints["suppress-sound"] = dbus.MakeVariant(true)
	}
	for _, a := range n.Actions {
		if a.Reply && a.Placeholder != "" && a.ID == s.reply {
			hints["x-kde-reply-placeholder-text"] = dbus.MakeVariant(a.Placeholder)
		}
	}
	lin.Lock()
	replaces := lin.byID[n.ID]
	lin.Unlock()
	var num uint32
	err = conn.Object(fdName, fdPath).Call(fdIface+".Notify", 0,
		appName(), replaces, "", n.Title, n.Body, actions, hints, int32(-1)).Store(&num)
	if err != nil {
		return err
	}
	lin.Lock()
	if replaces != 0 && replaces != num {
		delete(lin.byNum, replaces)
	}
	lin.byNum[num] = s
	lin.byID[n.ID] = num
	lin.Unlock()
	return nil
}

// appName is the name the service shows with notifications.
func appName() string {
	id := desktopID()
	if i := strings.LastIndexByte(id, '.'); i >= 0 && i < len(id)-1 {
		return id[i+1:]
	}
	return id
}

func remove(id string) error {
	conn, err := connect()
	if err != nil {
		return err
	}
	lin.Lock()
	num, ok := lin.byID[id]
	delete(lin.byID, id)
	delete(lin.byNum, num)
	lin.Unlock()
	if !ok {
		return nil
	}
	return conn.Object(fdName, fdPath).Call(fdIface+".CloseNotification", 0, num).Err
}

func removeAll() error {
	conn, err := connect()
	if err != nil {
		return err
	}
	lin.Lock()
	var nums []uint32
	for num := range lin.byNum {
		nums = append(nums, num)
	}
	lin.byNum = map[uint32]*shown{}
	lin.byID = map[string]uint32{}
	lin.Unlock()
	var errs []error
	for _, num := range nums {
		if err := conn.Object(fdName, fdPath).Call(fdIface+".CloseNotification", 0, num).Err; err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// setBadge sends the Unity launcher entry signal, which KDE's task
// manager and Ubuntu's dock show on the app's icon.
func setBadge(n int) error {
	conn, err := connect()
	if err != nil {
		return err
	}
	uri := "application://" + desktopID() + ".desktop"
	sum := md5.Sum([]byte(uri))
	path := dbus.ObjectPath("/com/canonical/unity/launcherentry/" + hex.EncodeToString(sum[:4]))
	props := map[string]dbus.Variant{
		"count":         dbus.MakeVariant(int64(n)),
		"count-visible": dbus.MakeVariant(n > 0),
	}
	return conn.Emit(path, "com.canonical.Unity.LauncherEntry.Update", uri, props)
}

func defineChannel(Channel) error           { return nil }
func defineCategory(string, []Action) error { return nil }
