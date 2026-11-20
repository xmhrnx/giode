//go:build linux && !android

package biometric

import (
	"context"
	"errors"

	"github.com/godbus/dbus/v5"

	"github.com/xmhrnx/giode"
)

// Linux reads fingerprints through fprintd on the system bus: the
// default reader, claimed for the current user, verifying against any
// enrolled finger. fprintd shows no prompt: the app shows its own ("Touch
// the sensor") while Authenticate waits. AllowPasscode has no effect.

const supported = true

const (
	fprintName    = "net.reactivated.Fprint"
	fprintManager = "net.reactivated.Fprint.Manager"
	fprintDevice  = "net.reactivated.Fprint.Device"
)

// defaultDevice returns the default reader, or an error without one.
func defaultDevice(conn *dbus.Conn) (dbus.BusObject, error) {
	var path dbus.ObjectPath
	err := conn.Object(fprintName, "/net/reactivated/Fprint/Manager").Call(fprintManager+".GetDefaultDevice", 0).Store(&path)
	if err != nil {
		return nil, err
	}
	return conn.Object(fprintName, path), nil
}

func available() Kind {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return None
	}
	defer conn.Close()
	dev, err := defaultDevice(conn)
	if err != nil {
		return None
	}
	var fingers []string
	if dev.Call(fprintDevice+".ListEnrolledFingers", 0, "").Store(&fingers) != nil || len(fingers) == 0 {
		return None
	}
	return Fingerprint
}

func authenticate(ctx context.Context, a *giode.App, opts Options) error {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return giode.ErrNotSupported
	}
	defer conn.Close()
	dev, err := defaultDevice(conn)
	if err != nil {
		return giode.ErrNotSupported
	}
	var fingers []string
	if err := dev.Call(fprintDevice+".ListEnrolledFingers", 0, "").Store(&fingers); err != nil || len(fingers) == 0 {
		return ErrNotEnrolled
	}
	if err := conn.AddMatchSignal(dbus.WithMatchInterface(fprintDevice), dbus.WithMatchMember("VerifyStatus"), dbus.WithMatchObjectPath(dev.Path())); err != nil {
		return err
	}
	signals := make(chan *dbus.Signal, 8)
	conn.Signal(signals)
	if err := dev.CallWithContext(ctx, fprintDevice+".Claim", 0, "").Err; err != nil {
		return err
	}
	defer dev.Call(fprintDevice+".Release", 0)
	if err := dev.CallWithContext(ctx, fprintDevice+".VerifyStart", 0, "any").Err; err != nil {
		return err
	}
	defer dev.Call(fprintDevice+".VerifyStop", 0)
	for {
		select {
		case <-ctx.Done():
			return ErrCanceled
		case s, ok := <-signals:
			if !ok {
				return ErrFailed
			}
			if s.Name != fprintDevice+".VerifyStatus" || len(s.Body) < 2 {
				continue
			}
			result, _ := s.Body[0].(string)
			done, _ := s.Body[1].(bool)
			switch result {
			case "verify-match":
				return nil
			case "verify-no-match":
				if done {
					return ErrFailed
				}
			case "verify-disconnected", "verify-unknown-error":
				return errors.New("biometric: the fingerprint reader failed (" + result + ")")
			}
			if done {
				return ErrFailed
			}
			// Retry messages (a short swipe, a finger not centred): the
			// reader keeps waiting.
		}
	}
}
