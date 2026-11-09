//go:build !darwin && !android && !ios && !js

package giode

// workspaceOpen is macOS's; the other desktops run a command.
func workspaceOpen(string, bool) error { return ErrNotSupported }
