//go:build linux && !android

package notification

import (
	"bufio"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
)

// fakeServer is a notification service on a private bus.
type fakeServer struct {
	conn *dbus.Conn
	mu   sync.Mutex
	next uint32
	got  []call
}

type call struct {
	method   string
	replaces uint32
	summary  string
	actions  []string
	hints    map[string]dbus.Variant
	id       uint32
}

func (s *fakeServer) GetCapabilities() ([]string, *dbus.Error) {
	return []string{"actions", "body", "inline-reply"}, nil
}

func (s *fakeServer) Notify(app string, replaces uint32, icon, summary, body string, actions []string, hints map[string]dbus.Variant, timeout int32) (uint32, *dbus.Error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := replaces
	if id == 0 {
		s.next++
		id = s.next
	}
	s.got = append(s.got, call{method: "Notify", replaces: replaces, summary: summary, actions: actions, hints: hints, id: id})
	return id, nil
}

func (s *fakeServer) CloseNotification(id uint32) *dbus.Error {
	s.mu.Lock()
	s.got = append(s.got, call{method: "Close", id: id})
	s.mu.Unlock()
	return nil
}

func (s *fakeServer) last() call {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.got[len(s.got)-1]
}

// privateBus starts a session bus for the test and points the package at
// it.
func privateBus(t *testing.T) {
	if _, err := exec.LookPath("dbus-daemon"); err != nil {
		t.Skip("no dbus-daemon")
	}
	cmd := exec.Command("dbus-daemon", "--session", "--nofork", "--print-address=1")
	out, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		t.Skip(err)
	}
	t.Cleanup(func() { cmd.Process.Kill(); cmd.Wait() })
	addr, err := bufio.NewReader(out).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", strings.TrimSpace(addr))
	lin.Lock()
	lin.conn, lin.err = nil, nil
	lin.Unlock()
	t.Cleanup(func() {
		lin.Lock()
		if lin.conn != nil {
			lin.conn.Close()
		}
		lin.conn, lin.err = nil, nil
		lin.Unlock()
	})
}

func startServer(t *testing.T) *fakeServer {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	s := &fakeServer{conn: conn}
	conn.Export(s, fdPath, fdIface)
	if r, err := conn.RequestName(fdName, dbus.NameFlagDoNotQueue); err != nil || r != dbus.RequestNameReplyPrimaryOwner {
		t.Fatal("can't own the name", err)
	}
	return s
}

func TestLinuxShowRespond(t *testing.T) {
	privateBus(t)
	srv := startServer(t)
	got := make(chan Response, 4)
	if err := Listen(nil, func(r Response) { got <- r }); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { mu.Lock(); handler = nil; mu.Unlock() })
	DefineChannel(Channel{ID: "chat", Name: "Chats", Importance: High})
	id, err := Show(Notification{
		ID: "m1", Title: "Ana", Body: "Lunch?", Channel: "chat", Data: map[string]string{"room": "7"},
		Actions: []Action{{ID: "like", Title: "Like"}, {ID: "reply", Title: "Reply", Reply: true, Placeholder: "Message"}},
	})
	if err != nil || id != "m1" {
		t.Fatal(id, err)
	}
	c := srv.last()
	want := []string{"default", "", "0:like", "Like", replyAction, "Reply"}
	if strings.Join(c.actions, "|") != strings.Join(want, "|") {
		t.Errorf("actions %q, want %q", c.actions, want)
	}
	if u := c.hints["urgency"].Value(); u != byte(2) {
		t.Errorf("urgency %v", u)
	}
	if p := c.hints["x-kde-reply-placeholder-text"].Value(); p != "Message" {
		t.Errorf("placeholder %v", p)
	}
	// Showing the same ID again replaces it.
	Show(Notification{ID: "m1", Title: "Ana", Body: "Lunch at 1?", Data: map[string]string{"room": "7"}, Actions: []Action{{ID: "like", Title: "Like"}, {ID: "reply", Title: "Reply", Reply: true}}})
	if c2 := srv.last(); c2.replaces != c.id {
		t.Errorf("replaces %d, want %d", c2.replaces, c.id)
	}

	emit := func(name string, args ...any) {
		if err := srv.conn.Emit(fdPath, fdIface+"."+name, args...); err != nil {
			t.Fatal(err)
		}
	}
	wait := func() Response {
		select {
		case r := <-got:
			return r
		case <-time.After(3 * time.Second):
			t.Fatal("no response")
		}
		return Response{}
	}
	emit("ActionInvoked", c.id, "0:like")
	if r := wait(); r.ID != "m1" || r.Action != "like" || r.Data["room"] != "7" {
		t.Errorf("button: %+v", r)
	}
	id2, _ := Show(Notification{Title: "Bo", Body: "Hi", Actions: []Action{{ID: "reply", Title: "Reply", Reply: true}}})
	c3 := srv.last()
	emit("NotificationReplied", c3.id, "on my way")
	if r := wait(); r.ID != id2 || r.Action != "reply" || r.Reply != "on my way" {
		t.Errorf("reply: %+v", r)
	}
	id3, _ := Show(Notification{Title: "Cy", Body: "Hey"})
	emit("ActionInvoked", srv.last().id, "default")
	if r := wait(); r.ID != id3 || r.Action != "" {
		t.Errorf("click: %+v", r)
	}

	id4, _ := Show(Notification{Title: "Di", Body: "Bye"})
	n4 := srv.last().id
	if err := Remove(id4); err != nil {
		t.Fatal(err)
	}
	if c := srv.last(); c.method != "Close" || c.id != n4 {
		t.Errorf("remove: %+v", c)
	}

	// The badge goes out as the Unity launcher entry signal.
	sig := make(chan *dbus.Signal, 4)
	srv.conn.AddMatchSignal(dbus.WithMatchInterface("com.canonical.Unity.LauncherEntry"))
	srv.conn.Signal(sig)
	if err := SetBadge(3); err != nil {
		t.Fatal(err)
	}
	select {
	case s := <-sig:
		props := s.Body[1].(map[string]dbus.Variant)
		if !strings.HasPrefix(s.Body[0].(string), "application://") || props["count"].Value() != int64(3) || props["count-visible"].Value() != true {
			t.Errorf("badge signal %v", s.Body)
		}
	case <-time.After(3 * time.Second):
		t.Error("no badge signal")
	}
}

func TestResponsesQueueUntilListen(t *testing.T) {
	mu.Lock()
	handler = nil
	mu.Unlock()
	respond(Response{ID: "early"})
	got := make(chan Response, 1)
	privateBus(t)
	startServer(t)
	Listen(nil, func(r Response) { got <- r })
	t.Cleanup(func() { mu.Lock(); handler = nil; mu.Unlock() })
	select {
	case r := <-got:
		if r.ID != "early" {
			t.Errorf("%+v", r)
		}
	case <-time.After(time.Second):
		t.Error("queued response not delivered")
	}
}
