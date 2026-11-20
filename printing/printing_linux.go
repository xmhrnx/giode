//go:build linux && !android

package printing

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"gioui.org/app"
	"github.com/godbus/dbus/v5"

	"github.com/xmhrnx/giode"
)

// Linux prints a PDF through the desktop portal's Print, which shows the
// desktop's print dialog (GNOME's, KDE's) and sends the job; other files,
// and PDFs without a portal, go to CUPS's lp, on the default printer.

const supported = true

// errNoPrinting says what's missing.
var errNoPrinting = errors.New("printing: no print portal and no lp command (CUPS)")

var lpCommand = "lp"

func printFile(ctx context.Context, a *giode.App, path string, opts Options) error {
	if isPDF(path) {
		err := portalPrint(ctx, a, path, opts)
		if err == nil || errors.Is(err, giode.ErrCanceled) || errors.Is(err, context.Canceled) {
			return err
		}
	}
	return lpPrint(ctx, path, opts)
}

func lpPrint(ctx context.Context, path string, opts Options) error {
	lp, err := exec.LookPath(lpCommand)
	if err != nil {
		return errNoPrinting
	}
	out, err := exec.CommandContext(ctx, lp, "-t", opts.JobName, "--", path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("printing: lp: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// portalRequest calls a portal method that answers with a Request
// object's Response signal, and returns its results.
func portalRequest(ctx context.Context, conn *dbus.Conn, method string, args ...any) (map[string]dbus.Variant, error) {
	var tok [12]byte
	rand.Read(tok[:])
	token := "giode" + hex.EncodeToString(tok[:])
	sender := strings.TrimPrefix(strings.ReplaceAll(conn.Names()[0], ".", "_"), ":")
	expected := dbus.ObjectPath("/org/freedesktop/portal/desktop/request/" + sender + "/" + token)
	if err := conn.AddMatchSignal(dbus.WithMatchObjectPath(expected), dbus.WithMatchInterface("org.freedesktop.portal.Request")); err != nil {
		return nil, err
	}
	signals := make(chan *dbus.Signal, 4)
	conn.Signal(signals)
	defer conn.RemoveSignal(signals)
	// The options are the last argument.
	opts := args[len(args)-1].(map[string]dbus.Variant)
	opts["handle_token"] = dbus.MakeVariant(token)
	var handle dbus.ObjectPath
	portal := conn.Object("org.freedesktop.portal.Desktop", "/org/freedesktop/portal/desktop")
	if err := portal.CallWithContext(ctx, method, 0, args...).Store(&handle); err != nil {
		return nil, err
	}
	if handle != expected {
		conn.AddMatchSignal(dbus.WithMatchObjectPath(handle), dbus.WithMatchInterface("org.freedesktop.portal.Request"))
	}
	for {
		select {
		case <-ctx.Done():
			conn.Object("org.freedesktop.portal.Desktop", handle).Call("org.freedesktop.portal.Request.Close", 0)
			return nil, ctx.Err()
		case sig, ok := <-signals:
			if !ok {
				return nil, errors.New("printing: the portal connection closed")
			}
			if sig.Path != handle || sig.Name != "org.freedesktop.portal.Request.Response" || len(sig.Body) < 2 {
				continue
			}
			if code, _ := sig.Body[0].(uint32); code != 0 {
				return nil, giode.ErrCanceled
			}
			results, _ := sig.Body[1].(map[string]dbus.Variant)
			return results, nil
		}
	}
}

// portalPrint shows the print dialog, then prints the PDF with the
// settings chosen.
func portalPrint(ctx context.Context, a *giode.App, path string, opts Options) error {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return err
	}
	defer conn.Close()
	parent := ""
	if a != nil {
		if v, ok := a.NativeView().(app.X11ViewEvent); ok && v.Window != 0 {
			parent = fmt.Sprintf("x11:%x", v.Window)
		}
	}
	prep, err := portalRequest(ctx, conn, "org.freedesktop.portal.Print.PreparePrint", parent, opts.JobName,
		map[string]dbus.Variant{}, map[string]dbus.Variant{}, map[string]dbus.Variant{"modal": dbus.MakeVariant(true)})
	if err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	options := map[string]dbus.Variant{}
	if t, ok := prep["token"]; ok {
		options["token"] = t
	}
	_, err = portalRequest(ctx, conn, "org.freedesktop.portal.Print.Print", parent, opts.JobName, dbus.UnixFD(f.Fd()), options)
	return err
}
