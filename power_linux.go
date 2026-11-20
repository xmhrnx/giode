//go:build linux && !android

package giode

import (
	"os"

	"github.com/godbus/dbus/v5"
)

// Sleep on Linux is systemd-logind's: its PrepareForSleep signal comes
// before (true) and after (false). A delay inhibitor lock, taken while
// awake, holds the sleep until OnSuspend has run.

const (
	logindName  = "org.freedesktop.login1"
	logindPath  = "/org/freedesktop/login1"
	logindIface = "org.freedesktop.login1.Manager"
)

func (a *App) watchPower(done <-chan struct{}, event func(suspend bool)) {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		a.logf(LogLevelDebug, "giode: no system bus for sleep notices: %v", err)
		return
	}
	if err := conn.AddMatchSignal(dbus.WithMatchInterface(logindIface), dbus.WithMatchMember("PrepareForSleep"), dbus.WithMatchObjectPath(logindPath)); err != nil {
		conn.Close()
		return
	}
	signals := make(chan *dbus.Signal, 4)
	conn.Signal(signals)
	hold := a.cfg.OnSuspend != nil
	var lock *os.File
	if hold {
		lock = sleepDelay(conn)
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
					lock = sleepDelay(conn)
				}
			}
		}
	}()
}

// sleepDelay takes a delay inhibitor lock on sleep, or returns nil.
func sleepDelay(conn *dbus.Conn) *os.File {
	var fd dbus.UnixFD
	err := conn.Object(logindName, logindPath).Call(logindIface+".Inhibit", 0,
		"sleep", appDisplayName(), "Saving its state before sleep", "delay").Store(&fd)
	if err != nil {
		return nil
	}
	return os.NewFile(uintptr(fd), "inhibit")
}

func appDisplayName() string {
	if a := currentApp.Load(); a != nil && a.cfg.Title != "" {
		return a.cfg.Title
	}
	exe, _ := os.Executable()
	return exe
}
