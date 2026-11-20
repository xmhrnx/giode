//go:build linux && !android

package giode

import (
	"bufio"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
)

type fakeLogind struct {
	inhibits chan string
	r        *os.File
}

func (l *fakeLogind) Inhibit(what, who, why, mode string) (dbus.UnixFD, *dbus.Error) {
	l.inhibits <- what + "/" + mode
	r, w, err := os.Pipe()
	if err != nil {
		return 0, dbus.MakeFailedError(err)
	}
	l.r = r
	return dbus.UnixFD(w.Fd()), nil
}

func TestSleepNotices(t *testing.T) {
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
	t.Setenv("DBUS_SYSTEM_BUS_ADDRESS", strings.TrimSpace(addr))

	conn, err := dbus.Connect(strings.TrimSpace(addr))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	l := &fakeLogind{inhibits: make(chan string, 4)}
	conn.Export(l, logindPath, logindIface)
	conn.RequestName(logindName, dbus.NameFlagDoNotQueue)

	a := &App{cfg: Config{OnSuspend: func() {}}}
	events := make(chan bool, 4)
	done := make(chan struct{})
	defer close(done)
	a.watchPower(done, func(s bool) { events <- s })
	select {
	case got := <-l.inhibits:
		if got != "sleep/delay" {
			t.Errorf("inhibit %q", got)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no delay lock")
	}
	emit := func(start bool) {
		if err := conn.Emit(logindPath, logindIface+".PrepareForSleep", start); err != nil {
			t.Fatal(err)
		}
	}
	emit(true)
	select {
	case s := <-events:
		if !s {
			t.Error("resume before suspend")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no suspend")
	}
	emit(false)
	select {
	case s := <-events:
		if s {
			t.Error("suspend again")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no resume")
	}
	// Awake again: a new delay lock.
	select {
	case <-l.inhibits:
	case <-time.After(3 * time.Second):
		t.Error("no new delay lock after resume")
	}
}
