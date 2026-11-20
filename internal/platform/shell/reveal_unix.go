//go:build (linux && !android) || freebsd || openbsd || netbsd || dragonfly

package shell

import (
	"net/url"
	"os/exec"
	"path/filepath"

	"github.com/godbus/dbus/v5"
)

// Reveal asks the file manager to show path selected, through the
// freedesktop FileManager1 D-Bus interface, and falls back to opening
// its folder with xdg-open.
func Reveal(path string) error {
	if conn, err := dbus.SessionBus(); err == nil {
		uri := (&url.URL{Scheme: "file", Path: path}).String()
		call := conn.Object("org.freedesktop.FileManager1", "/org/freedesktop/FileManager1").
			Call("org.freedesktop.FileManager1.ShowItems", 0, []string{uri}, "")
		if call.Err == nil {
			return nil
		}
	}
	return exec.Command("xdg-open", filepath.Dir(path)).Start()
}
