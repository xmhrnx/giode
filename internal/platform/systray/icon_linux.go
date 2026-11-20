//go:build linux && !android

package systray

import "fyne.io/systray"

// setIcon sets the tray icon from PNG bytes.
func setIcon(icon []byte, _ bool) {
	systray.SetIcon(icon)
}
