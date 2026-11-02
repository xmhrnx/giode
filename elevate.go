package giode

import (
	"errors"
	"os"
	"strconv"

	"github.com/xmhrnx/giode/host"
)

// ErrDeclined is returned by RestartElevated when the user refuses the
// administrator prompt.
var ErrDeclined = errors.New("giode: the user declined")

// IsElevated reports whether the app runs with administrator rights:
// elevated by UAC on Windows, as root on Linux and macOS.
func IsElevated() bool { return isElevated() }

// RestartElevated starts a new copy of the app with administrator
// rights and quits this one, for work that needs them (raw disk
// access, system settings). The system asks the user first: UAC on
// Windows, pkexec (polkit) on Linux, an administrator password prompt
// on macOS. args are the new copy's arguments, os.Args[1:] when nil.
//
// It returns once the new copy is starting, or ErrDeclined when the
// user refuses, leaving this copy running. With a SingleInstanceLock
// the new copy waits for this one to quit and takes the lock over.
// It returns ErrNotSupported on Android, iOS and the web.
//
// Running a whole GUI as root is what some disk tools need, but it's a
// big hammer: prefer a small elevated helper when the work allows.
func (a *App) RestartElevated(args ...string) error {
	if args == nil {
		args = os.Args[1:]
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	args = append([]string{host.WaitArg + strconv.Itoa(os.Getpid())}, args...)
	if err := startElevated(a, exe, args); err != nil {
		return err
	}
	a.Quit()
	return nil
}
