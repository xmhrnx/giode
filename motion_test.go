package giode

import (
	"testing"
	"time"

	"gioui.org/app"

	"github.com/xmhrnx/giode/components/kit"
)

func TestWatchMotion(t *testing.T) {
	defer kit.SetReducedMotion(false)
	reduced := false
	defer func(f func() (bool, error)) { readReducedMotion = f }(readReducedMotion)
	readReducedMotion = func() (bool, error) { return reduced, nil }

	a := &App{window: new(app.Window), themeCheck: make(chan struct{}, 1), motionCheck: make(chan struct{}, 1)}
	done := make(chan struct{})
	defer close(done)
	a.watchMotion(done)
	if ReducedMotion() {
		t.Fatal("reduced at startup with the system's animations on")
	}
	reduced = true
	a.checkSystemTheme()
	deadline := time.Now().Add(2 * time.Second)
	for !ReducedMotion() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !ReducedMotion() {
		t.Fatal("the system's change wasn't followed")
	}
	a.SetMotion(MotionFull)
	if ReducedMotion() {
		t.Error("MotionFull left motion reduced")
	}
	a.SetMotion(MotionSystem)
	if !ReducedMotion() {
		t.Error("MotionSystem didn't go back to the system's setting")
	}
}
