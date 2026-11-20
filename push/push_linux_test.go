//go:build linux && !android

package push

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
)

// fakeDistributor is a UnifiedPush distributor on a private bus.
type fakeDistributor struct {
	conn    *dbus.Conn
	mu      sync.Mutex
	service string
	token   string
}

func (d *fakeDistributor) get() (string, string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.service, d.token
}

func (d *fakeDistributor) Register(args map[string]dbus.Variant) (map[string]dbus.Variant, *dbus.Error) {
	d.mu.Lock()
	d.service, _ = args["service"].Value().(string)
	d.token, _ = args["token"].Value().(string)
	d.mu.Unlock()
	go func() {
		// The endpoint comes after Register returns.
		time.Sleep(20 * time.Millisecond)
		service, token := d.get()
		d.conn.Object(service, connectorPath).Call(connectorIface2+".NewEndpoint", 0, map[string]dbus.Variant{
			"token": dbus.MakeVariant(token), "endpoint": dbus.MakeVariant("https://push.example/up/abc"),
		})
	}()
	return map[string]dbus.Variant{"success": dbus.MakeVariant(registerOK)}, nil
}

func (d *fakeDistributor) Unregister(args map[string]dbus.Variant) (map[string]dbus.Variant, *dbus.Error) {
	return map[string]dbus.Variant{}, nil
}

func (d *fakeDistributor) send(t *testing.T, body []byte, id string) map[string]dbus.Variant {
	var res map[string]dbus.Variant
	service, token := d.get()
	err := d.conn.Object(service, connectorPath).Call(connectorIface2+".Message", 0, map[string]dbus.Variant{
		"token": dbus.MakeVariant(token), "message": dbus.MakeVariant(body), "id": dbus.MakeVariant(id),
	}).Store(&res)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

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
}

func TestUnifiedPush(t *testing.T) {
	privateBus(t)
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	lin.Lock()
	lin.dataDir, lin.st, lin.conn = filepath.Join(dir, "state"), state{}, nil
	lin.Unlock()
	t.Cleanup(func() {
		lin.Lock()
		if lin.conn != nil {
			lin.conn.Close()
		}
		lin.dataDir, lin.st, lin.conn = "", state{}, nil
		lin.Unlock()
		started, queued, handlers = false, nil, Handlers{}
	})

	if _, err := Token(context.Background()); err != ErrNoDistributor {
		t.Fatalf("without a distributor: %v", err)
	}
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	d := &fakeDistributor{conn: conn}
	conn.Export(d, distributorPath, distributorIface+"2")
	conn.RequestName(distributorName+"fake", dbus.NameFlagDoNotQueue)

	msgs, toks := make(chan Message, 4), make(chan string, 4)
	Listen(nil, Handlers{Message: func(m Message) { msgs <- m }, Token: func(s string) { toks <- s }})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ep, err := Token(ctx)
	if err != nil || ep != "https://push.example/up/abc" {
		t.Fatal(ep, err)
	}
	select {
	case s := <-toks:
		if s != ep {
			t.Error(s)
		}
	case <-time.After(2 * time.Second):
		t.Error("no Token event")
	}
	if service, _ := d.get(); service != busName() {
		t.Errorf("registered %q, bus name %q", service, busName())
	}
	// The service file lets the distributor start the app.
	svc, err := os.ReadFile(filepath.Join(dir, "dbus-1", "services", busName()+".service"))
	if err != nil || !strings.Contains(string(svc), "Name="+busName()) {
		t.Errorf("service file: %q %v", svc, err)
	}

	// An encrypted message, as a Web Push server sends it.
	k, _ := Keys()
	body, _ := encryptWebPush(k, []byte(`{"title":"Ana","body":"Lunch?","data":{"chat":"42"}}`))
	if res := d.send(t, body, "m1"); res["id"].Value() != "m1" {
		t.Errorf("ack %v", res)
	}
	wait := func() Message {
		select {
		case m := <-msgs:
			return m
		case <-time.After(3 * time.Second):
			t.Fatal("no message")
		}
		return Message{}
	}
	if m := wait(); m.Title != "Ana" || m.Body != "Lunch?" || m.Data["chat"] != "42" || m.ID != "m1" {
		t.Errorf("%+v", m)
	}
	// Plain text and flat JSON come through as data.
	d.send(t, []byte("ping"), "m2")
	if m := wait(); m.Data["message"] != "ping" {
		t.Errorf("%+v", m)
	}
	d.send(t, []byte(`{"chat":"7","n":3}`), "m3")
	if m := wait(); m.Data["chat"] != "7" || m.Data["n"] != "3" {
		t.Errorf("%+v", m)
	}
	// Another app's token is ignored.
	d.mu.Lock()
	d.token = "someone-else"
	d.mu.Unlock()
	d.send(t, []byte("x"), "m4")
	select {
	case m := <-msgs:
		t.Errorf("got another app's message %+v", m)
	case <-time.After(200 * time.Millisecond):
	}

	// The state survives a restart: same token and keys.
	lin.Lock()
	tok := lin.st.Token
	lin.st = state{}
	lin.Unlock()
	k2, _ := Keys()
	lin.Lock()
	if lin.st.Token != tok || k2 != k {
		t.Error("state not kept")
	}
	lin.Unlock()
	if err := Unregister(); err != nil {
		t.Error(err)
	}
}
