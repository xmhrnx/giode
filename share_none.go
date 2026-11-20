//go:build !android && !ios && !js && !windows && !(darwin && !ios)

package giode

// Linux has no system share sheet.
func (a *App) shareNative(title, text string, paths []string) error { return ErrNotSupported }
