package giode

import (
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/xmhrnx/giode/internal/platform/login"
)

// AppID is the app's reverse-DNS ID from giode.toml (app.id): the giode
// CLI sets it when it builds a desktop app (with -ldflags -X), as it
// does AppVersion. Login registration names its entry after it; without
// it, after the window title.
var AppID string

// loginArg is the argument a login registration starts the app with, so
// the app can tell it was started at login. It's removed from os.Args
// before main runs.
const loginArg = login.Arg

// launchedAtLogin records that the process was started by a login
// registration.
var launchedAtLogin atomic.Bool

func init() {
	var login bool
	os.Args, login = stripLoginArg(os.Args)
	launchedAtLogin.Store(login)
}

// stripLoginArg removes loginArg from args, reporting whether it was
// there.
func stripLoginArg(args []string) ([]string, bool) {
	for i, a := range args {
		if i > 0 && a == loginArg {
			return append(args[:i:i], args[i+1:]...), true
		}
	}
	return args, false
}

// LaunchedAtLogin reports whether the system started the app when the
// user signed in, from a registration made with App.SetLaunchAtLogin -
// to start quietly, in the tray (see Config.StartInTray), rather than
// as if the user had opened it.
func LaunchedAtLogin() bool {
	return launchedAtLogin.Load() || login.ByStartupTask()
}

// StartInTray says when the app starts with its window hidden in the
// tray (Config.StartInTray).
type StartInTray int

const (
	// StartInTrayNever shows the window at start (the default).
	StartInTrayNever StartInTray = iota
	// StartInTrayAtLogin starts in the tray when the app was started at
	// login, and shows the window when the user opens it.
	StartInTrayAtLogin
	// StartInTrayAlways always starts in the tray.
	StartInTrayAlways
)

// startsInTray reports whether this launch starts hidden in the tray.
func (s StartInTray) startsInTray() bool {
	switch s {
	case StartInTrayAlways:
		return traySupported
	case StartInTrayAtLogin:
		return traySupported && LaunchedAtLogin()
	}
	return false
}

// SetLaunchAtLogin registers the app to start when the user signs in,
// or removes the registration: a Run key entry on Windows (or, in an
// MSIX package, its startup task: set windows.msix.startup_task in
// giode.toml), a LaunchAgent on macOS (a login item with SMAppService
// in the App Sandbox, as Mac App Store builds are: macOS 13 and later,
// ErrNotSupported before), an XDG autostart entry on Linux
// (running the AppImage or the Flatpak when the app is one). The app
// then starts with LaunchedAtLogin true. It returns ErrNotSupported on
// Android, iOS and the web.
//
// The user can also change it in the system's settings, so read
// LaunchAtLogin for the current state rather than remembering it.
func (a *App) SetLaunchAtLogin(on bool) error {
	return login.Set(login.Entry{Name: a.loginName(), Display: a.displayName(), ID: AppID, Icon: a.cfg.Icon}, on)
}

// LaunchAtLogin reports whether the app is registered to start when the
// user signs in (and, on Windows, the user hasn't turned it off).
func (a *App) LaunchAtLogin() bool {
	return login.Enabled(a.loginName())
}

// displayName is the name the system's startup apps list shows:
// app.name, else the window title, else the entry's name.
func (a *App) displayName() string {
	if AppName != "" {
		return AppName
	}
	if t := a.Title(); t != "" {
		return t
	}
	return a.loginName()
}

// loginName names the app's login entry: its ID, else its title.
func (a *App) loginName() string {
	name := AppID
	if name == "" {
		name = a.Title()
	}
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(os.Args[0]), filepath.Ext(os.Args[0]))
	}
	return strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == ':' || r < ' ' {
			return '-'
		}
		return r
	}, name)
}
