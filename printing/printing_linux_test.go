//go:build linux && !android

package printing

import (
	"bufio"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/godbus/dbus/v5"
)

// fakePortal is the desktop portal's Print on a private bus.
type fakePortal struct {
	conn    *dbus.Conn
	cancel  atomic.Bool
	printed chan string
}

func (p *fakePortal) respond(sender dbus.Sender, opts map[string]dbus.Variant, results map[string]dbus.Variant) dbus.ObjectPath {
	token, _ := opts["handle_token"].Value().(string)
	s := strings.TrimPrefix(strings.ReplaceAll(string(sender), ".", "_"), ":")
	path := dbus.ObjectPath("/org/freedesktop/portal/desktop/request/" + s + "/" + token)
	code := uint32(0)
	if p.cancel.Load() {
		code = 1
	}
	go p.conn.Emit(path, "org.freedesktop.portal.Request.Response", code, results)
	return path
}

func (p *fakePortal) PreparePrint(sender dbus.Sender, parent, title string, settings, setup, opts map[string]dbus.Variant) (dbus.ObjectPath, *dbus.Error) {
	return p.respond(sender, opts, map[string]dbus.Variant{"token": dbus.MakeVariant(uint32(7))}), nil
}

func (p *fakePortal) Print(sender dbus.Sender, parent, title string, fd dbus.UnixFD, opts map[string]dbus.Variant) (dbus.ObjectPath, *dbus.Error) {
	f := os.NewFile(uintptr(fd), "print")
	b, _ := io.ReadAll(f)
	f.Close()
	tok, _ := opts["token"].Value().(uint32)
	p.printed <- title + "|" + string(b) + "|" + string(rune('0'+tok))
	return p.respond(sender, opts, map[string]dbus.Variant{}), nil
}

func TestLinuxPrint(t *testing.T) {
	if _, err := exec.LookPath("dbus-daemon"); err != nil {
		t.Skip("no dbus-daemon")
	}
	cmd := exec.Command("dbus-daemon", "--session", "--nofork", "--print-address=1")
	out, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		t.Skip(err)
	}
	defer func() { cmd.Process.Kill(); cmd.Wait() }()
	addr, _ := bufio.NewReader(out).ReadString('\n')
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", strings.TrimSpace(addr))
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	p := &fakePortal{conn: conn, printed: make(chan string, 2)}
	conn.Export(p, "/org/freedesktop/portal/desktop", "org.freedesktop.portal.Print")
	conn.RequestName("org.freedesktop.portal.Desktop", dbus.NameFlagDoNotQueue)

	dir := t.TempDir()
	pdf := filepath.Join(dir, "report.pdf")
	os.WriteFile(pdf, []byte("%PDF-1.4 test"), 0o644)
	if err := Print(context.Background(), nil, pdf, Options{}); err != nil {
		t.Fatal(err)
	}
	if got := <-p.printed; got != "report|%PDF-1.4 test|7" {
		t.Errorf("printed %q", got)
	}
	p.cancel.Store(true)
	if err := Print(context.Background(), nil, pdf, Options{}); err == nil {
		t.Error("canceled dialog printed")
	}

	// Other files go to lp.
	log := filepath.Join(dir, "lp.log")
	fake := filepath.Join(dir, "lp")
	os.WriteFile(fake, []byte("#!/bin/sh\necho \"$@\" > "+log+"\n"), 0o755)
	old := lpCommand
	lpCommand = fake
	defer func() { lpCommand = old }()
	png := filepath.Join(dir, "photo.png")
	os.WriteFile(png, []byte("png"), 0o644)
	if err := Print(context.Background(), nil, png, Options{JobName: "Photo"}); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(log); strings.TrimSpace(string(b)) != "-t Photo -- "+png {
		t.Errorf("lp got %q", b)
	}
}
