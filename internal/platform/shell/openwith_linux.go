//go:build linux && !android

package shell

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/godbus/dbus/v5"
	"github.com/xmhrnx/giode/internal/platform"
)

// OpenWith asks the XDG desktop portal's OpenURI to open the file, as
// Flatpak and Snap apps must and desktops otherwise do: its default
// app, or the portal's app chooser. Without a portal it runs xdg-open.
func OpenWith(ctx context.Context, h platform.Host, path, mime string) error {
	err := portalOpenFile(ctx, h, path)
	if err == nil || errors.Is(err, platform.ErrCanceled) || !errors.Is(err, errNoPortal) {
		return err
	}
	// The handler keeps running (or hands off and exits); either way
	// there's nothing to wait for.
	cmd := exec.Command("xdg-open", path)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("giode: no desktop portal and no xdg-open: %w", err)
	}
	go cmd.Wait()
	return nil
}

// errNoPortal is a portal that isn't there, or doesn't answer: try
// another way.
var errNoPortal = errors.New("giode: no desktop portal")

// portalOpenFile hands the portal the file, open for reading, through
// OpenURI.OpenFile, and waits for its answer: the app it started, or
// the user's choice in its app chooser.
func portalOpenFile(ctx context.Context, h platform.Host, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return fmt.Errorf("%w: %v", errNoPortal, err)
	}
	defer conn.Close()
	var tok [16]byte
	rand.Read(tok[:])
	token := "giode" + hex.EncodeToString(tok[:])
	sender := strings.TrimPrefix(strings.ReplaceAll(conn.Names()[0], ".", "_"), ":")
	expected := dbus.ObjectPath("/org/freedesktop/portal/desktop/request/" + sender + "/" + token)
	if err := conn.AddMatchSignal(dbus.WithMatchObjectPath(expected), dbus.WithMatchInterface("org.freedesktop.portal.Request")); err != nil {
		return fmt.Errorf("%w: %v", errNoPortal, err)
	}
	signals := make(chan *dbus.Signal, 4)
	conn.Signal(signals)

	parent := platform.PortalParent(h)
	options := map[string]dbus.Variant{
		"handle_token": dbus.MakeVariant(token),
		"writable":     dbus.MakeVariant(false),
	}
	var handle dbus.ObjectPath
	portal := conn.Object("org.freedesktop.portal.Desktop", "/org/freedesktop/portal/desktop")
	call := portal.CallWithContext(ctx, "org.freedesktop.portal.OpenURI.OpenFile", 0, parent, dbus.UnixFD(f.Fd()), options)
	if err := call.Store(&handle); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("%w: %v", errNoPortal, err)
	}
	if handle != expected {
		if err := conn.AddMatchSignal(dbus.WithMatchObjectPath(handle), dbus.WithMatchInterface("org.freedesktop.portal.Request")); err != nil {
			return err
		}
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case sig, ok := <-signals:
			if !ok {
				return errors.New("giode: the portal connection closed")
			}
			if sig.Path != handle || sig.Name != "org.freedesktop.portal.Request.Response" || len(sig.Body) < 1 {
				continue
			}
			switch code, _ := sig.Body[0].(uint32); code {
			case 0:
				return nil
			case 1:
				return platform.ErrCanceled
			default:
				return fmt.Errorf("giode: no app opened %s", path)
			}
		}
	}
}
