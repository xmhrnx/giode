//go:build linux && !android

package picker

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"strings"

	"github.com/godbus/dbus/v5"
	"github.com/xmhrnx/giode/internal/platform"
)

// PickFolder asks the desktop portal, else zenity, qarma or kdialog.
func PickFolder(h platform.Host, title, dir string) (string, error) {
	path, err := portalFolder(h, title, dir)
	if err == nil || errors.Is(err, platform.ErrCanceled) {
		return path, err
	}
	// No portal (a bare window manager, no xdg-desktop-portal): the
	// dialog tools desktops ship.
	for _, tool := range []struct {
		name string
		args []string
	}{
		{"zenity", []string{"--file-selection", "--directory", "--title=" + title, "--filename=" + dirArg(dir)}},
		{"qarma", []string{"--file-selection", "--directory", "--title=" + title, "--filename=" + dirArg(dir)}},
		{"kdialog", []string{"--getexistingdirectory", dir, "--title", title}},
	} {
		if _, err := exec.LookPath(tool.name); err != nil {
			continue
		}
		out, err := exec.Command(tool.name, tool.args...).Output()
		var ee *exec.ExitError
		if errors.As(err, &ee) && ee.ExitCode() == 1 {
			return "", platform.ErrCanceled
		}
		if err != nil {
			return "", err
		}
		return strings.TrimRight(string(out), "\n"), nil
	}
	return "", fmt.Errorf("giode: no folder picker (%v), and none of zenity, qarma or kdialog is installed: %w", err, platform.ErrNotSupported)
}

// dirArg makes zenity open inside dir rather than select it.
func dirArg(dir string) string {
	if dir == "" || strings.HasSuffix(dir, "/") {
		return dir
	}
	return dir + "/"
}

// portalFolder asks the XDG desktop portal's FileChooser for a
// directory.
func portalFolder(h platform.Host, title, dir string) (string, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return "", err
	}
	defer conn.Close()
	var tok [16]byte
	rand.Read(tok[:])
	token := "giode" + hex.EncodeToString(tok[:])
	sender := strings.TrimPrefix(strings.ReplaceAll(conn.Names()[0], ".", "_"), ":")
	expected := dbus.ObjectPath("/org/freedesktop/portal/desktop/request/" + sender + "/" + token)
	if err := conn.AddMatchSignal(dbus.WithMatchObjectPath(expected), dbus.WithMatchInterface("org.freedesktop.portal.Request")); err != nil {
		return "", err
	}
	signals := make(chan *dbus.Signal, 4)
	conn.Signal(signals)

	parent := platform.PortalParent(h)
	if title == "" {
		title = "Choose Folder"
	}
	options := map[string]dbus.Variant{
		"handle_token": dbus.MakeVariant(token),
		"directory":    dbus.MakeVariant(true),
		"modal":        dbus.MakeVariant(true),
	}
	if dir != "" {
		options["current_folder"] = dbus.MakeVariant(append([]byte(dir), 0))
	}
	var handle dbus.ObjectPath
	portal := conn.Object("org.freedesktop.portal.Desktop", "/org/freedesktop/portal/desktop")
	if err := portal.Call("org.freedesktop.portal.FileChooser.OpenFile", 0, parent, title, options).Store(&handle); err != nil {
		return "", err
	}
	if handle != expected {
		if err := conn.AddMatchSignal(dbus.WithMatchObjectPath(handle), dbus.WithMatchInterface("org.freedesktop.portal.Request")); err != nil {
			return "", err
		}
	}
	for sig := range signals {
		if sig.Path != handle || sig.Name != "org.freedesktop.portal.Request.Response" || len(sig.Body) < 2 {
			continue
		}
		if code, _ := sig.Body[0].(uint32); code != 0 {
			return "", platform.ErrCanceled
		}
		results, _ := sig.Body[1].(map[string]dbus.Variant)
		uris, _ := results["uris"].Value().([]string)
		if len(uris) == 0 {
			return "", platform.ErrCanceled
		}
		u, err := url.Parse(uris[0])
		if err != nil {
			return "", err
		}
		return u.Path, nil
	}
	return "", errors.New("giode: the portal connection closed")
}
