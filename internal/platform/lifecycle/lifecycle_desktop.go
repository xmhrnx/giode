//go:build !android && !ios && !js

package lifecycle

// On desktops, minimizing the window counts as going to the background:
// the app follows the window state itself.
const FromWindowState = true

func Watch(background, foreground func()) error { return nil }
