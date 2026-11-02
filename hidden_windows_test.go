//go:build windows

package giode

import (
	"testing"
	"time"
)

// TestPostWhileHidden_Windows checks that calls queued with Post (as
// tray menu items are) run while the window is hidden and draws no
// frames, and that a hidden window comes back and quits.
func TestPostWhileHidden_Windows(t *testing.T) {
	ready := make(chan struct{})
	a := New(Config{Title: "hidden post test", OnReady: func() { close(ready) }})
	done := make(chan error, 1)
	// The window loop itself: Run exits the process when the window
	// closes.
	go func() { done <- a.runLoop(func() Element { return Text("hidden") }) }()
	select {
	case <-ready:
	case err := <-done:
		t.Skipf("no window on this machine: %v", err)
	case <-time.After(20 * time.Second):
		t.Skip("the window never drew a frame on this machine")
	}
	a.HideWindow()
	if a.WindowVisible() {
		t.Fatal("visible after HideWindow")
	}
	time.Sleep(300 * time.Millisecond)
	for i := 0; i < 3; i++ {
		ran := make(chan struct{})
		a.Post(func() { close(ran) })
		select {
		case <-ran:
		case <-time.After(3 * time.Second):
			t.Fatal("Post didn't run while the window was hidden")
		}
	}
	a.Tray().clicked() // the default click shows it
	if !a.WindowVisible() {
		t.Error("the tray click didn't show the hidden window")
	}
	a.HideWindow()
	a.Post(a.Quit) // a tray "Quit" item, from the hidden window
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Quit through Post didn't end the app while hidden")
	}
}
