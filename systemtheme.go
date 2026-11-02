package giode

import (
	"time"

	"github.com/xmhrnx/giode/styles/properties"
)

// SystemDarkMode reports whether the platform is set to a dark
// appearance: on Android, iOS, macOS, Windows, the web, and Linux
// desktops that implement the XDG appearance setting (GNOME, KDE).
// Elsewhere it returns ErrNotSupported.
func SystemDarkMode() (bool, error) {
	return readSystemDark()
}

// readSystemDark is systemDark, replaceable in tests.
var readSystemDark = systemDark

// themePoll is how often the system appearance is checked while the
// app runs. Platforms don't notify Gio apps of changes, but it's also
// checked whenever the window regains focus, which is when a change
// made in system settings usually becomes visible.
const themePoll = 3 * time.Second

// watchSystemTheme applies the system appearance at startup and
// whenever it changes, when FollowSystemTheme or OnSystemThemeChange
// asks for it.
func (a *App) watchSystemTheme(done <-chan struct{}) {
	if !a.cfg.FollowSystemTheme && a.cfg.OnSystemThemeChange == nil {
		return
	}
	dark, err := readSystemDark()
	if err != nil {
		a.logf(LogLevelDebug, "giode: can't read the system theme: %v", err)
		return
	}
	if a.cfg.FollowSystemTheme {
		properties.SetMode(modeFor(dark))
	}
	go func() {
		t := time.NewTicker(themePoll)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
			case <-a.themeCheck:
			}
			now, err := readSystemDark()
			if err != nil || now == dark {
				continue
			}
			dark = now
			a.Post(func() {
				if a.cfg.FollowSystemTheme {
					properties.SetMode(modeFor(now))
				}
				if a.cfg.OnSystemThemeChange != nil {
					a.cfg.OnSystemThemeChange(now)
				}
			})
		}
	}()
}

func modeFor(dark bool) properties.Mode {
	if dark {
		return properties.ModeDark
	}
	return properties.ModeLight
}

// checkSystemTheme asks the watcher to check now, without waiting.
func (a *App) checkSystemTheme() {
	select {
	case a.themeCheck <- struct{}{}:
	default:
	}
}
