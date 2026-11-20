//go:build linux && !android

package biometric

import (
	"bufio"
	"context"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
)

// fakeReader is fprintd with one reader, on a private system bus.
type fakeReader struct {
	conn    *dbus.Conn
	mu      sync.Mutex
	fingers []string
	result  string
	calls   chan string
}

func (r *fakeReader) set(fingers []string, result string) {
	r.mu.Lock()
	r.fingers, r.result = fingers, result
	r.mu.Unlock()
}

const devPath = dbus.ObjectPath("/net/reactivated/Fprint/Device/0")

func (r *fakeReader) GetDefaultDevice() (dbus.ObjectPath, *dbus.Error) { return devPath, nil }
func (r *fakeReader) ListEnrolledFingers(user string) ([]string, *dbus.Error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.fingers, nil
}
func (r *fakeReader) Claim(user string) *dbus.Error { r.calls <- "claim"; return nil }
func (r *fakeReader) Release() *dbus.Error          { r.calls <- "release"; return nil }
func (r *fakeReader) VerifyStop() *dbus.Error       { r.calls <- "stop"; return nil }
func (r *fakeReader) VerifyStart(finger string) *dbus.Error {
	r.calls <- "start " + finger
	r.mu.Lock()
	result := r.result
	r.mu.Unlock()
	go func() {
		time.Sleep(20 * time.Millisecond)
		r.conn.Emit(devPath, fprintDevice+".VerifyStatus", "verify-retry-scan", false)
		if result != "" {
			r.conn.Emit(devPath, fprintDevice+".VerifyStatus", result, true)
		}
	}()
	return nil
}

func TestFprintd(t *testing.T) {
	if _, err := exec.LookPath("dbus-daemon"); err != nil {
		t.Skip("no dbus-daemon")
	}
	cmd := exec.Command("dbus-daemon", "--session", "--nofork", "--print-address=1")
	out, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		t.Skip(err)
	}
	defer func() { cmd.Process.Kill(); cmd.Wait() }()
	addr, _ := bufio.NewReader(out).ReadString('\n')
	addr = strings.TrimSpace(addr)
	t.Setenv("DBUS_SYSTEM_BUS_ADDRESS", addr)
	conn, err := dbus.Connect(addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	r := &fakeReader{conn: conn, calls: make(chan string, 32)}
	conn.Export(r, "/net/reactivated/Fprint/Manager", fprintManager)
	conn.Export(r, devPath, fprintDevice)
	conn.RequestName(fprintName, dbus.NameFlagDoNotQueue)

	if k := available(); k != None {
		t.Errorf("no fingers enrolled: %v", k)
	}
	if err := authenticate(context.Background(), nil, Options{}); err != ErrNotEnrolled {
		t.Errorf("not enrolled: %v", err)
	}
	r.set([]string{"right-index-finger"}, "")
	if k := available(); k != Fingerprint {
		t.Errorf("enrolled: %v", k)
	}
	r.set([]string{"right-index-finger"}, "verify-match")
	if err := authenticate(context.Background(), nil, Options{}); err != nil {
		t.Errorf("match: %v", err)
	}
	r.set([]string{"right-index-finger"}, "verify-no-match")
	if err := authenticate(context.Background(), nil, Options{}); err != ErrFailed {
		t.Errorf("no match: %v", err)
	}
	// Canceled while waiting for a finger.
	r.set([]string{"right-index-finger"}, "")
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if err := authenticate(ctx, nil, Options{}); err != ErrCanceled {
		t.Errorf("canceled: %v", err)
	}
	// Every session released the reader.
	var claims, releases int
	for len(r.calls) > 0 {
		switch <-r.calls {
		case "claim":
			claims++
		case "release":
			releases++
		}
	}
	if claims != 3 || releases != 3 {
		t.Errorf("claims %d, releases %d", claims, releases)
	}
}
