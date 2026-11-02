package giode

import (
	"net/url"
	"strings"
)

// handleLaunch handles a second instance's arguments in the running
// one.
func (a *App) handleLaunch(args []string) {
	if a.runContextAction(args) {
		return
	}
	// A link the system opened with the app (its URL scheme).
	if len(args) == 1 && strings.Contains(args[0], "://") && a.cfg.OnOpenURL != nil {
		if u, err := url.Parse(args[0]); err == nil && u.Scheme != "" {
			a.openedURL(u)
			a.showFromLaunch()
			return
		}
	}
	if l := a.cfg.SingleInstanceLock; l != nil && l.OnLaunchArgs != nil {
		a.Post(func() { l.OnLaunchArgs(args) })
	}
	a.showFromLaunch()
}

// showFromLaunch brings the window back for a second launch, directly
// rather than through the UI goroutine, so it works however the window
// was hidden.
func (a *App) showFromLaunch() {
	a.ShowWindow()
	go a.Raise()
}

func sanitizeID(id string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '-'
		}
	}, id)
}
