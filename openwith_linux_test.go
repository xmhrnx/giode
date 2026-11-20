//go:build linux && !android

package giode

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/godbus/dbus/v5"
)

// fakeOpenURI is the portal's OpenURI: it reads the file it's handed
// and answers with code.
type fakeOpenURI struct {
	conn *dbus.Conn
	code atomic.Uint32
	got  chan string
}

func (p *fakeOpenURI) OpenFile(sender dbus.Sender, parent string, fd dbus.UnixFD, opts map[string]dbus.Variant) (dbus.ObjectPath, *dbus.Error) {
	f := os.NewFile(uintptr(fd), "handed")
	b, _ := io.ReadAll(f)
	f.Close()
	p.got <- string(b)
	token, _ := opts["handle_token"].Value().(string)
	s := strings.TrimPrefix(strings.ReplaceAll(string(sender), ".", "_"), ":")
	handle := dbus.ObjectPath("/org/freedesktop/portal/desktop/request/" + s + "/" + token)
	go p.conn.Emit(handle, "org.freedesktop.portal.Request.Response", p.code.Load(), map[string]dbus.Variant{})
	return handle, nil
}

// OpenWith hands the file to the portal's OpenURI as a descriptor and
// reports its answer. It needs a session bus (dbus-run-session go test).
func TestPortalOpenFile(t *testing.T) {
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" {
		t.Skip("no session bus")
	}
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		t.Skip(err)
	}
	defer conn.Close()
	if r, err := conn.RequestName("org.freedesktop.portal.Desktop", dbus.NameFlagDoNotQueue); err != nil || r != dbus.RequestNameReplyPrimaryOwner {
		t.Skipf("can't stand in for the portal: %v %v", r, err)
	}
	p := &fakeOpenURI{conn: conn, got: make(chan string, 1)}
	if err := conn.Export(p, "/org/freedesktop/portal/desktop", "org.freedesktop.portal.OpenURI"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "notes.txt")
	os.WriteFile(path, []byte("hello"), 0o600)
	a := &App{}
	if err := a.OpenWith(context.Background(), path); err != nil {
		t.Fatal(err)
	}
	if got := <-p.got; got != "hello" {
		t.Errorf("the portal read %q", got)
	}
	p.code.Store(1)
	if err := a.OpenWith(context.Background(), path); !errors.Is(err, ErrCanceled) {
		t.Errorf("dismissed chooser: %v", err)
	}
	<-p.got
	p.code.Store(2)
	if err := a.OpenWith(context.Background(), path); err == nil {
		t.Error("no app: no error")
	}
}
