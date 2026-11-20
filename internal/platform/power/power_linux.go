//go:build linux && !android

package power

import (
	"os"

	"github.com/godbus/dbus/v5"
	"github.com/xmhrnx/giode/internal/platform"
)

// Sleep on Linux is systemd-logind's: its PrepareForSleep signal comes
// before (true) and after (false). A delay inhibitor lock, taken while
// awake, holds the sleep until OnSuspend has run.

const (
	logindName  = "org.freedesktop.login1"
	logindPath  = "/org/freedesktop/login1"
	logindIface = "org.freedesktop.login1.Manager"
)

func Watch(h platform.Host, done <-chan struct{}, event func(suspend bool), hold bool) {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		if h != nil {
			h.Logf(platform.Debug, "giode: no system bus for sleep notices: %v", err)
		}
		return
	}
	if err := conn.AddMatchSignal(dbus.WithMatchInterface(logindIface), dbus.WithMatchMember("PrepareForSleep"), dbus.WithMatchObjectPath(logindPath)); err != nil {
		conn.Close()
		return
	}
	signals := make(chan *dbus.Signal, 4)
	conn.Signal(signals)
	name := displayName(h)
	var lock *os.File
	if hold {
		lock = sleepDelay(conn, name)
	}
	go func() {
		defer conn.Close()
		for {
			select {
			case <-done:
				if lock != nil {
					lock.Close()
				}
				return
			case s, ok := <-signals:
				if !ok {
					return
				}
				if len(s.Body) < 1 {
					continue
				}
				start, _ := s.Body[0].(bool)
				event(start)
				if start {
					// Let the system sleep.
					if lock != nil {
						lock.Close()
						lock = nil
					}
				} else if hold && lock == nil {
					lock = sleepDelay(conn, name)
				}
			}
		}
	}()
}

// sleepDelay takes a delay inhibitor lock on sleep, or returns nil.
func sleepDelay(conn *dbus.Conn, name string) *os.File {
	var fd dbus.UnixFD
	err := conn.Object(logindName, logindPath).Call(logindIface+".Inhibit", 0,
		"sleep", name, "Saving its state before sleep", "delay").Store(&fd)
	if err != nil {
		return nil
	}
	return os.NewFile(uintptr(fd), "inhibit")
}

// displayName names the app to logind: its title, else its executable.
func displayName(h platform.Host) string {
	if h != nil && h.Title() != "" {
		return h.Title()
	}
	exe, _ := os.Executable()
	return exe
}
