// Package login registers the app to start when the user signs in.
package login

import (
	"os"
	"path/filepath"
)

// Arg is the argument a login registration starts the app with, so the
// app can tell it was started at login.
const Arg = "--giode-login"

// Entry is what a login registration is made from: the entry's file or
// key name, the name shown for it, the app's reverse-DNS ID (giode.toml's
// app.id, or ""), and the icon (PNG) shown where the system shows one.
type Entry struct {
	Name, Display, ID string
	Icon              []byte
}

// Command is the program a login registration runs and its arguments:
// the AppImage or Flatpak the app runs from when it does, else this
// executable.
func Command() ([]string, error) {
	if id := os.Getenv("FLATPAK_ID"); id != "" {
		return []string{"flatpak", "run", id, Arg}, nil
	}
	if img := os.Getenv("APPIMAGE"); img != "" {
		return []string{img, Arg}, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return []string{exe, Arg}, nil
}
