package giode

import (
	"net/url"
	"runtime"
	"strings"
)

// openedURL hands a link that opened the app to OnOpenURL, on the UI
// goroutine.
func (a *App) openedURL(u *url.URL) {
	if isShareScheme(u.Scheme) {
		return // the share extension's; listenShares delivers the share
	}
	if a.cfg.OnOpenURL == nil {
		a.logf(LogLevelWarning, "giode: opened with %s but Config.OnOpenURL isn't set", u.Redacted())
		return
	}
	a.Post(func() { a.cfg.OnOpenURL(u) })
}

// startupURL hands a link the app was started with to OnOpenURL on
// Linux, where desktops pass a registered scheme's URL as the argument
// (Gio reports it itself on the other platforms).
func (a *App) startupURL(args []string) {
	if runtime.GOOS != "linux" || len(args) != 1 || !strings.Contains(args[0], "://") {
		return
	}
	if u, err := url.Parse(args[0]); err == nil && u.Scheme != "" {
		a.openedURL(u)
	}
}
