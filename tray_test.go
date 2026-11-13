package giode

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
	"time"
)

// testPNG returns a small PNG.
func testPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for i := range img.Pix {
		img.Pix[i] = 0xff
	}
	img.Set(3, 3, color.NRGBA{A: 0xff})
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestTray_ItemHelpers(t *testing.T) {
	ran := false
	items := []TrayItem{
		TrayAction("Open", func() { ran = true }),
		TraySeparator(),
		TrayCheckbox("Mute", true, nil),
		TraySubmenu("More", TrayAction("About", nil)),
	}
	if items[0].Label != "Open" || items[0].OnClick == nil {
		t.Error("TrayAction")
	}
	items[0].OnClick()
	if !ran {
		t.Error("action didn't run")
	}
	if !items[1].Separator || !items[2].Checkable || !items[2].Checked || len(items[3].Items) != 1 {
		t.Errorf("helpers built %+v", items)
	}
}

// TestTray_ChangesDontWait checks that changing the tray never waits
// for the platform tray: on macOS it waits for the main thread, which
// waits for the UI goroutine during a frame, so a SetMenu from a menu
// callback deadlocked. Changes made meanwhile are applied after, the
// latest last.
func TestTray_ChangesDontWait(t *testing.T) {
	if !TraySupported() {
		t.Skip("no tray here")
	}
	release := make(chan struct{})
	applied := make(chan []TrayItem, 16)
	old := trayApplyFunc
	trayApplyFunc = func(_ *Tray, s trayState) {
		<-release // the main thread is busy
		applied <- s.items
	}
	defer func() { trayApplyFunc = old }()

	tr := &Tray{app: &App{}, started: true, ready: true}
	done := make(chan struct{})
	go func() {
		for _, label := range []string{"a", "b", "c", "d"} {
			tr.SetMenu(TrayAction(label, nil))
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("SetMenu waited for the platform tray")
	}

	close(release)
	var last []TrayItem
	n := 0
	for timeout := time.After(2 * time.Second); ; {
		select {
		case items := <-applied:
			last = items
			n++
			if items[0].Label == "d" {
				goto done
			}
		case <-timeout:
			t.Fatalf("the latest menu was never applied (last: %v)", last)
		}
	}
done:
	if n > 2 {
		t.Errorf("%d applies for changes made while one was applied, want them together", n)
	}
	tr.mu.Lock()
	for tr.applying {
		tr.mu.Unlock()
		time.Sleep(time.Millisecond)
		tr.mu.Lock()
	}
	tr.mu.Unlock()
	select {
	case items := <-applied:
		t.Errorf("applied again after the latest: %v", items)
	default:
	}
}
